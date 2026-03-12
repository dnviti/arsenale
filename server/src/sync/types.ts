import type { ConnectionType } from '../lib/prisma';

export interface SyncProviderConfig {
  url: string;
  apiToken: string;
  filters: Record<string, string>;
  platformMapping: Record<string, ConnectionType>;
  defaultProtocol: ConnectionType;
  defaultPort: Partial<Record<ConnectionType, number>>;
}

export interface DiscoveredDevice {
  externalId: string;
  name: string;
  host: string;
  port: number;
  protocol: ConnectionType;
  siteName?: string;
  rackName?: string;
  description?: string;
  metadata: Record<string, unknown>;
}

export interface ISyncProvider {
  readonly type: string;
  testConnection(config: SyncProviderConfig): Promise<{ ok: boolean; error?: string }>;
  discoverDevices(config: SyncProviderConfig): Promise<DiscoveredDevice[]>;
}

export interface SyncPlanEntry {
  device: DiscoveredDevice;
  connectionId?: string;
  changes?: string[];
  reason?: string;
  error?: string;
}

export interface SyncPlan {
  toCreate: DiscoveredDevice[];
  toUpdate: Array<{ device: DiscoveredDevice; connectionId: string; changes: string[] }>;
  toSkip: Array<{ device: DiscoveredDevice; reason: string }>;
  errors: Array<{ device: DiscoveredDevice; error: string }>;
}

export interface SyncResult {
  created: number;
  updated: number;
  skipped: number;
  failed: number;
  errors: Array<{ externalId: string; name: string; error: string }>;
}

export type ConflictStrategy = 'update' | 'skip' | 'overwrite';

export interface SyncProfileConfig {
  url: string;
  filters: Record<string, string>;
  platformMapping: Record<string, ConnectionType>;
  defaultProtocol: ConnectionType;
  defaultPort: Partial<Record<ConnectionType, number>>;
  conflictStrategy: ConflictStrategy;
}
