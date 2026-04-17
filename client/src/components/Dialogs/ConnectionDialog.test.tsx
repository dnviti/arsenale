import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ConnectionData } from '../../api/connections.api';
import { createConnection, updateConnection } from '../../api/connections.api';
import ConnectionDialog from './ConnectionDialog';
import { useAuthStore } from '../../store/authStore';
import { useConnectionsStore } from '../../store/connectionsStore';
import { useGatewayStore } from '../../store/gatewayStore';
import { useTenantStore } from '../../store/tenantStore';
import { useVaultStore } from '../../store/vaultStore';

const DEFAULT_CONNECTION_UPLOAD_LIMIT_BYTES = 100 * 1048576;

vi.mock('../../api/connections.api', async () => {
  const actual = await vi.importActual<typeof import('../../api/connections.api')>('../../api/connections.api');
  return {
    ...actual,
    createConnection: vi.fn(),
    updateConnection: vi.fn(),
  };
});

vi.mock('../../api/externalVault.api', () => ({
  listVaultProviders: vi.fn().mockResolvedValue([]),
}));

function buildConnection(overrides: Partial<ConnectionData> = {}): ConnectionData {
  return {
    id: 'connection-1',
    name: 'Sandbox SSH',
    type: 'SSH',
    host: 'ssh.example.com',
    port: 22,
    folderId: null,
    description: null,
    isFavorite: false,
    enableDrive: false,
    defaultCredentialMode: null,
    transferRetentionPolicy: null,
    isOwner: true,
    createdAt: '2026-04-15T00:00:00Z',
    updatedAt: '2026-04-15T00:00:00Z',
    ...overrides,
  } as ConnectionData;
}

describe('ConnectionDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({ user: null, isAuthenticated: false });
    useConnectionsStore.setState({
      fetchConnections: vi.fn().mockResolvedValue(undefined),
    });
    useGatewayStore.setState({
      gateways: [],
      fetchGateways: vi.fn().mockResolvedValue(undefined),
    });
    useTenantStore.setState({ tenant: null });
    useVaultStore.setState({ unlocked: false });
    vi.mocked(createConnection).mockResolvedValue(buildConnection());
    vi.mocked(updateConnection).mockResolvedValue(buildConnection());
  });

  it('defaults missing transfer retention policy to false and submits it on update', async () => {
    render(
      <ConnectionDialog
        open
        onClose={vi.fn()}
        connection={buildConnection({ transferRetentionPolicy: null })}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'File Transfer' }));

    const retentionToggle = screen.getByRole('checkbox', {
      name: 'Retain successful uploads in history',
    });
    expect(retentionToggle).not.toBeChecked();

    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(updateConnection).toHaveBeenCalledWith(
        'connection-1',
        expect.objectContaining({
          transferRetentionPolicy: expect.objectContaining({
            retainSuccessfulUploads: false,
            maxUploadSizeBytes: DEFAULT_CONNECTION_UPLOAD_LIMIT_BYTES,
          }),
        }),
      );
    });
  });

  it('round-trips transfer retention policy when enabled for a new SSH connection', async () => {
    render(<ConnectionDialog open onClose={vi.fn()} />);

    fireEvent.change(screen.getByLabelText('Name'), {
      target: { value: 'Upload Sandbox' },
    });
    fireEvent.change(screen.getByLabelText('Host'), {
      target: { value: 'ssh.example.com' },
    });

    fireEvent.click(screen.getByRole('button', { name: 'Credentials' }));
    fireEvent.change(screen.getByLabelText('Username'), {
      target: { value: 'demo' },
    });

    fireEvent.click(screen.getByRole('button', { name: 'File Transfer' }));
    fireEvent.click(screen.getByRole('checkbox', {
      name: 'Retain successful uploads in history',
    }));

    fireEvent.click(screen.getByRole('button', { name: 'Create' }));

    await waitFor(() => {
      expect(createConnection).toHaveBeenCalledWith(
        expect.objectContaining({
          name: 'Upload Sandbox',
          host: 'ssh.example.com',
          transferRetentionPolicy: expect.objectContaining({
            retainSuccessfulUploads: true,
            maxUploadSizeBytes: DEFAULT_CONNECTION_UPLOAD_LIMIT_BYTES,
          }),
        }),
      );
    });
  });

  it('shows the file transfer section for RDP with retention defaulted off', async () => {
    render(
      <ConnectionDialog
        open
        onClose={vi.fn()}
        connection={buildConnection({
          type: 'RDP',
          transferRetentionPolicy: null,
        })}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'File Transfer' }));

    expect(screen.getByRole('checkbox', {
      name: 'Retain successful uploads in history',
    })).not.toBeChecked();
  });
});
