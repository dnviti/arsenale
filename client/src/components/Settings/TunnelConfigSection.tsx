import { useCallback, useEffect, useState } from 'react';
import { RefreshCw } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useAuthStore } from '../../store/authStore';
import { useGatewayStore } from '../../store/gatewayStore';
import { useNotificationStore } from '../../store/notificationStore';
import { useTenantStore } from '../../store/tenantStore';
import { extractApiError } from '../../utils/apiError';
import NetworkEntryEditor from './NetworkEntryEditor';
import {
  SettingsButtonRow,
  SettingsFieldCard,
  SettingsFieldGroup,
  SettingsLoadingState,
  SettingsPanel,
  SettingsSectionBlock,
  SettingsSummaryGrid,
  SettingsSummaryItem,
  SettingsSwitchRow,
} from './settings-ui';

const MIN_TUNNEL_DAYS = 1;

export default function TunnelConfigSection() {
  const user = useAuthStore((state) => state.user);
  const tenant = useTenantStore((state) => state.tenant);
  const updateTenant = useTenantStore((state) => state.updateTenant);
  const fetchTenant = useTenantStore((state) => state.fetchTenant);
  const tunnelOverview = useGatewayStore((state) => state.tunnelOverview);
  const tunnelOverviewLoading = useGatewayStore((state) => state.tunnelOverviewLoading);
  const fetchTunnelOverview = useGatewayStore((state) => state.fetchTunnelOverview);
  const notify = useNotificationStore((state) => state.notify);

  const [tunnelDefaultEnabled, setTunnelDefaultEnabled] = useState(false);
  const [tunnelRequireForRemote, setTunnelRequireForRemote] = useState(false);
  const [tunnelAutoTokenRotation, setTunnelAutoTokenRotation] = useState(false);
  const [tunnelTokenRotationDays, setTunnelTokenRotationDays] = useState(90);
  const [tunnelTokenMaxLifetimeDays, setTunnelTokenMaxLifetimeDays] = useState<number | null>(null);
  const [tunnelAgentAllowedCidrs, setTunnelAgentAllowedCidrs] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (user?.tenantId && !tenant) {
      void fetchTenant();
    }
  }, [fetchTenant, tenant, user?.tenantId]);

  useEffect(() => {
    if (user?.tenantId) {
      void fetchTunnelOverview();
    }
  }, [fetchTunnelOverview, user?.tenantId]);

  const tunnelAgentAllowedCidrsKey = JSON.stringify(tenant?.tunnelAgentAllowedCidrs ?? []);
  useEffect(() => {
    if (!tenant) {
      return;
    }

    setTunnelDefaultEnabled(tenant.tunnelDefaultEnabled);
    setTunnelRequireForRemote(tenant.tunnelRequireForRemote);
    setTunnelAutoTokenRotation(tenant.tunnelAutoTokenRotation);
    setTunnelTokenRotationDays(tenant.tunnelTokenRotationDays ?? 90);
    setTunnelTokenMaxLifetimeDays(tenant.tunnelTokenMaxLifetimeDays ?? null);
    setTunnelAgentAllowedCidrs(JSON.parse(tunnelAgentAllowedCidrsKey) as string[]);
  }, [
    tenant,
    tunnelAgentAllowedCidrsKey,
  ]);

  const handleSave = useCallback(async () => {
    setSaving(true);
    setError('');

    try {
      await updateTenant({
        tunnelDefaultEnabled,
        tunnelRequireForRemote,
        tunnelAutoTokenRotation,
        tunnelTokenRotationDays,
        tunnelTokenMaxLifetimeDays,
        tunnelAgentAllowedCidrs,
      });
      await fetchTenant();
      notify('Tunnel configuration saved.', 'success');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to save tunnel configuration'));
    } finally {
      setSaving(false);
    }
  }, [
    fetchTenant,
    notify,
    tunnelAgentAllowedCidrs,
    tunnelAutoTokenRotation,
    tunnelDefaultEnabled,
    tunnelRequireForRemote,
    tunnelTokenMaxLifetimeDays,
    tunnelTokenRotationDays,
    updateTenant,
  ]);

  if (!tenant) {
    return (
      <SettingsPanel
        title="Tunnel Configuration"
        description="Zero-trust defaults, token security, and tunnel fleet health."
      >
        <SettingsLoadingState message="Loading tunnel configuration..." />
      </SettingsPanel>
    );
  }

  return (
    <SettingsPanel
      title="Tunnel Configuration"
      description="Define how gateways join the tunnel fabric, how tunnel tokens rotate, and which networks agents may connect from."
      contentClassName="space-y-4"
    >
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <SettingsSectionBlock
        title="Defaults"
        description="Set the baseline tunnel posture for new and remote gateways."
      >
        <SettingsFieldGroup>
          <SettingsSwitchRow
            title="Enable tunnels by default"
            description="New gateways will start with zero-trust tunneling enabled."
            checked={tunnelDefaultEnabled}
            disabled={saving}
            onCheckedChange={setTunnelDefaultEnabled}
          />
          <SettingsSwitchRow
            title="Require tunnels for remote gateways"
            description="Gateways outside the trusted local network must connect through the tunnel fabric."
            checked={tunnelRequireForRemote}
            disabled={saving}
            onCheckedChange={setTunnelRequireForRemote}
          />
        </SettingsFieldGroup>
      </SettingsSectionBlock>

      <SettingsSectionBlock
        title="Token Security"
        description="Rotate tunnel tokens automatically and cap how long any issued token can live."
      >
        <SettingsFieldGroup>
          <SettingsSwitchRow
            title="Auto-rotate tunnel tokens"
            description="Rotate issued tunnel credentials on a schedule to reduce long-lived exposure."
            checked={tunnelAutoTokenRotation}
            disabled={saving}
            onCheckedChange={setTunnelAutoTokenRotation}
          />
          <div className="grid gap-4 xl:grid-cols-2">
            <SettingsFieldCard
              label="Rotation interval"
              description="How often tokens rotate when auto-rotation is enabled."
            >
              <Input
                type="number"
                min={MIN_TUNNEL_DAYS}
                aria-label="Tunnel token rotation days"
                value={tunnelTokenRotationDays}
                disabled={saving || !tunnelAutoTokenRotation}
                onChange={(event) => {
                  const nextValue = Number.parseInt(event.target.value, 10) || MIN_TUNNEL_DAYS;
                  setTunnelTokenRotationDays(Math.max(MIN_TUNNEL_DAYS, nextValue));
                }}
              />
            </SettingsFieldCard>

            <SettingsFieldCard
              label="Maximum token lifetime"
              description="Leave empty to allow tokens to persist until rotated or revoked."
            >
              <Input
                type="number"
                min={MIN_TUNNEL_DAYS}
                aria-label="Tunnel token max lifetime days"
                value={tunnelTokenMaxLifetimeDays ?? ''}
                disabled={saving}
                placeholder="No limit"
                onChange={(event) => {
                  const { value } = event.target;
                  if (!value) {
                    setTunnelTokenMaxLifetimeDays(null);
                    return;
                  }
                  const nextValue = Number.parseInt(value, 10) || MIN_TUNNEL_DAYS;
                  setTunnelTokenMaxLifetimeDays(Math.max(MIN_TUNNEL_DAYS, nextValue));
                }}
              />
            </SettingsFieldCard>
          </div>
        </SettingsFieldGroup>
      </SettingsSectionBlock>

      <SettingsSectionBlock
        title="Agent Restrictions"
        description="Constrain tunnel agents to trusted source networks only."
      >
        <NetworkEntryEditor
          label="Allowed Agent Networks"
          description="Add IPv4 or IPv6 addresses and CIDR ranges that tunnel agents may connect from."
          inputLabel="Allowed Agent Network"
          placeholder="e.g. 10.0.0.0/8 or 2001:db8::/32"
          helperText="Leave empty to allow agents from any IP."
          emptyState="No restrictions are configured. Tunnel agents can connect from any network."
          entries={tunnelAgentAllowedCidrs}
          disabled={saving}
          onChange={setTunnelAgentAllowedCidrs}
        />
      </SettingsSectionBlock>

      <SettingsSectionBlock
        title="Fleet Overview"
        description="Live tunnel connectivity across the current tenant."
        className="space-y-4"
      >
        <SettingsButtonRow className="justify-end">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={tunnelOverviewLoading}
            onClick={() => void fetchTunnelOverview()}
          >
            <RefreshCw className={tunnelOverviewLoading ? 'animate-spin' : undefined} />
            Refresh
          </Button>
        </SettingsButtonRow>

        {tunnelOverviewLoading && !tunnelOverview ? (
          <SettingsLoadingState message="Loading tunnel overview..." />
        ) : tunnelOverview ? (
          <SettingsSummaryGrid className="xl:grid-cols-4">
            <SettingsSummaryItem label="Gateways" value={String(tunnelOverview.total)} />
            <SettingsSummaryItem label="Connected" value={String(tunnelOverview.connected)} />
            <SettingsSummaryItem label="Disconnected" value={String(tunnelOverview.disconnected)} />
            <SettingsSummaryItem
              label="Average RTT"
              value={tunnelOverview.avgRttMs != null ? `${tunnelOverview.avgRttMs} ms` : 'N/A'}
            />
          </SettingsSummaryGrid>
        ) : (
          <p className="text-sm leading-6 text-muted-foreground">
            No tunnel fleet data is available yet.
          </p>
        )}
      </SettingsSectionBlock>

      <SettingsButtonRow>
        <Button type="button" onClick={handleSave} disabled={saving}>
          {saving ? 'Saving...' : 'Save'}
        </Button>
      </SettingsButtonRow>
    </SettingsPanel>
  );
}
