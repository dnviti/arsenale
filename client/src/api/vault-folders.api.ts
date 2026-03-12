import api from './client';

export type VaultFolderScope = 'PERSONAL' | 'TEAM' | 'TENANT';

export interface VaultFolderInput {
  name: string;
  scope: VaultFolderScope;
  parentId?: string;
  teamId?: string;
}

export interface VaultFolderUpdate {
  name?: string;
  parentId?: string | null;
}

export interface VaultFolderData {
  id: string;
  name: string;
  parentId: string | null;
  scope: VaultFolderScope;
  sortOrder: number;
  userId: string;
  teamId: string | null;
  tenantId: string | null;
  teamName?: string | null;
}

export interface VaultFoldersResponse {
  personal: VaultFolderData[];
  team: VaultFolderData[];
  tenant: VaultFolderData[];
}

export async function listVaultFolders(): Promise<VaultFoldersResponse> {
  const { data } = await api.get('/vault-folders');
  return data;
}

export async function createVaultFolder(payload: VaultFolderInput): Promise<VaultFolderData> {
  const { data } = await api.post('/vault-folders', payload);
  return data;
}

export async function updateVaultFolder(
  id: string,
  payload: VaultFolderUpdate
): Promise<VaultFolderData> {
  const { data } = await api.put(`/vault-folders/${id}`, payload);
  return data;
}

export async function deleteVaultFolder(id: string): Promise<{ deleted: boolean }> {
  const { data } = await api.delete(`/vault-folders/${id}`);
  return data;
}
