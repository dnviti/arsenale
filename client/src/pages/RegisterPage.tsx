import { useState, useEffect, useRef } from 'react';
import { Link as RouterLink, useNavigate } from 'react-router-dom';
import {
  Box, TextField, Button, Typography, Alert, Link,
} from '@mui/material';
import { registerApi, getPublicConfig } from '../api/auth.api';
import { resendVerificationEmail } from '../api/email.api';
import OAuthButtons from '../components/OAuthButtons';
import PasswordStrengthMeter from '../components/common/PasswordStrengthMeter';
import { extractApiError } from '../utils/apiError';

export default function RegisterPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [registered, setRegistered] = useState(false);
  const [successMessage, setSuccessMessage] = useState('');
  const [registeredEmail, setRegisteredEmail] = useState('');
  const [recoveryKey, setRecoveryKey] = useState('');
  const [resendCountdown, setResendCountdown] = useState(0);
  const [signupDisabled, setSignupDisabled] = useState(false);
  const countdownRef = useRef<ReturnType<typeof setInterval>>(undefined);

  useEffect(() => {
    getPublicConfig()
      .then((cfg) => { if (!cfg.selfSignupEnabled) setSignupDisabled(true); })
      .catch(() => { /* fail-open: server guard is authoritative */ });
  }, []);

  const resendActive = resendCountdown > 0;
  useEffect(() => {
    if (!resendActive) {
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
  }, [resendActive]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    // eslint-disable-next-line security/detect-possible-timing-attacks -- client-side UI validation, not a security comparison
    if (password !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }
    if (password.length < 10) {
      setError('Password must be at least 10 characters');
      return;
    }

    setLoading(true);
    try {
      const result = await registerApi(email, password);
      if (!result.emailVerifyRequired) {
        navigate('/login?registered=true');
        return;
      }
      setRegisteredEmail(email);
      setSuccessMessage(result.message);
      if (result.recoveryKey) setRecoveryKey(result.recoveryKey);
      setRegistered(true);
      setResendCountdown(60);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Registration failed'));
    } finally {
      setLoading(false);
    }
  };

  const handleResend = async () => {
    try {
      await resendVerificationEmail(registeredEmail);
      setResendCountdown(60);
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
        bgcolor: 'background.default',
        background: (theme) => `radial-gradient(ellipse at 50% 40%, ${theme.palette.primary.main}0A 0%, ${theme.palette.background.default} 70%)`,
      }}
    >
      <Box sx={{
        width: 400,
        maxWidth: '90vw',
        bgcolor: 'background.paper',
        border: 1, borderColor: 'divider',
        borderRadius: 4,
        boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
        p: 3,
      }}>
          <Box sx={{ display: 'flex', justifyContent: 'center', mb: 1 }}>
            <Box sx={{ width: 24, height: 3, borderRadius: 1, bgcolor: 'primary.main' }} />
          </Box>
          <Typography variant="h5" gutterBottom align="center" sx={{
            fontFamily: (theme) => theme.typography.h5.fontFamily,
            fontSize: '2.25rem',
            color: 'text.primary',
            fontWeight: 400,
          }}>
            Create Account
          </Typography>

          {signupDisabled ? (
            <>
              <Alert severity="info" sx={{
                mb: 2,
                bgcolor: (theme) => `${theme.palette.info.main}14`,
                color: 'info.light',
                border: (theme) => `1px solid ${theme.palette.info.main}26`,
                '& .MuiAlert-icon': { color: 'info.light' },
              }}>
                Public registration is currently disabled. Please contact your organization administrator to get an account.
              </Alert>
              <Typography variant="body2" align="center" sx={{ color: 'text.secondary' }}>
                Already have an account?{' '}
                <Link component={RouterLink} to="/login" sx={{ color: 'primary.main', '&:hover': { color: 'secondary.main' } }}>Sign in</Link>
              </Typography>
            </>
          ) : registered ? (
            <>
              <Alert severity="success" sx={{
                mb: 2,
                bgcolor: (theme) => `${theme.palette.primary.main}14`,
                color: 'primary.main',
                border: (theme) => `1px solid ${theme.palette.primary.main}26`,
                '& .MuiAlert-icon': { color: 'primary.main' },
              }}>
                {successMessage}
              </Alert>
              {recoveryKey && (
                <Alert severity="warning" sx={{
                  mb: 2,
                  bgcolor: (theme) => `${theme.palette.warning.main}14`,
                  color: 'warning.light',
                  border: (theme) => `1px solid ${theme.palette.warning.main}26`,
                  '& .MuiAlert-icon': { color: 'warning.light' },
                }}>
                  <Typography variant="subtitle2" gutterBottom sx={{ color: 'warning.light' }}>
                    Save your vault recovery key:
                  </Typography>
                  <Typography
                    variant="body2"
                    sx={{
                      fontFamily: 'monospace',
                      wordBreak: 'break-all',
                      bgcolor: 'background.default',
                      color: 'text.primary',
                      p: 1,
                      borderRadius: 1,
                      border: 1, borderColor: 'divider',
                      userSelect: 'all',
                    }}
                  >
                    {recoveryKey}
                  </Typography>
                  <Typography variant="caption" sx={{ display: 'block', mt: 1, color: 'text.secondary' }}>
                    This key allows you to recover your encrypted vault if you forget your password.
                    Store it in a safe place. It is shown only once.
                  </Typography>
                </Alert>
              )}
              <Typography variant="body2" align="center" sx={{ mb: 2, color: 'text.secondary' }}>
                Didn't receive the email? Check your spam folder or resend it.
              </Typography>
              <Button
                fullWidth
                variant="outlined"
                onClick={handleResend}
                disabled={resendCountdown > 0}
                sx={{
                  mb: 1,
                  borderColor: 'divider',
                  color: 'text.secondary',
                  '&:hover': {
                    borderColor: 'primary.main',
                    color: 'primary.main',
                    bgcolor: (theme) => `${theme.palette.primary.main}0F`,
                  },
                  '&.Mui-disabled': {
                    borderColor: 'divider',
                    color: 'text.disabled',
                  },
                }}
              >
                {resendCountdown > 0
                  ? `Resend verification email (${resendCountdown}s)`
                  : 'Resend verification email'}
              </Button>
              <Typography variant="body2" align="center" sx={{ color: 'text.secondary' }}>
                <Link component={RouterLink} to="/login" sx={{ color: 'primary.main', '&:hover': { color: 'secondary.main' } }}>Go to Sign In</Link>
              </Typography>
            </>
          ) : (
            <>
              <Typography variant="body2" align="center" mb={3} sx={{ color: 'text.secondary' }}>
                Your password is also your vault key
              </Typography>
              {error && <Alert severity="error" sx={{
                mb: 2,
                bgcolor: (theme) => `${theme.palette.error.main}14`,
                color: 'error.light',
                border: (theme) => `1px solid ${theme.palette.error.main}26`,
                '& .MuiAlert-icon': { color: 'error.light' },
              }}>{error}</Alert>}
              <Box component="form" onSubmit={handleSubmit}>
                <OAuthButtons mode="register" />
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
                  helperText="Min 10 characters. This password also encrypts your saved credentials."
                />
                <PasswordStrengthMeter password={password} />
                <TextField
                  fullWidth
                  label="Confirm Password"
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  margin="normal"
                  required
                />
                <Button
                  fullWidth
                  type="submit"
                  variant="contained"
                  disabled={loading}
                  sx={{
                    mt: 2,
                    mb: 1,
                    bgcolor: 'primary.main',
                    color: (theme) => theme.palette.getContrastText(theme.palette.primary.main),
                    fontWeight: 600,
                    '&:hover': {
                      bgcolor: 'secondary.main',
                    },
                    '&.Mui-disabled': {
                      bgcolor: (theme) => `${theme.palette.primary.main}4D`,
                      color: (theme) => theme.palette.getContrastText(theme.palette.primary.main),
                    },
                  }}
                >
                  {loading ? 'Creating account...' : 'Sign Up'}
                </Button>
                <Typography variant="body2" align="center" sx={{ color: 'text.secondary' }}>
                  Already have an account?{' '}
                  <Link component={RouterLink} to="/login" sx={{ color: 'primary.main', '&:hover': { color: 'secondary.main' } }}>Sign in</Link>
                </Typography>
              </Box>
            </>
          )}
      </Box>
    </Box>
  );
}
