import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthStore } from '../../store/authStore';
import { useAccessPolicyStore } from '../../store/accessPolicyStore';
import AccessPolicySection from './AccessPolicySection';

const {
  listTeams,
  listFolders,
} = vi.hoisted(() => ({
  listTeams: vi.fn(),
  listFolders: vi.fn(),
}));

vi.mock('../../api/team.api', async () => {
  const actual = await vi.importActual<typeof import('../../api/team.api')>('../../api/team.api');
  return {
    ...actual,
    listTeams,
  };
});

vi.mock('../../api/folders.api', async () => {
  const actual = await vi.importActual<typeof import('../../api/folders.api')>('../../api/folders.api');
  return {
    ...actual,
    listFolders,
  };
});

describe('AccessPolicySection', () => {
  const fetchPolicies = vi.fn();
  const createPolicy = vi.fn();
  const updatePolicy = vi.fn();
  const deletePolicy = vi.fn();

  beforeEach(() => {
    vi.resetAllMocks();

    useAuthStore.setState({
      user: {
        id: 'user-1',
        email: 'admin@example.com',
        username: 'admin',
        tenantId: 'tenant-1',
        tenantRole: 'OWNER',
      } as never,
    });

    useAccessPolicyStore.setState({
      policies: [
        {
          id: 'policy-1',
          targetType: 'TEAM',
          targetId: 'team-1',
          allowedTimeWindows: '09:00-17:00',
          requireTrustedDevice: true,
          requireMfaStepUp: false,
          createdAt: '2026-04-07T12:00:00.000Z',
          updatedAt: '2026-04-07T12:00:00.000Z',
        },
      ],
      loading: false,
      error: null,
      fetchPolicies,
      createPolicy,
      updatePolicy,
      deletePolicy,
      reset: vi.fn(),
    });

    listTeams.mockResolvedValue([{ id: 'team-1', name: 'Platform Team' }]);
    listFolders.mockResolvedValue({ personal: [], team: [] });
    fetchPolicies.mockResolvedValue(undefined);
    createPolicy.mockResolvedValue(undefined);
    updatePolicy.mockResolvedValue(undefined);
    deletePolicy.mockResolvedValue(undefined);
  });

  it('renders existing policies and creates a tenant-scoped rule from the dialog', async () => {
    render(<AccessPolicySection />);

    expect(await screen.findByText('Team policy')).toBeInTheDocument();
    expect(screen.getByText('Applies to Platform Team. Policies stack, so the most restrictive applicable rule wins.')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Add Policy' }));

    fireEvent.change(screen.getByLabelText('Allowed time windows'), {
      target: { value: '08:00-18:00' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Create Policy' }));

    await waitFor(() => {
      expect(createPolicy).toHaveBeenCalledWith({
        targetType: 'TENANT',
        targetId: 'tenant-1',
        allowedTimeWindows: '08:00-18:00',
        requireTrustedDevice: false,
        requireMfaStepUp: false,
      });
    });
  });
});
