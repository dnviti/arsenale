import { logger } from '../utils/logger';
import type { ISyncProvider, SyncProviderConfig, DiscoveredDevice } from './types';

const log = logger.child('sync:netbox');

interface NetBoxPaginatedResponse<T> {
  count: number;
  next: string | null;
  results: T[];
}

interface NetBoxIpAddress {
  address: string;
  family: { value: number };
}

interface NetBoxDevice {
  id: number;
  name: string;
  display: string;
  primary_ip4?: NetBoxIpAddress | null;
  primary_ip6?: NetBoxIpAddress | null;
  platform?: { slug: string; name: string } | null;
  site?: { slug: string; name: string } | null;
  rack?: { name: string } | null;
  location?: { name: string } | null;
  status?: { value: string } | null;
  description?: string;
  custom_fields?: Record<string, unknown>;
}

interface NetBoxVm {
  id: number;
  name: string;
  display: string;
  primary_ip4?: NetBoxIpAddress | null;
  primary_ip6?: NetBoxIpAddress | null;
  platform?: { slug: string; name: string } | null;
  site?: { slug: string; name: string } | null;
  cluster?: { name: string } | null;
  status?: { value: string } | null;
  description?: string;
  custom_fields?: Record<string, unknown>;
}

function stripCidr(address: string): string {
  return address.split('/')[0];
}

function resolveIp(device: { primary_ip4?: NetBoxIpAddress | null; primary_ip6?: NetBoxIpAddress | null }): string | null {
  if (device.primary_ip4?.address) return stripCidr(device.primary_ip4.address);
  if (device.primary_ip6?.address) return stripCidr(device.primary_ip6.address);
  return null;
}

export class NetBoxProvider implements ISyncProvider {
  readonly type = 'NETBOX';

  async testConnection(config: SyncProviderConfig): Promise<{ ok: boolean; error?: string }> {
    try {
      const url = new URL('/api/status/', config.url);
      const response = await fetch(url.toString(), {
        headers: {
          Authorization: `Token ${config.apiToken}`,
          Accept: 'application/json',
        },
        signal: AbortSignal.timeout(10_000),
      });

      if (!response.ok) {
        return { ok: false, error: `NetBox returned HTTP ${response.status}` };
      }

      return { ok: true };
    } catch (err) {
      return { ok: false, error: (err as Error).message };
    }
  }

  async discoverDevices(config: SyncProviderConfig): Promise<DiscoveredDevice[]> {
    const devices: DiscoveredDevice[] = [];

    // Fetch physical devices
    const physicalDevices = await this.fetchAllPages<NetBoxDevice>(
      config.url,
      '/api/dcim/devices/',
      config.apiToken,
      config.filters,
    );

    for (const dev of physicalDevices) {
      const ip = resolveIp(dev);
      if (!ip) {
        log.debug(`Skipping device "${dev.name}" (id=${dev.id}): no primary IP`);
        continue;
      }

      const protocol = this.resolveProtocol(dev.platform?.slug, config);
      const port = config.defaultPort[protocol] ?? this.defaultPortForProtocol(protocol);

      devices.push({
        externalId: `device:${dev.id}`,
        name: dev.name || dev.display,
        host: ip,
        port,
        protocol,
        siteName: dev.site?.name,
        rackName: dev.rack?.name ?? dev.location?.name,
        description: dev.description || undefined,
        metadata: {
          type: 'device',
          netboxId: dev.id,
          platform: dev.platform?.slug,
          status: dev.status?.value,
          customFields: dev.custom_fields,
        },
      });
    }

    // Fetch virtual machines
    const vms = await this.fetchAllPages<NetBoxVm>(
      config.url,
      '/api/virtualization/virtual-machines/',
      config.apiToken,
      config.filters,
    );

    for (const vm of vms) {
      const ip = resolveIp(vm);
      if (!ip) {
        log.debug(`Skipping VM "${vm.name}" (id=${vm.id}): no primary IP`);
        continue;
      }

      const protocol = this.resolveProtocol(vm.platform?.slug, config);
      const port = config.defaultPort[protocol] ?? this.defaultPortForProtocol(protocol);

      devices.push({
        externalId: `vm:${vm.id}`,
        name: vm.name || vm.display,
        host: ip,
        port,
        protocol,
        siteName: vm.site?.name,
        rackName: vm.cluster?.name,
        description: vm.description || undefined,
        metadata: {
          type: 'vm',
          netboxId: vm.id,
          platform: vm.platform?.slug,
          status: vm.status?.value,
          customFields: vm.custom_fields,
        },
      });
    }

    log.info(`Discovered ${devices.length} devices/VMs from NetBox`);
    return devices;
  }

  private resolveProtocol(
    platformSlug: string | undefined | null,
    config: SyncProviderConfig,
  ) {
    if (platformSlug && config.platformMapping[platformSlug]) {
      return config.platformMapping[platformSlug];
    }
    return config.defaultProtocol;
  }

  private defaultPortForProtocol(protocol: string): number {
    switch (protocol) {
      case 'SSH': return 22;
      case 'RDP': return 3389;
      case 'VNC': return 5900;
      default: return 22;
    }
  }

  private async fetchAllPages<T>(
    baseUrl: string,
    path: string,
    apiToken: string,
    filters: Record<string, string>,
  ): Promise<T[]> {
    const results: T[] = [];
    const params = new URLSearchParams({ limit: '100', ...filters });
    let url: string | null = new URL(`${path}?${params.toString()}`, baseUrl).toString();

    while (url) {
      log.debug(`Fetching: ${url}`);
      const response = await fetch(url, {
        headers: {
          Authorization: `Token ${apiToken}`,
          Accept: 'application/json',
        },
        signal: AbortSignal.timeout(30_000),
      });

      if (!response.ok) {
        throw new Error(`NetBox API error: ${response.status} ${response.statusText}`);
      }

      const data = (await response.json()) as NetBoxPaginatedResponse<T>;
      results.push(...data.results);
      url = data.next;
    }

    return results;
  }
}
