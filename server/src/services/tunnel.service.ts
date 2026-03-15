/**
 * TunnelBroker — zero-trust WSS tunnel service.
 *
 * Manages a registry of connected tunnel agents (Map<gatewayId, TunnelConnection>),
 * multiplexes TCP streams over a binary-framed WebSocket, and provides an
 * openStream() API that returns a net.Duplex-compatible stream for transparent
 * use by SSH2 / guacamole-lite.
 *
 * Wire protocol (binary frames):
 *   4-byte header:
 *     byte 0 : message type  (OPEN=1, DATA=2, CLOSE=3, PING=4, PONG=5)
 *     byte 1 : flags         (reserved, set to 0)
 *     bytes 2-3 : streamId   (uint16 big-endian)
 *   followed by payload (variable length, 0 bytes for OPEN/CLOSE/PING/PONG)
 */

import crypto from 'crypto';
import net from 'net';
import { Duplex } from 'stream';
import type WebSocket from 'ws';
import prisma from '../lib/prisma';
import { encryptWithServerKey, hashToken } from './crypto.service';
import { logger } from '../utils/logger';
import * as auditService from './audit.service';

const log = logger.child('tunnel');

// ---------------------------------------------------------------------------
// Protocol constants
// ---------------------------------------------------------------------------

export const MsgType = {
  OPEN:      1,
  DATA:      2,
  CLOSE:     3,
  PING:      4,
  PONG:      5,
  HEARTBEAT: 6,
  CERT_RENEW: 7,
} as const;

export type MsgTypeValue = typeof MsgType[keyof typeof MsgType];

const HEADER_SIZE = 4;
const MAX_STREAM_ID = 0xffff;
const MAX_FRAME_SIZE = 1_048_576; // 1 MB
const HEARTBEAT_DB_INTERVAL_MS = 30_000; // Throttle heartbeat DB writes to once per 30s

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface PendingOpen {
  resolve: (stream: Duplex) => void;
  reject: (err: Error) => void;
  timer: ReturnType<typeof setTimeout>;
}

/** Per-heartbeat health metadata reported by the tunnel agent. */
export interface HeartbeatMetadata {
  /** Whether the agent considers the local service healthy */
  healthy: boolean;
  /** Agent-measured latency to the local service in ms (optional) */
  latencyMs?: number;
  /** Number of active streams at the agent side */
  activeStreams?: number;
}

