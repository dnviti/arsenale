import { useState, useEffect } from 'react';
import { Button, Stack, Divider, Typography, CircularProgress } from '@mui/material';
import GitHubIcon from '@mui/icons-material/GitHub';
import GoogleIcon from '@mui/icons-material/Google';
import { getOAuthProviders, initiateOAuthLogin, OAuthProviders } from '../api/oauth.api';

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

interface OAuthButtonsProps {
  mode: 'login' | 'register';
}

export default function OAuthButtons({ mode }: OAuthButtonsProps) {
  const [providers, setProviders] = useState<OAuthProviders | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    getOAuthProviders()
      .then(setProviders)
      .catch(() => setProviders(null))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <Stack alignItems="center" sx={{ my: 2 }}>
        <CircularProgress size={20} />
      </Stack>
    );
  }

  if (!providers || (!providers.google && !providers.microsoft && !providers.github)) {
    return null;
  }

  const label = mode === 'login' ? 'Sign in' : 'Sign up';

  return (
    <>
      <Stack spacing={1.5} sx={{ mb: 2 }}>
        {providers.google && (
          <Button
            fullWidth
            variant="outlined"
            startIcon={<GoogleIcon />}
            onClick={() => initiateOAuthLogin('google')}
          >
            {label} with Google
          </Button>
        )}
        {providers.microsoft && (
          <Button
            fullWidth
            variant="outlined"
            startIcon={<MicrosoftIcon />}
            onClick={() => initiateOAuthLogin('microsoft')}
          >
            {label} with Microsoft
          </Button>
        )}
        {providers.github && (
          <Button
            fullWidth
            variant="outlined"
            startIcon={<GitHubIcon />}
            onClick={() => initiateOAuthLogin('github')}
          >
            {label} with GitHub
          </Button>
        )}
      </Stack>

      <Divider sx={{ mb: 2 }}>
        <Typography variant="body2" color="text.secondary">
          or
        </Typography>
      </Divider>
    </>
  );
}
