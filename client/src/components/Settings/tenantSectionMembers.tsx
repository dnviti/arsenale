import { MoreHorizontal, Plus, Send, Settings2, Shield, UserMinus, UserRoundCog } from 'lucide-react';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import type { TenantUser } from '../../api/tenant.api';
import { ALL_ROLES, ROLE_LABELS, type TenantRole } from '../../utils/roles';
import {
  SettingsButtonRow,
  SettingsLoadingState,
  SettingsPanel,
  SettingsStatusBadge,
} from './settings-ui';

function getMemberName(user: TenantUser) {
  return user.username || user.email;
}

function getInitials(user: TenantUser) {
  return getMemberName(user)
    .split(/\s+/)
    .map((part) => part[0] || '')
    .join('')
    .slice(0, 2)
    .toUpperCase();
}

function formatExpiry(user: TenantUser) {
  if (!user.expiresAt) return 'No expiration';
  const label = new Date(user.expiresAt).toLocaleDateString();
  return user.expired ? `Expired ${label}` : `Expires ${label}`;
}

interface TenantMembersPanelProps {
  currentUserId?: string;
  isAdmin: boolean;
  loading: boolean;
  onChangeEmail: (user: TenantUser) => void;
  onChangePassword: (user: TenantUser) => void;
  onCreateUser: () => void;
  onEditExpiry: (user: TenantUser) => void;
  onEditPermissions: (user: TenantUser) => void;
  onInvite: () => void;
  onRemove: (user: TenantUser) => void;
  onRoleChange: (userId: string, role: TenantRole) => void;
  onToggleEnabled: (userId: string, enabled: boolean) => void;
  onViewUserProfile?: (userId: string) => void;
  togglingUserId?: string | null;
  users: TenantUser[];
}