/** Represents a single active tunnel WebSocket connection from a gateway agent. */
export interface TunnelConnection {
  gatewayId: string;
  ws: WebSocket;
  connectedAt: Date;
  clientVersion?: string;
  clientIp?: string;
  /** Map from streamId → Duplex stream for open multiplexed channels */
  streams: Map<number, Duplex>;
  /** Pending openStream() calls waiting for the remote OPEN acknowledgement */
  pendingOpens: Map<number, PendingOpen>;
  nextStreamId: number;
  /** Timestamp of last heartbeat persisted to DB (for throttling) */
  lastHeartbeatDbWrite: number;
  /** Timestamp of the last heartbeat received from the agent */
  lastHeartbeat?: Date;
  /** Round-trip latency derived from the last PING/PONG exchange (ms) */
  pingPongLatency?: number;
  /** Timestamp when the last PING was sent (used to compute RTT) */
  lastPingSentAt?: number;
  /** Cumulative bytes transferred across all streams */
  bytesTransferred: number;
  /** Health metadata from the most recent heartbeat frame */
  heartbeatMetadata?: HeartbeatMetadata;
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

/** Global registry: gatewayId → TunnelConnection */
const registry = new Map<string, TunnelConnection>();

export function getRegisteredTunnels(): string[] {
  return Array.from(registry.keys());
}

export function isTunnelConnected(gatewayId: string): boolean {
  const conn = registry.get(gatewayId);
  if (!conn) return false;
  // Check that the underlying WS is still open
  return conn.ws.readyState === 1 /* OPEN */;
}

/** Returns true if a live tunnel exists for the given gateway. Alias for isTunnelConnected. */
export function hasTunnel(gatewayId: string): boolean {
  return isTunnelConnected(gatewayId);
}

/** Returns a snapshot of tunnel connection metadata (without the raw WebSocket). */
export function getTunnelInfo(gatewayId: string): {
  connectedAt: Date;
  lastHeartbeat: Date | undefined;
  pingPongLatency: number | undefined;
  activeStreams: number;
  bytesTransferred: number;
  heartbeatMetadata: HeartbeatMetadata | undefined;
  clientVersion: string | undefined;
  clientIp: string | undefined;
} | null {
  const conn = registry.get(gatewayId);
  if (!conn || conn.ws.readyState !== 1 /* OPEN */) return null;
  return {
    connectedAt: conn.connectedAt,
    lastHeartbeat: conn.lastHeartbeat,
    pingPongLatency: conn.pingPongLatency,
    activeStreams: conn.streams.size,
    bytesTransferred: conn.bytesTransferred,
    heartbeatMetadata: conn.heartbeatMetadata,
    clientVersion: conn.clientVersion,
    clientIp: conn.clientIp,
  };
}

// ---------------------------------------------------------------------------
// Registration / deregistration
// ---------------------------------------------------------------------------

/**
 * Register a newly authenticated tunnel WebSocket for a specific gateway.
 * Called by the tunnel WebSocket handler after authentication succeeds.
 */
export function registerTunnel(
  gatewayId: string,
  ws: WebSocket,
  clientVersion?: string,
  clientIp?: string,
): TunnelConnection {
  // Evict any stale connection for the same gateway
  const existing = registry.get(gatewayId);
  if (existing) {
    log.warn(`[tunnel] Gateway ${gatewayId} reconnected — closing previous connection`);
    try { existing.ws.close(1001, 'replaced'); } catch { /* ignore */ }
    evictConnection(existing);
  }

  const conn: TunnelConnection = {
    gatewayId,
    ws,
    connectedAt: new Date(),
    clientVersion,
    clientIp,
    streams: new Map(),
    pendingOpens: new Map(),
    nextStreamId: 1,
    lastHeartbeatDbWrite: 0,
    bytesTransferred: 0,
  };

  registry.set(gatewayId, conn);
  attachFrameHandler(conn);
  log.info(`[tunnel] Gateway ${gatewayId} connected (ip=${clientIp ?? 'unknown'}, version=${clientVersion ?? 'unknown'})`);

  // Persist connection metadata
  prisma.gateway.update({
    where: { id: gatewayId },
    data: {
      tunnelConnectedAt: conn.connectedAt,
      tunnelLastHeartbeat: conn.connectedAt,
      tunnelClientVersion: clientVersion ?? null,
      tunnelClientIp: clientIp ?? null,
    },
  }).catch((err: unknown) => {
    log.warn(`[tunnel] Failed to persist connection state for gateway ${gatewayId}: ${(err as Error).message}`);
  });

  auditService.log({
    action: 'TUNNEL_CONNECT',
    targetType: 'Gateway',
    targetId: gatewayId,
    details: { clientVersion, clientIp },
    ipAddress: clientIp,
  });

  return conn;
}

export function deregisterTunnel(gatewayId: string): void {
  const conn = registry.get(gatewayId);
  if (!conn) return;

  evictConnection(conn);
  log.info(`[tunnel] Gateway ${gatewayId} disconnected`);

  prisma.gateway.update({
    where: { id: gatewayId },
    data: { tunnelConnectedAt: null, tunnelLastHeartbeat: null },
  }).catch((err: unknown) => {
    log.warn(`[tunnel] Failed to clear connection state for gateway ${gatewayId}: ${(err as Error).message}`);
  });

  auditService.log({
    action: 'TUNNEL_DISCONNECT',
    targetType: 'Gateway',
    targetId: gatewayId,
  });
}

function evictConnection(conn: TunnelConnection): void {
  registry.delete(conn.gatewayId);
  // Destroy all open streams
  for (const stream of conn.streams.values()) {
    if (!stream.destroyed) stream.destroy(new Error('tunnel closed'));
  }
  conn.streams.clear();
  // Reject all pending opens
  for (const pending of conn.pendingOpens.values()) {
    clearTimeout(pending.timer);
    pending.reject(new Error('tunnel closed'));
  }
  conn.pendingOpens.clear();
}

// ---------------------------------------------------------------------------
// openStream — public API for SSH2 / guacamole-lite
// ---------------------------------------------------------------------------

/**
 * Open a new multiplexed TCP stream through the tunnel to `host:port`.
 * Returns a net.Duplex-compatible stream once the remote agent acknowledges.
 *
 * @param gatewayId - The gateway that owns the tunnel.
 * @param host      - The target host (from the gateway's perspective).
 * @param port      - The target TCP port.
 * @param timeoutMs - How long to wait for the remote OPEN ack (default 10 s).
 */
export function openStream(
  gatewayId: string,
  host: string,
  port: number,
  timeoutMs = 10_000,
): Promise<Duplex> {
  // Validate host to prevent SSRF
  const BLOCKED_HOSTS = ['169.254.169.254', '0.0.0.0'];
  if (!host || BLOCKED_HOSTS.includes(host)) {
    return Promise.reject(new Error(`Blocked host: ${host}`));
  }
  // Validate port is a valid integer in TCP range
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    return Promise.reject(new Error(`Invalid port: ${port}`));
  }

