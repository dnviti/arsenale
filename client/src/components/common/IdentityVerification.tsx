import { useState } from 'react';
import { Fingerprint, KeyRound } from 'lucide-react';
import { startAuthentication } from '@simplewebauthn/browser';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { extractApiError } from '../../utils/apiError';
import {
  confirmIdentityVerification,
  type VerificationMethod,
} from '../../api/user.api';

interface IdentityVerificationProps {
  verificationId: string;
  method: VerificationMethod;
  metadata?: Record<string, unknown>;
  onVerified: (verificationId: string) => void;
  onCancel: () => void;
}

const methodLabels: Record<VerificationMethod, string> = {
  email: 'Enter the verification code sent to your email',
  totp: 'Enter the code from your authenticator app',
  sms: 'Enter the verification code sent to your phone',
  webauthn: 'Verify with your security key or passkey',
  password: 'Enter your current password',
};

export default function IdentityVerification({
  verificationId,
  method,
  metadata,
  onVerified,
  onCancel,
}: IdentityVerificationProps) {
  const [code, setCode] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const hint = method === 'email' && metadata?.maskedEmail
    ? `Code sent to ${metadata.maskedEmail}`
    : method === 'sms' && metadata?.maskedPhone
      ? `Code sent to ${metadata.maskedPhone}`
      : undefined;

  const handleSubmit = async () => {
    setError('');
    setLoading(true);

    try {
      let payload: { code?: string; credential?: unknown; password?: string } = {};

      switch (method) {
        case 'email':
        case 'totp':
        case 'sms':
          if (!code || code.length !== 6) {
            setError('Please enter a valid 6-digit code.');
            setLoading(false);
            return;
          }
          payload = { code };
          break;
        case 'webauthn': {
          const options = metadata?.options as Record<string, unknown> | undefined;
          if (!options) {
            setError('WebAuthn options not available.');
            setLoading(false);
            return;
          }
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          const credential = await startAuthentication({ optionsJSON: options as any });
          payload = { credential };
          break;
        }
        case 'password':
          if (!password) {
            setError('Please enter your password.');
            setLoading(false);
            return;
          }
          payload = { password };
          break;
      }

      const result = await confirmIdentityVerification(verificationId, payload);
      if (result.confirmed) {
        onVerified(verificationId);
      } else {
        setError('Verification failed. Please try again.');
      }
    } catch (err: unknown) {
      setError(extractApiError(err, 'Verification failed. Please try again.'));
    } finally {
      setLoading(false);
    }
  };

  const handleKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === 'Enter' && !loading) {
      void handleSubmit();
    }
  };

  return (
    <div className="space-y-4 rounded-xl border border-border/70 bg-background/60 p-4">
      <div className="flex items-center gap-2">
        <Fingerprint className="size-4 text-primary" />
        <h4 className="text-sm font-semibold text-foreground">Identity Verification</h4>
      </div>

      <div className="space-y-1">
        <p className="text-sm text-foreground">{methodLabels[method]}</p>
        {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {(method === 'email' || method === 'totp' || method === 'sms') && (
        <Input
          value={code}
          onChange={(event) => setCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
          onKeyDown={handleKeyDown}
          maxLength={6}
          inputMode="numeric"
          pattern="[0-9]*"
          autoFocus
          placeholder="6-digit code"
        />
      )}

      {method === 'password' && (
        <Input
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          onKeyDown={handleKeyDown}
          autoFocus
          placeholder="Current password"
        />
      )}

      <div className="flex flex-wrap gap-2">
        {method === 'webauthn' ? (
          <Button type="button" onClick={() => void handleSubmit()} disabled={loading}>
            <KeyRound className="size-4" />
            {loading ? 'Verifying...' : 'Verify with Security Key'}
          </Button>
        ) : (
          <Button type="button" onClick={() => void handleSubmit()} disabled={loading}>
            {loading ? 'Verifying...' : 'Verify'}
          </Button>
        )}
        <Button type="button" variant="outline" onClick={onCancel} disabled={loading}>
          Cancel
        </Button>
      </div>
    </div>
  );
}
