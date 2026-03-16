import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Box, Card, CardContent, TextField, Button, Typography, Alert } from '@mui/material';
import { setupVaultPassword } from '../api/oauth.api';
import { useVaultStore } from '../store/vaultStore';
import { useAuthStore } from '../store/authStore';
import { extractApiError } from '../utils/apiError';

export default function VaultSetupPage() {
  const [vaultPassword, setVaultPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const setVaultUnlocked = useVaultStore((s) => s.setUnlocked);
  const updateUser = useAuthStore((s) => s.updateUser);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (vaultPassword !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }
    if (vaultPassword.length < 8) {
      setError('Vault password must be at least 8 characters');
      return;
    }

    setLoading(true);
    try {
      await setupVaultPassword(vaultPassword);
      setVaultUnlocked(true);
      updateUser({ vaultSetupComplete: true });
      navigate('/', { replace: true });
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to set up vault'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box sx={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      bgcolor: 'background.default',
      background: (theme) => `radial-gradient(ellipse at 50% 0%, ${theme.palette.primary.main}08 0%, ${theme.palette.background.default} 70%)`,
    }}>
      <Card sx={{
        width: 450,
        maxWidth: '90vw',
        bgcolor: 'background.paper',
        border: 1, borderColor: 'divider',
        borderRadius: 4,
        boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
      }}>
        <CardContent sx={{ p: 4 }}>
          <Box sx={{ display: 'flex', justifyContent: 'center', mb: 2 }}>
            <Box sx={{
              width: 6,
              height: 6,
              borderRadius: '50%',
              bgcolor: 'primary.main',
              boxShadow: (theme) => `0 0 8px ${theme.palette.primary.main}66`,
            }} />
          </Box>
          <Typography variant="h4" gutterBottom align="center" sx={{
            fontFamily: (theme) => theme.typography.h4.fontFamily,
            color: 'text.primary',
            fontWeight: 600,
            letterSpacing: '-0.01em',
          }}>
            Set Up Your Vault
          </Typography>
          <Typography variant="body2" align="center" sx={{ mb: 1, color: 'text.secondary' }}>
            Your vault encrypts all saved connection credentials.
          </Typography>
          <Typography variant="body2" align="center" sx={{ mb: 3, color: 'text.secondary' }}>
            This vault password is separate from your OAuth login and cannot be recovered if lost.
          </Typography>

          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

          <Box component="form" onSubmit={handleSubmit}>
            <TextField
              fullWidth
              label="Vault Password"
              type="password"
              value={vaultPassword}
              onChange={(e) => setVaultPassword(e.target.value)}
              margin="normal"
              required
              helperText="Min 8 characters. This password encrypts your saved credentials."
              sx={{
                '& .MuiOutlinedInput-root': {
                  bgcolor: 'background.default',
                  '& fieldset': { borderColor: 'divider' },
                  '&:hover fieldset': { borderColor: 'text.disabled' },
                  '&.Mui-focused fieldset': { borderColor: 'primary.main' },
                },
                '& .MuiInputLabel-root': { color: 'text.secondary' },
                '& .MuiInputBase-input': { color: 'text.primary' },
                '& .MuiFormHelperText-root': { color: 'text.secondary' },
              }}
            />
            <TextField
              fullWidth
              label="Confirm Vault Password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              margin="normal"
              required
              sx={{
                '& .MuiOutlinedInput-root': {
                  bgcolor: 'background.default',
                  '& fieldset': { borderColor: 'divider' },
                  '&:hover fieldset': { borderColor: 'text.disabled' },
                  '&.Mui-focused fieldset': { borderColor: 'primary.main' },
                },
                '& .MuiInputLabel-root': { color: 'text.secondary' },
                '& .MuiInputBase-input': { color: 'text.primary' },
              }}
            />
            <Button
              fullWidth
              type="submit"
              variant="contained"
              disabled={loading}
              sx={{
                mt: 3,
                py: 1.4,
                bgcolor: 'primary.main',
                color: (theme) => theme.palette.getContrastText(theme.palette.primary.main),
                fontWeight: 600,
                textTransform: 'none',
                fontSize: '0.95rem',
                borderRadius: 2,
                '&:hover': { bgcolor: 'secondary.main' },
                '&.Mui-disabled': { bgcolor: (theme) => `${theme.palette.primary.main}4D`, color: (theme) => theme.palette.getContrastText(theme.palette.primary.main) },
              }}
            >
              {loading ? 'Setting up...' : 'Set Vault Password'}
            </Button>
          </Box>
        </CardContent>
      </Card>
    </Box>
  );
}