  const conn = registry.get(gatewayId);
  if (!conn || conn.ws.readyState !== 1 /* OPEN */) {
    return Promise.reject(new Error(`No active tunnel for gateway ${gatewayId}`));
  }

  return new Promise<Duplex>((resolve, reject) => {
    // Allocate a stream ID, wrapping around if needed
    let streamId = conn.nextStreamId;
    let attempts = 0;
    while (conn.streams.has(streamId) || conn.pendingOpens.has(streamId)) {
      streamId = (streamId % MAX_STREAM_ID) + 1;
      if (++attempts > MAX_STREAM_ID) {
        reject(new Error('No available stream IDs'));
        return;
      }
    }
    conn.nextStreamId = (streamId % MAX_STREAM_ID) + 1;

    // Set up timeout
    const timer = setTimeout(() => {
      conn.pendingOpens.delete(streamId);
      reject(new Error(`openStream timeout for gateway ${gatewayId} → ${host}:${port}`));
    }, timeoutMs);

    conn.pendingOpens.set(streamId, { resolve, reject, timer });

    // Send OPEN frame: header + "host:port" as UTF-8 payload
    const payload = Buffer.from(`${host}:${port}`, 'utf8');
    const frame = buildFrame(MsgType.OPEN, streamId, payload);
    conn.ws.send(frame, (err) => {
      if (err) {
        clearTimeout(timer);
        conn.pendingOpens.delete(streamId);
        reject(err);
      }
    });
  });
}

// ---------------------------------------------------------------------------
// Frame handling
// ---------------------------------------------------------------------------

function buildFrame(type: MsgTypeValue, streamId: number, payload?: Buffer): Buffer {
  const body = payload ?? Buffer.alloc(0);
  const frame = Buffer.allocUnsafe(HEADER_SIZE + body.length);
  frame[0] = type;
  frame[1] = 0; // flags
  frame.writeUInt16BE(streamId, 2);
  body.copy(frame, HEADER_SIZE);
  return frame;
}

