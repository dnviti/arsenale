import { Building2, Clock3, PencilLine, Shield, Trash2, Users } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn } from '@/lib/utils';
import type { TeamData, TeamMember } from '../../api/team.api';
import {
  SettingsButtonRow,
  SettingsStatusBadge,
} from './settings-ui';
import {
  TEAM_ROLES,
  type TeamRole,
  formatMemberExpiry,
  formatTeamDate,
  getInitials,
  getTeamMemberName,
  roleLabel,
} from './teamSectionUtils';

export function TeamTenantRequiredState({
  onNavigateToOrganization,
}: {
  onNavigateToOrganization: () => void;
}) {
  return (
    <Card className="border-dashed border-border/70 bg-muted/10 p-8 text-center">
      <div className="mx-auto flex max-w-xl flex-col items-center gap-4">
        <div className="flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
          <Building2 className="size-6" />
        </div>
        <div className="space-y-2">
          <h3 className="text-lg font-semibold text-foreground">Create or join an organization first</h3>
          <p className="text-sm leading-6 text-muted-foreground">
            Teams inherit organization membership. Finish your organization setup before you manage
            delegated access here.
          </p>
        </div>
        <Button type="button" onClick={onNavigateToOrganization}>
          Set Up Organization
        </Button>
      </div>
    </Card>
  );
}

export function TeamEmptyState({ onCreate }: { onCreate: () => void }) {
  return (
    <Card className="border-dashed border-border/70 bg-muted/10 p-8 text-center">
      <div className="mx-auto flex max-w-xl flex-col items-center gap-4">
        <div className="flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
          <Users className="size-6" />
        </div>
        <div className="space-y-2">
          <h3 className="text-lg font-semibold text-foreground">No teams yet</h3>
          <p className="text-sm leading-6 text-muted-foreground">
            Create a team when a group needs shared ownership, scoped collaboration, or temporary
            member access windows.
          </p>
        </div>
        <Button type="button" onClick={onCreate}>
          Create Team
        </Button>
      </div>
    </Card>
  );
}

export function TeamDirectory({
  teams,
  selectedTeamId,
  onSelect,
  onEdit,
  onDelete,
}: {
  teams: TeamData[];
  selectedTeamId: string | null;
  onSelect: (team: TeamData) => void;
  onEdit: (team: TeamData) => void;
  onDelete: (team: TeamData) => void;
}) {
  return (
    <div className="space-y-4 rounded-2xl border border-border/70 bg-background/40 p-4 md:p-5">
      <div className="space-y-1">
        <h3 className="text-sm font-semibold text-foreground">Team directory</h3>
        <p className="text-sm leading-6 text-muted-foreground">
          Keep teams short, role-driven, and easy to scan during sharing or operational reviews.
        </p>
      </div>

      <ScrollArea className="max-h-[32rem] pr-3">
        <div className="space-y-3">
          {teams.map((team) => {
            const isSelected = selectedTeamId === team.id;

            return (
              <Card
                key={team.id}
                className={cn(
                  'border-border/70 bg-background/70 p-4 transition-colors',
                  isSelected && 'border-primary/40 bg-primary/5 shadow-sm',
                )}
              >
                <div className="space-y-4">
                  <button
                    type="button"
                    className="w-full space-y-3 text-left"
                    onClick={() => onSelect(team)}
                  >
                    <div className="space-y-1">
                      <div className="text-sm font-semibold text-foreground">{team.name}</div>
                      <p className="line-clamp-2 text-sm leading-6 text-muted-foreground">
                        {team.description || 'No description yet.'}
                      </p>
                    </div>

                    <div className="flex flex-wrap gap-2">
                      <SettingsStatusBadge tone={isSelected ? 'success' : 'neutral'}>
                        {team.memberCount} member{team.memberCount === 1 ? '' : 's'}
                      </SettingsStatusBadge>
                      <SettingsStatusBadge tone={team.myRole === 'TEAM_ADMIN' ? 'success' : 'neutral'}>
                        {roleLabel(team.myRole)}
                      </SettingsStatusBadge>
                    </div>

                    <div className="text-xs text-muted-foreground">
                      Updated {formatTeamDate(team.updatedAt || team.createdAt)}
                    </div>
                  </button>

                  {team.myRole === 'TEAM_ADMIN' && (
                    <SettingsButtonRow className="justify-end">
                      <Button type="button" variant="outline" size="sm" onClick={() => onEdit(team)}>
                        <PencilLine className="size-4" />
                        Edit
                      </Button>
                      <Button type="button" variant="outline" size="sm" onClick={() => onDelete(team)}>
                        <Trash2 className="size-4" />
                        Delete
                      </Button>
                    </SettingsButtonRow>
                  )}
                </div>
              </Card>
            );
          })}
        </div>
      </ScrollArea>
    </div>
  );
}

export function TeamDetailPlaceholder() {
  return (
    <Card className="border-dashed border-border/70 bg-muted/10 p-8 text-center">
      <div className="mx-auto max-w-lg space-y-2">
        <h3 className="text-lg font-semibold text-foreground">Select a team</h3>
        <p className="text-sm leading-6 text-muted-foreground">
          Choose a team from the directory to manage members, role assignments, and temporary
          expiration windows.
        </p>
      </div>
    </Card>
  );
}

function TeamRoleSelect({
  value,
  onChange,
  ariaLabel,
  disabled = false,
}: {
  value: string;
  onChange: (value: TeamRole) => void;
  ariaLabel: string;
  disabled?: boolean;
}) {
  return (
    <Select value={value} onValueChange={(nextValue) => onChange(nextValue as TeamRole)} disabled={disabled}>
      <SelectTrigger aria-label={ariaLabel} className="w-full sm:w-[160px]">
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
  );
}

