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
      defaults: {
        canConnect: true,
        canCreateConnections: false,
        canManageConnections: false,
        canViewCredentials: false,
        canShareConnections: false,
        canViewAuditLog: false,
        canManageSessions: false,
        canManageGateways: false,
        canManageUsers: false,
        canManageSecrets: false,
        canManageTenantSettings: false,
      },
      overrides: {},
    });

    updateUserPermissions.mockResolvedValue({
      role: 'MEMBER',
      defaults: {
        canConnect: true,
        canCreateConnections: false,
        canManageConnections: false,
        canViewCredentials: false,
        canShareConnections: false,
        canViewAuditLog: false,
        canManageSessions: false,
        canManageGateways: false,
        canManageUsers: false,
        canManageSecrets: false,
        canManageTenantSettings: false,
      },
      overrides: { canManageUsers: true },
    });
  });

  it('groups permissions by concern and saves explicit overrides', async () => {
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

    fireEvent.click(screen.getByRole('switch', { name: 'Manage users' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save overrides' }));

    await waitFor(() => {
      expect(updateUserPermissions).toHaveBeenCalledWith('tenant-1', 'user-2', {
        canManageUsers: true,
      });
    });
  });
});
