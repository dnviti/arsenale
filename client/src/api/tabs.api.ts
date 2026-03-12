import api from './client';

export interface PersistedTab {
  connectionId: string;
  sortOrder: number;
  isActive: boolean;
}

export async function getPersistedTabs(): Promise<PersistedTab[]> {
  const { data } = await api.get('/tabs');
  return data;
}

export async function syncPersistedTabs(tabs: PersistedTab[]): Promise<PersistedTab[]> {
  const { data } = await api.put('/tabs', { tabs });
  return data;
}

export async function clearPersistedTabs(): Promise<void> {
  await api.delete('/tabs');
}
