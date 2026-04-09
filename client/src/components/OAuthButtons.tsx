import { useState, useEffect } from 'react';
import { LoaderCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import {
  AUTH_PROVIDER_LABELS,
  AUTH_PROVIDER_SERVER_KEYS,
  AuthProviderIcon,
  getEnabledAuthProviders,
  type AuthProviderKey,
} from './auth-provider-icons';
import {
  getOAuthProviders,
  initiateOAuthLogin,
  initiateSamlLogin,
  type OAuthProviders,
} from '../api/oauth.api';

interface OAuthButtonsProps {
  loading?: boolean;
  mode: 'login' | 'register';
  providers?: OAuthProviders | null;
}

export default function OAuthButtons({
  loading: controlledLoading,
  mode,
  providers: controlledProviders,
}: OAuthButtonsProps) {
  const [providers, setProviders] = useState<OAuthProviders | null>(null);
  const [loading, setLoading] = useState(true);
  const isControlled = controlledProviders !== undefined;

  useEffect(() => {
    if (isControlled) {
      return undefined;
    }

    getOAuthProviders()
      .then(setProviders)
      .catch(() => setProviders(null))
      .finally(() => setLoading(false));
    return undefined;
  }, [isControlled]);

  const resolvedProviders = isControlled ? controlledProviders : providers;
  const resolvedLoading = isControlled ? Boolean(controlledLoading) : loading;

  if (resolvedLoading) {
    return (
      <div className="my-4 flex items-center justify-center">
        <LoaderCircle className="size-5 animate-spin text-primary" />
      </div>
    );
  }

  const enabledProviders = resolvedProviders ? getEnabledAuthProviders(resolvedProviders) : [];

  if (!resolvedProviders || enabledProviders.length === 0) {
    return null;
  }

  const label = mode === 'login' ? 'Sign in' : 'Sign up';

  const handleProviderAction = (provider: AuthProviderKey) => {
    if (provider === 'SAML') {
      initiateSamlLogin();
      return;
    }

    initiateOAuthLogin(AUTH_PROVIDER_SERVER_KEYS[provider]);
  };

  return (
    <>
      <div className="mb-4 space-y-2">
        {enabledProviders.map((provider) => (
          <Button
            type="button"
            key={provider}
            variant="outline"
            onClick={() => handleProviderAction(provider)}
            className="w-full justify-start gap-3 border-border bg-card/60 text-foreground hover:bg-accent"
          >
            <AuthProviderIcon provider={provider} className="size-4" />
            {label} with {AUTH_PROVIDER_LABELS[provider]}
          </Button>
        ))}
      </div>

      <div className="relative mb-4">
        <Separator />
        <div className="absolute inset-0 flex items-center justify-center">
          <span className="bg-background px-2 text-xs uppercase tracking-[0.2em] text-muted-foreground">
            or
          </span>
        </div>
      </div>
    </>
  );
}
