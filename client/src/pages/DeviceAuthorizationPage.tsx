import { useMemo, useState, type FormEvent } from 'react';
import { Navigate, useLocation, useSearchParams } from 'react-router-dom';
import { LoaderCircle } from 'lucide-react';
import { authorizeCliDevice } from '@/api/cliAuth.api';
import AuthLayout from '@/components/auth/AuthLayout';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useAuth } from '@/hooks/useAuth';
import { extractApiError } from '@/utils/apiError';

function formatDeviceCode(value: string) {
  const normalized = value.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 8);
  if (normalized.length <= 4) {
    return normalized;
  }
  return `${normalized.slice(0, 4)}-${normalized.slice(4)}`;
}

function normalizedLength(value: string) {
  return value.replace(/[^A-Z0-9]/gi, '').length;
}

export default function DeviceAuthorizationPage() {
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const { isAuthenticated, loading: authLoading } = useAuth();
  const initialCode = useMemo(() => formatDeviceCode(searchParams.get('code') ?? ''), [searchParams]);
  const [code, setCode] = useState(initialCode);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const codeReady = normalizedLength(code) === 8;

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!codeReady) {
      setError('Enter the 8-character device code from the CLI.');
      return;
    }

    setSubmitting(true);
    setError('');
    setSuccess('');
    try {
      const result = await authorizeCliDevice(code);
      setSuccess(result.message || 'Device authorized. Return to your terminal.');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Device authorization failed.'));
    } finally {
      setSubmitting(false);
    }
  };

  if (authLoading) {
    return (
      <div className="flex min-h-screen w-screen items-center justify-center bg-background">
        <LoaderCircle className="size-8 animate-spin text-primary" aria-hidden="true" />
      </div>
    );
  }

  if (!isAuthenticated) {
    const redirect = encodeURIComponent(`${location.pathname}${location.search}`);
    return <Navigate to={`/login?redirect=${redirect}`} replace />;
  }

  return (
    <AuthLayout
      cardClassName="max-w-md"
      title="Authorize Device"
      titleClassName="text-3xl font-normal"
      description="Approve Arsenale CLI access for this account."
    >
      {success ? (
        <Alert variant="success">
          <AlertDescription className="text-foreground">{success}</AlertDescription>
        </Alert>
      ) : null}

      {error ? (
        <Alert variant="destructive">
          <AlertDescription className="text-foreground">{error}</AlertDescription>
        </Alert>
      ) : null}

      <form className="space-y-4" onSubmit={handleSubmit}>
        <div className="space-y-2">
          <Label htmlFor="device-code">Device code</Label>
          <Input
            id="device-code"
            autoComplete="one-time-code"
            autoFocus
            className="font-mono uppercase tracking-[0.3em]"
            inputMode="text"
            maxLength={9}
            placeholder="ABCD-1234"
            value={code}
            onChange={(event) => setCode(formatDeviceCode(event.target.value))}
          />
          <p className="text-xs leading-5 text-muted-foreground">
            Match this code to the one shown by `arsenale login`.
          </p>
        </div>

        <Button type="submit" className="w-full" disabled={submitting || !codeReady || Boolean(success)}>
          {submitting ? 'Authorizing...' : 'Authorize CLI'}
        </Button>
      </form>
    </AuthLayout>
  );
}
