/**
 * Authentication helpers for the tunnel agent.
 *
 * Builds the WebSocket connection options including:
 * - Authorization Bearer header
 * - X-Gateway-Id / X-Agent-Version headers
 * - Client certificate header and TLS options when certificate material is provided
 */

import type { ClientOptions } from 'ws';
import type { TunnelConfig } from './config';

/**
 * Build the `ws` ClientOptions for the TunnelBroker WebSocket connection,
 * incorporating auth headers and TLS certificates when certificate material is provided.
 *
 * `ClientOptions` extends `SecureContextOptions`, so `ca`, `cert`, and `key`
 * are flat properties — not nested under a `.tls` sub-object.
 */
export function buildWsOptions(cfg: TunnelConfig): ClientOptions {
  const headers: Record<string, string> = {
    Authorization: `Bearer ${cfg.token}`,
    'X-Gateway-Id': cfg.gatewayId,
    'X-Agent-Version': cfg.agentVersion,
    ...(cfg.clientCert ? { 'X-Client-Cert': encodeURIComponent(cfg.clientCert) } : {}),
  };

  return {
    headers,
    handshakeTimeout: 10_000,
    // CA cert for server verification (optional)
    ...(cfg.caCert ? { ca: cfg.caCert } : {}),
    // Client certificate + key for TLS client auth when both are provided.
    ...(cfg.clientCert && cfg.clientKey
      ? { cert: cfg.clientCert, key: cfg.clientKey }
      : {}),
  };
}
