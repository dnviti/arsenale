import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import SftpBrowser from './SftpBrowser';
import {
  createSshDirectory,
  deleteSshPath,
  downloadSshFile,
  listSshFiles,
  renameSshPath,
  uploadSshFile,
} from '../../api/sshFiles.api';
import {
  deleteSshHistoryItem,
  downloadSshHistoryItem,
  listSshHistory,
  restoreSshHistoryItem,
} from '../../api/managedHistory.api';

vi.mock('../../api/sshFiles.api', () => ({
  listSshFiles: vi.fn(),
  createSshDirectory: vi.fn(),
  deleteSshPath: vi.fn(),
  renameSshPath: vi.fn(),
  uploadSshFile: vi.fn(),
  downloadSshFile: vi.fn(),
}));

vi.mock('../../api/managedHistory.api', () => ({
  listSshHistory: vi.fn(),
  downloadSshHistoryItem: vi.fn(),
  restoreSshHistoryItem: vi.fn(),
  deleteSshHistoryItem: vi.fn(),
}));

describe('SftpBrowser', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listSshFiles).mockResolvedValue({
      entries: [
        {
          name: 'workspace-report.txt',
          size: 128,
          type: 'file',
          modifiedAt: '2026-04-15T00:00:00Z',
        },
      ],
    });
    vi.mocked(listSshHistory).mockResolvedValue([
      {
        id: 'history-1',
        fileName: 'retained-report.txt',
        size: 128,
        transferAt: '2026-04-15T00:00:00Z',
        protocol: 'ssh',
      },
    ]);
    vi.mocked(createSshDirectory).mockResolvedValue(undefined);
    vi.mocked(deleteSshPath).mockResolvedValue(undefined);
    vi.mocked(renameSshPath).mockResolvedValue(undefined);
    vi.mocked(uploadSshFile).mockResolvedValue(undefined);
    vi.mocked(downloadSshFile).mockResolvedValue(new Blob(['workspace']));
    vi.mocked(downloadSshHistoryItem).mockResolvedValue(new Blob(['history']));
    vi.mocked(restoreSshHistoryItem).mockResolvedValue({ restored: true });
    vi.mocked(deleteSshHistoryItem).mockResolvedValue({ deleted: true });
  });

  it('shows the sandbox banner and keeps history separate from the workspace list', async () => {
    const user = userEvent.setup();

    render(
      <SftpBrowser
        open
        onClose={vi.fn()}
        connectionId="connection-ssh-1"
      />,
    );

    expect(screen.getByText('This browser shows only the managed transfer sandbox for this connection.')).toBeInTheDocument();

    await waitFor(() => {
      expect(listSshFiles).toHaveBeenCalledWith({
        connectionId: 'connection-ssh-1',
        path: '',
      });
    });

    expect(screen.getByText('workspace-report.txt')).toBeInTheDocument();
    expect(screen.queryByText('retained-report.txt')).not.toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: 'History' }));

    await waitFor(() => {
      expect(listSshHistory).toHaveBeenCalledWith({
        connectionId: 'connection-ssh-1',
      });
    });

    expect(screen.getByText('retained-report.txt')).toBeInTheDocument();
    expect(screen.queryByText('workspace-report.txt')).not.toBeInTheDocument();
  });

  it('shows the sandbox-only rejection copy when legacy remote browsing is attempted', async () => {
    vi.mocked(listSshFiles).mockRejectedValueOnce(
      {
        response: {
          data: {
            error: 'Only sandbox-relative paths are allowed; remote filesystem browsing is disabled.',
          },
        },
      },
    );

    render(
      <SftpBrowser
        open
        onClose={vi.fn()}
        connectionId="connection-ssh-1"
      />,
    );

    await waitFor(() => {
      expect(screen.getByText('Remote filesystem browsing is disabled. Use sandbox-relative paths only.')).toBeInTheDocument();
    });
    expect(screen.queryByText('Only sandbox-relative paths are allowed; remote filesystem browsing is disabled.')).not.toBeInTheDocument();
  });
});
