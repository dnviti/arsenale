import { useState } from 'react';
import { useNavigate, Link as RouterLink } from 'react-router-dom';
import {
  Box, Card, CardContent, TextField, Button, Typography, Alert, Link,
} from '@mui/material';
import { loginApi, verifyTotpApi } from '../api/auth.api';
import { useAuthStore } from '../store/authStore';
import { useVaultStore } from '../store/vaultStore';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [step, setStep] = useState<'credentials' | 'totp'>('credentials');
  const [tempToken, setTempToken] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const navigate = useNavigate();
  const setAuth = useAuthStore((s) => s.setAuth);
  const setVaultUnlocked = useVaultStore((s) => s.setUnlocked);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = await loginApi(email, password);

      if ('requiresTOTP' in data && data.requiresTOTP) {
        setTempToken(data.tempToken);
        setStep('totp');
        setLoading(false);
        return;
      }

      if ('accessToken' in data) {
        setAuth(data.accessToken, data.refreshToken, data.user);
        setVaultUnlocked(true);
        navigate('/');
      }
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
        'Login failed';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  const handleTotpSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = await verifyTotpApi(tempToken, totpCode);
      setAuth(data.accessToken, data.refreshToken, data.user);
      setVaultUnlocked(true);
      navigate('/');
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { error?: string } } })?.response?.data?.error ||
        'Invalid code';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  const handleBackToCredentials = () => {
    setStep('credentials');
    setTotpCode('');
    setTempToken('');
    setError('');
  };

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <Card sx={{ width: 400, maxWidth: '90vw' }}>
        <CardContent>
          <Typography variant="h5" gutterBottom align="center">
            Remote Desktop Manager
          </Typography>
          <Typography variant="body2" color="text.secondary" align="center" mb={3}>
            {step === 'credentials'
              ? 'Sign in to manage your connections'
              : 'Enter the 6-digit code from your authenticator app'}
          </Typography>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

          {step === 'credentials' ? (
            <Box component="form" onSubmit={handleSubmit}>
              <TextField
                fullWidth
                label="Email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                margin="normal"
                required
              />
              <TextField
                fullWidth
                label="Password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                margin="normal"
                required
              />
              <Button
                fullWidth
                type="submit"
                variant="contained"
                disabled={loading}
                sx={{ mt: 2, mb: 1 }}
              >
                {loading ? 'Signing in...' : 'Sign In'}
              </Button>
              <Typography variant="body2" align="center">
                Don't have an account?{' '}
                <Link component={RouterLink} to="/register">Sign up</Link>
              </Typography>
            </Box>
          ) : (
            <Box component="form" onSubmit={handleTotpSubmit}>
              <TextField
                fullWidth
                label="Authenticator Code"
                type="text"
                inputMode="numeric"
                value={totpCode}
                onChange={(e) => setTotpCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                margin="normal"
                required
                autoFocus
                placeholder="000000"
                slotProps={{ htmlInput: { maxLength: 6 } }}
              />
              <Button
                fullWidth
                type="submit"
                variant="contained"
                disabled={loading || totpCode.length !== 6}
                sx={{ mt: 2, mb: 1 }}
              >
                {loading ? 'Verifying...' : 'Verify'}
              </Button>
              <Button
                fullWidth
                variant="text"
                onClick={handleBackToCredentials}
                sx={{ mb: 1 }}
              >
                Back
              </Button>
            </Box>
          )}
        </CardContent>
      </Card>
    </Box>
  );
}
