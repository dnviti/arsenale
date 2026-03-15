import { useState, useEffect } from 'react';
import {
  Card, CardContent, Typography, Button, Alert, Box, Chip, Stack, Divider,
} from '@mui/material';
import {
  CheckCircle as CheckIcon,
  Error as ErrorIcon,
  Sync as SyncIcon,
  NetworkCheck as TestIcon,
} from '@mui/icons-material';
import { getLdapStatus, testLdapConnection, triggerLdapSync } from '../../api/ldap.api';
import type { LdapStatus, LdapTestResult, LdapSyncResult } from '../../api/ldap.api';
import { extractApiError } from '../../utils/apiError';

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

  if (loading) return null;
  if (!status?.enabled) return null;

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          LDAP Authentication
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          LDAP configuration is managed via environment variables.
        </Typography>

        <Stack spacing={2}>
          <Stack direction="row" spacing={1} alignItems="center">
            <Chip
              icon={<CheckIcon />}
              label={status.providerName}
              color="success"
              variant="outlined"
            />
            {status.syncEnabled && (
              <Chip
                icon={<SyncIcon />}
                label="Auto-Sync"
                color="info"
                variant="outlined"
                size="small"
              />
            )}
          </Stack>

          <Box>
            <Typography variant="body2">
              Server: <code>{status.serverUrl}</code>
            </Typography>
            <Typography variant="body2">
              Base DN: <code>{status.baseDn}</code>
            </Typography>
            <Typography variant="body2">
              Auto-provision: {status.autoProvision ? 'Enabled' : 'Disabled'}
            </Typography>
            {status.syncEnabled && (
              <Typography variant="body2">
                Sync schedule: <code>{status.syncCron}</code>
              </Typography>
            )}
            {testResult && testResult.ok && testResult.userCount !== undefined && (
              <Typography variant="body2">
                Directory entries: {testResult.userCount} users{testResult.groupCount !== undefined ? `, ${testResult.groupCount} groups` : ''}
              </Typography>
            )}
          </Box>

          {error && <Alert severity="error">{error}</Alert>}

          <Divider />

          <Stack direction="row" spacing={1}>
            <Button
              variant="outlined"
              startIcon={<TestIcon />}
              onClick={handleTest}
              disabled={testing}
            >
              {testing ? 'Testing...' : 'Test Connection'}
            </Button>
            <Button
              variant="outlined"
              startIcon={<SyncIcon />}
              onClick={handleSync}
              disabled={syncing}
            >
              {syncing ? 'Syncing...' : 'Sync Now'}
            </Button>
          </Stack>

          {testResult && (
            <Alert
              severity={testResult.ok ? 'success' : 'error'}
              icon={testResult.ok ? <CheckIcon /> : <ErrorIcon />}
            >
              {testResult.message}
            </Alert>
          )}

          {syncResult && (
            <Alert severity={syncResult.errors.length > 0 ? 'warning' : 'success'}>
              Sync complete: {syncResult.created} created, {syncResult.updated} updated,
              {' '}{syncResult.disabled} disabled
              {syncResult.errors.length > 0 && (
                <Box component="ul" sx={{ mt: 1, mb: 0, pl: 2 }}>
                  {syncResult.errors.slice(0, 5).map((e, i) => (
                    <li key={i}>{e}</li>
                  ))}
                  {syncResult.errors.length > 5 && (
                    <li>...and {syncResult.errors.length - 5} more</li>
                  )}
                </Box>
              )}
            </Alert>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}
