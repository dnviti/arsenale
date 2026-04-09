import { useEffect, useState } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { useAuthStore } from '../../store/authStore';
import { getIpAllowlist, updateIpAllowlist, type IpAllowlistData } from '../../api/tenant.api';
import { useNotificationStore } from '../../store/notificationStore';
import { extractApiError } from '../../utils/apiError';
import NetworkEntryEditor from './NetworkEntryEditor';
import {
  SettingsFieldCard,
  SettingsFieldGroup,
  SettingsLoadingState,
  SettingsPanel,
  SettingsSectionBlock,
  SettingsSwitchRow,
} from './settings-ui';
import { isIpInCidr } from './networkAccessUtils';

export default function IpAllowlistSection() {
  const tenantId = useAuthStore((state) => state.user?.tenantId);
  const notify = useNotificationStore((state) => state.notify);

  const [config, setConfig] = useState<IpAllowlistData>({
    enabled: false,
    mode: 'flag',
    entries: [],
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [testIp, setTestIp] = useState('');
  const [testResult, setTestResult] = useState<'allowed' | 'blocked' | null>(null);

  useEffect(() => {
    if (!tenantId) {
      setLoading(false);
      return;
    }

    getIpAllowlist(tenantId)
      .then((nextConfig) => setConfig(nextConfig))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [tenantId]);

  const handleSave = async () => {
    if (!tenantId) {
      return;
    }

    setSaving(true);
    setError('');

    try {
      const nextConfig = await updateIpAllowlist(tenantId, config);
      setConfig(nextConfig);
      notify('IP allowlist saved.', 'success');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to save IP allowlist settings'));
    } finally {
      setSaving(false);
    }
  };

  const handleTestIp = () => {
    if (!testIp.trim()) {
      return;
    }

    if (config.entries.length === 0) {
      setTestResult('allowed');
      return;
    }

    const isAllowed = config.entries.some((entry) => isIpInCidr(testIp.trim(), entry));
    setTestResult(isAllowed ? 'allowed' : 'blocked');
  };

  if (loading) {
    return (
      <SettingsPanel
        title="IP Allowlist"
        description="Restrict tenant sign-ins to trusted IP addresses and CIDR ranges."
      >
        <SettingsLoadingState message="Loading IP allowlist..." />
      </SettingsPanel>
    );
  }

  if (!tenantId) {
    return null;
  }

  return (
    <SettingsPanel
      title="IP Allowlist"
      description="Decide whether unlisted IPs are only flagged or blocked entirely."
      contentClassName="space-y-4"
    >
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <SettingsSwitchRow
        title="Enable IP allowlist"
        description="When enabled, only listed addresses or CIDR ranges are trusted."
        checked={config.enabled}
        disabled={saving}
        onCheckedChange={(checked) => {
          setConfig((currentConfig) => ({ ...currentConfig, enabled: checked }));
          setTestResult(null);
        }}
      />

      {config.enabled && (
        <>
          <SettingsSectionBlock
            title="Enforcement Mode"
            description="Choose whether unlisted IPs are flagged in audits or rejected outright."
          >
            <SettingsFieldGroup>
              <SettingsFieldCard
                label="Mode"
                description="Flag mode keeps access open but marks suspicious sign-ins. Block mode rejects them."
              >
                <ToggleGroup
                  type="single"
                  value={config.mode}
                  onValueChange={(nextValue) => {
                    if (!nextValue) {
                      return;
                    }
                    setConfig((currentConfig) => ({
                      ...currentConfig,
                      mode: nextValue as IpAllowlistData['mode'],
                    }));
                    setTestResult(null);
                  }}
                  className="flex-wrap"
                >
                  <ToggleGroupItem value="flag" variant="outline">
                    Flag suspicious logins
                  </ToggleGroupItem>
                  <ToggleGroupItem value="block" variant="outline">
                    Block unauthorized logins
                  </ToggleGroupItem>
                </ToggleGroup>
              </SettingsFieldCard>

              {config.mode === 'block' && (
                <Alert>
                  <AlertDescription>
                    Block mode rejects all sign-ins from unlisted IPs. Make sure your own address is in the allowlist before saving.
                  </AlertDescription>
                </Alert>
              )}
            </SettingsFieldGroup>
          </SettingsSectionBlock>

          <SettingsSectionBlock
            title="Allowed Networks"
            description="Maintain the trusted IP and CIDR list used for login enforcement."
          >
            <NetworkEntryEditor
              label="Trusted IPs and CIDR Ranges"
              description="Use exact IPs for single hosts or CIDR ranges for office and VPN networks."
              inputLabel="Allowlist Entry"
              placeholder="e.g. 203.0.113.0/24 or 2001:db8::/32"
              emptyState={
                config.mode === 'block'
                  ? 'No entries are configured. Every login would be blocked.'
                  : 'No entries are configured. Unlisted logins would only be flagged.'
              }
              helperText="Add IPv4 or IPv6 addresses, with an optional CIDR prefix."
              entries={config.entries}
              disabled={saving}
              onChange={(entries) => {
                setConfig((currentConfig) => ({ ...currentConfig, entries }));
                setTestResult(null);
              }}
            />
          </SettingsSectionBlock>

          <SettingsSectionBlock
            title="Test an IP"
            description="Preview how the current allowlist would treat a specific client address."
          >
            <SettingsFieldCard
              label="Client Address"
              description="This uses the current unsaved form state, so you can validate changes before saving."
            >
              <div className="space-y-3">
                <div className="flex flex-col gap-2 sm:flex-row">
                  <Input
                    aria-label="Test IP Address"
                    value={testIp}
                    placeholder="e.g. 203.0.113.5"
                    onChange={(event) => {
                      setTestIp(event.target.value);
                      setTestResult(null);
                    }}
                  />
                  <Button type="button" variant="outline" onClick={handleTestIp} disabled={!testIp.trim()}>
                    Check
                  </Button>
                </div>

                {testResult && (
                  <Alert variant={testResult === 'allowed' ? 'default' : 'destructive'}>
                    <AlertDescription>
                      {testIp.trim()} would be <strong>{testResult}</strong> by the current allowlist.
                    </AlertDescription>
                  </Alert>
                )}
              </div>
            </SettingsFieldCard>
          </SettingsSectionBlock>
        </>
      )}

      <Button type="button" onClick={handleSave} disabled={saving}>
        {saving ? 'Saving...' : 'Save'}
      </Button>
    </SettingsPanel>
  );
}
