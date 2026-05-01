import { useEffect, useState } from 'react';
import { CheckCircle2, Loader2, Mail, TriangleAlert } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { getEmailStatus, sendTestEmail } from '../../api/admin.api';
import type { EmailStatus } from '../../api/admin.api';
import { useNotificationStore } from '../../store/notificationStore';
import { extractApiError } from '../../utils/apiError';
import { SettingsButtonRow, SettingsPanel, SettingsStatusBadge } from './settings-ui';

function EmailStatusSummary({ status }: { status: EmailStatus }) {
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
      <div className="rounded-xl border border-border/70 bg-background/70 p-4">
        <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Provider</div>
        <div className="mt-2 flex items-center gap-2">
          <SettingsStatusBadge tone={status.configured ? 'success' : 'warning'}>
            {status.configured ? <CheckCircle2 className="mr-1 size-3.5" /> : <TriangleAlert className="mr-1 size-3.5" />}
            {status.provider.toUpperCase()}
          </SettingsStatusBadge>
        </div>
      </div>
      <div className="rounded-xl border border-border/70 bg-background/70 p-4">
        <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">From</div>
        <div className="mt-2 break-all text-sm font-medium text-foreground">{status.from}</div>
      </div>
      <div className="rounded-xl border border-border/70 bg-background/70 p-4">
        <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Host</div>
        <div className="mt-2 text-sm font-medium text-foreground">
          {status.host ? `${status.host}:${status.port || 587}` : 'Console log fallback'}
        </div>
      </div>
      <div className="rounded-xl border border-border/70 bg-background/70 p-4">
        <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Transport</div>
        <div className="mt-2 text-sm font-medium text-foreground">
          {status.host ? (status.secure ? 'TLS' : 'Plain SMTP') : 'Not configured'}
        </div>
      </div>
    </div>
  );
}

export default function EmailProviderSection() {
  const notify = useNotificationStore((s) => s.notify);
  const [status, setStatus] = useState<EmailStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [testTo, setTestTo] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    getEmailStatus()
      .then(setStatus)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const handleSendTest = async () => {
    if (!testTo) return;
    setError('');
    setSending(true);

    try {
      const result = await sendTestEmail(testTo);
      notify(result.message, 'success');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to send test email'));
    } finally {
      setSending(false);
    }
  };

  return (
    <SettingsPanel
      title="Email Provider"
      description="Delivery state for account email, verification, password recovery, and notifications."
      contentClassName="space-y-4"
    >
      {loading && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Loading email provider status...
        </div>
      )}

      {status && (
        <>
          <EmailStatusSummary status={status} />

          {!status.configured && (
            <Alert variant="warning">
              <AlertDescription>
                Email delivery is not configured. Outbound messages currently fall back to console logging.
              </AlertDescription>
            </Alert>
          )}

          <div className="space-y-3 rounded-xl border border-border/70 bg-background/70 p-4">
            <div className="space-y-1">
              <div className="text-sm font-medium text-foreground">Send Test Email</div>
              <p className="text-sm leading-6 text-muted-foreground">
                Validate the configured provider without leaving settings.
              </p>
            </div>

            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}

            <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
              <Input
                type="email"
                aria-label="Recipient email"
                placeholder="Recipient email"
                value={testTo}
                onChange={(event) => setTestTo(event.target.value)}
                className="max-w-xl"
              />
              <SettingsButtonRow>
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleSendTest}
                  disabled={sending || !testTo}
                >
                  {sending ? <Loader2 className="animate-spin" /> : <Mail />}
                  Send Test
                </Button>
              </SettingsButtonRow>
            </div>
          </div>
        </>
      )}
    </SettingsPanel>
  );
}
