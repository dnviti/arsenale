import { beforeEach, describe, expect, it, vi } from 'vitest';
import api from './client';
import {
  deleteRdpHistoryItem,
  deleteSshHistoryItem,
  downloadRdpHistoryItem,
  downloadSshHistoryItem,
  listRdpHistory,
  listSshHistory,
  restoreRdpHistoryItem,
  restoreSshHistoryItem,
} from './managedHistory.api';

vi.mock('./client', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('managedHistory.api', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('lists RDP history through the history endpoint', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({ data: { items: [{ id: 'history-1' }] } });

    const result = await listRdpHistory('connection-1');

    expect(api.get).toHaveBeenCalledWith('/files/history', {
      params: { connectionId: 'connection-1' },
    });
    expect(result).toEqual([{ id: 'history-1' }]);
  });

  it('posts SSH history list requests with connection credentials', async () => {
    vi.mocked(api.post).mockResolvedValueOnce({ data: { items: [{ id: 'history-ssh-1' }] } });

    const result = await listSshHistory({
      connectionId: 'connection-ssh-1',
      username: 'demo',
      credentialMode: 'manual',
    });

    expect(api.post).toHaveBeenCalledWith('/files/ssh/history/list', {
      connectionId: 'connection-ssh-1',
      username: 'demo',
      credentialMode: 'manual',
    });
    expect(result).toEqual([{ id: 'history-ssh-1' }]);
  });

  it('downloads SSH history as a blob payload', async () => {
    vi.mocked(api.post).mockResolvedValueOnce({ data: 'blob-data' });

    const result = await downloadSshHistoryItem({
      connectionId: 'connection-ssh-1',
      id: 'history-ssh-1',
    });

    expect(api.post).toHaveBeenCalledWith(
      '/files/ssh/history/download',
      {
        connectionId: 'connection-ssh-1',
        id: 'history-ssh-1',
      },
      { responseType: 'blob' },
    );
    expect(result).toBeInstanceOf(Blob);
  });

  it('restores RDP history back into the workspace using the optional name parameter', async () => {
    vi.mocked(api.post).mockResolvedValueOnce({ data: { restored: true } });

    const result = await restoreRdpHistoryItem('connection-1', 'history-1', 'restored.txt');

    expect(api.post).toHaveBeenCalledWith('/files/history/history-1/restore', null, {
      params: {
        connectionId: 'connection-1',
        name: 'restored.txt',
      },
    });
    expect(result).toEqual({ restored: true });
  });

  it('restores SSH history with a sandbox-relative destination path', async () => {
    vi.mocked(api.post).mockResolvedValueOnce({ data: { restored: true } });

    const result = await restoreSshHistoryItem({
      connectionId: 'connection-ssh-1',
      id: 'history-ssh-1',
      path: 'docs/restored.txt',
    });

    expect(api.post).toHaveBeenCalledWith('/files/ssh/history/restore', {
      connectionId: 'connection-ssh-1',
      id: 'history-ssh-1',
      path: 'docs/restored.txt',
    });
    expect(result).toEqual({ restored: true });
  });

  it('deletes history entries through the matching RDP and SSH routes', async () => {
    vi.mocked(api.delete).mockResolvedValueOnce({ data: { deleted: true } });
    vi.mocked(api.post).mockResolvedValueOnce({ data: { deleted: true } });

    await deleteRdpHistoryItem('connection-1', 'history-1');
    await deleteSshHistoryItem({ connectionId: 'connection-ssh-1', id: 'history-ssh-1' });

    expect(api.delete).toHaveBeenCalledWith('/files/history/history-1', {
      params: { connectionId: 'connection-1' },
    });
    expect(api.post).toHaveBeenCalledWith('/files/ssh/history/delete', {
      connectionId: 'connection-ssh-1',
      id: 'history-ssh-1',
    });
  });

  it('downloads RDP history as a blob payload', async () => {
    vi.mocked(api.get).mockResolvedValueOnce({ data: 'blob-data' });

    const result = await downloadRdpHistoryItem('connection-1', 'history-1');

    expect(api.get).toHaveBeenCalledWith('/files/history/history-1', {
      params: { connectionId: 'connection-1' },
      responseType: 'blob',
    });
    expect(result).toBeInstanceOf(Blob);
  });
});
