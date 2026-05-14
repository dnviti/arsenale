import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useTenantStore } from '@/store/tenantStore';
import TenantSwitcher from './TenantSwitcher';

describe('TenantSwitcher', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useTenantStore.setState({
      tenant: null,
      users: [],
      memberships: [],
      loading: false,
      usersLoading: false,
      fetchTenant: vi.fn().mockResolvedValue(undefined),
      fetchMemberships: vi.fn().mockResolvedValue(undefined),
      switchTenant: vi.fn().mockResolvedValue(undefined),
      createTenant: vi.fn().mockResolvedValue({
        id: 'tenant-2',
        name: 'Second Org',
        slug: 'second-org',
      }),
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
  });

  it('keeps organization creation available with one active membership', async () => {
    const user = userEvent.setup();
    const fetchMemberships = vi.fn().mockResolvedValue(undefined);
    const createTenant = vi.fn().mockResolvedValue({
      id: 'tenant-2',
      name: 'Second Org',
      slug: 'second-org',
    });
    useTenantStore.setState({
      memberships: [
        {
          tenantId: 'tenant-1',
          name: 'First Org',
          slug: 'first-org',
          role: 'OWNER',
          status: 'ACCEPTED',
          pending: false,
          isActive: true,
          joinedAt: '2026-05-14T00:00:00.000Z',
        },
      ],
      fetchMemberships,
      createTenant,
    });

    render(<TenantSwitcher />);

    await waitFor(() => {
      expect(fetchMemberships).toHaveBeenCalledTimes(1);
    });
    fetchMemberships.mockClear();

    await user.click(screen.getByRole('button', { name: /First Org/i }));
    await user.click(await screen.findByText('Create organization'));
    fireEvent.change(await screen.findByLabelText('Organization name'), {
      target: { value: 'Second Org' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Create Organization' }));

    await waitFor(() => {
      expect(createTenant).toHaveBeenCalledWith('Second Org');
    });
    expect(fetchMemberships).not.toHaveBeenCalled();
  });

  it('keeps the create dialog open while creation is in flight', async () => {
    const user = userEvent.setup();
    let resolveCreate: (value: { id: string; name: string; slug: string }) => void = () => {};
    const createTenant = vi.fn().mockReturnValue(new Promise((resolve) => {
      resolveCreate = resolve;
    }));
    useTenantStore.setState({
      memberships: [],
      createTenant,
    });

    render(<TenantSwitcher />);

    await user.click(screen.getByRole('button', { name: /Create organization/i }));
    fireEvent.change(await screen.findByLabelText('Organization name'), {
      target: { value: 'Second Org' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Create Organization' }));

    expect(await screen.findByRole('button', { name: 'Creating...' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled();
    expect(screen.queryByRole('button', { name: 'Close' })).not.toBeInTheDocument();

    resolveCreate({
      id: 'tenant-2',
      name: 'Second Org',
      slug: 'second-org',
    });

    await waitFor(() => {
      expect(screen.queryByLabelText('Organization name')).not.toBeInTheDocument();
    });
  });
});
