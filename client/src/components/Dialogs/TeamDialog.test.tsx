import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { TeamData } from '../../api/team.api';
import { useConnectionsStore } from '../../store/connectionsStore';
import { useTeamStore } from '../../store/teamStore';
import TeamDialog from './TeamDialog';

describe('TeamDialog', () => {
  const createTeam = vi.fn();
  const updateTeam = vi.fn();
  const fetchConnections = vi.fn();

  const existingTeam: TeamData = {
    id: 'team-1',
    name: 'Database Operations',
    description: 'Owns database proxy workflows',
    memberCount: 3,
    myRole: 'TEAM_ADMIN',
    createdAt: '2026-04-07T12:00:00.000Z',
    updatedAt: '2026-04-07T12:00:00.000Z',
  };

  beforeEach(() => {
    vi.resetAllMocks();

    useTeamStore.setState({
      createTeam,
      updateTeam,
    });

    useConnectionsStore.setState({
      fetchConnections,
    });

    createTeam.mockResolvedValue(existingTeam);
    updateTeam.mockResolvedValue(undefined);
    fetchConnections.mockResolvedValue(undefined);
  });

  it('creates a team and refreshes connection state', async () => {
    const onClose = vi.fn();

    render(
      <TeamDialog
        open
        onClose={onClose}
      />,
    );

    fireEvent.change(screen.getByLabelText('Team name'), {
      target: { value: 'Platform Operations' },
    });
    fireEvent.change(screen.getByLabelText('Description'), {
      target: { value: 'Owns platform and gateway access' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Create Team' }));

    await waitFor(() => {
      expect(createTeam).toHaveBeenCalledWith(
        'Platform Operations',
        'Owns platform and gateway access',
      );
    });
    expect(fetchConnections).toHaveBeenCalledTimes(1);
    await waitFor(() => {
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it('updates only changed fields in edit mode', async () => {
    render(
      <TeamDialog
        open
        onClose={() => {}}
        team={existingTeam}
      />,
    );

    fireEvent.change(screen.getByLabelText('Team name'), {
      target: { value: 'Database Platform' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(updateTeam).toHaveBeenCalledWith('team-1', { name: 'Database Platform' });
    });
    expect(fetchConnections).not.toHaveBeenCalled();
  });
});
