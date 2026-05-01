import { useCallback, useEffect, useMemo, useState } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Switch } from '@/components/ui/switch';
import {
  getUserPermissions,
  updateUserPermissions,
  type PermissionFlag,
  type UserPermissionsData,
} from '../../api/tenant.api';
import { extractApiError } from '../../utils/apiError';
import { ROLE_LABELS, type TenantRole } from '../../utils/roles';
import {
  SettingsLoadingState,
  SettingsSectionBlock,
  SettingsStatusBadge,
  SettingsSummaryGrid,
  SettingsSummaryItem,
} from './settings-ui';

type EditablePermissionFlag = Exclude<PermissionFlag, 'canManageSessions'>;

const EDITABLE_PERMISSION_FLAGS: EditablePermissionFlag[] = [
  'canConnect',
  'canCreateConnections',
  'canManageConnections',
  'canViewCredentials',
  'canShareConnections',
  'canViewAuditLog',
  'canViewSessions',
  'canObserveSessions',
  'canControlSessions',
  'canManageGateways',
  'canManageUsers',
  'canManageSecrets',
  'canManageTenantSettings',
];

const PERMISSION_LABELS: Record<EditablePermissionFlag, string> = {
  canConnect: 'Connect to machines',
  canCreateConnections: 'Create connections',
  canManageConnections: 'Manage connections',
  canViewCredentials: 'View credentials',
  canShareConnections: 'Share connections',
  canViewAuditLog: 'View audit log',
  canViewSessions: 'View active sessions',
  canObserveSessions: 'Observe live sessions',
  canControlSessions: 'Control active sessions',
  canManageGateways: 'Manage gateways',
  canManageUsers: 'Manage users',
  canManageSecrets: 'Manage secrets',
  canManageTenantSettings: 'Manage tenant settings',
};

const PERMISSION_DESCRIPTIONS: Record<EditablePermissionFlag, string> = {
  canConnect: 'Allow the user to start remote sessions and database connections.',
  canCreateConnections: 'Allow creating new saved connection records.',
  canManageConnections: 'Allow editing and deleting saved connections.',
  canViewCredentials: 'Allow seeing stored credentials and secret-backed values.',
  canShareConnections: 'Allow granting other people access to shared resources.',
  canViewAuditLog: 'Allow opening organization-wide audit history.',
  canViewSessions: 'Allow opening the Active Sessions workspace and reviewing current session state.',
  canObserveSessions: 'Allow live session observation capabilities when observation tools are enabled.',
  canControlSessions: 'Allow pausing, resuming, and terminating active sessions.',
  canManageGateways: 'Allow editing gateway infrastructure and templates.',
  canManageUsers: 'Allow inviting, editing, and removing members.',
  canManageSecrets: 'Allow managing secrets in the keychain and external vault links.',
  canManageTenantSettings: 'Allow editing organization-level settings and policy.',
};

const PERMISSION_GROUPS: Array<{
  description: string;
  flags: EditablePermissionFlag[];
  title: string;
}> = [
  {
    title: 'Session access',
    description: 'What the user can open and directly use.',
    flags: ['canConnect', 'canViewCredentials', 'canShareConnections'],
  },
  {
    title: 'Workspace management',
    description: 'What the user can create or change in the shared workspace.',
    flags: ['canCreateConnections', 'canManageConnections', 'canManageSecrets'],
  },
  {
    title: 'Operations & review',
    description: 'Operational control over sessions, audit visibility, and gateway health.',
    flags: ['canViewAuditLog', 'canViewSessions', 'canObserveSessions', 'canControlSessions', 'canManageGateways'],
  },
  {
    title: 'Administration',
    description: 'Administrative control over members and tenant-wide settings.',
    flags: ['canManageUsers', 'canManageTenantSettings'],
  },
];

interface PermissionOverridesDialogProps {
  open: boolean;
  onClose: () => void;
  tenantId: string;
  userId: string;
  userName: string;
}