function attachFrameHandler(conn: TunnelConnection): void {
  conn.ws.on('message', (data: Buffer | ArrayBuffer | Buffer[]) => {
    const buf = Buffer.isBuffer(data)
      ? data
      : Array.isArray(data)
        ? Buffer.concat(data)
        : Buffer.from(data as ArrayBuffer);

    if (buf.length < HEADER_SIZE) {
      log.warn(`[tunnel] ${conn.gatewayId}: frame too short (${buf.length} bytes)`);
      return;
    }

    if (buf.length > MAX_FRAME_SIZE) {
      log.warn(`[tunnel] ${conn.gatewayId}: frame exceeds max size (${buf.length} bytes > ${MAX_FRAME_SIZE}), closing connection`);
      conn.ws.close(1009, 'frame too large');
      return;
    }

    const type = buf[0] as MsgTypeValue;
    const streamId = buf.readUInt16BE(2);
    const payload = buf.subarray(HEADER_SIZE);

    switch (type) {
      case MsgType.OPEN:
        handleOpenAck(conn, streamId);
        break;
      case MsgType.DATA:
        handleData(conn, streamId, payload);
        break;
      case MsgType.CLOSE:
        handleClose(conn, streamId);
        break;
      case MsgType.PING:
        handlePing(conn, streamId);
        break;
      case MsgType.PONG:
        handlePong(conn);
        break;
      case MsgType.HEARTBEAT:
        handleHeartbeat(conn, payload);
        break;
      default:
        log.warn(`[tunnel] ${conn.gatewayId}: unknown message type ${type}`);
    }
  });

  conn.ws.on('close', () => {
    deregisterTunnel(conn.gatewayId);
  });

  conn.ws.on('error', (err) => {
    log.error(`[tunnel] ${conn.gatewayId} WebSocket error: ${err.message}`);
    deregisterTunnel(conn.gatewayId);
  });
}

function handleOpenAck(conn: TunnelConnection, streamId: number): void {
  const pending = conn.pendingOpens.get(streamId);
  if (!pending) {
    log.warn(`[tunnel] ${conn.gatewayId}: unexpected OPEN ack for stream ${streamId}`);
    return;
  }
  clearTimeout(pending.timer);
  conn.pendingOpens.delete(streamId);

  const stream = createStream(conn, streamId);
  conn.streams.set(streamId, stream);
  pending.resolve(stream);
}

function handleData(conn: TunnelConnection, streamId: number, payload: Buffer): void {
  const stream = conn.streams.get(streamId);
  if (!stream) {
    log.warn(`[tunnel] ${conn.gatewayId}: DATA for unknown stream ${streamId}`);
    return;
  }
  conn.bytesTransferred += payload.length;
  if (!stream.push(payload)) {
    // Back-pressure: pause upstream until stream is drained
    // (the stream will resume via 'drain' event on the Duplex)
  }
}

function handleClose(conn: TunnelConnection, streamId: number): void {
  const stream = conn.streams.get(streamId);
  if (stream && !stream.destroyed) {
    stream.push(null); // signal EOF
    stream.destroy();
  }
  conn.streams.delete(streamId);
}

function handlePing(conn: TunnelConnection, streamId: number): void {
  // Respond with PONG and update heartbeat — do NOT set lastPingSentAt here
  // (lastPingSentAt should only be set when the server sends a PING)
  conn.lastHeartbeat = new Date();
  const frame = buildFrame(MsgType.PONG, streamId);
  conn.ws.send(frame, (err) => {
    if (err) log.warn(`[tunnel] ${conn.gatewayId}: failed to send PONG: ${err.message}`);
  });
}

function handlePong(conn: TunnelConnection): void {
  // Compute RTT if we recorded when the PING was sent
  if (conn.lastPingSentAt != null) {
    conn.pingPongLatency = Date.now() - conn.lastPingSentAt;
    conn.lastPingSentAt = undefined;
  }
  // Update heartbeat timestamp in memory
  const now = new Date();
  conn.lastHeartbeat = now;

  // Throttle heartbeat DB writes to avoid excessive queries
  const nowMs = Date.now();
  if (nowMs - conn.lastHeartbeatDbWrite < HEARTBEAT_DB_INTERVAL_MS) return;
  conn.lastHeartbeatDbWrite = nowMs;

  prisma.gateway.update({
    where: { id: conn.gatewayId },
    data: { tunnelLastHeartbeat: new Date(now) },
  }).catch(() => { /* best-effort */ });
}

