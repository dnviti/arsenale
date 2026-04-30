import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import FileBrowser from './FileBrowser';
import { deleteFile, downloadFile, listFiles, uploadFile } from '../../api/files.api';
import {
  deleteRdpHistoryItem,
  downloadRdpHistoryItem,
  listRdpHistory,
  restoreRdpHistoryItem,
} from '../../api/managedHistory.api';

vi.mock('../../api/files.api', () => ({
  listFiles: vi.fn(),
  uploadFile: vi.fn(),
  downloadFile: vi.fn(),
  deleteFile: vi.fn(),
}));

vi.mock('../../api/managedHistory.api', () => ({
  listRdpHistory: vi.fn(),
  downloadRdpHistoryItem: vi.fn(),
  restoreRdpHistoryItem: vi.fn(),
  deleteRdpHistoryItem: vi.fn(),
}));

describe('RDP FileBrowser', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listFiles).mockResolvedValue([
      {
        name: 'workspace-report.txt',
        size: 512,
        modifiedAt: '2026-04-15T00:00:00Z',
      },
    ]);
    vi.mocked(listRdpHistory).mockResolvedValue([
      {
        id: 'history-1',
        fileName: 'retained-report.txt',
        size: 512,
        transferAt: '2026-04-15T00:00:00Z',
        protocol: 'rdp',
      },
    ]);
    vi.mocked(uploadFile).mockResolvedValue([]);
    vi.mocked(downloadFile).mockResolvedValue(undefined);
    vi.mocked(deleteFile).mockResolvedValue(undefined);
    vi.mocked(downloadRdpHistoryItem).mockResolvedValue(new Blob(['history']));
    vi.mocked(restoreRdpHistoryItem).mockResolvedValue({ restored: true });
    vi.mocked(deleteRdpHistoryItem).mockResolvedValue({ deleted: true });
  });

  it('shows the sandbox banner and keeps retained history out of the workspace view', async () => {
    const user = userEvent.setup();

    render(
      <FileBrowser
        open
        onClose={vi.fn()}
        connectionId="connection-rdp-1"
      />,
    );

    expect(screen.getByText('This browser shows only the managed transfer sandbox for this connection.')).toBeInTheDocument();

    await waitFor(() => {
      expect(listFiles).toHaveBeenCalledWith('connection-rdp-1');
    });

    expect(screen.getByText('workspace-report.txt')).toBeInTheDocument();
    expect(screen.queryByText('retained-report.txt')).not.toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: 'History' }));

    await waitFor(() => {
      expect(listRdpHistory).toHaveBeenCalledWith('connection-rdp-1');
    });

    expect(screen.getByText('retained-report.txt')).toBeInTheDocument();
    expect(screen.queryByText('workspace-report.txt')).not.toBeInTheDocument();
  });
});
