import { useState, useEffect, useCallback } from 'react';
import {
  Dialog, DialogTitle, DialogContent, DialogActions,
  Button, Alert, CircularProgress, Box, Typography,
  Switch, FormControlLabel, Chip, Divider,
} from '@mui/material';
import {
  getUserPermissions, updateUserPermissions,
  type PermissionFlag, type UserPermissionsData,
} from '../../api/tenant.api';
import { extractApiError } from '../../utils/apiError';
import { ROLE_LABELS, type TenantRole } from '../../utils/roles';

const PERMISSION_LABELS: Record<PermissionFlag, string> = {
  canConnect: 'Connect to machines',
  canCreateConnections: 'Create connections',
  canManageConnections: 'Manage connections',
  canViewCredentials: 'View credentials',
  canShareConnections: 'Share connections',
  canViewAuditLog: 'View audit log',
  canManageSessions: 'Manage sessions',
  canManageGateways: 'Manage gateways',
  canManageUsers: 'Manage users',
  canManageSecrets: 'Manage secrets',
  canManageTenantSettings: 'Manage tenant settings',
};

const ALL_FLAGS: PermissionFlag[] = [
  'canConnect', 'canCreateConnections', 'canManageConnections',
  'canViewCredentials', 'canShareConnections', 'canViewAuditLog',
  'canManageSessions', 'canManageGateways', 'canManageUsers',
  'canManageSecrets', 'canManageTenantSettings',
];

interface PermissionOverridesDialogProps {
  open: boolean;
  onClose: () => void;
  tenantId: string;
  userId: string;
  userName: string;
}

export default function PermissionOverridesDialog({
  open, onClose, tenantId, userId, userName,
}: PermissionOverridesDialogProps) {
  const [data, setData] = useState<UserPermissionsData | null>(null);
  const [localOverrides, setLocalOverrides] = useState<Record<string, boolean>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const result = await getUserPermissions(tenantId, userId);
      setData(result);
      setLocalOverrides(result.overrides ?? {});
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to load permissions'));
    } finally {
      setLoading(false);
    }
  }, [tenantId, userId]);

  useEffect(() => {
    if (open) load();
  }, [open, load]);

  const isOwner = data?.role === 'OWNER';

  const getRoleDefault = (flag: PermissionFlag): boolean => {
    return data?.defaults[flag] ?? false;
  };

  const handleToggle = (flag: PermissionFlag, checked: boolean) => {
    setLocalOverrides((prev) => {
      const roleDefault = getRoleDefault(flag);
      if (roleDefault === checked) {
        // Remove the override — value matches role default
        const { [flag]: _, ...rest } = prev;
        void _;
        return rest;
      }
      return { ...prev, [flag]: checked };
    });
  };

  const getEffectiveValue = (flag: PermissionFlag): boolean => {
    if (flag in localOverrides) return localOverrides[flag];
    return getRoleDefault(flag);
  };

  const isOverridden = (flag: PermissionFlag): boolean => {
    return flag in localOverrides;
  };

  const handleSave = async () => {
    setSaving(true);
    setError('');
    try {
      const overrides = Object.keys(localOverrides).length > 0 ? localOverrides : null;
      const result = await updateUserPermissions(tenantId, userId, overrides);
      setData(result);
      setLocalOverrides(result.overrides ?? {});
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
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        Permissions — {userName}
        {data && (
          <Chip
            label={ROLE_LABELS[data.role as TenantRole] ?? data.role}
            size="small"
            variant="outlined"
            sx={{ ml: 1, verticalAlign: 'middle' }}
          />
        )}
      </DialogTitle>
      <DialogContent>
        {error && <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError('')}>{error}</Alert>}
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
            <CircularProgress />
          </Box>
        ) : data ? (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, mt: 1 }}>
            {isOwner && (
              <Alert severity="info" sx={{ mb: 1 }}>
                Owner permissions cannot be reduced.
              </Alert>
            )}
            <Typography variant="caption" color="text.secondary" sx={{ mb: 1 }}>
              Overridden permissions are highlighted. Toggle to grant or revoke individual flags.
            </Typography>
            <Divider sx={{ mb: 1 }} />
            {ALL_FLAGS.map((flag) => {
              const effective = getEffectiveValue(flag);
              const overridden = isOverridden(flag);
              return (
                <FormControlLabel
                  key={flag}
                  disabled={isOwner && getRoleDefault(flag)}
                  control={
                    <Switch
                      size="small"
                      checked={effective}
                      onChange={(_, checked) => handleToggle(flag, checked)}
                    />
                  }
                  label={
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Typography
                        variant="body2"
                        sx={{ fontWeight: overridden ? 600 : 400 }}
                      >
                        {PERMISSION_LABELS[flag]}
                      </Typography>
                      {overridden && (
                        <Chip label="overridden" size="small" color="warning" variant="outlined" sx={{ height: 20, fontSize: '0.65rem' }} />
                      )}
                    </Box>
                  }
                  sx={{
                    mx: 0,
                    py: 0.25,
                    bgcolor: overridden ? 'action.hover' : 'transparent',
                    borderRadius: 1,
                    px: 1,
                  }}
                />
              );
            })}
          </Box>
        ) : null}
      </DialogContent>
      <DialogActions>
        {data && Object.keys(localOverrides).length > 0 && (
          <Button color="warning" onClick={handleReset} disabled={saving}>
            Reset to Role Defaults
          </Button>
        )}
        <Box sx={{ flex: 1 }} />
        <Button onClick={onClose}>Cancel</Button>
        <Button variant="contained" onClick={handleSave} disabled={saving || loading}>
          {saving ? 'Saving...' : 'Save'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}
