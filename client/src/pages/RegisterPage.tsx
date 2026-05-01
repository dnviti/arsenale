import { useEffect, useRef, useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import AuthLayout from '@/components/auth/AuthLayout';
import AuthLink from '@/components/auth/AuthLink';
import OAuthButtons from '@/components/OAuthButtons';
import PasswordStrengthMeter from '@/components/common/PasswordStrengthMeter';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { getPublicConfig, registerApi } from '../api/auth.api';
import { resendVerificationEmail } from '../api/email.api';
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
      .then((config) => {
        if (!config.selfSignupEnabled) {
          setSignupDisabled(true);
        }
      })
      .catch(() => {
        // Fail open in the UI. The server remains authoritative.
      });
  }, []);

  const resendActive = resendCountdown > 0;
  useEffect(() => {
    if (!resendActive) {
      clearInterval(countdownRef.current);
      return undefined;
    }

    countdownRef.current = setInterval(() => {
      setResendCountdown((previous) => {
        if (previous <= 1) {
          clearInterval(countdownRef.current);
          return 0;
        }
        return previous - 1;
      });
    }, 1000);

    return () => clearInterval(countdownRef.current);
  }, [resendActive]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');

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
      setRecoveryKey(result.recoveryKey ?? '');
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
      // Server returns 200 for any valid email format.
    }
  };

  const passwordsMismatch = confirmPassword.length > 0 && password !== confirmPassword;

  return (
    <AuthLayout
      cardClassName="max-w-md"
      title="Create Account"
      description="Your password is also your vault key."
    >
      {signupDisabled ? (
        <>
          <Alert variant="info">
            <AlertDescription className="text-foreground">
              Public registration is currently disabled. Please contact your organization administrator to get an account.
            </AlertDescription>
          </Alert>
          <p className="text-center text-sm text-muted-foreground">
            Already have an account? <AuthLink to="/login">Sign in</AuthLink>
          </p>
        </>
      ) : registered ? (
        <>
          <Alert variant="success">
            <AlertDescription className="text-foreground">{successMessage}</AlertDescription>
          </Alert>

          {recoveryKey ? (
            <Alert variant="warning">
              <AlertTitle>Save your vault recovery key</AlertTitle>
              <AlertDescription className="space-y-3 text-foreground">
                <div className="rounded-lg border bg-background/80 p-3">
                  <p className="break-all font-mono text-xs leading-6">{recoveryKey}</p>
                </div>
                <p className="text-xs text-muted-foreground">
                  This key allows you to recover your encrypted vault if you forget your password.
                  Store it in a safe place. It is shown only once.
                </p>
              </AlertDescription>
            </Alert>
          ) : null}

          <p className="text-center text-sm text-muted-foreground">
            Didn&apos;t receive the email? Check your spam folder or resend it.
          </p>

          <Button
            type="button"
            variant="outline"
            className="w-full"
            disabled={resendCountdown > 0}
            onClick={() => void handleResend()}
          >
            {resendCountdown > 0
              ? `Resend verification email (${resendCountdown}s)`
              : 'Resend verification email'}
          </Button>

          <p className="text-center text-sm text-muted-foreground">
            <AuthLink to="/login">Go to Sign In</AuthLink>
          </p>
        </>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-4">
          {error ? (
            <Alert variant="destructive">
              <AlertDescription className="text-foreground">{error}</AlertDescription>
            </Alert>
          ) : null}

          <OAuthButtons mode="register" />

          <div className="space-y-2">
            <Label htmlFor="register-email">Email</Label>
            <Input
              id="register-email"
              autoFocus
              required
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="register-password">Password</Label>
            <Input
              id="register-password"
              required
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
            <p className="text-xs text-muted-foreground">Minimum 10 characters.</p>
          </div>

          <PasswordStrengthMeter password={password} />

          <div className="space-y-2">
            <Label htmlFor="register-confirm-password">Confirm Password</Label>
            <Input
              id="register-confirm-password"
              required
              type="password"
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
            />
            {passwordsMismatch ? (
              <p className="text-xs text-destructive">Passwords do not match</p>
            ) : null}
          </div>

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? 'Creating account...' : 'Create Account'}
          </Button>

          <p className="text-center text-sm text-muted-foreground">
            Already have an account? <AuthLink to="/login">Sign in</AuthLink>
          </p>
        </form>
      )}
    </AuthLayout>
  );
}
