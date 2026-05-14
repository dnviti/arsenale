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
    expect(bundle.envContent).toContain('TUNNEL_SERVER_URL="https://arsenale.example.com"');
    expect(bundle.envContent).toContain('TUNNEL_TOKEN="tok-secret"');
    expect(bundle.envContent).toContain('TUNNEL_GATEWAY_ID="gateway-1"');
    expect(bundle.envContent).toContain('TUNNEL_LOCAL_PORT="2222"');
    expect(bundle.envContent).toContain('SSH_PORT="2222"');
    expect(bundle.dockerCompose).not.toContain('container_name:');
    expect(bundle.installCommands).toContain('umask 077');
    expect(bundle.installCommands).toContain("mkdir -p 'arsenale-gateway-gateway-1/certs'");
    expect(bundle.installCommands).toContain("cd 'arsenale-gateway-gateway-1'");
    expect(bundle.installCommands).toContain('chmod 600 tunnel.env');
    expect(bundle.installCommands).toContain('docker compose --env-file tunnel.env up -d');
    expect(bundle.installCommands).toContain('-----BEGIN CERTIFICATE-----');
    expect(bundle.installCommands).toContain('-----BEGIN PRIVATE KEY-----');
  });

  it('uses the SSH gateway runtime for SSH bastion bundles', () => {
    const bundle = buildTunnelInstallBundle({
      gateway: { ...gateway, type: 'SSH_BASTION', port: 2022 },
      tokenBundle: { ...tokenBundle, gatewayType: 'SSH_BASTION', tunnelLocalPort: 2022 },
      serverUrl: 'https://arsenale.example.com',
    });

    expect(bundle.serviceName).toBe('ssh-gateway');
    expect(bundle.gatewayImage).toBe('ghcr.io/dnviti/arsenale/ssh-gateway:stable');
    expect(bundle.dockerCompose).toContain('image: ghcr.io/dnviti/arsenale/ssh-gateway:stable');
    expect(bundle.dockerCompose).toContain('SSH_PORT: "${SSH_PORT:-2022}"');
    expect(bundle.envContent).toContain('TUNNEL_LOCAL_PORT="2022"');
    expect(bundle.envContent).toContain('SSH_PORT="2022"');
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

  it('points GUACD bundles at the configured tunnel listener', () => {
    const bundle = buildTunnelInstallBundle({
      gateway: { ...gateway, type: 'GUACD', port: 14822 },
      tokenBundle: { ...tokenBundle, gatewayType: 'GUACD', tunnelLocalPort: 14822 },
      serverUrl: 'https://arsenale.example.com',
    });

    expect(bundle.serviceName).toBe('guacd');
    expect(bundle.gatewayImage).toBe('ghcr.io/dnviti/arsenale/guacd:stable');
    expect(bundle.envContent).toContain('TUNNEL_LOCAL_PORT="14822"');
    expect(bundle.envContent).toContain('GUACD_PORT="14822"');
    expect(bundle.dockerCompose).toContain('image: ghcr.io/dnviti/arsenale/guacd:stable');
    expect(bundle.dockerCompose).toContain('GUACD_PORT: "${GUACD_PORT:-14822}"');
    expect(bundle.dockerCompose).toContain('GUACD_SSL: "${GUACD_SSL:-true}"');
    expect(bundle.dockerCompose).toContain('./certs/guacd-server-cert.pem:/certs/guacd-server-cert.pem:ro');
    expect(bundle.dockerCompose).toContain('./certs/guacd-server-key.pem:/certs/guacd-server-key.pem:ro');
    const topLevelVolumes = bundle.dockerCompose.split('\nvolumes:\n')[1] ?? '';
    expect(topLevelVolumes).toContain('  guacd-drive:');
    expect(topLevelVolumes).toContain('  guacd-recordings:');
    expect(topLevelVolumes).not.toContain('./certs/guacd-server-cert.pem');
    expect(topLevelVolumes).not.toContain('./certs/guacd-server-key.pem');
    expect(bundle.installCommands).toContain('openssl req -x509');
    expect(bundle.installCommands).toContain('chmod 600 ./certs/guacd-server-key.pem');
  });
});