export default function PermissionOverridesDialog({
  open,
  onClose,
  tenantId,
  userId,
  userName,
}: PermissionOverridesDialogProps) {
  const [data, setData] = useState<UserPermissionsData | null>(null);
  const [localOverrides, setLocalOverrides] = useState<Partial<Record<EditablePermissionFlag, boolean>>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');

    try {
      const result = await getUserPermissions(tenantId, userId);
      setData(result);
      setLocalOverrides(sanitizePermissionOverrides(result.overrides));
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to load permissions'));
    } finally {
      setLoading(false);
    }
  }, [tenantId, userId]);

  useEffect(() => {
    if (open) {
      void load();
    }
  }, [load, open]);

  const isOwner = data?.role === 'OWNER';
  const overrideCount = Object.keys(localOverrides).length;

  const summary = useMemo(() => {
    if (!data) {
      return { granted: 0, inherited: 0, overridden: 0 };
    }

    const flags = PERMISSION_GROUPS.flatMap((group) => group.flags);
    return flags.reduce(
      (accumulator, flag) => {
        const effective = flag in localOverrides ? localOverrides[flag] : (data.defaults[flag] ?? false);
        if (effective) {
          accumulator.granted += 1;
        }
        if (flag in localOverrides) {
          accumulator.overridden += 1;
        } else {
          accumulator.inherited += 1;
        }
        return accumulator;
      },
      { granted: 0, inherited: 0, overridden: 0 },
    );
  }, [data, localOverrides]);

  const getRoleDefault = (flag: EditablePermissionFlag) => data?.defaults[flag] ?? false;

  const getEffectiveValue = (flag: EditablePermissionFlag) =>
    (flag in localOverrides ? localOverrides[flag] : getRoleDefault(flag));

  const isOverridden = (flag: EditablePermissionFlag) => flag in localOverrides;

  const handleToggle = (flag: EditablePermissionFlag, checked: boolean) => {
    setLocalOverrides((current) => {
      const roleDefault = getRoleDefault(flag);
      if (roleDefault === checked) {
        const { [flag]: _removed, ...rest } = current;
        void _removed;
        return rest;
      }
      return { ...current, [flag]: checked };
    });
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');

    try {
      const overrides = overrideCount > 0 ? localOverrides : null;
      const result = await updateUserPermissions(tenantId, userId, overrides);
      setData(result);
      setLocalOverrides(sanitizePermissionOverrides(result.overrides));
      onClose();
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to update permissions'));
    } finally {
      setSaving(false);
    }
  };

  const handleReset = async () => {
    setSaving(true);
    setError('');

    try {
      const result = await updateUserPermissions(tenantId, userId, null);
      setData(result);
      setLocalOverrides({});
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to reset permissions'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}>
      <DialogContent className="max-h-[92vh] overflow-y-auto sm:max-w-4xl">
        <DialogHeader>
          <DialogTitle>Permission overrides for {userName}</DialogTitle>
          <DialogDescription>
            Start from the member&apos;s role defaults, then only override the few permissions that need an exception.
          </DialogDescription>
        </DialogHeader>

        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {loading ? (
          <SettingsLoadingState message="Loading effective permissions..." />
        ) : data ? (
          <div className="space-y-5">
            <SettingsSummaryGrid className="xl:grid-cols-4">
              <SettingsSummaryItem
                label="Role"
                value={ROLE_LABELS[data.role as TenantRole] ?? data.role}
              />
              <SettingsSummaryItem label="Granted flags" value={summary.granted} />
              <SettingsSummaryItem label="Inherited" value={summary.inherited} />
              <SettingsSummaryItem label="Overrides" value={summary.overridden} />
            </SettingsSummaryGrid>

            {isOwner && (
              <Alert>
                <AlertDescription>
                  Organization owners keep their role-level permissions. You can still review the effective policy, but owner defaults cannot be reduced here.
                </AlertDescription>
              </Alert>
            )}

            {PERMISSION_GROUPS.map((group) => (
              <SettingsSectionBlock
                key={group.title}
                title={group.title}
                description={group.description}
              >
                <div className="space-y-3">
                  {group.flags.map((flag) => {
                    const effective = getEffectiveValue(flag);
                    const overridden = isOverridden(flag);
                    const lockedByRole = Boolean(isOwner && getRoleDefault(flag));

                    return (
                      <label
                        key={flag}
                        className="flex flex-col gap-4 rounded-xl border border-border/70 bg-background/70 px-4 py-4 lg:flex-row lg:items-start lg:justify-between"
                      >
                        <div className="space-y-2">
                          <div className="flex flex-wrap items-center gap-2">
                            <div className="text-sm font-medium text-foreground">
                              {PERMISSION_LABELS[flag]}
                            </div>
                            {overridden && (
                              <SettingsStatusBadge tone="warning">Overridden</SettingsStatusBadge>
                            )}
                            <SettingsStatusBadge tone={getRoleDefault(flag) ? 'success' : 'neutral'}>
                              Role default: {getRoleDefault(flag) ? 'Allowed' : 'Blocked'}
                            </SettingsStatusBadge>
                            {lockedByRole && (
                              <SettingsStatusBadge tone="neutral">Owner locked</SettingsStatusBadge>
                            )}
                          </div>
                          <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
                            {PERMISSION_DESCRIPTIONS[flag]}
                          </p>
                        </div>

                        <div className="flex items-center gap-3 lg:shrink-0">
                          <SettingsStatusBadge tone={effective ? 'success' : 'neutral'}>
                            {effective ? 'Allowed' : 'Blocked'}
                          </SettingsStatusBadge>
                          <Switch
                            checked={effective}
                            disabled={lockedByRole || saving}
                            aria-label={PERMISSION_LABELS[flag]}
                            onCheckedChange={(checked) => handleToggle(flag, checked)}
                          />
                        </div>
                      </label>
                    );
                  })}
                </div>
              </SettingsSectionBlock>
            ))}
          </div>
        ) : null}

        <DialogFooter className="sm:justify-between">
          <div>
            {overrideCount > 0 && (
              <Button type="button" variant="outline" onClick={handleReset} disabled={saving}>
                Reset to role defaults
              </Button>
            )}
          </div>
          <div className="flex gap-2">
            <Button type="button" variant="outline" onClick={onClose} disabled={saving}>
              Cancel
            </Button>
            <Button type="button" onClick={handleSave} disabled={saving || loading || !data}>
              {saving ? 'Saving...' : 'Save overrides'}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function sanitizePermissionOverrides(
  overrides: Record<string, boolean> | null | undefined,
): Partial<Record<EditablePermissionFlag, boolean>> {
  if (!overrides) {
    return {};
  }

  return EDITABLE_PERMISSION_FLAGS.reduce<Partial<Record<EditablePermissionFlag, boolean>>>((accumulator, flag) => {
    if (flag in overrides) {
      accumulator[flag] = overrides[flag];
    }
    return accumulator;
  }, {});
}
