import type { SyncProvider } from '../lib/prisma';
import type { ISyncProvider } from './types';
import { NetBoxProvider } from './netbox.provider';

const providers: Record<string, ISyncProvider> = {
  NETBOX: new NetBoxProvider(),
};

export function createSyncProvider(type: SyncProvider): ISyncProvider {
  const provider = providers[type];
  if (!provider) {
    throw new Error(`Unknown sync provider: ${type}`);
  }
  return provider;
}
