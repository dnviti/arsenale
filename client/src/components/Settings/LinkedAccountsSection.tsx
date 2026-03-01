import { useState, useEffect } from 'react';
import {
  Card, CardContent, Typography, Button, Alert, Stack, Chip, IconButton,
  List, ListItem, ListItemIcon, ListItemText, ListItemSecondaryAction,
} from '@mui/material';
import GitHubIcon from '@mui/icons-material/GitHub';
import GoogleIcon from '@mui/icons-material/Google';
import LinkOffIcon from '@mui/icons-material/LinkOff';
import {
  getOAuthProviders, getLinkedAccounts, unlinkOAuthAccount, initiateOAuthLink,
  OAuthProviders, LinkedAccount,
} from '../../api/oauth.api';

function MicrosoftIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 21 21" xmlns="http://www.w3.org/2000/svg">
      <rect x="1" y="1" width="9" height="9" fill="#f25022" />
      <rect x="11" y="1" width="9" height="9" fill="#7fba00" />
      <rect x="1" y="11" width="9" height="9" fill="#00a4ef" />
      <rect x="11" y="11" width="9" height="9" fill="#ffb900" />
    </svg>
  );
}

function OidcIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg">
      <path d="M18 8h-1V6c0-2.76-2.24-5-5-5S7 3.24 7 6v2H6c-1.1 0-2 .9-2 2v10c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V10c0-1.1-.9-2-2-2zM12 17c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zM15.1 8H8.9V6c0-1.71 1.39-3.1 3.1-3.1s3.1 1.39 3.1 3.1v2z" />
    </svg>
  );
}

const providerIcons: Record<string, React.ReactNode> = {
  GOOGLE: <GoogleIcon />,
  MICROSOFT: <MicrosoftIcon />,
  GITHUB: <GitHubIcon />,
  OIDC: <OidcIcon />,
};

const providerLabels: Record<string, string> = {
  GOOGLE: 'Google',
  MICROSOFT: 'Microsoft',
  GITHUB: 'GitHub',
  OIDC: 'SSO',
};

interface LinkedAccountsSectionProps {
  hasPassword: boolean;
}

export default function LinkedAccountsSection({ hasPassword }: LinkedAccountsSectionProps) {
  const [providers, setProviders] = useState<OAuthProviders | null>(null);
  const [accounts, setAccounts] = useState<LinkedAccount[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  useEffect(() => {
    Promise.all([getOAuthProviders(), getLinkedAccounts()])
      .then(([p, a]) => {
        setProviders(p);
        setAccounts(a);
      })
      .catch(() => setError('Failed to load linked accounts'))
      .finally(() => setLoading(false));
  }, []);

  const linkedProviders = new Set(accounts.map((a) => a.provider));
  const totalAuthMethods = accounts.length + (hasPassword ? 1 : 0);

  // Compute labels with dynamic OIDC provider name
  const labels: Record<string, string> = providers?.oidcProviderName
    ? { ...providerLabels, OIDC: providers.oidcProviderName }
    : providerLabels;

  const handleUnlink = async (provider: string) => {
    setError('');
    setSuccess('');
    try {
      await unlinkOAuthAccount(provider);
      setAccounts((prev) => prev.filter((a) => a.provider !== provider));
      setSuccess(`${labels[provider] ?? provider} account unlinked`);
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
        'Failed to unlink account';
      setError(msg);
    }
  };

  if (loading) return null;
  if (!providers) return null;

  const availableProviders = (['GOOGLE', 'MICROSOFT', 'GITHUB', 'OIDC'] as const).filter(
    (p) => providers[p.toLowerCase() as keyof OAuthProviders] && !linkedProviders.has(p),
  );

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>Linked Accounts</Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Link external accounts for faster sign-in. You need at least one authentication method.
        </Typography>

        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        {success && <Alert severity="success" sx={{ mb: 2 }}>{success}</Alert>}

        {accounts.length > 0 && (
          <List disablePadding>
            {accounts.map((account) => (
              <ListItem key={account.id} disableGutters>
                <ListItemIcon sx={{ minWidth: 40 }}>
                  {providerIcons[account.provider] ?? null}
                </ListItemIcon>
                <ListItemText
                  primary={
                    <Stack direction="row" alignItems="center" spacing={1}>
                      <span>{labels[account.provider] ?? account.provider}</span>
                      <Chip label="Linked" color="success" size="small" />
                    </Stack>
                  }
                  secondary={account.providerEmail}
                />
                <ListItemSecondaryAction>
                  <IconButton
                    edge="end"
                    title="Unlink"
                    disabled={totalAuthMethods <= 1}
                    onClick={() => handleUnlink(account.provider)}
                  >
                    <LinkOffIcon />
                  </IconButton>
                </ListItemSecondaryAction>
              </ListItem>
            ))}
          </List>
        )}

        {availableProviders.length > 0 && (
          <Stack spacing={1} sx={{ mt: 2 }}>
            {availableProviders.map((provider) => (
              <Button
                key={provider}
                variant="outlined"
                startIcon={providerIcons[provider]}
                onClick={() => initiateOAuthLink(provider.toLowerCase())}
              >
                Link {labels[provider]}
              </Button>
            ))}
          </Stack>
        )}

        {accounts.length === 0 && availableProviders.length === 0 && (
          <Typography variant="body2" color="text.secondary">
            No OAuth providers are configured on this server.
          </Typography>
        )}
      </CardContent>
    </Card>
  );
}
