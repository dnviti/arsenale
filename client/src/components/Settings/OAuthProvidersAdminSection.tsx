import { useEffect, useState } from 'react';
import { Ban, CheckCircle2, Loader2 } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { AuthProviderIcon, isAuthProviderKey } from '../auth-provider-icons';
import { getAuthProviderDetails } from '../../api/admin.api';
import type { AuthProviderDetail } from '../../api/admin.api';
import { extractApiError } from '../../utils/apiError';
import { SettingsPanel, SettingsStatusBadge } from './settings-ui';

function ProviderState({ provider }: { provider: AuthProviderDetail }) {
  const enabledLabel = provider.providerName
    ? `Enabled as ${provider.providerName}`
    : 'Enabled';

  return (
    <SettingsStatusBadge tone={provider.enabled ? 'success' : 'neutral'}>
      {provider.enabled ? <CheckCircle2 className="mr-1 size-3.5" /> : <Ban className="mr-1 size-3.5" />}
      {provider.enabled ? enabledLabel : 'Disabled'}
    </SettingsStatusBadge>
  );
}

export default function OAuthProvidersAdminSection() {
  const [providers, setProviders] = useState<AuthProviderDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    getAuthProviderDetails()
      .then(setProviders)
      .catch((err: unknown) => {
        setError(extractApiError(err, 'Failed to load authentication providers'));
      })
      .finally(() => setLoading(false));
  }, []);

  return (
    <SettingsPanel
      title="Authentication Providers"
      description="OAuth and SSO provider state. Configuration lives in system settings, and environment variables override editable values."
      contentClassName="space-y-3"
    >
      {loading && (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Loading provider state...
        </div>
      )}

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {!loading && !error && (
        <div className="space-y-2">
          {providers.map((provider) => (
            <div
              key={provider.key}
              className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border/70 bg-background/70 px-4 py-3"
            >
              <div className="flex items-center gap-3">
                {isAuthProviderKey(provider.key)
                  ? <AuthProviderIcon provider={provider.key} className="size-5 text-foreground" />
                  : <div className="size-5 rounded-full border border-border/80" />}
                <div className="space-y-0.5">
                  <div className="text-sm font-medium text-foreground">{provider.label}</div>
                  <div className="text-xs text-muted-foreground">{provider.key}</div>
                </div>
              </div>
              <ProviderState provider={provider} />
            </div>
          ))}

          {providers.length === 0 && (
            <div className="rounded-xl border border-dashed border-border/80 px-4 py-5 text-sm text-muted-foreground">
              No authentication providers are configured.
            </div>
          )}
        </div>
      )}
    </SettingsPanel>
  );
}