const MAX_HEARTBEAT_PAYLOAD_BYTES = 4096; // 4 KB

function handleHeartbeat(conn: TunnelConnection, payload: Buffer): void {
  const now = new Date();
  conn.lastHeartbeat = now;

  // Reject oversized payloads to prevent abuse
  if (payload.length > MAX_HEARTBEAT_PAYLOAD_BYTES) {
    log.warn(`[tunnel] ${conn.gatewayId}: heartbeat payload too large (${payload.length} bytes) — ignored`);
    conn.heartbeatMetadata = { healthy: true };
    return;
  }

  // Parse optional JSON metadata from the heartbeat payload
  if (payload.length > 0) {
    try {
      const meta = JSON.parse(payload.toString('utf8')) as Partial<HeartbeatMetadata>;
      conn.heartbeatMetadata = {
        healthy: meta.healthy ?? true,
        latencyMs: meta.latencyMs,
        activeStreams: meta.activeStreams,
      };
    } catch {
      // Payload is not JSON — treat as a healthy heartbeat
      conn.heartbeatMetadata = { healthy: true };
    }
  } else {
    conn.heartbeatMetadata = { healthy: true };
  }

  // Persist heartbeat timestamp and update per-instance health if metadata was provided
  prisma.gateway.update({
    where: { id: conn.gatewayId },
    data: { tunnelLastHeartbeat: now },
  }).catch(() => { /* best-effort */ });

  // If the heartbeat carries per-instance health, update ManagedGatewayInstance records
  if (conn.heartbeatMetadata) {
    const healthStatus = conn.heartbeatMetadata.healthy ? 'healthy' : 'unhealthy';
    prisma.managedGatewayInstance.updateMany({
      where: { gatewayId: conn.gatewayId, status: 'RUNNING' },
      data: { healthStatus, lastHealthCheck: now },
    }).catch(() => { /* best-effort */ });
  }

  log.debug(`[tunnel] ${conn.gatewayId}: heartbeat received (healthy=${conn.heartbeatMetadata.healthy})`);
}

// ---------------------------------------------------------------------------
// Duplex stream factory
// ---------------------------------------------------------------------------

function createStream(conn: TunnelConnection, streamId: number): Duplex {
  const stream = new Duplex({
    read() {
      // Readable side is push-driven by handleData()
    },
    write(chunk: Buffer, _encoding, callback) {
      if (conn.ws.readyState !== 1 /* OPEN */) {
        callback(new Error('tunnel WebSocket is closed'));
        return;
      }
      conn.bytesTransferred += chunk.length;
      const frame = buildFrame(MsgType.DATA, streamId, chunk);
      conn.ws.send(frame, (err) => callback(err ?? null));
    },
    destroy(err, callback) {
      if (conn.streams.has(streamId)) {
        conn.streams.delete(streamId);
        const frame = buildFrame(MsgType.CLOSE, streamId);
        try { conn.ws.send(frame); } catch { /* ignore */ }
      }
      callback(err ?? null);
    },
  });

  stream.once('close', () => {
    conn.streams.delete(streamId);
  });

  return stream;
}

// ---------------------------------------------------------------------------
// Token / certificate management
// ---------------------------------------------------------------------------

