import { useEffect, useState } from 'react';
import { Link2, Link2Off } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  AUTH_PROVIDER_LABELS,
  AUTH_PROVIDER_SERVER_KEYS,
  AuthProviderIcon,
  getEnabledAuthProviders,
  isAuthProviderKey,
} from '../auth-provider-icons';
import {
  getLinkedAccounts,
  getOAuthProviders,
  initiateOAuthLink,
  initiateSamlLink,
  type LinkedAccount,
  type OAuthProviders,
  unlinkOAuthAccount,
} from '../../api/oauth.api';
import { useNotificationStore } from '../../store/notificationStore';
import { extractApiError } from '../../utils/apiError';
import {
  SettingsButtonRow,
  SettingsPanel,
  SettingsStatusBadge,
} from './settings-ui';

interface LinkedAccountsSectionProps {
  hasPassword: boolean;
}

function providerLabel(provider: string) {
  return isAuthProviderKey(provider) ? AUTH_PROVIDER_LABELS[provider] : provider;
}

function linkedAccountEmail(account: LinkedAccount) {
  return account.providerEmail || 'No email address returned by this provider.';
}

export default function LinkedAccountsSection({
  hasPassword,
}: LinkedAccountsSectionProps) {
  const notify = useNotificationStore((state) => state.notify);
  const [providers, setProviders] = useState<OAuthProviders | null>(null);
  const [accounts, setAccounts] = useState<LinkedAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [pendingProvider, setPendingProvider] = useState<string | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    Promise.all([getOAuthProviders(), getLinkedAccounts()])
      .then(([availableProviders, linkedAccounts]) => {
        setProviders(availableProviders);
        setAccounts(linkedAccounts);
      })
      .catch(() => setError('Failed to load linked accounts.'))
      .finally(() => setLoading(false));
  }, []);

  const totalAuthMethods = accounts.length + (hasPassword ? 1 : 0);
  const linkedProviders = new Set(accounts.map((account) => account.provider));
  const availableProviders = providers
    ? getEnabledAuthProviders(providers).filter((provider) => !linkedProviders.has(provider))
    : [];

  const handleLink = async (provider: (typeof availableProviders)[number]) => {
    setError('');
    setPendingProvider(provider);

    try {
      if (provider === 'SAML') {
        await initiateSamlLink();
      } else {
        await initiateOAuthLink(AUTH_PROVIDER_SERVER_KEYS[provider]);
      }
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to initiate account linking.'));
      setPendingProvider(null);
    }
  };

  const handleUnlink = async (provider: string) => {
    setError('');
    setPendingProvider(provider);

    try {
      await unlinkOAuthAccount(provider);
      setAccounts((current) => current.filter((account) => account.provider !== provider));
      notify(`${providerLabel(provider)} account unlinked`, 'success');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to unlink account.'));
    } finally {
      setPendingProvider(null);
    }
  };

  if (loading) {
    return (
      <SettingsPanel
        title="Linked Accounts"
        description="Connect external identities for faster sign-in."
      >
        <p className="text-sm text-muted-foreground">Loading linked accounts...</p>
      </SettingsPanel>
    );
  }

  return (
    <SettingsPanel
      title="Linked Accounts"
      description="Connect external identities for faster sign-in. Keep at least one active sign-in method on your account."
      heading={
        <SettingsStatusBadge tone={accounts.length > 0 ? 'success' : 'neutral'}>
          {accounts.length} linked
        </SettingsStatusBadge>
      }
    >
      <div className="space-y-4">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {accounts.length > 0 && (
          <div className="space-y-3">
            {accounts.map((account) => {
              const canUnlink = totalAuthMethods > 1;
              const label = providerLabel(account.provider);

              return (
                <div
                  key={account.id}
                  className="flex flex-wrap items-start justify-between gap-4 rounded-xl border border-border/70 bg-background/60 p-4"
                >
                  <div className="flex min-w-0 items-start gap-3">
                    <div className="flex size-10 shrink-0 items-center justify-center rounded-lg border border-border/70 bg-card">
                      {isAuthProviderKey(account.provider) ? (
                        <AuthProviderIcon provider={account.provider} className="size-4" />
                      ) : (
                        <Link2 className="size-4 text-muted-foreground" />
                      )}
                    </div>
                    <div className="min-w-0 space-y-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="text-sm font-medium text-foreground">{label}</p>
                        <Badge variant="default">Linked</Badge>
                      </div>
                      <p className="break-all text-sm text-muted-foreground">
                        {linkedAccountEmail(account)}
                      </p>
                    </div>
                  </div>

                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    disabled={!canUnlink || pendingProvider === account.provider}
                    onClick={() => void handleUnlink(account.provider)}
                  >
                    <Link2Off className="size-4" />
                    {pendingProvider === account.provider ? 'Unlinking...' : 'Unlink'}
                  </Button>
                </div>
              );
            })}
          </div>
        )}

        {accounts.length > 0 && totalAuthMethods <= 1 && (
          <Alert variant="warning">
            <AlertDescription>
              Add another sign-in method before unlinking your last remaining account.
            </AlertDescription>
          </Alert>
        )}

        {availableProviders.length > 0 && (
          <div className="space-y-3">
            <p className="text-sm font-medium text-foreground">
              Available providers
            </p>
            <SettingsButtonRow className="gap-3">
              {availableProviders.map((provider) => (
                <Button
                  key={provider}
                  type="button"
                  variant="outline"
                  className="justify-start gap-3"
                  disabled={pendingProvider === provider}
                  onClick={() => void handleLink(provider)}
                >
                  <AuthProviderIcon provider={provider} className="size-4" />
                  {pendingProvider === provider
                    ? 'Connecting...'
                    : `Link ${AUTH_PROVIDER_LABELS[provider]}`}
                </Button>
              ))}
            </SettingsButtonRow>
          </div>
        )}

        {accounts.length === 0 && availableProviders.length === 0 && (
          <p className="text-sm text-muted-foreground">
            No external identity providers are configured on this server.
          </p>
        )}
      </div>
    </SettingsPanel>
  );
}
