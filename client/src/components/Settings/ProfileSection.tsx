import { useState, useEffect, useRef } from 'react';
import {
  Card, CardContent, Typography, TextField, Button, Alert, Avatar, Stack, Box,
} from '@mui/material';
import { useAuthStore } from '../../store/authStore';
import { getProfile, updateProfile, uploadAvatar } from '../../api/user.api';

interface ProfileSectionProps {
  onHasPasswordResolved: (hasPassword: boolean) => void;
  linkedProvider?: string | null;
}

export default function ProfileSection({ onHasPasswordResolved, linkedProvider }: ProfileSectionProps) {
  const updateUser = useAuthStore((s) => s.updateUser);

  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    getProfile().then((profile) => {
      setUsername(profile.username ?? '');
      setEmail(profile.email);
      setAvatarPreview(profile.avatarData);
      onHasPasswordResolved(profile.hasPassword);
    }).catch(() => {
      setError('Failed to load profile');
    });
  }, []);

  useEffect(() => {
    if (linkedProvider) {
      setSuccess(`${linkedProvider.charAt(0).toUpperCase() + linkedProvider.slice(1)} account linked successfully`);
    }
  }, [linkedProvider]);

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (file.size > 200 * 1024) {
      setError('Avatar must be under 200KB');
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      const dataUrl = reader.result as string;
      setAvatarPreview(dataUrl);
      setError('');
      uploadAvatar(dataUrl).then((result) => {
        updateUser({ avatarData: result.avatarData });
        setSuccess('Avatar updated');
      }).catch(() => {
        setError('Failed to upload avatar');
      });
    };
    reader.readAsDataURL(file);
  };

  const handleProfileSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');
    setLoading(true);
    try {
      const result = await updateProfile({
        username: username || undefined,
        email,
      });
      updateUser({ email: result.email, username: result.username });
      setSuccess('Profile updated successfully');
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error || 'Failed to update profile';
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>Profile</Typography>
        <Stack direction="row" alignItems="center" spacing={2} sx={{ mb: 2 }}>
          <Avatar
            src={avatarPreview ?? undefined}
            sx={{ width: 64, height: 64, cursor: 'pointer' }}
            onClick={() => fileInputRef.current?.click()}
          />
          <Button variant="outlined" size="small" onClick={() => fileInputRef.current?.click()}>
            Change Avatar
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            hidden
            onChange={handleAvatarChange}
          />
        </Stack>

        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        {success && <Alert severity="success" sx={{ mb: 2 }}>{success}</Alert>}

        <Box component="form" onSubmit={handleProfileSave}>
          <TextField
            fullWidth label="Username" value={username}
            onChange={(e) => setUsername(e.target.value)}
            margin="normal"
            placeholder="Optional display name"
          />
          <TextField
            fullWidth label="Email" type="email" value={email}
            onChange={(e) => setEmail(e.target.value)}
            margin="normal" required
          />
          <Button
            type="submit" variant="contained" disabled={loading}
            sx={{ mt: 2 }}
          >
            {loading ? 'Saving...' : 'Save Profile'}
          </Button>
        </Box>
      </CardContent>
    </Card>
  );
}
