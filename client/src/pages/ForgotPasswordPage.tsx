import { useState } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import {
  Box, Card, CardContent, TextField, Button, Typography, Alert, Link,
} from '@mui/material';
import { forgotPasswordApi } from '../api/passwordReset.api';
import { extractApiError } from '../utils/apiError';

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      await forgotPasswordApi(email);
      setSent(true);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Request failed. Please try again.'));
    } finally {
      setLoading(false);
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
        background: (theme) => `radial-gradient(ellipse at 50% 0%, ${theme.palette.primary.main}0A 0%, ${theme.palette.background.default} 70%)`,
      }}
    >
      <Card sx={{
        width: 400,
        maxWidth: '90vw',
        bgcolor: 'background.paper',
        border: 1, borderColor: 'divider',
        borderRadius: 4,
        boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
      }}>
        <CardContent sx={{ p: 4 }}>
          <Box sx={{ display: 'flex', justifyContent: 'center', mb: 2 }}>
            <Box sx={{ width: 32, height: 3, borderRadius: 1, bgcolor: 'primary.main' }} />
          </Box>
          <Typography variant="h5" gutterBottom align="center" sx={{
            fontFamily: (theme) => theme.typography.h5.fontFamily,
            fontSize: '1.75rem',
            color: 'text.primary',
          }}>
            Reset Password
          </Typography>

          {sent ? (
            <>
              <Alert severity="success" sx={{ mb: 2, bgcolor: (theme) => `${theme.palette.primary.main}14`, color: 'primary.main', '& .MuiAlert-icon': { color: 'primary.main' } }}>
                If an account exists with that email, a password reset link has been sent.
                Check your inbox and spam folder.
              </Alert>
              <Typography variant="body2" align="center">
                <Link component={RouterLink} to="/login" sx={{ color: 'primary.main', '&:hover': { color: 'secondary.main' } }}>Back to Sign In</Link>
              </Typography>
            </>
          ) : (
            <>
              <Typography variant="body2" align="center" mb={3} sx={{ color: 'text.secondary' }}>
                Enter your email address and we'll send you a link to reset your password.
              </Typography>
              {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
              <Box component="form" onSubmit={handleSubmit}>
                <TextField
                  fullWidth
                  label="Email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  margin="normal"
                  required
                  autoFocus
                  sx={{
                    '& .MuiOutlinedInput-root': {
                      bgcolor: 'background.default',
                      color: 'text.primary',
                      '& fieldset': { borderColor: 'divider' },
                      '&:hover fieldset': { borderColor: 'text.disabled' },
                      '&.Mui-focused fieldset': { borderColor: 'primary.main' },
                    },
                    '& .MuiInputLabel-root': { color: 'text.secondary' },
                    '& .MuiInputLabel-root.Mui-focused': { color: 'primary.main' },
                  }}
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
                    '&:hover': { bgcolor: 'secondary.main' },
                    '&.Mui-disabled': { bgcolor: (theme) => `${theme.palette.primary.main}4D`, color: (theme) => theme.palette.getContrastText(theme.palette.primary.main) },
                  }}
                >
                  {loading ? 'Sending...' : 'Send Reset Link'}
                </Button>
                <Typography variant="body2" align="center">
                  <Link component={RouterLink} to="/login" sx={{ color: 'primary.main', '&:hover': { color: 'secondary.main' } }}>Back to Sign In</Link>
                </Typography>
              </Box>
            </>
          )}
        </CardContent>
      </Card>
    </Box>
  );
}
