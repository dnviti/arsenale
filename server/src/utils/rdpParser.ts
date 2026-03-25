export interface RdpConnection {
  fullAddress: string;
  hostname: string;
  port: number;
  username?: string;
}

export function parseRdpFile(content: string): RdpConnection {
  const properties = new Map<string, string>();

  const lines = content.split(/\r?\n/);

  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith(';')) {
      continue;
    }

    const match = trimmed.match(/^([^:]+):([^:]+):(.*)$/);
    if (match) {
      const [, key, _type, value] = match;
      properties.set(key.trim(), value.trim());
    }
  }

  const fullAddress = properties.get('full address') || properties.get('address') || '';
  const username = properties.get('username');

  const { hostname, port } = parseAddress(fullAddress);

  return {
    fullAddress,
    hostname,
    port,
    username,
  };
}

function parseAddress(address: string): { hostname: string; port: number } {
  if (!address) {
    return { hostname: '', port: 3389 };
  }

  const lastColon = address.lastIndexOf(':');
  if (lastColon !== -1) {
    const portStr = address.slice(lastColon + 1);
    const port = parseInt(portStr, 10);
    if (!isNaN(port) && port > 0 && port <= 65535) {
      return {
        hostname: address.slice(0, lastColon),
        port,
      };
    }
  }

  return {
    hostname: address,
    port: 3389,
  };
}
