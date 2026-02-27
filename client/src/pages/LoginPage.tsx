import { useState, useEffect, useRef } from 'react';
import { useNavigate, Link as RouterLink, useSearchParams } from 'react-router-dom';
import {
  Box, Card, CardContent, TextField, Button, Typography, Alert, Link,
} from '@mui/material';
import { loginApi, verifyTotpApi } from '../api/auth.api';
import { resendVerificationEmail } from '../api/email.api';
import { useAuthStore } from '../store/authStore';
import { useVaultStore } from '../store/vaultStore';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);
  const [step, setStep] = useState<'credentials' | 'totp'>('credentials');
  const [tempToken, setTempToken] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [showResend, setShowResend] = useState(false);
  const [resendCountdown, setResendCountdown] = useState(0);
  const countdownRef = useRef<ReturnType<typeof setInterval>>(undefined);
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const setAuth = useAuthStore((s) => s.setAuth);
  const setVaultUnlocked = useVaultStore((s) => s.setUnlocked);

  useEffect(() => {
    if (searchParams.get('verified') === 'true') {
      setSuccess('Email verified successfully! You can now sign in.');
      searchParams.delete('verified');
      setSearchParams(searchParams, { replace: true });
    }
    const verifyError = searchParams.get('verifyError');
    if (verifyError) {
      setError(verifyError);
      searchParams.delete('verifyError');
      setSearchParams(searchParams, { replace: true });
    }
  }, []);

  useEffect(() => {
    if (resendCountdown <= 0) {
      clearInterval(countdownRef.current);
      return;
    }
    countdownRef.current = setInterval(() => {
      setResendCountdown((prev) => {
        if (prev <= 1) {
          clearInterval(countdownRef.current);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
    return () => clearInterval(countdownRef.current);
  }, [resendCountdown > 0]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setShowResend(false);
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
      const axiosErr = err as { response?: { status?: number; data?: { error?: string } } };
      const msg = axiosErr?.response?.data?.error || 'Login failed';
      setError(msg);
      if (axiosErr?.response?.status === 403) {
        setShowResend(true);
      }
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

  const handleResend = async () => {
    try {
      await resendVerificationEmail(email);
      setResendCountdown(60);
      setSuccess('Verification email sent! Check your inbox.');
      setError('');
      setShowResend(false);
    } catch {
      // Server always returns 200 for valid format
    }
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
          {success && <Alert severity="success" sx={{ mb: 2 }}>{success}</Alert>}
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          {showResend && (
            <Button
              fullWidth
              variant="outlined"
              size="small"
              onClick={handleResend}
              disabled={resendCountdown > 0}
              sx={{ mb: 2 }}
            >
              {resendCountdown > 0
                ? `Resend verification email (${resendCountdown}s)`
                : 'Resend verification email'}
            </Button>
          )}

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
