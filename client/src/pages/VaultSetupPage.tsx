import { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import AuthLayout from '@/components/auth/AuthLayout';
import PasswordStrengthMeter from '@/components/common/PasswordStrengthMeter';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { setupVaultPassword } from '../api/oauth.api';
import { useAuthStore } from '../store/authStore';
import { useVaultStore } from '../store/vaultStore';
import { extractApiError } from '../utils/apiError';

export default function VaultSetupPage() {
  const [vaultPassword, setVaultPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();
  const setVaultUnlocked = useVaultStore((state) => state.setUnlocked);
  const updateUser = useAuthStore((state) => state.updateUser);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
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
    <AuthLayout
      cardClassName="max-w-lg"
      title="Set Up Your Vault"
      description="Your vault encrypts all saved connection credentials. This vault password is separate from your OAuth login and cannot be recovered if lost."
    >
      {error ? (
        <Alert variant="destructive">
          <AlertDescription className="text-foreground">{error}</AlertDescription>
        </Alert>
      ) : null}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="vault-password">Vault Password</Label>
          <Input
            id="vault-password"
            autoFocus
            required
            type="password"
            value={vaultPassword}
            onChange={(event) => setVaultPassword(event.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            Minimum 8 characters. This password encrypts your saved credentials.
          </p>
        </div>

        <PasswordStrengthMeter password={vaultPassword} />

        <div className="space-y-2">
          <Label htmlFor="vault-confirm-password">Confirm Vault Password</Label>
          <Input
            id="vault-confirm-password"
            required
            type="password"
            value={confirmPassword}
            onChange={(event) => setConfirmPassword(event.target.value)}
          />
        </div>

        <Button type="submit" className="w-full" disabled={loading}>
          {loading ? 'Setting up...' : 'Set Vault Password'}
        </Button>
      </form>
    </AuthLayout>
  );
}