/** Generate a 256-bit token, store it encrypted + hashed in the DB, return plain token. */
export async function generateTunnelToken(
  gatewayId: string,
  operatorUserId?: string,
): Promise<{ token: string; tunnelEnabled: boolean }> {
  const raw = crypto.randomBytes(32).toString('hex'); // 64 hex chars
  const hash = hashToken(raw);
  const enc = encryptWithServerKey(raw);

  await prisma.gateway.update({
    where: { id: gatewayId },
    data: {
      tunnelEnabled: true,
      encryptedTunnelToken: enc.ciphertext,
      tunnelTokenIV: enc.iv,
      tunnelTokenTag: enc.tag,
      tunnelTokenHash: hash,
    },
  });

  log.info(`[tunnel] Token generated for gateway ${gatewayId} by user ${operatorUserId ?? 'system'}`);

  auditService.log({
    userId: operatorUserId ?? null,
    action: 'TUNNEL_TOKEN_GENERATE',
    targetType: 'Gateway',
    targetId: gatewayId,
  });

  return { token: raw, tunnelEnabled: true };
}

/** Revoke (delete) the tunnel token for a gateway and disable tunnelling. */
export async function revokeTunnelToken(
  gatewayId: string,
  operatorUserId?: string,
): Promise<void> {
  await prisma.gateway.update({
    where: { id: gatewayId },
    data: {
      tunnelEnabled: false,
      encryptedTunnelToken: null,
      tunnelTokenIV: null,
      tunnelTokenTag: null,
      tunnelTokenHash: null,
    },
  });

  // Disconnect active tunnel if any
  deregisterTunnel(gatewayId);

  log.info(`[tunnel] Token revoked for gateway ${gatewayId} by user ${operatorUserId ?? 'system'}`);

  auditService.log({
    userId: operatorUserId ?? null,
    action: 'TUNNEL_TOKEN_ROTATE',
    targetType: 'Gateway',
    targetId: gatewayId,
    details: { revoked: true },
  });
}

/**
 * Authenticate an incoming tunnel WebSocket connection.
 * Returns the gateway record if authentication succeeds, null otherwise.
 *
 * The agent presents:
 *   Authorization: Bearer <token>
 *   X-Gateway-Id:  <uuid>
 *   X-Agent-Version: <version string>   (optional)
 */
export async function authenticateTunnelRequest(
  gatewayId: string,
  bearerToken: string,
): Promise<{ id: string; tenantId: string } | null> {
  if (!gatewayId || !bearerToken) return null;
  if (bearerToken.length > 128) return null; // Reject obviously oversized tokens early

  const gateway = await prisma.gateway.findUnique({
    where: { id: gatewayId },
    select: {
      id: true,
      tenantId: true,
      tunnelEnabled: true,
      encryptedTunnelToken: true,
      tunnelTokenIV: true,
      tunnelTokenTag: true,
      tunnelTokenHash: true,
    },
  });

  if (!gateway || !gateway.tunnelEnabled) return null;
  if (!gateway.tunnelTokenHash) return null;

  // Constant-time comparison against the stored hash
  const incomingHash = hashToken(bearerToken);
  const storedHashBuf = Buffer.from(gateway.tunnelTokenHash, 'hex');
  const incomingHashBuf = Buffer.from(incomingHash, 'hex');

  if (
    storedHashBuf.length !== incomingHashBuf.length ||
    !crypto.timingSafeEqual(storedHashBuf, incomingHashBuf)
  ) {
    return null;
  }

  return { id: gateway.id, tenantId: gateway.tenantId };
}

// ---------------------------------------------------------------------------
// TCP proxy — create a local TCP server that proxies to a gateway via tunnel
// ---------------------------------------------------------------------------

/**
 * Create a local TCP server that forwards every connection through the
 * zero-trust tunnel to `targetHost:targetPort` on the gateway side.
 *
 * Returns the local server and the assigned port.
 * Used to provide a `host:port` tuple to guacamole-lite for GUACD tunnels.
 */
