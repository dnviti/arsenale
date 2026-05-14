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
  defaultLocalPort: number;
  listenerEnvKey: string;
  containerUid: number;
  containerGid: number;
  setupCommands: string[];
  extraEnvironment: string[];
  publicMountPaths: string[];
  privateMountPaths: string[];
  volumes: string[];
}

const certPath = './certs/tunnel-client-cert.pem';
const keyPath = './certs/tunnel-client-key.pem';
const guacdCertPath = './certs/guacd-server-cert.pem';
const guacdKeyPath = './certs/guacd-server-key.pem';
const containerCertPath = '/tunnel-certs/client-cert.pem';
const containerKeyPath = '/tunnel-certs/client-key.pem';
const containerGuacdCertPath = '/certs/guacd-server-cert.pem';
const containerGuacdKeyPath = '/certs/guacd-server-key.pem';

export function buildTunnelInstallBundle({
  gateway,
  tokenBundle,
  serverUrl,
}: BuildTunnelInstallBundleOptions): TunnelInstallBundle {
  const runtime = gatewayRuntimeInstall(gateway.type);
  const localHost = tokenBundle.tunnelLocalHost || '127.0.0.1';
  const localPort = tokenBundle.tunnelLocalPort || gateway.port || runtime.defaultLocalPort;
  const gatewayID = tokenBundle.gatewayId || gateway.id;
  const envContent = [
    envLine('TUNNEL_SERVER_URL', trimServerUrl(serverUrl)),
    envLine('TUNNEL_TOKEN', tokenBundle.token),
    envLine('TUNNEL_GATEWAY_ID', gatewayID),
    envLine('TUNNEL_LOCAL_HOST', localHost),
    envLine('TUNNEL_LOCAL_PORT', String(localPort)),
    envLine('TUNNEL_CLIENT_CERT_FILE', containerCertPath),
    envLine('TUNNEL_CLIENT_KEY_FILE', containerKeyPath),
    envLine(runtime.listenerEnvKey, String(localPort)),
  ].join('\n') + '\n';

  const dockerCompose = buildDockerCompose(runtime, localPort);
  const installCommands = buildInstallCommands(
    tokenBundle,
    envContent,
    dockerCompose,
    runtime,
    gatewayInstallDirectory(gatewayID, runtime.serviceName),
  );

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
        defaultLocalPort: 2222,
        listenerEnvKey: 'SSH_PORT',
        containerUid: 1000,
        containerGid: 1000,
        setupCommands: [],
        extraEnvironment: [],
        publicMountPaths: [],
        privateMountPaths: [],
        volumes: [],
      };
    case 'DB_PROXY':
      return {
        serviceName: 'db-proxy',
        image: 'ghcr.io/dnviti/arsenale/db-proxy:stable',
        defaultLocalPort: 5432,
        listenerEnvKey: 'DB_LISTEN_PORT',
        containerUid: 100,
        containerGid: 101,
        setupCommands: [],
        extraEnvironment: [],
        publicMountPaths: [],
        privateMountPaths: [],
        volumes: [],
      };
    case 'SSH_BASTION':
      return {
        serviceName: 'ssh-gateway',
        image: 'ghcr.io/dnviti/arsenale/ssh-gateway:stable',
        defaultLocalPort: 2222,
        listenerEnvKey: 'SSH_PORT',
        containerUid: 1000,
        containerGid: 1000,
        setupCommands: [],
        extraEnvironment: [],
        publicMountPaths: [],
        privateMountPaths: [],
        volumes: [],
      };
    case 'GUACD':
    default:
      return {
        serviceName: 'guacd',
        image: 'ghcr.io/dnviti/arsenale/guacd:stable',
        defaultLocalPort: 4822,
        listenerEnvKey: 'GUACD_PORT',
        containerUid: 100,
        containerGid: 101,
        setupCommands: [
          `if [ ! -f ${guacdCertPath} ] || [ ! -f ${guacdKeyPath} ]; then`,
          '  command -v openssl >/dev/null || { echo "openssl is required to generate GUACD TLS certificates" >&2; exit 1; }',
          `  openssl req -x509 -newkey rsa:3072 -nodes -days 365 -subj "/CN=arsenale-guacd" -keyout ${guacdKeyPath} -out ${guacdCertPath}`,
          'fi',
          `chmod 644 ${guacdCertPath}`,
          `chmod 600 ${guacdKeyPath}`,
        ],
        extraEnvironment: ['GUACD_SSL: "true"', `GUACD_SSL_CERT: ${containerGuacdCertPath}`, `GUACD_SSL_KEY: ${containerGuacdKeyPath}`],
        publicMountPaths: [guacdCertPath],
        privateMountPaths: [guacdKeyPath],
        volumes: [
          'guacd-drive:/guacd-drive',
          'guacd-recordings:/recordings',
          `${guacdCertPath}:${containerGuacdCertPath}:ro`,
          `${guacdKeyPath}:${containerGuacdKeyPath}:ro`,
        ],
      };
  }
}

