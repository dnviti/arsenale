import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  ChevronDown,
  ChevronUp,
  Edit3,
  FlaskConical,
  Loader2,
  Plus,
  RefreshCw,
  Trash2,
} from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { getSyncLogs, triggerSync, testSyncConnection } from '../../api/sync.api';
import {
  createSyncProfile,
  deleteSyncProfile,
  listSyncProfiles,
  updateSyncProfile,
} from '../../api/sync.api';
import type {
  CreateSyncProfileInput,
  SyncLogEntry,
  SyncPlanData,
  SyncProfileData,
  UpdateSyncProfileInput,
} from '../../api/sync.api';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import { useNotificationStore } from '../../store/notificationStore';
import { extractApiError } from '../../utils/apiError';
import SyncPreviewDialog from './SyncPreviewDialog';
import {
  SettingsButtonRow,
  SettingsLoadingState,
  SettingsPanel,
  SettingsStatusBadge,
  SettingsSummaryGrid,
  SettingsSummaryItem,
} from './settings-ui';

interface SyncProfileFormState {
  name: string;
  url: string;
  apiToken: string;
  filters: string;
  platformMapping: string;
  defaultProtocol: string;
  conflictStrategy: string;
  cronExpression: string;
  teamId: string;
}

const emptyForm: SyncProfileFormState = {
  name: '',
  url: '',
  apiToken: '',
  filters: '',
  platformMapping: '',
  defaultProtocol: 'SSH',
  conflictStrategy: 'update',
  cronExpression: '',
  teamId: '',
};

const syncStatusTones = {
  SUCCESS: 'success',
  ERROR: 'destructive',
  PARTIAL: 'warning',
  RUNNING: 'neutral',
  PENDING: 'neutral',
} as const;

function createFormState(profile: SyncProfileData | null): SyncProfileFormState {
  if (!profile) return emptyForm;

  return {
    name: profile.name,
    url: profile.config.url,
    apiToken: '',
    filters: Object.entries(profile.config.filters ?? {}).map(([key, value]) => `${key}=${value}`).join(', '),
    platformMapping: Object.entries(profile.config.platformMapping ?? {}).map(([key, value]) => `${key}=${value}`).join(', '),
    defaultProtocol: profile.config.defaultProtocol || 'SSH',
    conflictStrategy: profile.config.conflictStrategy || 'update',
    cronExpression: profile.cronExpression || '',
    teamId: profile.teamId || '',
  };
}

function parseKeyValuePairs(input: string): Record<string, string> {
  if (!input.trim()) return {};

  return Object.fromEntries(
    input
      .split(',')
      .map((pair) => {
        const [key, ...rest] = pair.split('=');
        return [key.trim(), rest.join('=').trim()];
      })
      .filter(([key]) => key),
  );
}

function formatDateTime(value: string | null): string {
  return value ? new Date(value).toLocaleString() : 'Never';
}

function formatLogDetails(details: Record<string, unknown> | null): string {
  if (!details) return 'No additional details';
  if (details.dryRun) return '(dry run)';
  if (details.error) return `Error: ${details.error}`;

  const parts: string[] = [];
  if (typeof details.created === 'number') parts.push(`+${details.created}`);
  if (typeof details.updated === 'number') parts.push(`~${details.updated}`);
  if (typeof details.skipped === 'number') parts.push(`=${details.skipped}`);
  if (typeof details.failed === 'number' && details.failed > 0) parts.push(`!${details.failed}`);
  return parts.join(' ') || 'No additional details';
}

