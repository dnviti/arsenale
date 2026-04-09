import { useState, useEffect } from 'react';
import {
  CheckCircle2,
  Loader2,
  RefreshCw,
  TestTubeDiagonal,
} from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { getLdapStatus, testLdapConnection, triggerLdapSync } from '../../api/ldap.api';
import type { LdapStatus, LdapTestResult, LdapSyncResult } from '../../api/ldap.api';
import { extractApiError } from '../../utils/apiError';
import {
  SettingsButtonRow,
  SettingsLoadingState,
  SettingsPanel,
  SettingsStatusBadge,
  SettingsSummaryGrid,
  SettingsSummaryItem,
} from './settings-ui';

export default function LdapConfigSection() {
  const [status, setStatus] = useState<LdapStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [testing, setTesting] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [testResult, setTestResult] = useState<LdapTestResult | null>(null);
  const [syncResult, setSyncResult] = useState<LdapSyncResult | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    getLdapStatus()
      .then(setStatus)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const handleTest = async () => {
    setError('');
    setTestResult(null);
    setTesting(true);
    try {
      const result = await testLdapConnection();
      setTestResult(result);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to test LDAP connection'));
    } finally {
      setTesting(false);
    }
  };

  const handleSync = async () => {
    setError('');
    setSyncResult(null);
    setSyncing(true);
    try {
      const result = await triggerLdapSync();
      setSyncResult(result);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to run LDAP sync'));
    } finally {
      setSyncing(false);
    }
  };

  if (loading) {
    return (
      <SettingsPanel
        title="LDAP"
        description="Directory-backed authentication and scheduled identity sync."
      >
        <SettingsLoadingState message="Loading LDAP status..." />
      </SettingsPanel>
    );
  }

  if (!status?.enabled) return null;

  return (
    <SettingsPanel
      title="LDAP"
      description="Directory-backed authentication and scheduled identity sync."
      heading={(
        <div className="flex flex-wrap items-center gap-2">
          <SettingsStatusBadge tone="success">
            <CheckCircle2 className="mr-1 size-3.5" />
            {status.providerName}
          </SettingsStatusBadge>
          {status.syncEnabled && <SettingsStatusBadge tone="neutral">Auto-Sync</SettingsStatusBadge>}
        </div>
      )}
      contentClassName="space-y-4"
    >
      <SettingsSummaryGrid className="xl:grid-cols-2">
        <SettingsSummaryItem label="Server" value={<code className="text-xs">{status.serverUrl}</code>} />
        <SettingsSummaryItem label="Base DN" value={<code className="text-xs">{status.baseDn}</code>} />
        <SettingsSummaryItem label="Auto-Provision" value={status.autoProvision ? 'Enabled' : 'Disabled'} />
        <SettingsSummaryItem
          label="Sync Schedule"
          value={status.syncEnabled ? <code className="text-xs">{status.syncCron}</code> : 'Manual only'}
        />
        {testResult?.ok && testResult.userCount !== undefined && (
          <SettingsSummaryItem
            label="Directory Entries"
            className="xl:col-span-2"
            value={`${testResult.userCount} users${testResult.groupCount !== undefined ? `, ${testResult.groupCount} groups` : ''}`}
          />
        )}
      </SettingsSummaryGrid>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <SettingsButtonRow>
        <Button
          type="button"
          variant="outline"
          onClick={handleTest}
          disabled={testing}
        >
          {testing ? <Loader2 className="animate-spin" /> : <TestTubeDiagonal />}
          {testing ? 'Testing...' : 'Test Connection'}
        </Button>
        <Button
          type="button"
          variant="outline"
          onClick={handleSync}
          disabled={syncing}
        >
          {syncing ? <Loader2 className="animate-spin" /> : <RefreshCw />}
          {syncing ? 'Syncing...' : 'Sync Now'}
        </Button>
      </SettingsButtonRow>

      {testResult && (
        <Alert variant={testResult.ok ? 'success' : 'destructive'}>
          <AlertDescription>{testResult.message}</AlertDescription>
        </Alert>
      )}

      {syncResult && (
        <Alert variant={syncResult.errors.length > 0 ? 'warning' : 'success'}>
          <AlertDescription>
            <div className="space-y-2">
              <div>
                Sync complete:
                {' '}
                {syncResult.created}
                {' '}
                created,
                {' '}
                {syncResult.updated}
                {' '}
                updated,
                {' '}
                {syncResult.disabled}
                {' '}
                disabled.
              </div>
              {syncResult.errors.length > 0 && (
                <ul className="list-disc space-y-1 pl-5">
                  {syncResult.errors.slice(0, 5).map((entry) => (
                    <li key={entry}>{entry}</li>
                  ))}
                  {syncResult.errors.length > 5 && (
                    <li>...and {syncResult.errors.length - 5} more</li>
                  )}
                </ul>
              )}
            </div>
          </AlertDescription>
        </Alert>
      )}
    </SettingsPanel>
  );
}
