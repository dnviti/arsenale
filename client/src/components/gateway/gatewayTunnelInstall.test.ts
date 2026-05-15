import { describe, expect, it } from 'vitest';
import type { GatewayData, TunnelTokenResponse } from '../../api/gateway.api';
import { buildTunnelInstallBundle } from './gatewayTunnelInstall';

const tokenBundle: TunnelTokenResponse = {
  token: 'tok-secret',
  tunnelEnabled: true,
  tunnelConnected: false,
  gatewayId: 'gateway-1',
  gatewayType: 'MANAGED_SSH',
  tunnelLocalHost: '127.0.0.1',
  tunnelLocalPort: 2222,
  tunnelClientCert: '-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----',
  tunnelClientKey: '-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----',
  tunnelClientCertExp: '2026-08-14T00:00:00.000Z',
};

const guacdServiceTLS = {
  tunnelServiceCert: '-----BEGIN CERTIFICATE-----\nguacd-cert\n-----END CERTIFICATE-----',
  tunnelServiceKey: '-----BEGIN PRIVATE KEY-----\nguacd-key\n-----END PRIVATE KEY-----',
  tunnelServiceCaCert: '-----BEGIN CERTIFICATE-----\nguacd-ca\n-----END CERTIFICATE-----',
  tunnelServiceCertExp: '2026-08-14T00:00:00.000Z',
};

const gateway: Pick<GatewayData, 'id' | 'type' | 'host' | 'port'> = {
  id: 'gateway-1',
  type: 'MANAGED_SSH',
  host: '',
  port: 2222,
};

