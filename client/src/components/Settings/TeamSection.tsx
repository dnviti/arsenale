import { useEffect, useMemo, useState } from 'react';
import { Plus } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { TeamData, TeamMember } from '../../api/team.api';
import type { UserSearchResult } from '../../api/user.api';
import { useAuthStore } from '../../store/authStore';
import { useTeamStore } from '../../store/teamStore';
import { extractApiError } from '../../utils/apiError';
import TeamDialog from '../Dialogs/TeamDialog';
import UserPicker from '../UserPicker';
import {
  SettingsFieldCard,
  SettingsLoadingState,
  SettingsPanel,
  SettingsSummaryGrid,
  SettingsSummaryItem,
  SettingsStatusBadge,
} from './settings-ui';
import {
  TeamConfirmDialog,
  TeamDetailPlaceholder,
  TeamDirectory,
  TeamEmptyState,
  TeamMemberExpiryDialog,
  TeamMembersList,
  TeamTenantRequiredState,
} from './teamSectionUi';
import {
  TEAM_ROLES,
  type TeamRole,
  formatTeamDate,
  fromDateTimeLocalValue,
  getTeamMemberName,
  roleLabel,
  toDateTimeLocalValue,
} from './teamSectionUtils';

interface TeamSectionProps {
  onNavigateToTab?: (tabId: string) => void;
}

interface DeleteTarget {
  id: string;
  name: string;
}

interface MemberActionTarget {
  teamId: string;
  userId: string;
  name: string;
}

interface MemberExpiryTarget extends MemberActionTarget {
  expiresAt: string | null;
}

