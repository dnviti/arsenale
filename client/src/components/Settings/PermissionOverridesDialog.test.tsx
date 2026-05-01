import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import PermissionOverridesDialog from './PermissionOverridesDialog';

const {
  getUserPermissions,
  updateUserPermissions,
} = vi.hoisted(() => ({
  getUserPermissions: vi.fn(),
  updateUserPermissions: vi.fn(),
}));

vi.mock('../../api/tenant.api', async () => {
  const actual = await vi.importActual<typeof import('../../api/tenant.api')>('../../api/tenant.api');
  return {
    ...actual,
    getUserPermissions,
    updateUserPermissions,
  };
});

describe('PermissionOverridesDialog', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    getUserPermissions.mockResolvedValue({
      role: 'MEMBER',
      permissions: {
        canConnect: true,
        canCreateConnections: false,
        canManageConnections: false,
        canViewCredentials: false,
        canShareConnections: false,
        canViewAuditLog: false,
        canViewSessions: false,
        canObserveSessions: false,
        canControlSessions: false,
        canManageSessions: false,
        canManageGateways: false,
        canManageUsers: false,
        canManageSecrets: false,
        canManageTenantSettings: false,
      },
      defaults: {
        canConnect: true,
        canCreateConnections: false,
        canManageConnections: false,
        canViewCredentials: false,
        canShareConnections: false,
        canViewAuditLog: false,
        canViewSessions: false,
        canObserveSessions: false,
        canControlSessions: false,
        canManageSessions: false,
        canManageGateways: false,
        canManageUsers: false,
        canManageSecrets: false,
        canManageTenantSettings: false,
      },
      overrides: { canViewSessions: true, canManageSessions: true },
    });

    updateUserPermissions.mockResolvedValue({
      role: 'MEMBER',
      permissions: {
        canConnect: true,
        canCreateConnections: false,
        canManageConnections: false,
        canViewCredentials: false,
        canShareConnections: false,
        canViewAuditLog: false,
        canViewSessions: true,
        canObserveSessions: false,
        canControlSessions: true,
        canManageSessions: true,
        canManageGateways: false,
        canManageUsers: false,
        canManageSecrets: false,
        canManageTenantSettings: false,
      },
      defaults: {
        canConnect: true,
        canCreateConnections: false,
        canManageConnections: false,
        canViewCredentials: false,
        canShareConnections: false,
        canViewAuditLog: false,
        canViewSessions: false,
        canObserveSessions: false,
        canControlSessions: false,
        canManageSessions: false,
        canManageGateways: false,
        canManageUsers: false,
        canManageSecrets: false,
        canManageTenantSettings: false,
      },
      overrides: { canViewSessions: true, canControlSessions: true, canManageSessions: true },
    });
  });

  it('groups split session permissions and saves explicit overrides without the legacy alias', async () => {
    render(
      <PermissionOverridesDialog
        open
        onClose={vi.fn()}
        tenantId="tenant-1"
        userId="user-2"
        userName="Jamie"
      />,
    );

    expect(await screen.findByText('Session access')).toBeInTheDocument();
    expect(screen.getByText('Administration')).toBeInTheDocument();
    expect(screen.getByText('View active sessions')).toBeInTheDocument();
    expect(screen.getByText('Observe live sessions')).toBeInTheDocument();
    expect(screen.getByText('Control active sessions')).toBeInTheDocument();
    expect(screen.queryByText('Manage sessions')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('switch', { name: 'Control active sessions' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save overrides' }));

    await waitFor(() => {
      expect(updateUserPermissions).toHaveBeenCalledWith('tenant-1', 'user-2', {
        canViewSessions: true,
        canControlSessions: true,
      });
    });
  });
});