describe('buildTunnelInstallBundle', () => {
  it('generates a remote gateway compose bundle with discovery credentials', () => {
    const bundle = buildTunnelInstallBundle({
      gateway,
      tokenBundle,
      serverUrl: 'https://arsenale.example.com/',
    });

    expect(bundle.gatewayImage).toBe('ghcr.io/dnviti/arsenale/ssh-gateway:stable');
    expect(bundle.dockerCompose).toContain('image: ghcr.io/dnviti/arsenale/ssh-gateway:stable');
    expect(bundle.dockerCompose).toContain('pull_policy: always');
    expect(bundle.dockerCompose).toContain('user: "0:0"');
    expect(bundle.envContent).toContain('TUNNEL_SERVER_URL="https://arsenale.example.com"');
    expect(bundle.envContent).toContain('TUNNEL_TOKEN="tok-secret"');
    expect(bundle.envContent).toContain('TUNNEL_GATEWAY_ID="gateway-1"');
    expect(bundle.envContent).toContain('TUNNEL_LOCAL_PORT="2222"');
    expect(bundle.envContent).toContain('SSH_PORT="2222"');
    expect(bundle.dockerCompose).toContain('env_file:');
    expect(bundle.dockerCompose).not.toContain('TUNNEL_SERVER_URL: "${TUNNEL_SERVER_URL}"');
    expect(bundle.dockerCompose).not.toContain('container_name:');
    expect(bundle.installCommands).toContain('umask 077');
    expect(bundle.installCommands).toContain("mkdir -p 'arsenale-gateway-gateway-1/certs'");
    expect(bundle.installCommands).toContain("cd 'arsenale-gateway-gateway-1'");
    expect(bundle.installCommands).toContain('chmod 600 tunnel.env');
    expect(bundle.installCommands).toContain('chmod 644 ./certs/tunnel-client-cert.pem');
    expect(bundle.installCommands).toContain('chmod 600 ./certs/tunnel-client-key.pem');
    expect(bundle.installCommands).not.toContain('chmod 644 ./certs/tunnel-client-*.pem');
    expect(bundle.installCommands).toContain('compose_cmd="docker compose"');
    expect(bundle.installCommands).toContain('compose_cmd="podman-compose"');
    expect(bundle.installCommands).toContain('podman unshare chown 0:0 "$path" 2>/dev/null || true');
    expect(bundle.installCommands).toContain('podman unshare chown 1000:1000 ./certs/tunnel-client-key.pem');
    expect(bundle.installCommands).toContain('$compose_cmd --env-file tunnel.env up -d');
    expect(bundle.installCommands).toContain('-----BEGIN CERTIFICATE-----');
    expect(bundle.installCommands).toContain('-----BEGIN PRIVATE KEY-----');
  });

  it('uses the SSH gateway runtime for SSH bastion bundles', () => {
    const bundle = buildTunnelInstallBundle({
      gateway: { ...gateway, type: 'SSH_BASTION', port: 2022 },
      tokenBundle: { ...tokenBundle, gatewayType: 'SSH_BASTION', tunnelLocalPort: 2222 },
      serverUrl: 'https://arsenale.example.com',
    });

    expect(bundle.serviceName).toBe('ssh-gateway');
    expect(bundle.gatewayImage).toBe('ghcr.io/dnviti/arsenale/ssh-gateway:stable');
    expect(bundle.dockerCompose).toContain('image: ghcr.io/dnviti/arsenale/ssh-gateway:stable');
    expect(bundle.dockerCompose).not.toContain('SSH_PORT: "${SSH_PORT:-2022}"');
    expect(bundle.envContent).toContain('TUNNEL_LOCAL_PORT="2222"');
    expect(bundle.envContent).toContain('SSH_PORT="2222"');
  });

  it('uses a gateway-specific install directory for multiple remote enrollments', () => {
    const bundle = buildTunnelInstallBundle({
      gateway: { ...gateway, id: 'tenant/gateway two', port: 2222 },
      tokenBundle: { ...tokenBundle, gatewayId: 'tenant/gateway two' },
      serverUrl: 'https://arsenale.example.com',
    });

    expect(bundle.envContent).toContain('TUNNEL_GATEWAY_ID="tenant/gateway two"');
    expect(bundle.installCommands).toContain("mkdir -p 'arsenale-gateway-tenant-gateway-two/certs'");
    expect(bundle.installCommands).toContain("cd 'arsenale-gateway-tenant-gateway-two'");
  });

  it('points GUACD bundles at the runtime-managed tunnel listener', () => {
    const bundle = buildTunnelInstallBundle({
      gateway: { ...gateway, type: 'GUACD', port: 14822 },
      tokenBundle: { ...tokenBundle, ...guacdServiceTLS, gatewayType: 'GUACD', tunnelLocalPort: 4822 },
      serverUrl: 'https://arsenale.example.com',
    });

    expect(bundle.serviceName).toBe('guacd');
    expect(bundle.gatewayImage).toBe('ghcr.io/dnviti/arsenale/guacd:stable');
    expect(bundle.envContent).toContain('TUNNEL_LOCAL_PORT="4822"');
    expect(bundle.envContent).toContain('GUACD_PORT="4822"');
    expect(bundle.dockerCompose).toContain('image: ghcr.io/dnviti/arsenale/guacd:stable');
    expect(bundle.dockerCompose).toContain('pull_policy: always');
    expect(bundle.dockerCompose).toContain('user: "0:0"');
    expect(bundle.dockerCompose).not.toContain('GUACD_PORT: "${GUACD_PORT:-14822}"');
    expect(bundle.dockerCompose).toContain('GUACD_SSL: "true"');
    expect(bundle.dockerCompose).toContain('./certs/guacd-server-cert.pem:/certs/guacd-server-cert.pem:ro');
    expect(bundle.dockerCompose).toContain('./certs/guacd-server-key.pem:/certs/guacd-server-key.pem:ro');
    const topLevelVolumes = bundle.dockerCompose.split('\nvolumes:\n')[1] ?? '';
    expect(topLevelVolumes).toContain('  guacd-drive:');
    expect(topLevelVolumes).toContain('  guacd-recordings:');
    expect(topLevelVolumes).not.toContain('./certs/guacd-server-cert.pem');
    expect(topLevelVolumes).not.toContain('./certs/guacd-server-key.pem');
    expect(bundle.installCommands).not.toContain('openssl req -x509');
    expect(bundle.installCommands).toContain('guacd-cert');
    expect(bundle.installCommands).toContain('guacd-key');
    expect(bundle.installCommands).toContain('guacd-ca');
    expect(bundle.installCommands).toContain('openssl verify -CAfile ./certs/guacd-ca.pem ./certs/guacd-server-cert.pem');
    expect(bundle.installCommands).toContain('chmod 600 ./certs/guacd-server-key.pem');
    expect(bundle.installCommands).toContain('podman unshare chown 100:101 ./certs/tunnel-client-key.pem');
    expect(bundle.installCommands).toContain('podman unshare chown 100:101 ./certs/guacd-server-key.pem');
    expect(bundle.installCommands).not.toContain('podman unshare chown 1000:1000 ./certs/tunnel-client-key.pem');
  });

  it('does not use the gateway direct port when the token omits a managed tunnel port', () => {
    const bundle = buildTunnelInstallBundle({
      gateway: { ...gateway, type: 'GUACD', port: 3389 },
      tokenBundle: { ...tokenBundle, ...guacdServiceTLS, gatewayType: 'GUACD', tunnelLocalPort: 0 },
      serverUrl: 'https://arsenale.example.com',
    });

    expect(bundle.envContent).toContain('TUNNEL_LOCAL_PORT="4822"');
    expect(bundle.envContent).toContain('GUACD_PORT="4822"');
  });

  it('uses the DB proxy runtime user for database tunnel bundles', () => {
    const bundle = buildTunnelInstallBundle({
      gateway: { ...gateway, type: 'DB_PROXY', port: 15432 },
      tokenBundle: { ...tokenBundle, gatewayType: 'DB_PROXY', tunnelLocalPort: 5432 },
      serverUrl: 'https://arsenale.example.com',
    });

    expect(bundle.serviceName).toBe('db-proxy');
    expect(bundle.gatewayImage).toBe('ghcr.io/dnviti/arsenale/db-proxy:stable');
    expect(bundle.dockerCompose).toContain('pull_policy: always');
    expect(bundle.dockerCompose).toContain('user: "0:0"');
    expect(bundle.envContent).toContain('DB_LISTEN_PORT="5432"');
    expect(bundle.installCommands).toContain('podman unshare chown 100:101 ./certs/tunnel-client-key.pem');
  });

  it('requires platform-issued service TLS for GUACD bundles', () => {
    expect(() =>
      buildTunnelInstallBundle({
        gateway: { ...gateway, type: 'GUACD', port: 3389 },
        tokenBundle: { ...tokenBundle, gatewayType: 'GUACD', tunnelLocalPort: 4822 },
        serverUrl: 'https://arsenale.example.com',
      }),
    ).toThrow('service TLS material');
  });
});
