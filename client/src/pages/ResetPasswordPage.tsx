import { useCallback, useEffect, useState, type FormEvent } from 'react';
import { LoaderCircle } from 'lucide-react';
import { useSearchParams } from 'react-router-dom';
import AuthCodeInput from '@/components/auth/AuthCodeInput';
import AuthLayout from '@/components/auth/AuthLayout';
import AuthLink from '@/components/auth/AuthLink';
import PasswordStrengthMeter from '@/components/common/PasswordStrengthMeter';
import RecoveryKeyConfirmDialog from '@/components/common/RecoveryKeyConfirmDialog';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  completePasswordResetApi,
  requestResetSmsCodeApi,
  validateResetTokenApi,
} from '../api/passwordReset.api';
import { extractApiError } from '../utils/apiError';

type Step = 'validating' | 'sms' | 'form' | 'recovery-key' | 'success' | 'error';

const STEP_DESCRIPTIONS: Record<Step, string> = {
  validating: 'Checking your reset link.',
  sms: 'Verify your phone number before resetting your password.',
  form: 'Choose a new password for your account.',
  'recovery-key': 'Save your new recovery key before continuing.',
  success: 'Your password has been updated.',
  error: 'This reset link is invalid or has expired.',
};

export default function ResetPasswordPage() {
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') || '';

  const [step, setStep] = useState<Step>('validating');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [requiresSms, setRequiresSms] = useState(false);
  const [maskedPhone, setMaskedPhone] = useState('');
  const [smsCode, setSmsCode] = useState('');
  const [smsSent, setSmsSent] = useState(false);
  const [smsSending, setSmsSending] = useState(false);
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [hasRecoveryKey, setHasRecoveryKey] = useState(false);
  const [showRecoveryInput, setShowRecoveryInput] = useState(false);
  const [recoveryKey, setRecoveryKey] = useState('');
  const [vaultPreserved, setVaultPreserved] = useState(false);
  const [newRecoveryKey, setNewRecoveryKey] = useState('');

  const validateToken = useCallback(async () => {
    if (!token) {
      setError('No reset token provided.');
      setStep('error');
      return;
    }

    try {
      const result = await validateResetTokenApi(token);
      if (!result.valid) {
        setError('This reset link is invalid or has expired.');
        setStep('error');
        return;
      }

      setRequiresSms(result.requiresSmsVerification);
      setMaskedPhone(result.maskedPhone || '');
      setHasRecoveryKey(result.hasRecoveryKey);
      setStep(result.requiresSmsVerification ? 'sms' : 'form');
    } catch {
      setError('This reset link is invalid or has expired.');
      setStep('error');
    }
  }, [token]);

  useEffect(() => {
    void validateToken();
  }, [validateToken]);

  const handleSendSms = async () => {
    setSmsSending(true);
    setError('');
    try {
      await requestResetSmsCodeApi(token);
      setSmsSent(true);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to send SMS code'));
    } finally {
      setSmsSending(false);
    }
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');

    if (newPassword !== confirmPassword) {
      setError('Passwords do not match.');
      return;
    }
    if (newPassword.length < 10) {
      setError('Password must be at least 10 characters.');
      return;
    }

    setLoading(true);
    try {
      const result = await completePasswordResetApi({
        token,
        newPassword,
        smsCode: requiresSms ? smsCode : undefined,
        recoveryKey: recoveryKey || undefined,
      });
      setVaultPreserved(result.vaultPreserved);
      setNewRecoveryKey(result.newRecoveryKey || '');
      setStep(result.newRecoveryKey ? 'recovery-key' : 'success');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Password reset failed. Please try again.'));
    } finally {
      setLoading(false);
    }
  };

  const passwordsMismatch = confirmPassword.length > 0 && newPassword !== confirmPassword;

  return (
    <>
      <AuthLayout
        cardClassName="max-w-lg"
        title="Reset Password"
        description={STEP_DESCRIPTIONS[step]}
      >
        {step === 'validating' ? (
          <div className="flex justify-center py-6">
            <LoaderCircle className="size-6 animate-spin text-primary" />
          </div>
        ) : null}

        {step === 'error' ? (
          <>
            <Alert variant="destructive">
              <AlertDescription className="text-foreground">{error}</AlertDescription>
            </Alert>
            <p className="text-center text-sm text-muted-foreground">
              <AuthLink to="/forgot-password">Request a new reset link</AuthLink>
            </p>
          </>
        ) : null}

        {step === 'sms' ? (
          <div className="space-y-4">
            <p className="text-center text-sm text-muted-foreground">
              Your account has SMS verification enabled. Please verify your phone number to continue.
            </p>

            {error ? (
              <Alert variant="destructive">
                <AlertDescription className="text-foreground">{error}</AlertDescription>
              </Alert>
            ) : null}

            {!smsSent ? (
              <>
                <Alert variant="info">
                  <AlertDescription className="text-foreground">
                    A verification code will be sent to {maskedPhone}.
                  </AlertDescription>
                </Alert>
                <Button type="button" className="w-full" disabled={smsSending} onClick={() => void handleSendSms()}>
                  {smsSending ? 'Sending...' : 'Send SMS Code'}
                </Button>
              </>
            ) : (
              <>
                <Alert variant="info">
                  <AlertDescription className="text-foreground">
                    A verification code has been sent to {maskedPhone}.
                  </AlertDescription>
                </Alert>
                <AuthCodeInput
                  autoFocus
                  label="SMS Code"
                  value={smsCode}
                  onChange={setSmsCode}
                />
                <Button
                  type="button"
                  className="w-full"
                  disabled={smsCode.length !== 6}
                  onClick={() => setStep('form')}
                >
                  Continue
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  className="w-full"
                  disabled={smsSending}
                  onClick={() => void handleSendSms()}
                >
                  Resend Code
                </Button>
              </>
            )}
          </div>
        ) : null}

        {step === 'form' ? (
          <form onSubmit={handleSubmit} className="space-y-4">
            <p className="text-center text-sm text-muted-foreground">Enter your new password.</p>

            {error ? (
              <Alert variant="destructive">
                <AlertDescription className="text-foreground">{error}</AlertDescription>
              </Alert>
            ) : null}

            <div className="space-y-2">
              <Label htmlFor="reset-new-password">New Password</Label>
              <Input
                id="reset-new-password"
                autoFocus
                required
                type="password"
                value={newPassword}
                onChange={(event) => setNewPassword(event.target.value)}
              />
              <p className="text-xs text-muted-foreground">Minimum 10 characters.</p>
            </div>

            <PasswordStrengthMeter password={newPassword} />

            <div className="space-y-2">
              <Label htmlFor="reset-confirm-password">Confirm New Password</Label>
              <Input
                id="reset-confirm-password"
                required
                type="password"
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
              />
              {passwordsMismatch ? (
                <p className="text-xs text-destructive">Passwords do not match.</p>
              ) : null}
            </div>

            {hasRecoveryKey ? (
              <div className="space-y-3">
                <Button
                  type="button"
                  variant="ghost"
                  className="px-0 text-primary hover:text-primary/80"
                  onClick={() => setShowRecoveryInput((previous) => !previous)}
                >
                  {showRecoveryInput ? 'Hide recovery key input' : 'I have a vault recovery key'}
                </Button>

                {showRecoveryInput ? (
                  <div className="space-y-3">
                    <Alert variant="info">
                      <AlertDescription className="text-foreground">
                        Enter your vault recovery key to preserve your saved credentials.
                        Without it, your encrypted vault data will be reset.
                      </AlertDescription>
                    </Alert>
                    <div className="space-y-2">
                      <Label htmlFor="reset-recovery-key">Vault Recovery Key</Label>
                      <Input
                        id="reset-recovery-key"
                        placeholder="Enter your recovery key"
                        type="text"
                        value={recoveryKey}
                        onChange={(event) => setRecoveryKey(event.target.value.trim())}
                      />
                    </div>
                  </div>
                ) : null}
              </div>
            ) : null}

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? 'Resetting...' : 'Reset Password'}
            </Button>
          </form>
        ) : null}

        {step === 'success' ? (
          <>
            <Alert variant="success">
              <AlertDescription className="text-foreground">
                Your password has been reset successfully.
              </AlertDescription>
            </Alert>
            {vaultPreserved ? (
              <Alert variant="info">
                <AlertDescription className="text-foreground">
                  Your vault data has been preserved.
                </AlertDescription>
              </Alert>
            ) : (
              <Alert variant="warning">
                <AlertDescription className="text-foreground">
                  Your vault is locked. Enter your recovery key in Keychain to restore access to your credentials.
                </AlertDescription>
              </Alert>
            )}
            <p className="text-center text-sm text-muted-foreground">
              <AuthLink to="/login?passwordReset=true">Go to Sign In</AuthLink>
            </p>
          </>
        ) : null}
      </AuthLayout>

      <RecoveryKeyConfirmDialog
        open={step === 'recovery-key'}
        recoveryKey={newRecoveryKey}
        onConfirmed={() => {
          setNewRecoveryKey('');
          setStep('success');
        }}
      />
    </>
  );
}
