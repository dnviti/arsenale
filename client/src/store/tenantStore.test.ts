import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { TenantData, TenantMembership } from '../api/tenant.api';
import { createTenant as createTenantApi, getMyTenants } from '../api/tenant.api';
import { useAuthStore } from './authStore';
import { useConnectionsStore } from './connectionsStore';
import { useTabsStore } from './tabsStore';
import { useTenantStore } from './tenantStore';

vi.mock('../api/tenant.api', () => ({
  getMyTenant: vi.fn(),
  createTenant: vi.fn(),
  updateTenant: vi.fn(),
  deleteTenant: vi.fn(),
  listTenantUsers: vi.fn(),
  inviteUser: vi.fn(),
  updateUserRole: vi.fn(),
  removeUser: vi.fn(),
  createTenantUser: vi.fn(),
  toggleUserEnabled: vi.fn(),
  getMyTenants: vi.fn(),
  switchTenant: vi.fn(),
  updateMembershipExpiry: vi.fn(),
}));

function tenantData(overrides: Partial<TenantData> = {}): TenantData {
  return {
    id: 'tenant-2',
    name: 'Second Org',
    slug: 'second-org',
    mfaRequired: false,
    vaultAutoLockMaxMinutes: null,
    userCount: 1,
    defaultSessionTimeoutSeconds: 3600,
    maxConcurrentSessions: 10,
    absoluteSessionTimeoutSeconds: 28800,
    dlpDisableCopy: false,
    dlpDisablePaste: false,
    dlpDisableDownload: false,
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
    recordingEnabled: false,
    recordingRetentionDays: null,
    fileUploadMaxSizeBytes: null,
    userDriveQuotaBytes: null,
    teamCount: 0,
    createdAt: '2026-05-14T00:00:00.000Z',
    updatedAt: '2026-05-14T00:00:00.000Z',
    ...overrides,
  };
}

describe('useTenantStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({
      accessToken: null,
      csrfToken: null,
      user: null,
      isAuthenticated: false,
    });
    useTenantStore.setState({
      tenant: tenantData({ id: 'tenant-1', name: 'First Org', slug: 'first-org' }),
      users: [
        {
          id: 'user-old',
          email: 'old@example.com',
          username: 'old',
          avatarData: null,
          role: 'OWNER',
          status: 'ACCEPTED',
          pending: false,
          totpEnabled: false,
          smsMfaEnabled: false,
          enabled: true,
          createdAt: '2026-05-14T00:00:00.000Z',
          expiresAt: null,
          expired: false,
        },
      ],
      memberships: [],
      loading: false,
      usersLoading: false,
    });
  });

  it('resets tenant-scoped workspace state after creating an organization', async () => {
    const tenant = tenantData();
    const memberships: TenantMembership[] = [
      {
        tenantId: tenant.id,
        name: tenant.name,
        slug: tenant.slug,
        role: 'OWNER',
        status: 'ACCEPTED',
        pending: false,
        isActive: true,
        joinedAt: '2026-05-14T00:00:00.000Z',
      },
    ];
    const clearAll = vi.fn().mockResolvedValue(undefined);
    const resetConnections = vi.fn();
    const fetchConnections = vi.fn().mockResolvedValue(undefined);
    vi.mocked(createTenantApi).mockResolvedValue({
      tenant,
      accessToken: 'access-token-2',
      csrfToken: 'csrf-token-2',
      user: {
        id: 'user-1',
        email: 'user@example.com',
        username: 'user',
        avatarData: null,
        tenantId: tenant.id,
        tenantRole: 'OWNER',
      },
    });
    vi.mocked(getMyTenants).mockResolvedValue(memberships);
    useTabsStore.setState({ clearAll });
    useConnectionsStore.setState({ reset: resetConnections, fetchConnections });

    const result = await useTenantStore.getState().createTenant('Second Org');

    expect(result).toEqual(tenant);
    expect(useAuthStore.getState()).toMatchObject({
      accessToken: 'access-token-2',
      csrfToken: 'csrf-token-2',
      user: { tenantId: tenant.id, tenantRole: 'OWNER' },
      isAuthenticated: true,
    });
    expect(clearAll).toHaveBeenCalledTimes(1);
    expect(resetConnections).toHaveBeenCalledTimes(1);
    expect(fetchConnections).toHaveBeenCalledTimes(1);
    expect(getMyTenants).toHaveBeenCalledTimes(1);
    expect(useTenantStore.getState()).toMatchObject({
      tenant,
      users: [],
      memberships,
    });
  });
});
