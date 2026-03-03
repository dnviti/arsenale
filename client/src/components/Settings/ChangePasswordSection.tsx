import { useState } from 'react';
import {
  Card, CardContent, Typography, TextField, Button, Alert, Box,
} from '@mui/material';
import { useAuthStore } from '../../store/authStore';
import { changePassword } from '../../api/user.api';

interface ChangePasswordSectionProps {
  hasPassword: boolean;
}

export default function ChangePasswordSection({ hasPassword }: ChangePasswordSectionProps) {
  const authLogout = useAuthStore((s) => s.logout);

  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  if (!hasPassword) return null;

  const handlePasswordChange = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    if (newPassword !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    setLoading(true);
    try {
      await changePassword(oldPassword, newPassword);
      setSuccess('Password changed. You will be signed out...');
      setTimeout(() => {
        authLogout();
      }, 2000);
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to change password';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>Change Password</Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Changing your password will lock your vault and sign you out of all devices.
        </Typography>

        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        {success && <Alert severity="success" sx={{ mb: 2 }}>{success}</Alert>}

        <Box component="form" onSubmit={handlePasswordChange}>
          <TextField
            fullWidth label="Current Password" type="password"
            value={oldPassword} onChange={(e) => setOldPassword(e.target.value)}
            margin="normal" required
          />
          <TextField
            fullWidth label="New Password" type="password"
            value={newPassword} onChange={(e) => setNewPassword(e.target.value)}
            margin="normal" required
            helperText="Minimum 8 characters"
          />
          <TextField
            fullWidth label="Confirm New Password" type="password"
            value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)}
            margin="normal" required
          />
          <Button
            type="submit" variant="contained" color="warning"
            disabled={loading}
            sx={{ mt: 2 }}
          >
            {loading ? 'Changing...' : 'Change Password'}
          </Button>
        </Box>
      </CardContent>
    </Card>
  );
}