export default function TeamSection({ onNavigateToTab }: TeamSectionProps) {
  const user = useAuthStore((state) => state.user);
  const teams = useTeamStore((state) => state.teams);
  const loading = useTeamStore((state) => state.loading);
  const selectedTeam = useTeamStore((state) => state.selectedTeam);
  const members = useTeamStore((state) => state.members);
  const membersLoading = useTeamStore((state) => state.membersLoading);
  const fetchTeams = useTeamStore((state) => state.fetchTeams);
  const selectTeam = useTeamStore((state) => state.selectTeam);
  const clearSelectedTeam = useTeamStore((state) => state.clearSelectedTeam);
  const deleteTeam = useTeamStore((state) => state.deleteTeam);
  const addMember = useTeamStore((state) => state.addMember);
  const updateMemberRole = useTeamStore((state) => state.updateMemberRole);
  const removeMember = useTeamStore((state) => state.removeMember);
  const updateMemberExpiry = useTeamStore((state) => state.updateMemberExpiry);

  const [teamDialogOpen, setTeamDialogOpen] = useState(false);
  const [editingTeam, setEditingTeam] = useState<TeamData | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<DeleteTarget | null>(null);
  const [removeTarget, setRemoveTarget] = useState<MemberActionTarget | null>(null);
  const [expiryTarget, setExpiryTarget] = useState<MemberExpiryTarget | null>(null);
  const [expiryValue, setExpiryValue] = useState('');
  const [expiryError, setExpiryError] = useState('');
  const [error, setError] = useState('');
  const [addMemberRole, setAddMemberRole] = useState<TeamRole>('TEAM_VIEWER');
  const [addingMember, setAddingMember] = useState(false);
  const [deletingTeam, setDeletingTeam] = useState(false);
  const [savingExpiry, setSavingExpiry] = useState(false);
  const [updatingUserId, setUpdatingUserId] = useState<string | null>(null);
  const [removingUserId, setRemovingUserId] = useState<string | null>(null);

  const hasTenant = Boolean(user?.tenantId);
  const isTeamAdmin = selectedTeam?.myRole === 'TEAM_ADMIN';
  const totalMembers = useMemo(
    () => teams.reduce((count, team) => count + team.memberCount, 0),
    [teams],
  );
  const currentRoleLabel = selectedTeam ? roleLabel(selectedTeam.myRole) : 'No team selected';
  const existingMemberIds = useMemo(() => members.map((member) => member.userId), [members]);

  useEffect(() => {
    if (!hasTenant) {
      clearSelectedTeam();
      return;
    }

    void fetchTeams();
  }, [clearSelectedTeam, fetchTeams, hasTenant]);

  useEffect(() => {
    if (!hasTenant || loading || selectedTeam || teams.length === 0) {
      return;
    }

    void selectTeam(teams[0].id);
  }, [hasTenant, loading, selectTeam, selectedTeam, teams]);

  const handleSelectTeam = async (team: TeamData) => {
    setError('');
    await selectTeam(team.id);
  };

  const handleDeleteTeam = async () => {
    if (!deleteTarget) {
      return;
    }

    setDeletingTeam(true);
    setError('');
    try {
      await deleteTeam(deleteTarget.id);
      if (selectedTeam?.id === deleteTarget.id) {
        clearSelectedTeam();
      }
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to delete team'));
    } finally {
      setDeletingTeam(false);
      setDeleteTarget(null);
    }
  };

  const handleAddMember = async (selectedUser: UserSearchResult | null) => {
    if (!selectedUser || !selectedTeam) {
      return;
    }

    setAddingMember(true);
    setError('');
    try {
      await addMember(selectedTeam.id, selectedUser.id, addMemberRole);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to add member'));
    } finally {
      setAddingMember(false);
    }
  };

  const handleRoleChange = async (member: TeamMember, nextRole: TeamRole) => {
    if (!selectedTeam) {
      return;
    }

    setUpdatingUserId(member.userId);
    setError('');
    try {
      await updateMemberRole(selectedTeam.id, member.userId, nextRole);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to update role'));
    } finally {
      setUpdatingUserId(null);
    }
  };

  const handleRemoveMember = async () => {
    if (!removeTarget) {
      return;
    }

    setRemovingUserId(removeTarget.userId);
    setError('');
    try {
      await removeMember(removeTarget.teamId, removeTarget.userId);
      setRemoveTarget(null);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to remove member'));
    } finally {
      setRemovingUserId(null);
    }
  };

  const openExpiryDialog = (member: TeamMember) => {
    if (!selectedTeam) {
      return;
    }

    setExpiryTarget({
      teamId: selectedTeam.id,
      userId: member.userId,
      name: getTeamMemberName(member),
      expiresAt: member.expiresAt,
    });
    setExpiryValue(toDateTimeLocalValue(member.expiresAt));
    setExpiryError('');
  };

  const closeExpiryDialog = () => {
    setExpiryTarget(null);
    setExpiryValue('');
    setExpiryError('');
  };

  const handleSaveExpiry = async () => {
    if (!expiryTarget) {
      return;
    }

    const nextExpiry = fromDateTimeLocalValue(expiryValue);
    if (!nextExpiry) {
      setExpiryError('Choose a valid expiration date and time.');
      return;
    }

    setSavingExpiry(true);
    setError('');
    setExpiryError('');
    try {
      await updateMemberExpiry(expiryTarget.teamId, expiryTarget.userId, nextExpiry);
      closeExpiryDialog();
    } catch (err: unknown) {
      const message = extractApiError(err, 'Failed to update member expiration');
      setError(message);
      setExpiryError(message);
    } finally {
      setSavingExpiry(false);
    }
  };

  const handleClearExpiry = async () => {
    if (!expiryTarget) {
      return;
    }

    setSavingExpiry(true);
    setError('');
    setExpiryError('');
    try {
      await updateMemberExpiry(expiryTarget.teamId, expiryTarget.userId, null);
      closeExpiryDialog();
    } catch (err: unknown) {
      const message = extractApiError(err, 'Failed to remove member expiration');
      setError(message);
      setExpiryError(message);
    } finally {
      setSavingExpiry(false);
    }
  };

  if (!hasTenant) {
    return (
      <TeamTenantRequiredState
        onNavigateToOrganization={() => onNavigateToTab?.('organization')}
      />
    );
  }

  return (
    <>
      <SettingsPanel
        title="Team Collaboration"
        description="Organize shared access by team instead of scattering roles and temporary membership changes across unrelated menus."
        heading={(
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => {
              setEditingTeam(null);
              setTeamDialogOpen(true);
            }}
          >
            <Plus />
            New Team
          </Button>
        )}
        contentClassName="space-y-5"
      >
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {loading ? (
          <SettingsLoadingState message="Loading teams..." />
        ) : teams.length === 0 ? (
          <TeamEmptyState onCreate={() => setTeamDialogOpen(true)} />
        ) : (
          <>
            <SettingsSummaryGrid>
              <SettingsSummaryItem
                label="Teams"
                value={`${teams.length} configured`}
              />
              <SettingsSummaryItem
                label="Members"
                value={`${totalMembers} memberships`}
              />
              <SettingsSummaryItem
                label="Selected Role"
                value={currentRoleLabel}
              />
              <SettingsSummaryItem
                label="Last Updated"
                value={selectedTeam ? formatTeamDate(selectedTeam.updatedAt || selectedTeam.createdAt) : 'Select a team'}
              />
            </SettingsSummaryGrid>

            <div className="grid gap-4 xl:grid-cols-[340px_minmax(0,1fr)]">
              <TeamDirectory
                teams={teams}
                selectedTeamId={selectedTeam?.id || null}
                onSelect={(team) => {
                  void handleSelectTeam(team);
                }}
                onEdit={(team) => {
                  setEditingTeam(team);
                  setTeamDialogOpen(true);
                }}
                onDelete={(team) => setDeleteTarget({ id: team.id, name: team.name })}
              />

              {selectedTeam ? (
                <div className="space-y-4 rounded-2xl border border-border/70 bg-background/40 p-4 md:p-5">
                  <div className="space-y-3">
                    <div className="space-y-1">
                      <h3 className="text-lg font-semibold text-foreground">{selectedTeam.name}</h3>
                      <p className="text-sm leading-6 text-muted-foreground">
                        {selectedTeam.description || 'Manage members, role assignments, and temporary access windows for this team.'}
                      </p>
                    </div>

                    <div className="flex flex-wrap gap-2">
                      <SettingsStatusBadge tone="neutral">
                        {selectedTeam.memberCount} member{selectedTeam.memberCount === 1 ? '' : 's'}
                      </SettingsStatusBadge>
                      <SettingsStatusBadge tone={isTeamAdmin ? 'success' : 'neutral'}>
                        {roleLabel(selectedTeam.myRole)}
                      </SettingsStatusBadge>
                    </div>
                  </div>

                  {isTeamAdmin && (
                    <SettingsFieldCard
                      label="Add members"
                      description="Search the organization once, choose the default team role, and add the member directly from here."
                    >
                      <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_180px]">
                        <UserPicker
                          scope="tenant"
                          placeholder="Add a member by name or email"
                          excludeUserIds={existingMemberIds}
                          clearAfterSelect
                          onSelect={(selectedUser) => {
                            void handleAddMember(selectedUser);
                          }}
                        />

                        <div className="space-y-2">
                          <Label htmlFor="team-member-role">Member role</Label>
                          <Select
                            value={addMemberRole}
                            onValueChange={(value) => setAddMemberRole(value as TeamRole)}
                          >
                            <SelectTrigger id="team-member-role" aria-label="Member role">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              {TEAM_ROLES.map((role) => (
                                <SelectItem key={role} value={role}>
                                  {roleLabel(role)}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </div>
                      </div>

                      {addingMember && (
                        <SettingsLoadingState message="Adding member..." />
                      )}
                    </SettingsFieldCard>
                  )}

                  {membersLoading ? (
                    <SettingsLoadingState message="Loading members..." />
                  ) : (
                    <TeamMembersList
                      members={members}
                      currentUserId={user?.id}
                      canManageMembers={Boolean(isTeamAdmin)}
                      updatingUserId={updatingUserId || removingUserId}
                      onRoleChange={(member, role) => {
                        void handleRoleChange(member, role);
                      }}
                      onEditExpiry={openExpiryDialog}
                      onRemoveMember={(member) => setRemoveTarget({
                        teamId: selectedTeam.id,
                        userId: member.userId,
                        name: getTeamMemberName(member),
                      })}
                    />
                  )}
                </div>
              ) : (
                <TeamDetailPlaceholder />
              )}
            </div>
          </>
        )}
      </SettingsPanel>

      <TeamDialog
        open={teamDialogOpen}
        onClose={() => {
          setTeamDialogOpen(false);
          setEditingTeam(null);
        }}
        team={editingTeam}
      />

      <TeamConfirmDialog
        open={Boolean(deleteTarget)}
        title="Delete team"
        description={deleteTarget
          ? `Delete ${deleteTarget.name}? Team-owned folders and connections will become unassigned.`
          : ''}
        confirmLabel="Delete team"
        busy={deletingTeam}
        onConfirm={() => {
          void handleDeleteTeam();
        }}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteTarget(null);
          }
        }}
      />

      <TeamConfirmDialog
        open={Boolean(removeTarget)}
        title="Remove member"
        description={removeTarget
          ? `Remove ${removeTarget.name} from this team? Their team-scoped access will be revoked immediately.`
          : ''}
        confirmLabel="Remove member"
        busy={Boolean(removingUserId)}
        onConfirm={() => {
          void handleRemoveMember();
        }}
        onOpenChange={(open) => {
          if (!open) {
            setRemoveTarget(null);
          }
        }}
      />

      <TeamMemberExpiryDialog
        open={Boolean(expiryTarget)}
        memberName={expiryTarget?.name || 'this member'}
        value={expiryValue}
        error={expiryError}
        saving={savingExpiry}
        canClear={Boolean(expiryTarget?.expiresAt)}
        onValueChange={setExpiryValue}
        onSave={() => {
          void handleSaveExpiry();
        }}
        onClear={() => {
          void handleClearExpiry();
        }}
        onOpenChange={(open) => {
          if (!open) {
            closeExpiryDialog();
          }
        }}
      />
    </>
  );
}
