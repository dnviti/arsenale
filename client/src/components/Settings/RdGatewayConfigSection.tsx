import { useState, useEffect, useCallback } from 'react';
import { Loader2 } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useAuthStore } from '../../store/authStore';
import { useNotificationStore } from '../../store/notificationStore';
import { extractApiError } from '../../utils/apiError';
import { getRdGatewayConfig, updateRdGatewayConfig, getRdGatewayStatus } from '../../api/rdGateway.api';
import type { RdGatewayConfig, RdGatewayStatus } from '../../api/rdGateway.api';
import { isAdminOrAbove } from '../../utils/roles';
import {
  SettingsButtonRow,
  SettingsLoadingState,
  SettingsPanel,
  SettingsStatusBadge,
  SettingsSummaryGrid,
  SettingsSummaryItem,
  SettingsSwitchRow,
} from './settings-ui';

export default function RdGatewayConfigSection() {
  const user = useAuthStore((s) => s.user);
  const isAdmin = isAdminOrAbove(user?.tenantRole);

  const [config, setConfig] = useState<RdGatewayConfig | null>(null);
  const [status, setStatus] = useState<RdGatewayStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  // Local form state
  const [enabled, setEnabled] = useState(false);
  const [externalHostname, setExternalHostname] = useState('');
  const [port, setPort] = useState(443);
  const [idleTimeoutSeconds, setIdleTimeoutSeconds] = useState(3600);

  const notify = useNotificationStore((s) => s.notify);

  // Load config and status on mount
  useEffect(() => {
    if (!user?.tenantId || !isAdmin) return;

    Promise.all([
      getRdGatewayConfig(),
      getRdGatewayStatus().catch(() => null),
    ]).then(([cfg, sts]) => {
      setConfig(cfg);
      setEnabled(cfg.enabled);
      setExternalHostname(cfg.externalHostname);
      setPort(cfg.port);
      setIdleTimeoutSeconds(cfg.idleTimeoutSeconds);
      if (sts) setStatus(sts);
      setLoading(false);
    }).catch((err: unknown) => {
      setError(extractApiError(err, 'Failed to load RD Gateway configuration'));
      setLoading(false);
    });
  }, [user?.tenantId, isAdmin]);

  const handleSave = useCallback(async () => {
    setSaving(true);
    setError('');
    try {
      const updated = await updateRdGatewayConfig({
        enabled,
        externalHostname,
        port,
        idleTimeoutSeconds,
      });
      setConfig(updated);
      setEnabled(updated.enabled);
      setExternalHostname(updated.externalHostname);
      setPort(updated.port);
      setIdleTimeoutSeconds(updated.idleTimeoutSeconds);
      notify('RD Gateway configuration saved', 'success');
    } catch (err) {
      setError(extractApiError(err, 'Failed to save RD Gateway configuration'));
    } finally {
      setSaving(false);
    }
  }, [enabled, externalHostname, port, idleTimeoutSeconds, notify]);

  if (!isAdmin || !user?.tenantId) return null;

  if (loading) {
    return (
      <SettingsPanel
        title="Native RDP Access"
        description="RD Gateway settings for native Windows and macOS RDP clients."
      >
        <SettingsLoadingState message="Loading RD Gateway configuration..." />
      </SettingsPanel>
    );
  }

  if (!config) {
    return (
      <SettingsPanel
        title="Native RDP Access"
        description="RD Gateway settings for native Windows and macOS RDP clients."
      >
        <Alert variant="destructive">
          <AlertDescription>{error || 'RD Gateway configuration is unavailable.'}</AlertDescription>
        </Alert>
      </SettingsPanel>
    );
  }

  const hasChanges = (
    config.enabled !== enabled ||
    config.externalHostname !== externalHostname ||
    config.port !== port ||
    config.idleTimeoutSeconds !== idleTimeoutSeconds
  );

  return (
    <SettingsPanel
      title="Native RDP Access"
      description="Enable MS-TSGU RD Gateway so native clients can tunnel RDP sessions through Arsenale."
      contentClassName="space-y-4"
    >
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <SettingsSwitchRow
        title="Enable RD Gateway"
        description="Expose the RD Gateway endpoint for native RDP clients like mstsc.exe and Microsoft Remote Desktop."
        checked={enabled}
        disabled={saving}
        onCheckedChange={setEnabled}
      />

      <div className="grid gap-4 md:grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)]">
        <div className="space-y-2">
          <Label htmlFor="rd-gateway-hostname">External Hostname</Label>
          <Input
            id="rd-gateway-hostname"
            value={externalHostname}
            onChange={(event) => setExternalHostname(event.target.value)}
            placeholder="rdgw.example.com"
            disabled={!enabled || saving}
          />
          <p className="text-xs leading-5 text-muted-foreground">
            The public hostname RDP clients use to reach the gateway.
          </p>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="rd-gateway-port">Port</Label>
            <Input
              id="rd-gateway-port"
              type="number"
              min={1}
              max={65535}
              value={port}
              onChange={(event) => {
                const nextPort = Number.parseInt(event.target.value, 10);
                setPort(Number.isFinite(nextPort) ? nextPort : 443);
              }}
              disabled={!enabled || saving}
            />
            <p className="text-xs leading-5 text-muted-foreground">HTTPS port for the gateway endpoint.</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="rd-gateway-idle-timeout">Idle Timeout (seconds)</Label>
            <Input
              id="rd-gateway-idle-timeout"
              type="number"
              min={60}
              max={86400}
              value={idleTimeoutSeconds}
              onChange={(event) => {
                const nextTimeout = Number.parseInt(event.target.value, 10);
                setIdleTimeoutSeconds(Number.isFinite(nextTimeout) ? nextTimeout : 3600);
              }}
              disabled={!enabled || saving}
            />
            <p className="text-xs leading-5 text-muted-foreground">Maximum idle time before tunnel teardown.</p>
          </div>
        </div>
      </div>

      {status && enabled && (
        <SettingsSummaryGrid className="xl:grid-cols-2">
          <SettingsSummaryItem
            label="Gateway Status"
            value={<SettingsStatusBadge tone="success">Running</SettingsStatusBadge>}
          />
          <SettingsSummaryItem
            label="Live Usage"
            value={`${status.activeTunnels} tunnels / ${status.activeChannels} channels`}
          />
        </SettingsSummaryGrid>
      )}

      <SettingsButtonRow>
        <Button
          type="button"
          onClick={handleSave}
          disabled={saving || !hasChanges}
        >
          {saving && <Loader2 className="animate-spin" />}
          Save Changes
        </Button>
      </SettingsButtonRow>
    </SettingsPanel>
  );
}
