import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useFeatureFlagsStore } from '../../store/featureFlagsStore';
import { useTenantStore } from '../../store/tenantStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import TenantAuditLogSection from './TenantAuditLogSection';

const {
  getTenantAuditCountries,
  getTenantAuditGateways,
  getTenantAuditLogs,
} = vi.hoisted(() => ({
  getTenantAuditCountries: vi.fn(),
  getTenantAuditGateways: vi.fn(),
  getTenantAuditLogs: vi.fn(),
}));

vi.mock('../../api/audit.api', async () => {
  const actual = await vi.importActual<typeof import('../../api/audit.api')>('../../api/audit.api');
  return {
    ...actual,
    getTenantAuditCountries,
    getTenantAuditGateways,
    getTenantAuditLogs,
  };
});

vi.mock('../Audit/AuditGeoMap', () => ({
  default: () => <div>Audit geo map</div>,
}));

describe('TenantAuditLogSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    useFeatureFlagsStore.setState({
      ipGeolocationEnabled: true,
    });

    useUiPreferencesStore.setState({
      tenantAuditLogAction: '',
      tenantAuditLogSearch: '',
      tenantAuditLogTargetType: '',
      tenantAuditLogGatewayId: '',
      tenantAuditLogUserId: '',
      tenantAuditLogSortBy: 'createdAt',
      tenantAuditLogSortOrder: 'desc',
      tenantAuditLogViewMode: 'table',
    });

    useTenantStore.setState({
      tenant: {
        id: 'tenant-1',
        name: 'Acme',
        slug: 'acme',
        mfaRequired: false,
        vaultAutoLockMaxMinutes: null,
        userCount: 1,
        defaultSessionTimeoutSeconds: 1800,
        maxConcurrentSessions: 2,
        absoluteSessionTimeoutSeconds: 7200,
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
        recordingEnabled: true,
        recordingRetentionDays: 30,
        fileUploadMaxSizeBytes: null,
        userDriveQuotaBytes: null,
        teamCount: 0,
        createdAt: '2026-04-08T00:00:00.000Z',
        updatedAt: '2026-04-08T00:00:00.000Z',
      },
      users: [
        {
          id: 'user-1',
          email: 'admin@example.com',
          username: 'Admin',
          avatarData: null,
          role: 'OWNER',
          status: 'ACCEPTED',
          pending: false,
          totpEnabled: true,
          smsMfaEnabled: false,
          enabled: true,
          createdAt: '2026-04-08T00:00:00.000Z',
          expiresAt: null,
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

    getTenantAuditGateways.mockResolvedValue([{ id: 'gateway-1', name: 'Tunnel SSH' }]);
    getTenantAuditCountries.mockResolvedValue(['United States']);
    getTenantAuditLogs.mockResolvedValue({
      data: [
        {
          id: 'log-1',
          action: 'LOGIN',
          targetType: 'User',
          targetId: 'user-1',
          details: { provider: 'password' },
          ipAddress: '8.8.8.8',
          gatewayId: 'gateway-1',
          geoCountry: 'United States',
          geoCity: 'New York',
          geoCoords: [],
          flags: ['IMPOSSIBLE_TRAVEL'],
          createdAt: '2026-04-08T00:00:00.000Z',
          userId: 'user-1',
          userName: 'Admin',
          userEmail: 'admin@example.com',
        },
      ],
      total: 1,
      page: 1,
      limit: 25,
      totalPages: 1,
    });
  });

  it('renders grouped filters and card-based activity entries', async () => {
    render(<TenantAuditLogSection />);

    await waitFor(() => {
      expect(getTenantAuditLogs).toHaveBeenCalledTimes(1);
    });

    expect(screen.getByText('Organization Audit Log')).toBeInTheDocument();
    expect(screen.getByLabelText('Search activity')).toBeInTheDocument();
    expect(screen.getByText('Flagged Only')).toBeInTheDocument();
    expect(screen.getAllByText('Admin').length).toBeGreaterThan(0);
    expect(screen.getByText('Flagged')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Show Details' }));
    expect(await screen.findByText('Structured details')).toBeInTheDocument();
  });
});
