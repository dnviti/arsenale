import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  CheckCircle2,
  Database,
  Loader2,
  RefreshCw,
  Settings2,
  XCircle,
} from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { getAdminDbStatus, getSystemSettings } from '../../api/systemSettings.api';
import type { DbStatusResponse, SettingGroup, SettingValue } from '../../api/systemSettings.api';
import { extractApiError } from '../../utils/apiError';
import SettingField from './SettingField';
import { SettingsPanel, SettingsStatusBadge } from './settings-ui';

function LoadingState() {
  return (
    <SettingsPanel
      title="System Settings"
      description="Global runtime defaults and control-plane behavior."
    >
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        Loading system settings...
      </div>
    </SettingsPanel>
  );
}

function DbStatusCard({
  dbStatus,
  onRefresh,
}: {
  dbStatus: DbStatusResponse;
  onRefresh: () => void;
}) {
  const [openItem, setOpenItem] = useState('database');
  const connected = dbStatus.connected;

  return (
    <Accordion type="single" collapsible value={openItem} onValueChange={setOpenItem}>
      <AccordionItem value="database">
        <AccordionTrigger>
          <div className="flex flex-wrap items-center gap-2">
            <div className="inline-flex items-center gap-2">
              <Database className="size-4 text-muted-foreground" />
              <span>Database Status</span>
            </div>
            <SettingsStatusBadge tone={connected ? 'success' : 'destructive'}>
              {connected ? <CheckCircle2 className="mr-1 size-3.5" /> : <XCircle className="mr-1 size-3.5" />}
              {connected ? 'Connected' : 'Disconnected'}
            </SettingsStatusBadge>
          </div>
        </AccordionTrigger>
        <AccordionContent>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <div className="rounded-lg border border-border/70 bg-background/70 p-3">
              <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Host</div>
              <div className="mt-1 text-sm font-medium">{dbStatus.host || '—'}</div>
            </div>
            <div className="rounded-lg border border-border/70 bg-background/70 p-3">
              <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Port</div>
              <div className="mt-1 text-sm font-medium">{dbStatus.port}</div>
            </div>
            <div className="rounded-lg border border-border/70 bg-background/70 p-3">
              <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Database</div>
              <div className="mt-1 text-sm font-medium">{dbStatus.database || '—'}</div>
            </div>
            <div className="rounded-lg border border-border/70 bg-background/70 p-3">
              <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Version</div>
              <div className="mt-1 text-sm font-medium">{dbStatus.version || '—'}</div>
            </div>
          </div>
          <div className="mt-4">
            <Button type="button" variant="outline" size="sm" onClick={onRefresh}>
              <RefreshCw />
              Refresh Status
            </Button>
          </div>
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  );
}

export default function SystemSettingsSection() {
  const [settings, setSettings] = useState<SettingValue[]>([]);
  const [groups, setGroups] = useState<SettingGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [dbStatus, setDbStatus] = useState<DbStatusResponse | null>(null);

  const refreshDbStatus = useCallback(() => {
    getAdminDbStatus().then(setDbStatus).catch(() => {});
  }, []);

  useEffect(() => {
    Promise.all([getSystemSettings(), getAdminDbStatus().catch(() => null)])
      .then(([data, db]) => {
        setSettings(data.settings);
        setGroups(data.groups);
        if (db) setDbStatus(db);
        setLoading(false);
      })
      .catch((err: unknown) => {
        setError(extractApiError(err, 'Failed to load system settings'));
        setLoading(false);
      });
  }, []);

  const handleUpdated = useCallback((key: string, value: unknown) => {
    setSettings((current) =>
      current.map((setting) =>
        setting.key === key
          ? { ...setting, value, source: 'db' as const, envLocked: false }
          : setting,
      ),
    );
  }, []);

  const sortedGroups = useMemo(() => {
    const grouped = new Map<string, SettingValue[]>();

    for (const setting of settings) {
      const current = grouped.get(setting.group) ?? [];
      current.push(setting);
      grouped.set(setting.group, current);
    }

    return groups
      .filter((group) => grouped.has(group.key))
      .sort((left, right) => left.order - right.order)
      .map((group) => ({
        ...group,
        settings: grouped.get(group.key) ?? [],
      }));
  }, [groups, settings]);

  if (loading) {
    return <LoadingState />;
  }

  if (error) {
    return (
      <SettingsPanel
        title="System Settings"
        description="Global runtime defaults and control-plane behavior."
      >
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      </SettingsPanel>
    );
  }

  return (
    <SettingsPanel
      title="System Settings"
      description="Global runtime defaults, feature flags, and control-plane behavior."
      heading={(
        <div className="flex items-center gap-2">
          <Settings2 className="size-4 text-muted-foreground" />
          <Badge variant="outline">{settings.length} settings</Badge>
        </div>
      )}
      contentClassName="space-y-4"
    >
      <Alert variant="info">
        <AlertDescription>
          Settings locked by environment variables are read-only. Editable values apply immediately unless a restart badge is shown.
        </AlertDescription>
      </Alert>

      {dbStatus && <DbStatusCard dbStatus={dbStatus} onRefresh={refreshDbStatus} />}

      <Accordion
        type="multiple"
        defaultValue={sortedGroups.slice(0, 2).map((group) => group.key)}
        className="space-y-3"
      >
        {sortedGroups.map((group) => {
          const envCount = group.settings.filter((setting) => setting.envLocked).length;

          return (
            <AccordionItem key={group.key} value={group.key}>
              <AccordionTrigger>
                <div className="flex flex-wrap items-center gap-2">
                  <span>{group.label}</span>
                  <Badge variant="outline">{group.settings.length}</Badge>
                  {envCount > 0 && (
                    <SettingsStatusBadge tone="warning">{envCount} locked</SettingsStatusBadge>
                  )}
                </div>
              </AccordionTrigger>
              <AccordionContent>
                <div className="space-y-3">
                  {group.settings.map((setting) => (
                    <SettingField
                      key={setting.key}
                      setting={setting}
                      onUpdated={handleUpdated}
                    />
                  ))}
                </div>
              </AccordionContent>
            </AccordionItem>
          );
        })}
      </Accordion>
    </SettingsPanel>
  );
}
