import { useState } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import IdentityVerification from '../common/IdentityVerification';
import PasswordStrengthMeter from '../common/PasswordStrengthMeter';
import RecoveryKeyConfirmDialog from '../common/RecoveryKeyConfirmDialog';
import { SettingsButtonRow, SettingsFieldGroup, SettingsPanel } from './settings-ui';
import { useAuthStore } from '../../store/authStore';
import {
  changePassword,
  initiatePasswordChange,
  type VerificationMethod,
} from '../../api/user.api';
import { useAsyncAction } from '../../hooks/useAsyncAction';

interface ChangePasswordSectionProps {
  hasPassword: boolean;
}

type Phase =
  | 'idle'
  | 'verifying-identity'
  | 'entering-password'
  | 'showing-recovery-key';

export default function ChangePasswordSection({
  hasPassword,
}: ChangePasswordSectionProps) {
  const authLogout = useAuthStore((state) => state.logout);
  const [phase, setPhase] = useState<Phase>('idle');
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [recoveryKey, setRecoveryKey] = useState('');
  const [skipVerification, setSkipVerification] = useState(false);
  const [verificationId, setVerificationId] = useState('');
  const [verificationMethod, setVerificationMethod] =
    useState<VerificationMethod>('password');
  const [verificationMetadata, setVerificationMetadata] =
    useState<Record<string, unknown>>();
  const [completedVerificationId, setCompletedVerificationId] = useState<
    string | undefined
  >();
  const { loading, error, setError, run } = useAsyncAction();

  if (!hasPassword) return null;

  const resetForm = () => {
    setPhase('idle');
    setOldPassword('');
    setNewPassword('');
    setConfirmPassword('');
    setCompletedVerificationId(undefined);
    setError('');
  };

  const handleStartPasswordChange = async () => {
    await run(async () => {
      const result = await initiatePasswordChange();
      if (result.skipVerification) {
        setSkipVerification(true);
        setPhase('entering-password');
        return;
      }

      setSkipVerification(false);
      setVerificationId(result.verificationId ?? '');
      setVerificationMethod(result.method ?? 'password');
      setVerificationMetadata(result.metadata);
      setPhase('verifying-identity');
    }, 'Failed to initiate password change');
  };

  const handlePasswordChange = async (event: React.FormEvent) => {
    event.preventDefault();
    if (newPassword !== confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    const ok = await run(async () => {
      const result = await changePassword(
        skipVerification ? oldPassword : '',
        newPassword,
        completedVerificationId,
      );
      setRecoveryKey(result.recoveryKey);
    }, 'Failed to change password');

    if (ok) {
      setPhase('showing-recovery-key');
    }
  };

  return (
    <SettingsPanel
      title="Change Password"
      description="Changing your password locks the vault and signs you out on every device."
    >
      <div className="space-y-4">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {phase === 'idle' && (
          <Button type="button" onClick={() => void handleStartPasswordChange()} disabled={loading}>
            {loading ? 'Loading...' : 'Change Password'}
          </Button>
        )}

        {phase === 'verifying-identity' && (
          <IdentityVerification
            verificationId={verificationId}
            method={verificationMethod}
            metadata={verificationMetadata}
            onVerified={(verifiedId) => {
              setCompletedVerificationId(verifiedId);
              setPhase('entering-password');
            }}
            onCancel={resetForm}
          />
        )}

        {phase === 'entering-password' && (
          <form className="space-y-4" onSubmit={handlePasswordChange}>
            <SettingsFieldGroup>
              {skipVerification && (
                <Input
                  type="password"
                  value={oldPassword}
                  onChange={(event) => setOldPassword(event.target.value)}
                  placeholder="Current password"
                  required
                />
              )}
              <div className="space-y-2">
                <Input
                  type="password"
                  value={newPassword}
                  onChange={(event) => setNewPassword(event.target.value)}
                  placeholder="New password"
                  minLength={10}
                  autoFocus={!skipVerification}
                  required
                />
                <p className="text-xs text-muted-foreground">
                  Minimum 10 characters.
                </p>
                <PasswordStrengthMeter password={newPassword} />
              </div>
              <Input
                type="password"
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                placeholder="Confirm new password"
                required
              />
            </SettingsFieldGroup>

            <SettingsButtonRow>
              <Button type="submit" disabled={loading}>
                {loading ? 'Changing...' : 'Change Password'}
              </Button>
              <Button type="button" variant="outline" onClick={resetForm}>
                Cancel
              </Button>
            </SettingsButtonRow>
          </form>
        )}
      </div>

      <RecoveryKeyConfirmDialog
        open={phase === 'showing-recovery-key'}
        recoveryKey={recoveryKey}
        onConfirmed={() => {
          setRecoveryKey('');
          authLogout();
        }}
      />
    </SettingsPanel>
  );
}