function MemberActionMenu({
  onChangeEmail,
  onChangePassword,
  onEditExpiry,
  onRemove,
  user,
}: {
  onChangeEmail: (user: TenantUser) => void;
  onChangePassword: (user: TenantUser) => void;
  onEditExpiry: (user: TenantUser) => void;
  onRemove: (user: TenantUser) => void;
  user: TenantUser;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" variant="ghost" size="icon" className="size-9">
          <MoreHorizontal className="size-4" />
          <span className="sr-only">More actions for {getMemberName(user)}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onSelect={() => onChangeEmail(user)}>
          <UserRoundCog className="size-4" />
          Change Email
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={() => onChangePassword(user)}>
          <Shield className="size-4" />
          Change Password
        </DropdownMenuItem>
        <DropdownMenuItem onSelect={() => onEditExpiry(user)}>
          <Settings2 className="size-4" />
          {user.expiresAt ? 'Change Expiration' : 'Set Expiration'}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem className="text-destructive focus:text-destructive" onSelect={() => onRemove(user)}>
          <UserMinus className="size-4" />
          Remove Member
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export default function TenantMembersPanel({
  currentUserId,
  isAdmin,
  loading,
  onChangeEmail,
  onChangePassword,
  onCreateUser,
  onEditExpiry,
  onEditPermissions,
  onInvite,
  onRemove,
  onRoleChange,
  onToggleEnabled,
  onViewUserProfile,
  togglingUserId,
  users,
}: TenantMembersPanelProps) {
  return (
    <SettingsPanel
      title="Members"
      description="Keep people, roles, and membership lifecycle controls in one place."
      heading={
        isAdmin ? (
          <SettingsButtonRow>
            <Button type="button" variant="outline" size="sm" onClick={onInvite}>
              <Send className="size-4" />
              Invite
            </Button>
            <Button type="button" size="sm" onClick={onCreateUser}>
              <Plus className="size-4" />
              Create User
            </Button>
          </SettingsButtonRow>
        ) : null
      }
      contentClassName="space-y-4"
    >
      {loading ? (
        <SettingsLoadingState message="Loading members..." />
      ) : users.length === 0 ? (
        <Card className="border-dashed border-border/70 bg-muted/10 p-8 text-center">
          <div className="space-y-2">
            <div className="text-lg font-semibold text-foreground">No members yet</div>
            <p className="text-sm leading-6 text-muted-foreground">
              Invite collaborators once the organization structure and tenant-wide policy are in place.
            </p>
          </div>
        </Card>
      ) : (
        <div className="grid gap-4 xl:grid-cols-2">
          {users.map((member) => {
            const name = getMemberName(member);
            const isCurrentUser = member.id === currentUserId;
            const canEditRole = isAdmin && !isCurrentUser;
            const isEnabled = member.enabled !== false;

            return (
              <Card key={member.id} className="border-border/70 bg-background/70 p-5">
                <div className="space-y-4">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex min-w-0 items-center gap-3">
                      <Avatar className="size-11">
                        <AvatarImage src={member.avatarData || undefined} alt={name} />
                        <AvatarFallback>{getInitials(member)}</AvatarFallback>
                      </Avatar>
                      <div className="min-w-0 space-y-1">
                        <button
                          type="button"
                          className="max-w-full truncate text-left text-sm font-semibold text-foreground hover:underline"
                          onClick={() => onViewUserProfile?.(member.id)}
                          disabled={!onViewUserProfile}
                        >
                          {name}
                        </button>
                        <div className="truncate text-sm text-muted-foreground">{member.email}</div>
                        <div className="text-xs text-muted-foreground">
                          Joined {new Date(member.createdAt).toLocaleDateString()}
                        </div>
                      </div>
                    </div>

                    {isAdmin && !isCurrentUser ? (
                      <MemberActionMenu
                        user={member}
                        onChangeEmail={onChangeEmail}
                        onChangePassword={onChangePassword}
                        onEditExpiry={onEditExpiry}
                        onRemove={onRemove}
                      />
                    ) : null}
                  </div>

                  <div className="flex flex-wrap gap-2">
                    {isCurrentUser ? <SettingsStatusBadge tone="neutral">You</SettingsStatusBadge> : null}
                    {member.pending ? <SettingsStatusBadge tone="warning">Pending Invite</SettingsStatusBadge> : null}
                    {member.expired ? <SettingsStatusBadge tone="destructive">Expired</SettingsStatusBadge> : null}
                    <SettingsStatusBadge tone={member.totpEnabled || member.smsMfaEnabled ? 'success' : 'neutral'}>
                      {member.totpEnabled || member.smsMfaEnabled ? 'MFA Active' : 'No MFA'}
                    </SettingsStatusBadge>
                    <SettingsStatusBadge tone={isEnabled ? 'success' : 'destructive'}>
                      {isEnabled ? 'Active' : 'Disabled'}
                    </SettingsStatusBadge>
                    <SettingsStatusBadge tone={member.expired ? 'destructive' : member.expiresAt ? 'warning' : 'neutral'}>
                      {formatExpiry(member)}
                    </SettingsStatusBadge>
                  </div>

                  <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
                    <div className="space-y-2">
                      <div className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
                        Organization Role
                      </div>
                      {canEditRole ? (
                        <Select value={member.role} onValueChange={(value) => onRoleChange(member.id, value as TenantRole)}>
                          <SelectTrigger aria-label={`Role for ${name}`} className="w-full sm:w-[180px]">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            {ALL_ROLES.map((role) => (
                              <SelectItem key={role} value={role}>
                                {ROLE_LABELS[role]}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      ) : (
                        <SettingsStatusBadge tone={member.role === 'OWNER' || member.role === 'ADMIN' ? 'success' : 'neutral'}>
                          {ROLE_LABELS[member.role as TenantRole] ?? member.role}
                        </SettingsStatusBadge>
                      )}
                    </div>

                    {isAdmin ? (
                      <div className="flex flex-wrap items-center gap-3">
                        <Button type="button" variant="outline" size="sm" onClick={() => onEditPermissions(member)}>
                          <Settings2 className="size-4" />
                          Permissions
                        </Button>
                        {!isCurrentUser ? (
                          <label className="flex items-center gap-2 text-sm text-muted-foreground">
                            <span>Enabled</span>
                            <Switch
                              checked={isEnabled}
                              disabled={togglingUserId === member.id}
                              aria-label={`Toggle ${name}`}
                              onCheckedChange={(checked) => onToggleEnabled(member.id, checked)}
                            />
                          </label>
                        ) : null}
                      </div>
                    ) : null}
                  </div>
                </div>
              </Card>
            );
          })}
        </div>
      )}
    </SettingsPanel>
  );
}
