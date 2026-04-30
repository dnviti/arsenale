import { useEffect, useState } from 'react';
import { LoaderCircle } from 'lucide-react';
import { useParams } from 'react-router-dom';
import AuthCodeInput from '@/components/auth/AuthCodeInput';
import AuthLayout from '@/components/auth/AuthLayout';
import SecretPayloadView from '../components/secrets/SecretPayloadView';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  accessExternalShare,
  getExternalShareInfo,
  type ExternalShareInfo,
  type SecretPayload,
} from '../api/secrets.api';
import { extractApiError } from '../utils/apiError';

export default function PublicSharePage() {
  const { token } = useParams<{ token: string }>();
  const [info, setInfo] = useState<ExternalShareInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [accessing, setAccessing] = useState(false);
  const [error, setError] = useState('');
  const [pin, setPin] = useState('');
  const [data, setData] = useState<SecretPayload | null>(null);
  const [secretName, setSecretName] = useState('');

  useEffect(() => {
    if (!token) {
      return;
    }
    void loadInfo();
  }, [token]);

  const accessShare = async (pinValue?: string) => {
    setAccessing(true);
    setError('');
    try {
      const result = await accessExternalShare(token as string, pinValue);
      setData(result.data);
      setSecretName(result.secretName);
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status;
      setError(extractApiError(err, status === 403 ? 'Invalid PIN' : 'Failed to access share'));
    } finally {
      setAccessing(false);
    }
  };

  const loadInfo = async () => {
    setLoading(true);
    setError('');
    try {
      const shareInfo = await getExternalShareInfo(token as string);
      setInfo(shareInfo);
      if (!shareInfo.hasPin && !shareInfo.isExpired && !shareInfo.isExhausted && !shareInfo.isRevoked) {
        await accessShare();
      }
    } catch (err: unknown) {
      setError(extractApiError(err, 'Share not found or no longer available'));
    } finally {
      setLoading(false);
    }
  };

  const handlePinSubmit = () => {
    if (!/^\d{4,8}$/.test(pin)) {
      setError('PIN must be 4-8 digits');
      return;
    }
    void accessShare(pin);
  };

  const isUnavailable = info && (info.isExpired || info.isExhausted || info.isRevoked);
  const unavailableReason = info?.isRevoked
    ? 'This share link has been revoked.'
    : info?.isExpired
      ? 'This share link has expired.'
      : info?.isExhausted
        ? 'This share link has reached its access limit.'
        : '';

  return (
    <AuthLayout
      cardClassName="max-w-xl"
      title="Arsenale"
      titleClassName="text-2xl font-semibold"
      description="Shared secret access"
    >
      {loading ? (
        <div className="flex justify-center py-6">
          <LoaderCircle className="size-6 animate-spin text-primary" />
        </div>
      ) : error && !data && !info ? (
        <Alert variant="destructive">
          <AlertDescription className="text-foreground">{error}</AlertDescription>
        </Alert>
      ) : isUnavailable ? (
        <Alert variant="warning">
          <AlertDescription className="text-foreground">{unavailableReason}</AlertDescription>
        </Alert>
      ) : data ? (
        <div className="space-y-4">
          <div className="space-y-1">
            <h2 className="text-lg font-semibold text-foreground">{secretName}</h2>
            <p className="text-sm text-muted-foreground">
              This shared data may expire or become unavailable. Save what you need.
            </p>
          </div>
          <SecretPayloadView data={data} />
        </div>
      ) : info?.hasPin ? (
        <div className="space-y-4">
          <div className="space-y-1">
            <h2 className="text-lg font-semibold text-foreground">{info.secretName}</h2>
            <p className="text-sm text-muted-foreground">
              This secret is protected with a PIN. Enter the PIN to access it.
            </p>
          </div>

          {error ? (
            <Alert variant="destructive">
              <AlertDescription className="text-foreground">{error}</AlertDescription>
            </Alert>
          ) : null}

          <AuthCodeInput
            label="PIN"
            maxLength={8}
            placeholder="Enter PIN"
            value={pin}
            onChange={setPin}
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault();
                handlePinSubmit();
              }
            }}
          />

          <Button type="button" className="w-full" disabled={accessing} onClick={handlePinSubmit}>
            {accessing ? 'Decrypting...' : 'Decrypt'}
          </Button>
        </div>
      ) : (
        <div className="flex justify-center py-6">
          <LoaderCircle className="size-6 animate-spin text-primary" />
        </div>
      )}
    </AuthLayout>
  );
}
