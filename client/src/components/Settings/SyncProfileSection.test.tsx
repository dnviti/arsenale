import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import SyncProfileSection from './SyncProfileSection';
import { useNotificationStore } from '../../store/notificationStore';

const {
  listSyncProfiles,
  createSyncProfile,
  updateSyncProfile,
  deleteSyncProfile,
  testSyncConnection,
  triggerSync,
  getSyncLogs,
} = vi.hoisted(() => ({
  listSyncProfiles: vi.fn(),
  createSyncProfile: vi.fn(),
  updateSyncProfile: vi.fn(),
  deleteSyncProfile: vi.fn(),
  testSyncConnection: vi.fn(),
  triggerSync: vi.fn(),
  getSyncLogs: vi.fn(),
}));

vi.mock('../../api/sync.api', () => ({
  listSyncProfiles,
  createSyncProfile,
  updateSyncProfile,
  deleteSyncProfile,
  testSyncConnection,
  triggerSync,
  getSyncLogs,
}));

describe('SyncProfileSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useNotificationStore.setState({ notification: null });

    listSyncProfiles.mockResolvedValue([]);
    createSyncProfile.mockResolvedValue({});
    updateSyncProfile.mockResolvedValue({});
    deleteSyncProfile.mockResolvedValue(undefined);
    testSyncConnection.mockResolvedValue({ ok: true });
    triggerSync.mockResolvedValue({
      plan: {
        toCreate: [],
        toUpdate: [],
        toSkip: [],
        errors: [],
      },
    });
    getSyncLogs.mockResolvedValue({ logs: [], total: 0 });
  });

  it('creates a sync profile from the editor dialog', async () => {
    render(<SyncProfileSection />);

    fireEvent.click(await screen.findByRole('button', { name: 'Add Profile' }));
    fireEvent.change(screen.getByLabelText('Name'), {
      target: { value: 'NetBox Import' },
    });
    fireEvent.change(screen.getByLabelText('NetBox URL'), {
      target: { value: 'https://netbox.example.com' },
    });
    fireEvent.change(screen.getByLabelText('API Token'), {
      target: { value: 'secret-token' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save Profile' }));

    await waitFor(() => {
      expect(createSyncProfile).toHaveBeenCalledWith({
        name: 'NetBox Import',
        provider: 'NETBOX',
        url: 'https://netbox.example.com',
        apiToken: 'secret-token',
        filters: {},
        platformMapping: {},
        defaultProtocol: 'SSH',
        conflictStrategy: 'update',
        cronExpression: undefined,
        teamId: undefined,
      });
    });
  });

  it('opens the preview dialog and confirms a sync run', async () => {
    listSyncProfiles.mockResolvedValue([
      {
        id: 'profile-1',
        name: 'NetBox Import',
        provider: 'NETBOX',
        config: {
          url: 'https://netbox.example.com',
          filters: {},
          platformMapping: {},
          defaultProtocol: 'SSH',
          defaultPort: {},
          conflictStrategy: 'update',
        },
        cronExpression: null,
        enabled: true,
        teamId: null,
        lastSyncAt: null,
        lastSyncStatus: null,
        lastSyncDetails: null,
        hasApiToken: true,
        createdAt: '2026-04-07T00:00:00.000Z',
        updatedAt: '2026-04-07T00:00:00.000Z',
      },
    ]);
    triggerSync
      .mockResolvedValueOnce({
        plan: {
          toCreate: [
            {
              externalId: 'device-1',
              name: 'web-1',
              host: '10.0.0.10',
              port: 22,
              protocol: 'SSH',
            },
          ],
          toUpdate: [],
          toSkip: [],
          errors: [],
        },
      })
      .mockResolvedValueOnce({
        plan: {
          toCreate: [],
          toUpdate: [],
          toSkip: [],
          errors: [],
        },
      });

    render(<SyncProfileSection />);

    fireEvent.click(await screen.findByRole('button', { name: 'Preview Sync' }));
    expect(await screen.findByText('Sync Preview')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Confirm Import' }));

    await waitFor(() => {
      expect(triggerSync).toHaveBeenNthCalledWith(1, 'profile-1', true);
      expect(triggerSync).toHaveBeenNthCalledWith(2, 'profile-1', false);
    });

    expect(useNotificationStore.getState().notification).toMatchObject({
      message: 'Sync completed successfully',
      severity: 'success',
    });
  });
});
