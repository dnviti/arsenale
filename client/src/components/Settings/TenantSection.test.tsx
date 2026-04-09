import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthStore } from '../../store/authStore';
import { useTenantStore } from '../../store/tenantStore';
import TenantSection from './TenantSection';

const { getTenantMfaStats } = vi.hoisted(() => ({
  getTenantMfaStats: vi.fn(),
}));

vi.mock('../../api/tenant.api', async () => {
  const actual = await vi.importActual<typeof import('../../api/tenant.api')>('../../api/tenant.api');
  return {
    ...actual,
    getTenantMfaStats,
  };
});

describe('TenantSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    useAuthStore.setState({
      user: {
        id: 'user-1',
        email: 'admin@example.com',
        username: 'admin',
        avatarData: null,
        tenantId: 'tenant-1',
        tenantRole: 'OWNER',
      },
    });

    getTenantMfaStats.mockResolvedValue({ total: 2, withoutMfa: 1 });
  });

  it('shows the tenant creation state when no tenant exists', () => {
    useTenantStore.setState({
      tenant: null,
      users: [],
      memberships: [],
      loading: false,
      usersLoading: false,
      fetchTenant: vi.fn().mockResolvedValue(undefined),
      fetchMemberships: vi.fn().mockResolvedValue(undefined),
      switchTenant: vi.fn().mockResolvedValue(undefined),
      createTenant: vi.fn().mockResolvedValue(undefined),
      updateTenant: vi.fn().mockResolvedValue(undefined),
      deleteTenant: vi.fn().mockResolvedValue(undefined),
      fetchUsers: vi.fn().mockResolvedValue(undefined),
      inviteUser: vi.fn().mockResolvedValue(undefined),
      updateUserRole: vi.fn().mockResolvedValue(undefined),
      removeUser: vi.fn().mockResolvedValue(undefined),
      createUser: vi.fn().mockResolvedValue(undefined),
      toggleUserEnabled: vi.fn().mockResolvedValue(undefined),
      updateMembershipExpiry: vi.fn().mockResolvedValue(undefined),
      reset: vi.fn(),
    });

    render(<TenantSection />);

    expect(screen.getByText('Create Your Organization')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create Organization' })).toBeInTheDocument();
  });

  it('renders split organization and member panels without embedding vault providers', async () => {
    useTenantStore.setState({
      tenant: {
        id: 'tenant-1',
        name: 'Acme Ops',
        slug: 'acme-ops',
        mfaRequired: true,
        vaultAutoLockMaxMinutes: 15,
        userCount: 2,
        defaultSessionTimeoutSeconds: 1800,
        maxConcurrentSessions: 2,
        absoluteSessionTimeoutSeconds: 43200,
        dlpDisableCopy: true,
        dlpDisablePaste: false,
        dlpDisableDownload: true,
        dlpDisableUpload: false,
        enforcedConnectionSettings: null,
        tunnelDefaultEnabled: false,
        tunnelAutoTokenRotation: false,
        tunnelTokenRotationDays: 30,
        tunnelRequireForRemote: false,
        tunnelTokenMaxLifetimeDays: null,
        tunnelAgentAllowedCidrs: [],
        loginRateLimitWindowMs: null,
        loginRateLimitMaxAttempts: null,
        accountLockoutThreshold: null,
        accountLockoutDurationMs: null,
        impossibleTravelSpeedKmh: null,
        jwtExpiresInSeconds: null,
        jwtRefreshExpiresInSeconds: null,
        vaultDefaultTtlMinutes: null,
        recordingEnabled: true,
        recordingRetentionDays: 30,
        fileUploadMaxSizeBytes: 10485760,
        userDriveQuotaBytes: 52428800,
        teamCount: 3,
        createdAt: '2026-04-07T12:00:00.000Z',
        updatedAt: '2026-04-07T12:00:00.000Z',
      },
      users: [
        {
          id: 'user-1',
          email: 'admin@example.com',
          username: 'admin',
          avatarData: null,
          role: 'OWNER',
          status: 'ACCEPTED',
          pending: false,
          totpEnabled: true,
          smsMfaEnabled: false,
          enabled: true,
          createdAt: '2026-04-07T12:00:00.000Z',
          expiresAt: null,
          expired: false,
        },
        {
          id: 'user-2',
          email: 'jamie@example.com',
          username: 'jamie',
          avatarData: null,
          role: 'ADMIN',
          status: 'ACCEPTED',
          pending: false,
          totpEnabled: false,
          smsMfaEnabled: false,
          enabled: true,
          createdAt: '2026-04-07T12:00:00.000Z',
          expiresAt: '2026-04-30T12:00:00.000Z',
          expired: false,
        },
      ],
      memberships: [],
      loading: false,
      usersLoading: false,
      fetchTenant: vi.fn().mockResolvedValue(undefined),
      fetchMemberships: vi.fn().mockResolvedValue(undefined),
      switchTenant: vi.fn().mockResolvedValue(undefined),
      createTenant: vi.fn().mockResolvedValue(undefined),
      updateTenant: vi.fn().mockResolvedValue(undefined),
      deleteTenant: vi.fn().mockResolvedValue(undefined),
      fetchUsers: vi.fn().mockResolvedValue(undefined),
      inviteUser: vi.fn().mockResolvedValue(undefined),
      updateUserRole: vi.fn().mockResolvedValue(undefined),
      removeUser: vi.fn().mockResolvedValue(undefined),
      createUser: vi.fn().mockResolvedValue(undefined),
      toggleUserEnabled: vi.fn().mockResolvedValue(undefined),
      updateMembershipExpiry: vi.fn().mockResolvedValue(undefined),
      reset: vi.fn(),
    });

    render(<TenantSection />);

    await waitFor(() => {
      expect(getTenantMfaStats).toHaveBeenCalledWith('tenant-1');
    });

    expect(screen.getByText('Workspace identity')).toBeInTheDocument();
    expect(screen.getAllByText('Members').length).toBeGreaterThan(0);
    expect(screen.getByText('Security & Session Policy')).toBeInTheDocument();
    expect(screen.getByText('jamie@example.com')).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'Permissions' }).length).toBeGreaterThan(0);
    expect(screen.queryByText('External Vault Providers')).not.toBeInTheDocument();
  });
});
