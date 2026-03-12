import api from './client';

export interface FolderInput {
  name: string;
  parentId?: string;
  teamId?: string;
}

export interface FolderUpdate {
  name?: string;
  parentId?: string | null;
}

export interface FolderData {
  id: string;
  name: string;
  parentId: string | null;
  sortOrder: number;
  teamId?: string | null;
  teamName?: string | null;
  scope?: 'private' | 'team';
}

export interface FoldersResponse {
  personal: FolderData[];
  team: FolderData[];
}

export async function listFolders(): Promise<FoldersResponse> {
  const { data } = await api.get('/folders');
  return data;
}

export async function createFolder(payload: FolderInput): Promise<FolderData> {
  const { data } = await api.post('/folders', payload);
  return data;
}

export async function updateFolder(
  id: string,
  payload: FolderUpdate
): Promise<FolderData> {
  const { data } = await api.put(`/folders/${id}`, payload);
  return data;
}

export async function deleteFolder(id: string): Promise<{ deleted: boolean }> {
  const { data } = await api.delete(`/folders/${id}`);
  return data;
}
