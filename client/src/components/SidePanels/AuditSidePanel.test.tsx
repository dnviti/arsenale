import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import AuditSidePanel from './AuditSidePanel';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';

const { getAuditLogs } = vi.hoisted(() => ({
  getAuditLogs: vi.fn(),
}));

vi.mock('../../api/audit.api', async () => {
  const actual = await vi.importActual<typeof import('../../api/audit.api')>('../../api/audit.api');
  return {
    ...actual,
    getAuditLogs,
  };
});

describe('AuditSidePanel', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    localStorage.clear();

    useUiPreferencesStore.setState({
      auditLogAction: '',
      auditLogSearch: '',
      auditLogSortBy: 'createdAt',
      auditLogSortOrder: 'desc',
    });

    getAuditLogs.mockResolvedValue({
      data: [
        {
          id: 'log-file-upload',
          action: 'FILE_UPLOAD',
          targetType: 'Connection',
          targetId: 'conn-1',
          details: {
            protocol: 'ssh',
            transferMode: 'managed-payload',
            transferId: 'corr-123',
            objectKey: 'shared-files/ssh-upload/stage/key',
            remotePath: '/tmp/report.txt',
            fileName: 'report.txt',
            size: 42,
            checksumSha256: 'abc123',
            policyDecision: 'allowed',
            scanResult: 'clean',
            result: 'success',
          },
          ipAddress: '127.0.0.1',
          gatewayId: null,
          geoCountry: null,
          geoCity: null,
          geoCoords: [],
          flags: [],
          createdAt: '2026-04-15T00:00:00.000Z',
        },
        {
          id: 'log-legacy-download',
          action: 'SFTP_DOWNLOAD',
          targetType: 'Connection',
          targetId: 'conn-1',
          details: {
            path: '/legacy/archive.zip',
            filename: 'archive.zip',
          },
          ipAddress: '127.0.0.1',
          gatewayId: null,
          geoCountry: null,
          geoCity: null,
          geoCoords: [],
          flags: [],
          createdAt: '2026-04-14T00:00:00.000Z',
        },
      ],
      total: 2,
      page: 1,
      limit: 30,
      totalPages: 1,
    });
  });

  it('renders unified file actions and keeps legacy sftp actions readable', async () => {
    render(<AuditSidePanel />);

    await waitFor(() => {
      expect(getAuditLogs).toHaveBeenCalledTimes(1);
    });

    expect(screen.getByText('File Upload')).toBeInTheDocument();
    expect(screen.getByText('SFTP Download (Legacy)')).toBeInTheDocument();

    fireEvent.click(screen.getByText('File Upload'));

    expect(await screen.findByText('remotePath:')).toBeInTheDocument();
    expect(screen.getByText('/tmp/report.txt')).toBeInTheDocument();
    expect(screen.getByText('checksumSha256:')).toBeInTheDocument();
    expect(screen.getByText('abc123')).toBeInTheDocument();
  });
});
