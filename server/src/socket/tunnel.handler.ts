/**
 * Tunnel WebSocket handler.
 *
 * Attaches a raw `ws` WebSocket server to the HTTP server at the path
 * `/api/tunnel/connect`. Gateway agents connect here with:
 *
 *   GET /api/tunnel/connect
 *   Upgrade: websocket
 *   Authorization: Bearer <tunnel-token>
 *   X-Gateway-Id: <gateway-uuid>
 *   X-Agent-Version: <semver>  (optional)
 *
 * The handler authenticates the request, then delegates to the TunnelBroker
 * (tunnel.service.ts) for frame multiplexing.
 */

import crypto from 'crypto';
import http from 'http';
import type https from 'https';
import type tls from 'tls';
import { WebSocketServer } from 'ws';
import type { WebSocket } from 'ws';
import {
  authenticateTunnelRequest,
  registerTunnel,
} from '../services/tunnel.service';
import * as auditService from '../services/audit.service';
import { config } from '../config';
import { logger } from '../utils/logger';

const log = logger.child('tunnel-handler');

export function setupTunnelHandler(server: http.Server | https.Server): WebSocketServer {
  const wss = new WebSocketServer({ noServer: true });

  // Handle the HTTP upgrade
  server.on('upgrade', (req: http.IncomingMessage, socket: import('net').Socket, head: Buffer) => {
    if (req.url !== '/api/tunnel/connect') {
      return; // Let other upgrade handlers (e.g. Socket.IO) process this request
    }

    // Extract auth headers
    const authHeader = req.headers['authorization'] ?? '';
    const gatewayId  = (req.headers['x-gateway-id'] as string | undefined) ?? '';
    const agentVersion = (req.headers['x-agent-version'] as string | undefined) ?? undefined;

    const bearerToken = authHeader.startsWith('Bearer ')
      ? authHeader.slice('Bearer '.length).trim()
      : '';

    if (!bearerToken || !gatewayId) {
      log.warn('[tunnel] Upgrade rejected: missing Authorization or X-Gateway-Id header');
      socket.write('HTTP/1.1 401 Unauthorized\r\nConnection: close\r\n\r\n');
      socket.destroy();
      return;
    }

    // -----------------------------------------------------------------------
    // mTLS enforcement — verify client certificate identity
    // -----------------------------------------------------------------------
    let clientCertCn: string | undefined;
    let clientCertPem: string | undefined;

    const tlsSocket = req.socket as tls.TLSSocket;
    if (typeof tlsSocket.getPeerCertificate === 'function') {
      const peerCert = tlsSocket.getPeerCertificate(false);
      if (peerCert && peerCert.subject && peerCert.subject.CN) {
        // TLS-terminated connection with client cert
        const cnRaw = peerCert.subject.CN;
        const cn = Array.isArray(cnRaw) ? cnRaw[0] : cnRaw;
        if (!cn || !timingSafeStringEqual(cn, gatewayId)) {
          log.warn(`[tunnel] Upgrade rejected: client cert CN does not match X-Gateway-Id "${gatewayId}"`);
          auditService.log({ action: 'TUNNEL_MTLS_REJECTED', targetType: 'Gateway', targetId: gatewayId, details: { reason: 'cn_mismatch' }, ipAddress: req.socket.remoteAddress });
          socket.write('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\nClient certificate CN mismatch');
          socket.destroy();
          return;
        }
        // Check cert expiry
        if (peerCert.valid_to) {
          const expiryDate = new Date(peerCert.valid_to);
          if (expiryDate.getTime() < Date.now()) {
            log.warn(`[tunnel] Upgrade rejected: client cert for gateway ${gatewayId} has expired`);
            auditService.log({ action: 'TUNNEL_MTLS_REJECTED', targetType: 'Gateway', targetId: gatewayId, details: { reason: 'cert_expired', expiry: peerCert.valid_to }, ipAddress: req.socket.remoteAddress });
            socket.write('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\nClient certificate expired');
            socket.destroy();
            return;
          }
        }
        clientCertCn = cn;
        // Convert raw DER to PEM for CA chain validation downstream
        if (peerCert.raw) {
          const b64 = peerCert.raw.toString('base64');
          clientCertPem = `-----BEGIN CERTIFICATE-----\n${b64.match(/.{1,64}/g)!.join('\n')}\n-----END CERTIFICATE-----`;
        }
      }
    }

    // If no TLS cert, check proxy-forwarded client cert headers (only when proxy is trusted)
    if (!clientCertCn && config.trustProxy) {
      // Verify the request originates from a trusted proxy IP
      const sourceIp = req.socket.remoteAddress ?? '';
      const trustedIps = config.tunnelTrustedProxyIps;
      if (trustedIps.length > 0 && !trustedIps.includes(sourceIp)) {
        log.warn(`[tunnel] Upgrade rejected: proxy IP ${sourceIp} not in trusted proxy list for gateway ${gatewayId}`);
        auditService.log({ action: 'TUNNEL_MTLS_REJECTED', targetType: 'Gateway', targetId: gatewayId, details: { reason: 'untrusted_proxy_ip', sourceIp }, ipAddress: sourceIp });
        socket.write('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\nUntrusted proxy IP');
        socket.destroy();
        return;
      }

      const proxyCnRaw = req.headers['x-client-cert-cn'];
      const proxyCn = Array.isArray(proxyCnRaw) ? proxyCnRaw[0] : proxyCnRaw;
      const proxyVerifiedRaw = req.headers['x-client-cert-verified'];
      const proxyVerified = Array.isArray(proxyVerifiedRaw) ? proxyVerifiedRaw[0] : proxyVerifiedRaw;

      // Require the full client cert PEM for CA chain validation
      const proxyClientCertRaw = req.headers['x-client-cert'];
      const proxyClientCert = Array.isArray(proxyClientCertRaw) ? proxyClientCertRaw[0] : proxyClientCertRaw;

      if (proxyCn && proxyVerified === 'SUCCESS') {
        if (!timingSafeStringEqual(proxyCn, gatewayId)) {
          log.warn(`[tunnel] Upgrade rejected: proxy cert CN does not match X-Gateway-Id "${gatewayId}"`);
          auditService.log({ action: 'TUNNEL_MTLS_REJECTED', targetType: 'Gateway', targetId: gatewayId, details: { reason: 'proxy_cn_mismatch' }, ipAddress: sourceIp });
          socket.write('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\nClient certificate CN mismatch');
          socket.destroy();
          return;
        }
        clientCertCn = proxyCn;
        // Decode URL-encoded PEM from the proxy header
        if (proxyClientCert) {
          try {
            clientCertPem = decodeURIComponent(proxyClientCert);
          } catch {
            log.warn(`[tunnel] Upgrade rejected: malformed x-client-cert header for gateway ${gatewayId}`);
            auditService.log({ action: 'TUNNEL_MTLS_REJECTED', targetType: 'Gateway', targetId: gatewayId, details: { reason: 'malformed_proxy_cert' }, ipAddress: sourceIp });
            socket.write('HTTP/1.1 400 Bad Request\r\nConnection: close\r\n\r\nMalformed client certificate header');
            socket.destroy();
            return;
          }
        }
      }
    }

    // No client certificate at all — reject
    if (!clientCertCn) {
      log.warn(`[tunnel] Upgrade rejected: no client certificate for gateway ${gatewayId}`);
      socket.write('HTTP/1.1 401 Unauthorized\r\nConnection: close\r\n\r\nClient certificate required');
      socket.destroy();
      return;
    }

    // Authenticate asynchronously before completing the upgrade
    authenticateTunnelRequest(gatewayId, bearerToken, clientCertCn, clientCertPem)
      .then((result) => {
        if (!result) {
          log.warn(`[tunnel] Upgrade rejected: invalid credentials for gateway ${gatewayId}`);
          socket.write('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\n');
          socket.destroy();
          return;
        }

        // Complete WebSocket upgrade
        wss.handleUpgrade(req, socket, head, (ws: WebSocket) => {
          wss.emit('connection', ws, req, result.id, agentVersion);
        });
      })
      .catch((err: Error) => {
        log.error(`[tunnel] Auth error for gateway ${gatewayId}: ${err.message}`);
        socket.write('HTTP/1.1 500 Internal Server Error\r\nConnection: close\r\n\r\n');
        socket.destroy();
      });
  });

  // Handle authenticated connections
  wss.on('connection', (ws: WebSocket, _req: http.IncomingMessage, gatewayId: string, agentVersion?: string) => {
    const clientIp = extractRemoteIp(_req);
    registerTunnel(gatewayId, ws, agentVersion, clientIp);
    // close/error handlers are attached in attachFrameHandler (tunnel.service.ts)
  });

  log.info('[tunnel] WebSocket handler attached at /api/tunnel/connect');
  return wss;
}

function extractRemoteIp(req: http.IncomingMessage): string | undefined {
  const socketAddr = req.socket.remoteAddress ?? undefined;
  const forwarded = req.headers['x-forwarded-for'];
  if (forwarded) {
    const first = Array.isArray(forwarded) ? forwarded[0] : forwarded.split(',')[0];
    const forwardedIp = first?.trim();
    if (forwardedIp && forwardedIp !== socketAddr) {
      log.info(`[tunnel] Remote IP ${socketAddr ?? 'unknown'}, X-Forwarded-For: ${forwardedIp}`);
    }
  }
  return socketAddr;
}

/** Constant-time string comparison to prevent timing side-channels on CN values. */
function timingSafeStringEqual(a: string, b: string): boolean {
  const aBuf = Buffer.from(a, 'utf8');
  const bBuf = Buffer.from(b, 'utf8');
  if (aBuf.length !== bBuf.length) return false;
  return crypto.timingSafeEqual(aBuf, bBuf);
}