export function createTcpProxy(
  gatewayId: string,
  targetHost: string,
  targetPort: number,
): Promise<{ server: net.Server; localPort: number }> {
  return new Promise((resolve, reject) => {
    const server = net.createServer(async (socket) => {
      try {
        const remote = await openStream(gatewayId, targetHost, targetPort);
        socket.pipe(remote);
        remote.pipe(socket);

        const cleanup = () => {
          socket.destroy();
          remote.destroy();
        };
        socket.once('close', cleanup);
        remote.once('close', cleanup);
        socket.once('error', cleanup);
        remote.once('error', cleanup);
      } catch (err) {
        log.error(`[tunnel] TCP proxy: failed to open stream for gateway ${gatewayId}: ${(err as Error).message}`);
        socket.destroy();
      }
    });

    server.listen(0, '127.0.0.1', () => {
      const addr = server.address();
      if (!addr || typeof addr === 'string') {
        server.close();
        reject(new Error('Failed to determine TCP proxy port'));
        return;
      }
      resolve({ server, localPort: addr.port });
    });

    server.on('error', reject);
  });
}

// ---------------------------------------------------------------------------
// Certificate rotation
// ---------------------------------------------------------------------------

const CERT_ROTATION_THRESHOLD_DAYS = 7;
const CERT_ROTATION_CHECK_INTERVAL_MS = 6 * 60 * 60 * 1000; // every 6 hours

/**
 * Send a CERT_RENEW response frame through the active tunnel for a gateway.
 * The payload is a JSON object containing the new PEM-encoded certificate.
 */
export function sendCertRenew(gatewayId: string, newClientCert: string): boolean {
  const conn = registry.get(gatewayId);
  if (!conn || conn.ws.readyState !== 1 /* OPEN */) return false;

  const payload = Buffer.from(JSON.stringify({ clientCert: newClientCert }), 'utf8');
  const frame = buildFrame(MsgType.CERT_RENEW, 0, payload);
  conn.ws.send(frame, (err) => {
    if (err) {
      log.warn(`[tunnel] ${gatewayId}: failed to send CERT_RENEW: ${err.message}`);
    } else {
      log.info(`[tunnel] ${gatewayId}: CERT_RENEW frame sent`);
    }
  });
  return true;
}

/**
 * Check all tunneled gateways for certificate expiry.
 * When a cert expires within CERT_ROTATION_THRESHOLD_DAYS, generate a new one
 * signed by the gateway's CA and send it via CERT_RENEW through the active tunnel.
 * For managed gateways, trigger a rolling restart via the orchestrator.
 */
