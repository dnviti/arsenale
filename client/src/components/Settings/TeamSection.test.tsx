import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthStore } from '../../store/authStore';
import { useTeamStore } from '../../store/teamStore';
import TeamSection from './TeamSection';

const { searchUsers } = vi.hoisted(() => ({
  searchUsers: vi.fn(),
}));

vi.mock('../../api/user.api', async () => {
  const actual = await vi.importActual<typeof import('../../api/user.api')>('../../api/user.api');
  return {
    ...actual,
    searchUsers,
  };
});

describe('TeamSection', () => {
  const fetchTeams = vi.fn();
  const createTeam = vi.fn();
  const updateTeam = vi.fn();
  const deleteTeam = vi.fn();
  const selectTeam = vi.fn();
  const clearSelectedTeam = vi.fn();
  const fetchMembers = vi.fn();
  const addMember = vi.fn();
  const updateMemberRole = vi.fn();
  const removeMember = vi.fn();
  const updateMemberExpiry = vi.fn();
  const reset = vi.fn();

  beforeEach(() => {
    vi.resetAllMocks();
    localStorage.clear();

    useAuthStore.setState({
      user: {
        id: 'user-1',
        email: 'admin@example.com',
        username: 'admin',
        avatarData: null,
        tenantId: 'tenant-1',
      },
    });

    const team = {
      id: 'team-1',
      name: 'Platform Operations',
      description: 'Owns gateway and platform changes',
      memberCount: 2,
      myRole: 'TEAM_ADMIN',
      createdAt: '2026-04-07T12:00:00.000Z',
      updatedAt: '2026-04-07T12:00:00.000Z',
    };

    fetchTeams.mockResolvedValue(undefined);
    createTeam.mockResolvedValue(team);
    updateTeam.mockResolvedValue(undefined);
    deleteTeam.mockResolvedValue(undefined);
    fetchMembers.mockResolvedValue(undefined);
    addMember.mockResolvedValue(undefined);
    updateMemberRole.mockResolvedValue(undefined);
    removeMember.mockResolvedValue(undefined);
    updateMemberExpiry.mockResolvedValue(undefined);

    selectTeam.mockImplementation(async (teamId: string) => {
      if (teamId !== team.id) {
        return;
      }

      useTeamStore.setState({
        selectedTeam: team,
        members: [
          {
            userId: 'user-1',
            email: 'admin@example.com',
            username: 'admin',
            avatarData: null,
            role: 'TEAM_ADMIN',
            joinedAt: '2026-04-07T12:00:00.000Z',
            expiresAt: null,
            expired: false,
          },
        ],
        membersLoading: false,
      });
    });

    useTeamStore.setState({
      teams: [team],
      loading: false,
      selectedTeam: null,
      members: [],
      membersLoading: false,
      fetchTeams,
      createTeam,
      updateTeam,
      deleteTeam,
      selectTeam,
      clearSelectedTeam,
      fetchMembers,
      addMember,
      updateMemberRole,
      removeMember,
      updateMemberExpiry,
      reset,
    });

    searchUsers.mockResolvedValue([
      {
        id: 'user-2',
        email: 'jamie@example.com',
        username: 'jamie',
        avatarData: null,
      },
    ]);
  });

  it('auto-selects the first team and adds members from the picker', async () => {
    render(<TeamSection />);

    await waitFor(() => {
      expect(selectTeam).toHaveBeenCalledWith('team-1');
    });

    fireEvent.change(screen.getByLabelText('Add a member by name or email'), {
      target: { value: 'jamie' },
    });

    await waitFor(() => {
      expect(searchUsers).toHaveBeenCalledWith('jamie', 'tenant', undefined);
    }, { timeout: 5000 });

    fireEvent.click(await screen.findByRole('button', { name: 'Select jamie' }, { timeout: 5000 }));

    await waitFor(() => {
      expect(addMember).toHaveBeenCalledWith('team-1', 'user-2', 'TEAM_VIEWER');
    }, { timeout: 5000 });
  }, 10000);
});