export function TeamMembersList({
  members,
  currentUserId,
  canManageMembers,
  updatingUserId,
  onRoleChange,
  onEditExpiry,
  onRemoveMember,
}: {
  members: TeamMember[];
  currentUserId?: string;
  canManageMembers: boolean;
  updatingUserId?: string | null;
  onRoleChange: (member: TeamMember, role: TeamRole) => void;
  onEditExpiry: (member: TeamMember) => void;
  onRemoveMember: (member: TeamMember) => void;
}) {
  if (members.length === 0) {
    return (
      <Card className="border-dashed border-border/70 bg-muted/10 p-6 text-center">
        <div className="space-y-2">
          <div className="text-sm font-semibold text-foreground">No members yet</div>
          <p className="text-sm leading-6 text-muted-foreground">
            Add people from your organization to start sharing connections, folders, and team-scoped access.
          </p>
        </div>
      </Card>
    );
  }

  return (
    <div className="space-y-3">
      {members.map((member) => {
        const isCurrentUser = member.userId === currentUserId;
        const memberName = getTeamMemberName(member);
        const memberRoleLabel = roleLabel(member.role);
        const expiryTone = member.expired ? 'destructive' : (member.expiresAt ? 'warning' : 'neutral');

        return (
          <Card key={member.userId} className="border-border/70 bg-background/70 p-4">
            <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
              <div className="flex min-w-0 items-center gap-3">
                <Avatar className="size-10">
                  <AvatarImage src={member.avatarData || undefined} alt={memberName} />
                  <AvatarFallback>{getInitials(memberName)}</AvatarFallback>
                </Avatar>
                <div className="min-w-0 space-y-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <div className="truncate text-sm font-semibold text-foreground">{memberName}</div>
                    {isCurrentUser && (
                      <SettingsStatusBadge tone="neutral">You</SettingsStatusBadge>
                    )}
                    {member.expired && (
                      <SettingsStatusBadge tone="destructive">Expired</SettingsStatusBadge>
                    )}
                  </div>
                  {member.username && (
                    <div className="truncate text-sm text-muted-foreground">{member.email}</div>
                  )}
                  <div className="text-xs text-muted-foreground">
                    Joined {formatTeamDate(member.joinedAt)}
                  </div>
                </div>
              </div>

              <div className="flex flex-col gap-3 xl:items-end">
                <div className="flex flex-wrap items-center gap-2">
                  {canManageMembers && !isCurrentUser ? (
                    <TeamRoleSelect
                      value={member.role}
                      ariaLabel={`Role for ${memberName}`}
                      disabled={updatingUserId === member.userId}
                      onChange={(role) => onRoleChange(member, role)}
                    />
                  ) : (
                    <SettingsStatusBadge tone={member.role === 'TEAM_ADMIN' ? 'success' : 'neutral'}>
                      {memberRoleLabel}
                    </SettingsStatusBadge>
                  )}

                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={!canManageMembers || isCurrentUser}
                    onClick={() => onEditExpiry(member)}
                  >
                    <Clock3 className="size-4" />
                    {member.expiresAt ? formatMemberExpiry(member.expiresAt, member.expired) : 'Set expiration'}
                  </Button>

                  {canManageMembers && !isCurrentUser && (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => onRemoveMember(member)}
                    >
                      <Trash2 className="size-4" />
                      Remove
                    </Button>
                  )}
                </div>

                <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                  <SettingsStatusBadge tone={expiryTone}>
                    {member.expiresAt ? formatMemberExpiry(member.expiresAt, member.expired) : 'No expiration'}
                  </SettingsStatusBadge>
                  {member.role === 'TEAM_ADMIN' && (
                    <SettingsStatusBadge tone="success">
                      <Shield className="mr-1 size-3.5" />
                      Admin access
                    </SettingsStatusBadge>
                  )}
                </div>
              </div>
            </div>
          </Card>
        );
      })}
    </div>
  );
}

export function TeamConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  busy = false,
  onConfirm,
  onOpenChange,
}: {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  busy?: boolean;
  onConfirm: () => void;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" variant="destructive" disabled={busy} onClick={onConfirm}>
            {busy ? 'Working...' : confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function TeamMemberExpiryDialog({
  open,
  memberName,
  value,
  error,
  saving,
  canClear,
  onValueChange,
  onSave,
  onClear,
  onOpenChange,
}: {
  open: boolean;
  memberName: string;
  value: string;
  error?: string;
  saving?: boolean;
  canClear: boolean;
  onValueChange: (value: string) => void;
  onSave: () => void;
  onClear: () => void;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Member expiration</DialogTitle>
          <DialogDescription>
            Control how long {memberName} keeps access to this team.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="team-member-expiration">Expires at</Label>
            <Input
              id="team-member-expiration"
              type="datetime-local"
              value={value}
              onChange={(event) => onValueChange(event.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Leave it empty to keep access indefinitely.
            </p>
          </div>
        </div>

        <DialogFooter className="sm:justify-between">
          <div className="flex flex-col-reverse gap-2 sm:flex-row">
            {canClear && (
              <Button type="button" variant="outline" disabled={saving} onClick={onClear}>
                Clear expiration
              </Button>
            )}
          </div>
          <div className="flex flex-col-reverse gap-2 sm:flex-row">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="button" disabled={saving || !value} onClick={onSave}>
              {saving ? 'Saving...' : 'Save expiration'}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
