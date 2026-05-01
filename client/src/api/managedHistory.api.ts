import api from './client';
import type { SshFileCredentials } from './sshFiles.api';

export interface ManagedHistoryEntry {
  id: string;
  fileName: string;
  restoredName?: string;
  size: number;
  contentType?: string;
  transferAt: string;
  actorId?: string;
  protocol: string;
  transferId?: string;
  checksumSha256?: string;
  policyDecision?: string;
  scanResult?: string;
}

export interface ManagedHistoryMutationResult {
  deleted?: boolean;
  restored?: boolean;
  item?: ManagedHistoryEntry;
}

export interface ManagedRdpHistoryRestoreResult extends ManagedHistoryMutationResult {
  files?: Array<{ name: string; size: number; modifiedAt: string }>;
}

export async function listRdpHistory(connectionId: string): Promise<ManagedHistoryEntry[]> {
  const { data } = await api.get('/files/history', {
    params: { connectionId },
  });
  return data.items ?? [];
}

export async function downloadRdpHistoryItem(connectionId: string, id: string): Promise<Blob> {
  const { data } = await api.get(`/files/history/${encodeURIComponent(id)}`, {
    params: { connectionId },
    responseType: 'blob',
  });
  return new Blob([data]);
}

export async function restoreRdpHistoryItem(
  connectionId: string,
  id: string,
  name?: string,
): Promise<ManagedRdpHistoryRestoreResult> {
  const { data } = await api.post(`/files/history/${encodeURIComponent(id)}/restore`, null, {
    params: {
      connectionId,
      ...(name ? { name } : {}),
    },
  });
  return data;
}

export async function deleteRdpHistoryItem(connectionId: string, id: string): Promise<ManagedHistoryMutationResult> {
  const { data } = await api.delete(`/files/history/${encodeURIComponent(id)}`, {
    params: { connectionId },
  });
  return data;
}

export async function listSshHistory(payload: SshFileCredentials): Promise<ManagedHistoryEntry[]> {
  const { data } = await api.post('/files/ssh/history/list', payload);
  return data.items ?? [];
}

export async function downloadSshHistoryItem(payload: SshFileCredentials & { id: string }): Promise<Blob> {
  const { data } = await api.post('/files/ssh/history/download', payload, {
    responseType: 'blob',
  });
  return new Blob([data]);
}

export async function restoreSshHistoryItem(
  payload: SshFileCredentials & { id: string; path: string },
): Promise<ManagedHistoryMutationResult> {
  const { data } = await api.post('/files/ssh/history/restore', payload);
  return data;
}

export async function deleteSshHistoryItem(
  payload: SshFileCredentials & { id: string },
): Promise<ManagedHistoryMutationResult> {
  const { data } = await api.post('/files/ssh/history/delete', payload);
  return data;
}
