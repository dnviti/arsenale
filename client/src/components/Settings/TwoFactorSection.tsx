import { useEffect, useState } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  SettingsButtonRow,
  SettingsPanel,
  SettingsStatusBadge,
} from './settings-ui';
import {
  disable2FA,
  get2FAStatus,
  setup2FA,
  verify2FA,
} from '../../api/twofa.api';
import { useNotificationStore } from '../../store/notificationStore';
import { extractApiError } from '../../utils/apiError';

type Phase = 'idle' | 'setup' | 'disabling';

export default function TwoFactorSection() {
  const notify = useNotificationStore((state) => state.notify);
  const [enabled, setEnabled] = useState(false);
  const [statusLoading, setStatusLoading] = useState(true);
  const [phase, setPhase] = useState<Phase>('idle');
  const [otpauthUri, setOtpauthUri] = useState('');
  const [secret, setSecret] = useState('');
  const [code, setCode] = useState('');
  const [disableCode, setDisableCode] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    get2FAStatus()
      .then(({ enabled: isEnabled }) => setEnabled(isEnabled))
      .catch(() => {})
      .finally(() => setStatusLoading(false));
  }, []);

  if (statusLoading) return null;

  const resetSetup = () => {
    setPhase('idle');
    setCode('');
    setSecret('');
    setOtpauthUri('');
    setError('');
  };

  const handleStartSetup = async () => {
    setError('');
    setLoading(true);
    try {
      const result = await setup2FA();
      setSecret(result.secret);
      setOtpauthUri(result.otpauthUri);
      setPhase('setup');
    } catch {
      setError('Failed to initialize 2FA setup');
    } finally {
      setLoading(false);
    }
  };

  const handleVerifyAndEnable = async () => {
    setError('');
    setLoading(true);
    try {
      await verify2FA(code);
      setEnabled(true);
      notify('Two-factor authentication enabled successfully', 'success');
      resetSetup();
    } catch (err: unknown) {
      setError(extractApiError(err, 'Invalid code'));
    } finally {
      setLoading(false);
    }
  };

  const handleDisable = async () => {
    setError('');
    setLoading(true);
    try {
      await disable2FA(disableCode);
      setEnabled(false);
      notify('Two-factor authentication disabled', 'success');
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
      title="Two-Factor Authentication"
      description="Protect your account with a time-based authenticator app."
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
          <Button type="button" onClick={() => void handleStartSetup()} disabled={loading}>
            {loading ? 'Setting up...' : 'Enable Two-Factor Authentication'}
          </Button>
        )}

        {phase === 'setup' && (
          <div className="space-y-4">
            <div className="space-y-2">
              <p className="text-sm font-medium text-foreground">
                1. Scan this QR code with your authenticator app.
              </p>
              <div className="inline-flex rounded-xl border bg-background p-4">
                <QRCodeSVG value={otpauthUri} size={200} />
              </div>
            </div>

            <div className="space-y-2">
              <p className="text-sm font-medium text-foreground">
                2. Or enter this code manually.
              </p>
              <Input value={secret} readOnly className="font-mono" />
            </div>

            <div className="space-y-2">
              <p className="text-sm font-medium text-foreground">
                3. Enter the 6-digit code from your app.
              </p>
              <Input
                value={code}
                onChange={(event) =>
                  setCode(event.target.value.replace(/\D/g, '').slice(0, 6))
                }
                inputMode="numeric"
                maxLength={6}
                placeholder="000000"
              />
            </div>

            <SettingsButtonRow>
              <Button
                type="button"
                disabled={loading || code.length !== 6}
                onClick={() => void handleVerifyAndEnable()}
              >
                {loading ? 'Verifying...' : 'Confirm & Enable'}
              </Button>
              <Button type="button" variant="outline" disabled={loading} onClick={resetSetup}>
                Cancel
              </Button>
            </SettingsButtonRow>
          </div>
        )}

        {enabled && phase === 'idle' && (
          <Button type="button" variant="outline" onClick={() => setPhase('disabling')}>
            Disable Two-Factor Authentication
          </Button>
        )}

        {enabled && phase === 'disabling' && (
          <div className="space-y-4">
            <div className="space-y-2">
              <p className="text-sm font-medium text-foreground">
                Enter your current authenticator code to disable 2FA.
              </p>
              <Input
                value={disableCode}
                onChange={(event) =>
                  setDisableCode(event.target.value.replace(/\D/g, '').slice(0, 6))
                }
                inputMode="numeric"
                maxLength={6}
                placeholder="000000"
              />
            </div>

            <SettingsButtonRow>
              <Button
                type="button"
                disabled={loading || disableCode.length !== 6}
                onClick={() => void handleDisable()}
              >
                {loading ? 'Verifying...' : 'Disable 2FA'}
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
