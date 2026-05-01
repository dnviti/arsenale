import { useEffect, useState } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  SettingsButtonRow,
  SettingsPanel,
  SettingsStatusBadge,
} from './settings-ui';
import { useNotificationStore } from '../../store/notificationStore';
import { extractApiError } from '../../utils/apiError';
import {
  disableSmsMfa,
  enableSmsMfa,
  getSmsMfaStatus,
  sendSmsMfaDisableCode,
  setupSmsPhone,
  verifySmsPhone,
} from '../../api/smsMfa.api';

type Phase = 'idle' | 'phone-input' | 'verify-phone' | 'disabling';

const PHONE_PATTERN = /^\+[1-9]\d{1,14}$/;

export default function SmsMfaSection() {
  const notify = useNotificationStore((state) => state.notify);
  const [enabled, setEnabled] = useState(false);
  const [phoneNumber, setPhoneNumber] = useState<string | null>(null);
  const [statusLoading, setStatusLoading] = useState(true);
  const [phase, setPhase] = useState<Phase>('idle');
  const [phoneInput, setPhoneInput] = useState('');
  const [code, setCode] = useState('');
  const [disableCode, setDisableCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    getSmsMfaStatus()
      .then((status) => {
        setEnabled(status.enabled);
        setPhoneNumber(status.phoneNumber);
      })
      .catch(() => {})
      .finally(() => setStatusLoading(false));
  }, []);

  if (statusLoading) return null;

  const resetSetup = () => {
    setPhase('idle');
    setPhoneInput('');
    setCode('');
    setError('');
  };

  const handleSubmitPhone = async () => {
    setError('');
    setLoading(true);
    try {
      await setupSmsPhone(phoneInput);
      setPhase('verify-phone');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to send verification code'));
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyAndEnable = async () => {
    setError('');
    setLoading(true);
    try {
      await verifySmsPhone(code);
      await enableSmsMfa();
      setEnabled(true);
      notify('SMS MFA enabled successfully', 'success');
      resetSetup();
    } catch (err: unknown) {
      setError(extractApiError(err, 'Invalid code'));
    } finally {
      setLoading(false);
    }
  };

  const handleStartDisable = async () => {
    setError('');
    setLoading(true);
    try {
      await sendSmsMfaDisableCode();
      setPhase('disabling');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to send verification code'));
    } finally {
      setLoading(false);
    }
  };

  const handleDisable = async () => {
    setError('');
    setLoading(true);
    try {
      await disableSmsMfa(disableCode);
      setEnabled(false);
      setPhoneNumber(null);
      notify('SMS MFA disabled', 'success');
      setPhase('idle');
      setDisableCode('');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Invalid code'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <SettingsPanel
      title="SMS Authentication"
      description="Receive sign-in codes by SMS as a fallback or alternative second factor."
      heading={
        <SettingsStatusBadge tone={enabled ? 'success' : 'neutral'}>
          {enabled ? 'Enabled' : 'Disabled'}
        </SettingsStatusBadge>
      }
    >
      <div className="space-y-4">
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {!enabled && phase === 'idle' && (
          <Button type="button" onClick={() => setPhase('phone-input')} disabled={loading}>
            {loading ? 'Setting up...' : 'Enable SMS Authentication'}
          </Button>
        )}

        {phase === 'phone-input' && (
          <div className="space-y-4">
            <div className="space-y-2">
              <p className="text-sm font-medium text-foreground">
                Enter your phone number in international format.
              </p>
              <Input
                value={phoneInput}
                onChange={(event) => setPhoneInput(event.target.value)}
                placeholder="+1234567890"
              />
              <p className="text-xs text-muted-foreground">
                Use E.164 format, for example `+1234567890`.
              </p>
            </div>

            <SettingsButtonRow>
              <Button
                type="button"
                disabled={loading || !PHONE_PATTERN.test(phoneInput)}
                onClick={() => void handleSubmitPhone()}
              >
                {loading ? 'Sending...' : 'Send Verification Code'}
              </Button>
              <Button type="button" variant="outline" disabled={loading} onClick={resetSetup}>
                Cancel
              </Button>
            </SettingsButtonRow>
          </div>
        )}

        {phase === 'verify-phone' && (
          <div className="space-y-4">
            <Alert variant="info">
              <AlertDescription>
                A verification code has been sent to {phoneInput}.
              </AlertDescription>
            </Alert>

            <Input
              value={code}
              onChange={(event) => setCode(event.target.value.replace(/\D/g, '').slice(0, 6))}
              inputMode="numeric"
              maxLength={6}
              placeholder="000000"
            />

            <SettingsButtonRow>
              <Button
                type="button"
                disabled={loading || code.length !== 6}
                onClick={() => void handleVerifyAndEnable()}
              >
                {loading ? 'Verifying...' : 'Verify & Enable'}
              </Button>
              <Button type="button" variant="outline" disabled={loading} onClick={resetSetup}>
                Cancel
              </Button>
            </SettingsButtonRow>
          </div>
        )}

        {enabled && phase === 'idle' && (
          <div className="space-y-3">
            {phoneNumber && (
              <p className="text-sm text-foreground">
                Phone: <span className="font-medium">{phoneNumber}</span>
              </p>
            )}
            <Button type="button" variant="outline" disabled={loading} onClick={() => void handleStartDisable()}>
              {loading ? 'Sending code...' : 'Disable SMS Authentication'}
            </Button>
          </div>
        )}

        {enabled && phase === 'disabling' && (
          <div className="space-y-4">
            <Alert variant="info">
              <AlertDescription>
                A verification code has been sent to your phone.
              </AlertDescription>
            </Alert>

            <Input
              value={disableCode}
              onChange={(event) =>
                setDisableCode(event.target.value.replace(/\D/g, '').slice(0, 6))
              }
              inputMode="numeric"
              maxLength={6}
              placeholder="000000"
            />

            <SettingsButtonRow>
              <Button
                type="button"
                disabled={loading || disableCode.length !== 6}
                onClick={() => void handleDisable()}
              >
                {loading ? 'Verifying...' : 'Disable SMS MFA'}
              </Button>
              <Button
                type="button"
                variant="outline"
                disabled={loading}
                onClick={() => {
                  setPhase('idle');
                  setDisableCode('');
                  setError('');
                }}
              >
                Cancel
              </Button>
            </SettingsButtonRow>
          </div>
        )}
      </div>
    </SettingsPanel>
  );
}