export async function processCertRotations(): Promise<void> {
  const thresholdDate = new Date(Date.now() + CERT_ROTATION_THRESHOLD_DAYS * 24 * 60 * 60 * 1000);

  const candidates = await prisma.gateway.findMany({
    where: {
      tunnelEnabled: true,
      tunnelClientCertExp: { not: null, lte: thresholdDate },
    },
    select: {
      id: true,
      name: true,
      tenantId: true,
      isManaged: true,
      tunnelCaCert: true,
      tunnelCaKey: true,
      tunnelCaKeyIV: true,
      tunnelCaKeyTag: true,
      tunnelClientCertExp: true,
    },
  });

  if (candidates.length === 0) return;

  log.info(`[tunnel] cert-rotation: ${candidates.length} gateway(s) due for cert renewal`);

  for (const gw of candidates) {
    try {
      // Only rotate if the CA key is available for signing
      if (!gw.tunnelCaKey || !gw.tunnelCaKeyIV || !gw.tunnelCaKeyTag || !gw.tunnelCaCert) {
        log.warn(`[tunnel] cert-rotation: gateway ${gw.id} (${gw.name}) missing CA key — skipping`);
        continue;
      }

      // Decrypt the CA private key
      let caKeyPem: string;
      try {
        caKeyPem = decryptWithServerKey({
          ciphertext: gw.tunnelCaKey,
          iv: gw.tunnelCaKeyIV,
          tag: gw.tunnelCaKeyTag,
        });
      } catch (decryptErr) {
        log.error(`[tunnel] cert-rotation: failed to decrypt CA key for gateway ${gw.id}: ${(decryptErr as Error).message}`);
        continue;
      }

      // Generate a new client certificate valid for 90 days
      const validityDays = 90;
      let newClientCert: string;
      let expiry: Date;
      try {
        const result = generateClientCert(gw.tunnelCaCert, caKeyPem, validityDays);
        newClientCert = result.cert;
        expiry = result.expiry;
      } catch (genErr) {
        log.warn(`[tunnel] cert-rotation: skipping gateway ${gw.id} — ${(genErr as Error).message}`);
        continue;
      }

      // Persist only the new client cert and expiry — no need to re-encrypt
      // the CA key on every rotation cycle
      await prisma.gateway.update({
        where: { id: gw.id },
        data: {
          tunnelClientCert: newClientCert,
          tunnelClientCertExp: expiry,
        },
      });

      log.info(`[tunnel] cert-rotation: new cert generated for gateway ${gw.id} (expires ${expiry.toISOString()})`);

      auditService.log({
        action: 'TUNNEL_TOKEN_ROTATE',
        targetType: 'Gateway',
        targetId: gw.id,
        details: { certRotation: true, newExpiry: expiry.toISOString() },
      });

      // Try to deliver the new cert via the active tunnel
      const delivered = sendCertRenew(gw.id, newClientCert);
      log.info(`[tunnel] cert-rotation: CERT_RENEW for gateway ${gw.id} delivered=${delivered}`);

      // For managed gateways the mTLS handshake cert cannot be hot-swapped —
      // trigger a rolling restart so instances pick up the new cert from env.
      if (gw.isManaged) {
        try {
          const { rollingRestartForCertRotation } = await import('./managedGateway.service');
          await rollingRestartForCertRotation(gw.id, newClientCert);
          log.info(`[tunnel] cert-rotation: rolling restart triggered for managed gateway ${gw.id}`);
        } catch (restartErr) {
          log.warn(`[tunnel] cert-rotation: rolling restart failed for gateway ${gw.id}: ${(restartErr as Error).message}`);
        }
      }
    } catch (err) {
      log.error(`[tunnel] cert-rotation: failed for gateway ${gw.id}: ${(err as Error).message}`);
    }
  }
}

/**
 * Generate a CA-signed X.509 client certificate.
 *
 * TODO: This function is a stub. The previous implementation only generated an
 * RSA public key PEM, NOT a valid X.509 certificate, and the CA parameters
 * were unused. Proper implementation requires an X.509 signing library such as
 * `node-forge` or `@peculiar/x509`. Until that dependency is added, this
 * function throws to prevent silently distributing invalid certificates.
 */
function generateClientCert(
  _caCertPem: string,
  _caKeyPem: string,
  _validityDays: number,
): { cert: string; expiry: Date } {
  throw new Error('Client certificate generation not yet implemented — requires X.509 signing library');
}

let certRotationTimer: ReturnType<typeof setInterval> | null = null;

/** Start the certificate rotation background scheduler. */
export function startCertRotationScheduler(): void {
  if (certRotationTimer) return;
  certRotationTimer = setInterval(() => {
    processCertRotations().catch((err) => {
      log.error('[tunnel] cert-rotation scheduler error:', (err as Error).message);
    });
  }, CERT_ROTATION_CHECK_INTERVAL_MS);

  // Run once immediately at startup
  processCertRotations().catch((err) => {
    log.error('[tunnel] cert-rotation initial check error:', (err as Error).message);
  });

  log.info(`[tunnel] cert-rotation scheduler started (interval=${CERT_ROTATION_CHECK_INTERVAL_MS / 1000}s)`);
}

/** Stop the certificate rotation background scheduler. */
export function stopCertRotationScheduler(): void {
  if (certRotationTimer) {
    clearInterval(certRotationTimer);
    certRotationTimer = null;
    log.info('[tunnel] cert-rotation scheduler stopped');
  }
}
