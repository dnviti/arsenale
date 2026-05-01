import { useState, useEffect } from 'react';
import {
  AlertCircle,
  CheckCircle2,
  Copy,
  CopyCheck,
} from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { getSshProxyStatus } from '../../api/sessions.api';
import type { SshProxyStatus } from '../../api/sessions.api';
import {
  SettingsButtonRow,
  SettingsLoadingState,
  SettingsPanel,
  SettingsStatusBadge,
  SettingsSummaryGrid,
  SettingsSummaryItem,
} from './settings-ui';

export default function NativeSshSection() {
  const [status, setStatus] = useState<SshProxyStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    getSshProxyStatus()
      .then((s) => { setStatus(s); setLoading(false); })
      .catch(() => { setError('Unable to fetch SSH proxy status'); setLoading(false); });
  }, []);

  const handleCopy = async (text: string) => {
    if (!navigator.clipboard?.writeText) return;

    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      // Ignore clipboard failures and leave the snippet visible.
    }
  };

  if (loading) {
    return (
      <SettingsPanel
        title="Native SSH Access"
        description="Connect through the Arsenale SSH proxy with a native OpenSSH client."
      >
        <SettingsLoadingState message="Loading SSH proxy status..." />
      </SettingsPanel>
    );
  }

  const sshConfigSnippet = status ? `Host arsenale-proxy
  HostName <server-host>
  Port ${status.port}
  User <connection-id>
  # Use the proxy token as password when prompted` : '';

  return (
    <SettingsPanel
      title="Native SSH Access"
      description="Connect through the Arsenale SSH proxy with a native OpenSSH client."
      contentClassName="space-y-4"
    >
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {status && (
        <>
          <SettingsSummaryGrid>
            <SettingsSummaryItem
              label="Proxy Status"
              value={(
                <SettingsStatusBadge
                  tone={status.enabled ? (status.listening ? 'success' : 'warning') : 'neutral'}
                >
                  {status.enabled
                    ? (status.listening ? <CheckCircle2 className="mr-1 size-3.5" /> : <AlertCircle className="mr-1 size-3.5" />)
                    : <AlertCircle className="mr-1 size-3.5" />}
                  {status.enabled ? (status.listening ? 'Running' : 'Enabled, not listening') : 'Disabled'}
                </SettingsStatusBadge>
              )}
            />
            <SettingsSummaryItem label="Proxy Port" value={status.port} />
            <SettingsSummaryItem label="Active Proxy Sessions" value={status.activeSessions} />
            <SettingsSummaryItem
              label="Auth Methods"
              value={(
                <div className="flex flex-wrap gap-2">
                  {status.allowedAuthMethods.map((method) => (
                    <Badge key={method} variant="outline">{method}</Badge>
                  ))}
                </div>
              )}
            />
          </SettingsSummaryGrid>

          {status.enabled && (
            <div className="space-y-3 rounded-xl border border-border/70 bg-background/70 p-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="space-y-1">
                  <div className="text-sm font-medium text-foreground">SSH Config Snippet</div>
                  <p className="text-sm leading-6 text-muted-foreground">
                    Drop this into your local SSH config and use a proxy token as the password.
                  </p>
                </div>
                <SettingsButtonRow>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => handleCopy(sshConfigSnippet)}
                    disabled={!navigator.clipboard?.writeText}
                  >
                    {copied ? <CopyCheck /> : <Copy />}
                    {copied ? 'Copied' : 'Copy Snippet'}
                  </Button>
                </SettingsButtonRow>
              </div>

              <Textarea
                readOnly
                value={sshConfigSnippet}
                rows={5}
                className="min-h-0 font-mono text-xs leading-6"
              />
            </div>
          )}

          {status.enabled ? (
            <Alert variant="info">
              <AlertDescription>
                <div className="space-y-2">
                  <div className="font-medium text-foreground">How to connect</div>
                  <ol className="list-decimal space-y-1 pl-5">
                    <li>Generate a proxy token from the connection menu or API.</li>
                    <li>Use the token as the password when connecting through the SSH proxy.</li>
                    <li>The proxy authenticates you, resolves vault credentials, and forwards the session.</li>
                  </ol>
                </div>
              </AlertDescription>
            </Alert>
          ) : (
            <Alert variant="info">
              <AlertDescription>
                The SSH proxy is currently disabled. Set
                {' '}
                <code className="rounded bg-background/80 px-1.5 py-0.5 text-xs text-foreground">SSH_PROXY_ENABLED=true</code>
                {' '}
                and restart the server to enable native SSH access.
              </AlertDescription>
            </Alert>
          )}
        </>
      )}
    </SettingsPanel>
  );
}