function buildDockerCompose(runtime: GatewayRuntimeInstall, _localPort: number): string {
  const environmentLines = runtime.extraEnvironment;
  const volumeLines = [
    `${certPath}:${containerCertPath}:ro`,
    `${keyPath}:${containerKeyPath}:ro`,
    ...runtime.volumes,
  ];

  const compose = [
    'services:',
    `  ${runtime.serviceName}:`,
    `    image: ${runtime.image}`,
    '    pull_policy: always',
    '    user: "0:0"',
    '    restart: unless-stopped',
    '    env_file:',
    '      - tunnel.env',
  ];
  if (environmentLines.length > 0) {
    compose.push('    environment:', ...environmentLines.map((line) => `      ${line}`));
  }
  compose.push('    volumes:', ...volumeLines.map((line) => `      - ${line}`));

  const namedVolumes = collectNamedVolumeMounts(runtime.volumes);
  if (namedVolumes.length > 0) {
    compose.push('', 'volumes:');
    for (const volumeName of namedVolumes) {
      compose.push(`  ${volumeName}:`);
    }
  }

  return compose.join('\n') + '\n';
}

function collectNamedVolumeMounts(volumes: string[]): string[] {
  const namedVolumes = new Set<string>();
  for (const volume of volumes) {
    const source = volume.split(':', 1)[0]?.trim();
    if (!source || source.startsWith('.') || source.startsWith('/') || source.startsWith('~') || source.includes('$')) {
      continue;
    }
    namedVolumes.add(source);
  }
  return Array.from(namedVolumes);
}

function buildInstallCommands(
  tokenBundle: TunnelTokenResponse,
  envContent: string,
  dockerCompose: string,
  runtime: GatewayRuntimeInstall,
  installDirectory: string,
): string {
  const quotedInstallDirectory = shellQuote(installDirectory);
  const quotedCertsDirectory = shellQuote(`${installDirectory}/certs`);
  const publicMountPaths = [certPath, ...runtime.publicMountPaths];
  const privateMountPaths = [keyPath, ...runtime.privateMountPaths];
  return [
    'umask 077',
    `mkdir -p ${quotedCertsDirectory}`,
    `cd ${quotedInstallDirectory}`,
    ...restorePodmanRootlessOwnershipCommands([...publicMountPaths, ...privateMountPaths]),
    ...runtime.setupCommands,
    `cat > ${certPath} <<'EOF'`,
    stringsTrimWithNewline(tokenBundle.tunnelClientCert),
    'EOF',
    `cat > ${keyPath} <<'EOF'`,
    stringsTrimWithNewline(tokenBundle.tunnelClientKey),
    'EOF',
    `chmod 644 ${certPath}`,
    `chmod 600 ${keyPath}`,
    'if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then',
    '  compose_cmd="docker compose"',
    'elif command -v podman-compose >/dev/null 2>&1; then',
    '  compose_cmd="podman-compose"',
    'else',
    '  echo "docker compose or podman-compose is required" >&2',
    '  exit 1',
    'fi',
    ...podmanRootlessPrivateFileCommands(privateMountPaths, runtime),
    "cat > tunnel.env <<'EOF'",
    stringsTrimWithNewline(envContent),
    'EOF',
    'chmod 600 tunnel.env',
    "cat > docker-compose.yml <<'EOF'",
    stringsTrimWithNewline(dockerCompose),
    'EOF',
    '$compose_cmd --env-file tunnel.env up -d',
  ].join('\n');
}

function restorePodmanRootlessOwnershipCommands(paths: string[]): string[] {
  if (paths.length === 0) {
    return [];
  }
  return [
    'if command -v podman >/dev/null 2>&1; then',
    `  for path in ${paths.map(shellQuote).join(' ')}; do`,
    '    if [ -e "$path" ]; then',
    '      podman unshare chown 0:0 "$path" 2>/dev/null || true',
    '    fi',
    '  done',
    'fi',
  ];
}

function podmanRootlessPrivateFileCommands(paths: string[], runtime: GatewayRuntimeInstall): string[] {
  if (paths.length === 0) {
    return [];
  }
  return [
    'if [ "$compose_cmd" = "podman-compose" ] && command -v podman >/dev/null 2>&1; then',
    ...paths.flatMap((path) => [
      `  podman unshare chown ${runtime.containerUid}:${runtime.containerGid} ${path}`,
      `  podman unshare chmod 600 ${path}`,
    ]),
    'fi',
  ];
}

function gatewayInstallDirectory(gatewayID: string, serviceName: string): string {
  const raw = (gatewayID || serviceName || 'remote').trim().toLowerCase();
  const slug = raw
    .replace(/[^a-z0-9._-]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 80);
  return `arsenale-gateway-${slug || 'remote'}`;
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, "'\\''")}'`;
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
