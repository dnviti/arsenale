import { useState, useEffect } from 'react';
import { CheckCircle2, ExternalLink, ShieldCheck } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { getAuthProviderDetails } from '../../api/admin.api';
import { extractApiError } from '../../utils/apiError';
import { SettingsPanel, SettingsStatusBadge } from './settings-ui';

export default function SamlConfigSection() {
  const [enabled, setEnabled] = useState(false);
  const [providerName, setProviderName] = useState('SAML');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    getAuthProviderDetails()
      .then((providers) => {
        const saml = providers.find((p) => p.key === 'saml');
        setEnabled(!!saml?.enabled);
        if (saml?.providerName) {
          setProviderName(saml.providerName);
        }
      })
      .catch((err: unknown) => {
        setError(extractApiError(err, 'Failed to load SAML configuration'));
      })
      .finally(() => setLoading(false));
  }, []);

  if (loading) return null;
  if (!enabled && !error) return null;

  return (
    <SettingsPanel
      title="SAML Single Sign-On"
      description="Environment-managed trust for enterprise SSO providers."
      heading={enabled ? (
        <SettingsStatusBadge tone="success">
          <CheckCircle2 className="mr-1 size-3.5" />
          {providerName}
        </SettingsStatusBadge>
      ) : undefined}
      contentClassName="space-y-4"
    >
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {enabled && (
        <>
          <div className="rounded-xl border border-border/70 bg-background/70 p-4">
            <div className="flex items-start gap-3">
              <ShieldCheck className="mt-0.5 size-4 text-muted-foreground" />
              <p className="text-sm leading-6 text-muted-foreground">
                Users can authenticate through the configured SAML identity provider. Provisioning
                and attribute mapping are controlled by the server&apos;s environment configuration.
              </p>
            </div>
          </div>

          <Button asChild type="button" variant="outline" size="sm">
            <a href="/api/saml/metadata" target="_blank" rel="noopener noreferrer">
              <ExternalLink />
              View SP Metadata
            </a>
          </Button>
        </>
      )}
    </SettingsPanel>
  );
}