export default function SyncProfileSection() {
  const notify = useNotificationStore((s) => s.notify);
  const [profiles, setProfiles] = useState<SyncProfileData[]>([]);
  const [loading, setLoading] = useState(true);
  const [editOpen, setEditOpen] = useState(false);
  const [editingProfile, setEditingProfile] = useState<SyncProfileData | null>(null);
  const [form, setForm] = useState<SyncProfileFormState>(emptyForm);
  const [expandedLogsId, setExpandedLogsId] = useState<string | null>(null);
  const [loadingLogsId, setLoadingLogsId] = useState<string | null>(null);
  const [logsByProfile, setLogsByProfile] = useState<Record<string, SyncLogEntry[]>>({});
  const [previewPlan, setPreviewPlan] = useState<SyncPlanData | null>(null);
  const [previewProfileId, setPreviewProfileId] = useState<string | null>(null);
  const [previewOpen, setPreviewOpen] = useState(false);
  const [sectionError, setSectionError] = useState('');
  const { loading: actionLoading, error: actionError, run } = useAsyncAction();

  const loadProfiles = useCallback(async () => {
    try {
      const data = await listSyncProfiles();
      setProfiles(data);
    } catch {
      // Keep the section usable even if the initial fetch fails transiently.
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadProfiles();
  }, [loadProfiles]);

  const isSaveDisabled = useMemo(
    () => actionLoading || !form.name || !form.url || (!editingProfile && !form.apiToken),
    [actionLoading, editingProfile, form.apiToken, form.name, form.url],
  );

  const openCreate = () => {
    setEditingProfile(null);
    setForm(emptyForm);
    setEditOpen(true);
  };

  const openEdit = (profile: SyncProfileData) => {
    setEditingProfile(profile);
    setForm(createFormState(profile));
    setEditOpen(true);
  };

  const handleSave = async () => {
    const filters = parseKeyValuePairs(form.filters);
    const platformMapping = parseKeyValuePairs(form.platformMapping);

    if (editingProfile) {
      const input: UpdateSyncProfileInput = {
        name: form.name,
        url: form.url,
        filters,
        platformMapping,
        defaultProtocol: form.defaultProtocol,
        conflictStrategy: form.conflictStrategy,
        cronExpression: form.cronExpression || null,
        teamId: form.teamId || null,
      };
      if (form.apiToken) input.apiToken = form.apiToken;

      const ok = await run(async () => {
        await updateSyncProfile(editingProfile.id, input);
      }, 'Failed to update sync profile');

      if (ok) {
        setEditOpen(false);
        await loadProfiles();
      }
      return;
    }

    const input: CreateSyncProfileInput = {
      name: form.name,
      provider: 'NETBOX',
      url: form.url,
      apiToken: form.apiToken,
      filters,
      platformMapping,
      defaultProtocol: form.defaultProtocol,
      conflictStrategy: form.conflictStrategy,
      cronExpression: form.cronExpression || undefined,
      teamId: form.teamId || undefined,
    };

    const ok = await run(async () => {
      await createSyncProfile(input);
    }, 'Failed to create sync profile');

    if (ok) {
      setEditOpen(false);
      await loadProfiles();
    }
  };

  const handleDelete = async (id: string) => {
    const ok = await run(async () => {
      await deleteSyncProfile(id);
    }, 'Failed to delete sync profile');

    if (ok) {
      await loadProfiles();
    }
  };

  const handleToggle = async (profile: SyncProfileData) => {
    const ok = await run(async () => {
      await updateSyncProfile(profile.id, { enabled: !profile.enabled });
    }, 'Failed to toggle sync profile');

    if (ok) {
      await loadProfiles();
    }
  };

  const handleTest = async (id: string) => {
    setSectionError('');
    try {
      const result = await testSyncConnection(id);
      if (result.ok) {
        notify('Connection successful', 'success');
      } else {
        setSectionError(`Connection failed: ${result.error}`);
      }
    } catch (err) {
      setSectionError(extractApiError(err, 'Connection test failed'));
    }
  };

  const handlePreviewSync = async (id: string) => {
    setSectionError('');
    try {
      const result = await triggerSync(id, true);
      setPreviewPlan(result.plan);
      setPreviewProfileId(id);
      setPreviewOpen(true);
    } catch (err) {
      setSectionError(extractApiError(err, 'Failed to run sync preview'));
    }
  };

  const handleConfirmSync = async () => {
    if (!previewProfileId) return;

    try {
      await triggerSync(previewProfileId, false);
      setPreviewOpen(false);
      setPreviewPlan(null);
      setPreviewProfileId(null);
      notify('Sync completed successfully', 'success');
      await loadProfiles();
    } catch (err) {
      setSectionError(extractApiError(err, 'Sync failed'));
    }
  };

  const handleToggleLogs = async (profileId: string) => {
    if (expandedLogsId === profileId) {
      setExpandedLogsId(null);
      return;
    }

    setExpandedLogsId(profileId);
    if (logsByProfile[profileId]) return;

    setLoadingLogsId(profileId);
    try {
      const result = await getSyncLogs(profileId, 1, 10);
      setLogsByProfile((current) => ({ ...current, [profileId]: result.logs }));
    } catch {
      setLogsByProfile((current) => ({ ...current, [profileId]: [] }));
    } finally {
      setLoadingLogsId(null);
    }
  };

  if (loading) {
    return (
      <SettingsPanel
        title="Sync Profiles"
        description="Import connections from external sources like NetBox."
      >
        <SettingsLoadingState message="Loading sync profiles..." />
      </SettingsPanel>
    );
  }

  return (
    <>
      <SettingsPanel
        title="Sync Profiles"
        description="Import connections from external sources like NetBox."
        heading={(
          <Button type="button" size="sm" onClick={openCreate}>
            <Plus />
            Add Profile
          </Button>
        )}
        contentClassName="space-y-4"
      >
        {sectionError && (
          <Alert variant="destructive">
            <AlertDescription>{sectionError}</AlertDescription>
          </Alert>
        )}

        {actionError && (
          <Alert variant="destructive">
            <AlertDescription>{actionError}</AlertDescription>
          </Alert>
        )}

        {profiles.length === 0 ? (
          <div className="rounded-xl border border-dashed border-border/80 px-4 py-8 text-center text-sm text-muted-foreground">
            No sync profiles configured. Add one to start importing connections.
          </div>
        ) : (
          <div className="space-y-4">
            {profiles.map((profile) => {
              const logs = logsByProfile[profile.id] ?? [];
              const statusTone = profile.lastSyncStatus
                ? syncStatusTones[profile.lastSyncStatus as keyof typeof syncStatusTones] ?? 'neutral'
                : 'neutral';

              return (
                <div
                  key={profile.id}
                  className="space-y-4 rounded-xl border border-border/70 bg-background/70 p-4"
                >
                  <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
                    <div className="space-y-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <div className="text-base font-medium text-foreground">{profile.name}</div>
                        <Badge variant="outline">{profile.provider}</Badge>
                        <SettingsStatusBadge tone={statusTone}>
                          {profile.lastSyncStatus || 'Not synced'}
                        </SettingsStatusBadge>
                        <Badge variant="outline">
                          {profile.cronExpression ? 'Scheduled' : 'Manual'}
                        </Badge>
                      </div>
                      <p className="break-all text-sm leading-6 text-muted-foreground">
                        {profile.config.url}
                      </p>
                    </div>

                    <label className="flex items-center gap-3 rounded-xl border border-border/70 bg-background px-3 py-2">
                      <span className="text-sm font-medium text-foreground">Enabled</span>
                      <Switch
                        checked={profile.enabled}
                        onCheckedChange={() => handleToggle(profile)}
                        disabled={actionLoading}
                        aria-label={`Enable ${profile.name}`}
                      />
                    </label>
                  </div>

                  <SettingsSummaryGrid className="xl:grid-cols-5">
                    <SettingsSummaryItem label="Last Sync" value={formatDateTime(profile.lastSyncAt)} />
                    <SettingsSummaryItem label="Default Protocol" value={profile.config.defaultProtocol || 'SSH'} />
                    <SettingsSummaryItem label="Conflict Strategy" value={profile.config.conflictStrategy || 'update'} />
                    <SettingsSummaryItem label="Schedule" value={profile.cronExpression || 'Manual only'} />
                    <SettingsSummaryItem label="Team" value={profile.teamId || 'Personal'} />
                  </SettingsSummaryGrid>

                  <SettingsButtonRow>
                    <Button type="button" variant="outline" size="sm" onClick={() => handleTest(profile.id)} disabled={actionLoading}>
                      <FlaskConical />
                      Test Connection
                    </Button>
                    <Button type="button" variant="outline" size="sm" onClick={() => handlePreviewSync(profile.id)} disabled={actionLoading}>
                      <RefreshCw />
                      Preview Sync
                    </Button>
                    <Button type="button" variant="outline" size="sm" onClick={() => handleToggleLogs(profile.id)}>
                      {expandedLogsId === profile.id ? <ChevronUp /> : <ChevronDown />}
                      Recent Activity
                    </Button>
                    <Button type="button" variant="outline" size="sm" onClick={() => openEdit(profile)}>
                      <Edit3 />
                      Edit
                    </Button>
                    <Button type="button" variant="destructive" size="sm" onClick={() => handleDelete(profile.id)} disabled={actionLoading}>
                      <Trash2 />
                      Delete
                    </Button>
                  </SettingsButtonRow>

                  {expandedLogsId === profile.id && (
                    <div className="space-y-3 rounded-xl border border-border/70 bg-background p-4">
                      <div className="text-sm font-medium text-foreground">Recent Activity</div>
                      {loadingLogsId === profile.id ? (
                        <SettingsLoadingState message="Loading sync history..." />
                      ) : logs.length === 0 ? (
                        <div className="text-sm text-muted-foreground">No sync history.</div>
                      ) : (
                        <div className="space-y-2">
                          {logs.map((logEntry) => (
                            <div key={logEntry.id} className="rounded-lg border border-border/60 bg-background/70 p-3">
                              <div className="flex flex-wrap items-center justify-between gap-2">
                                <SettingsStatusBadge tone={syncStatusTones[logEntry.status as keyof typeof syncStatusTones] ?? 'neutral'}>
                                  {logEntry.status}
                                </SettingsStatusBadge>
                                <span className="text-xs text-muted-foreground">{formatDateTime(logEntry.startedAt)}</span>
                              </div>
                              <p className="mt-2 text-sm text-muted-foreground">{formatLogDetails(logEntry.details)}</p>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </SettingsPanel>

      <Dialog open={editOpen} onOpenChange={(next) => { if (!next) setEditOpen(false); }}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{editingProfile ? 'Edit Sync Profile' : 'Create Sync Profile'}</DialogTitle>
            <DialogDescription>
              Configure a NetBox source and review the import plan before syncing.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-1 md:grid-cols-2">
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="sync-profile-name">Name</Label>
              <Input id="sync-profile-name" value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
            </div>
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="sync-profile-url">NetBox URL</Label>
              <Input id="sync-profile-url" value={form.url} placeholder="https://netbox.example.com" onChange={(event) => setForm((current) => ({ ...current, url: event.target.value }))} />
            </div>
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="sync-profile-token">API Token</Label>
              <Input id="sync-profile-token" type="password" value={form.apiToken} placeholder={editingProfile ? 'Leave empty to keep current' : ''} onChange={(event) => setForm((current) => ({ ...current, apiToken: event.target.value }))} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="sync-profile-filters">Filters</Label>
              <Input id="sync-profile-filters" value={form.filters} placeholder="site=dc1, status=active" onChange={(event) => setForm((current) => ({ ...current, filters: event.target.value }))} />
              <p className="text-xs leading-5 text-muted-foreground">Comma-separated `key=value` pairs.</p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="sync-profile-platform-mapping">Platform Mapping</Label>
              <Input id="sync-profile-platform-mapping" value={form.platformMapping} placeholder="linux=SSH, windows=RDP" onChange={(event) => setForm((current) => ({ ...current, platformMapping: event.target.value }))} />
              <p className="text-xs leading-5 text-muted-foreground">Map NetBox platform slugs to protocols.</p>
            </div>
            <div className="space-y-2">
              <Label>Default Protocol</Label>
              <Select value={form.defaultProtocol} onValueChange={(value) => setForm((current) => ({ ...current, defaultProtocol: value }))}>
                <SelectTrigger aria-label="Default Protocol">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="SSH">SSH</SelectItem>
                  <SelectItem value="RDP">RDP</SelectItem>
                  <SelectItem value="VNC">VNC</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Conflict Strategy</Label>
              <Select value={form.conflictStrategy} onValueChange={(value) => setForm((current) => ({ ...current, conflictStrategy: value }))}>
                <SelectTrigger aria-label="Conflict Strategy">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="update">Update</SelectItem>
                  <SelectItem value="skip">Skip</SelectItem>
                  <SelectItem value="overwrite">Overwrite</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="sync-profile-cron">Cron Expression</Label>
              <Input id="sync-profile-cron" value={form.cronExpression} placeholder="0 */6 * * *" onChange={(event) => setForm((current) => ({ ...current, cronExpression: event.target.value }))} />
              <p className="text-xs leading-5 text-muted-foreground">Leave empty for manual sync only.</p>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setEditOpen(false)}>
              Cancel
            </Button>
            <Button type="button" onClick={handleSave} disabled={isSaveDisabled}>
              {actionLoading ? <Loader2 className="animate-spin" /> : null}
              {editingProfile ? 'Save Changes' : 'Save Profile'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <SyncPreviewDialog
        open={previewOpen}
        onClose={() => {
          setPreviewOpen(false);
          setPreviewPlan(null);
          setPreviewProfileId(null);
        }}
        onConfirm={handleConfirmSync}
        plan={previewPlan}
        confirming={actionLoading}
      />
    </>
  );
}
