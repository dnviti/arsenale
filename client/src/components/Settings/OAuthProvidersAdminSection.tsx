import { useState, useEffect } from 'react';
import {
  Card, CardContent, Typography, Chip, Stack, Alert,
} from '@mui/material';
import {
  CheckCircle as CheckIcon,
  Block as BlockIcon,
} from '@mui/icons-material';
import { getOAuthProviders } from '../../api/oauth.api';
import type { OAuthProviders } from '../../api/oauth.api';
import { extractApiError } from '../../utils/apiError';

interface ProviderRow {
  label: string;
  enabled: boolean;
  providerName?: string;
}

function buildProviderRows(providers: OAuthProviders): ProviderRow[] {
  return [
    { label: 'Google', enabled: providers.google },
    { label: 'Microsoft', enabled: providers.microsoft },
    { label: 'GitHub', enabled: providers.github },
    { label: 'OIDC', enabled: !!providers.oidc, providerName: providers.oidcProviderName },
    { label: 'SAML', enabled: !!providers.saml, providerName: providers.samlProviderName },
    { label: 'LDAP', enabled: !!providers.ldap, providerName: providers.ldapProviderName },
  ];
}

export default function OAuthProvidersAdminSection() {
  const [providers, setProviders] = useState<OAuthProviders | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    getOAuthProviders()
      .then(setProviders)
      .catch((err: unknown) => {
        setError(extractApiError(err, 'Failed to load authentication providers'));
      })
      .finally(() => setLoading(false));
  }, []);

  if (loading) return null;

  const rows = providers ? buildProviderRows(providers) : [];

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          Authentication Providers
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          OAuth and SSO provider configuration is managed via environment variables.
        </Typography>

        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

        {providers && (
          <Stack spacing={1}>
            {rows.map((row) => (
              <Stack key={row.label} direction="row" spacing={1} alignItems="center">
                <Typography variant="body2" sx={{ minWidth: 80 }}>
                  {row.label}
                </Typography>
                <Chip
                  icon={row.enabled ? <CheckIcon /> : <BlockIcon />}
                  label={
                    row.enabled
                      ? row.providerName
                        ? `Enabled — ${row.providerName}`
                        : 'Enabled'
                      : 'Disabled'
                  }
                  color={row.enabled ? 'success' : 'default'}
                  variant="outlined"
                  size="small"
                />
              </Stack>
            ))}
          </Stack>
        )}
      </CardContent>
    </Card>
  );
}
