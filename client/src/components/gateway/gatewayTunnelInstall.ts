import type { GatewayData, TunnelTokenResponse } from '../../api/gateway.api';

export interface TunnelInstallBundle {
  envContent: string;
  dockerCompose: string;
  installCommands: string;
  gatewayImage: string;
  serviceName: string;
}

interface BuildTunnelInstallBundleOptions {
  gateway: Pick<GatewayData, 'id' | 'type' | 'host' | 'port'>;
  tokenBundle: TunnelTokenResponse;
  serverUrl: string;
}

interface GatewayRuntimeInstall {
  serviceName: string;
  image: string;
  localPort: number;
  extraEnvironment: string[];
  volumes: string[];
}

const certPath = './certs/tunnel-client-cert.pem';
const keyPath = './certs/tunnel-client-key.pem';
const containerCertPath = '/tunnel-certs/client-cert.pem';
const containerKeyPath = '/tunnel-certs/client-key.pem';

export function buildTunnelInstallBundle({
  gateway,
  tokenBundle,
  serverUrl,
}: BuildTunnelInstallBundleOptions): TunnelInstallBundle {
  const runtime = gatewayRuntimeInstall(gateway.type);
  const localHost = tokenBundle.tunnelLocalHost || '127.0.0.1';
  const localPort = runtime.localPort || tokenBundle.tunnelLocalPort || gateway.port;
  const envContent = [
    envLine('TUNNEL_SERVER_URL', trimServerUrl(serverUrl)),
    envLine('TUNNEL_TOKEN', tokenBundle.token),
    envLine('TUNNEL_GATEWAY_ID', tokenBundle.gatewayId || gateway.id),
    envLine('TUNNEL_LOCAL_HOST', localHost),
    envLine('TUNNEL_LOCAL_PORT', String(localPort)),
    envLine('TUNNEL_CLIENT_CERT_FILE', containerCertPath),
    envLine('TUNNEL_CLIENT_KEY_FILE', containerKeyPath),
  ].join('\n') + '\n';

  const dockerCompose = buildDockerCompose(runtime);
  const installCommands = buildInstallCommands(tokenBundle, envContent, dockerCompose);

  return {
    envContent,
    dockerCompose,
    installCommands,
    gatewayImage: runtime.image,
    serviceName: runtime.serviceName,
  };
}

function gatewayRuntimeInstall(type: GatewayData['type']): GatewayRuntimeInstall {
  switch (type) {
    case 'MANAGED_SSH':
      return {
        serviceName: 'ssh-gateway',
        image: 'ghcr.io/dnviti/arsenale/ssh-gateway:stable',
        localPort: 2222,
        extraEnvironment: ['SSH_PORT: "${SSH_PORT:-2222}"'],
        volumes: [],
      };
    case 'DB_PROXY':
      return {
        serviceName: 'db-proxy',
        image: 'ghcr.io/dnviti/arsenale/db-proxy:stable',
        localPort: 5432,
        extraEnvironment: ['DB_LISTEN_PORT: "${DB_LISTEN_PORT:-5432}"'],
        volumes: [],
      };
    case 'SSH_BASTION':
      return {
        serviceName: 'ssh-gateway',
        image: 'ghcr.io/dnviti/arsenale/ssh-gateway:stable',
        localPort: 2222,
        extraEnvironment: ['SSH_PORT: "${SSH_PORT:-2222}"'],
        volumes: [],
      };
    case 'GUACD':
    default:
      return {
        serviceName: 'guacd',
        image: 'ghcr.io/dnviti/arsenale/guacd:stable',
        localPort: 4822,
        extraEnvironment: ['GUACD_SSL: "${GUACD_SSL:-false}"'],
        volumes: ['guacd-drive:/guacd-drive', 'guacd-recordings:/recordings'],
      };
  }
}

function buildDockerCompose(runtime: GatewayRuntimeInstall): string {
  const environmentLines = [
    'TUNNEL_SERVER_URL: "${TUNNEL_SERVER_URL}"',
    'TUNNEL_TOKEN: "${TUNNEL_TOKEN}"',
    'TUNNEL_GATEWAY_ID: "${TUNNEL_GATEWAY_ID}"',
    'TUNNEL_LOCAL_HOST: "${TUNNEL_LOCAL_HOST}"',
    'TUNNEL_LOCAL_PORT: "${TUNNEL_LOCAL_PORT}"',
    'TUNNEL_CLIENT_CERT_FILE: "${TUNNEL_CLIENT_CERT_FILE}"',
    'TUNNEL_CLIENT_KEY_FILE: "${TUNNEL_CLIENT_KEY_FILE}"',
    ...runtime.extraEnvironment,
  ];
  const volumeLines = [
    `${certPath}:${containerCertPath}:ro`,
    `${keyPath}:${containerKeyPath}:ro`,
    ...runtime.volumes,
  ];

  const compose = [
    'services:',
    `  ${runtime.serviceName}:`,
    `    image: ${runtime.image}`,
    `    container_name: arsenale-${runtime.serviceName}`,
    '    restart: unless-stopped',
    '    env_file:',
    '      - tunnel.env',
    '    environment:',
    ...environmentLines.map((line) => `      ${line}`),
    '    volumes:',
    ...volumeLines.map((line) => `      - ${line}`),
  ];

  if (runtime.volumes.length > 0) {
    compose.push('', 'volumes:');
    for (const volume of runtime.volumes) {
      const volumeName = volume.split(':', 1)[0];
      compose.push(`  ${volumeName}:`);
    }
  }

  return compose.join('\n') + '\n';
}

function buildInstallCommands(
  tokenBundle: TunnelTokenResponse,
  envContent: string,
  dockerCompose: string,
): string {
  return [
    'umask 077',
    'mkdir -p arsenale-gateway/certs',
    'cd arsenale-gateway',
    `cat > ${certPath} <<'EOF'`,
    stringsTrimWithNewline(tokenBundle.tunnelClientCert),
    'EOF',
    `cat > ${keyPath} <<'EOF'`,
    stringsTrimWithNewline(tokenBundle.tunnelClientKey),
    'EOF',
    "chmod 600 ./certs/tunnel-client-*.pem",
    "cat > tunnel.env <<'EOF'",
    stringsTrimWithNewline(envContent),
    'EOF',
    'chmod 600 tunnel.env',
    "cat > docker-compose.yml <<'EOF'",
    stringsTrimWithNewline(dockerCompose),
    'EOF',
    'docker compose --env-file tunnel.env up -d',
  ].join('\n');
}

function envLine(key: string, value: string): string {
  return `${key}=${JSON.stringify(value.trim())}`;
}

function trimServerUrl(serverUrl: string): string {
  return serverUrl.trim().replace(/\/+$/, '');
}

function stringsTrimWithNewline(value: string): string {
  return value.trim() + '\n';
}
