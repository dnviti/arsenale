import type { FormEventHandler } from 'react';
import { Building2, CheckCircle2, Fingerprint, LoaderCircle } from 'lucide-react';
import { QRCodeSVG } from 'qrcode.react';
import AuthCodeInput from '@/components/auth/AuthCodeInput';
import AuthLink from '@/components/auth/AuthLink';
import OAuthButtons from '@/components/OAuthButtons';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';
import {
  loginApi,
  mfaSetupInitApi,
  type AuthSuccessResponse,
  type TenantMembershipInfo,
} from '@/api/auth.api';
import type { OAuthProviders } from '@/api/oauth.api';

interface ProviderPanelProps {
  authProviders?: OAuthProviders | null;
  authProvidersLoading?: boolean;
}

interface PasskeyPanelProps extends ProviderPanelProps {
  loading: boolean;
  onRetry: () => void | Promise<void>;
  onUsePasswordFallback: () => void;
  passkeyFailures: number;
}

export function PasskeyPanel({
  authProviders,
  authProvidersLoading,
  loading,
  onRetry,
  onUsePasswordFallback,
  passkeyFailures,
}: PasskeyPanelProps) {
  return (
    <div className="space-y-4">
      <OAuthButtons
        mode="login"
        loading={authProvidersLoading}
        providers={authProviders}
      />

      <Alert variant="info">
        <AlertDescription className="text-foreground">
          {loading
            ? 'Waiting for your passkey confirmation...'
            : 'Use a passkey to sign in without entering your email and password first.'}
        </AlertDescription>
      </Alert>

      {passkeyFailures > 0 ? (
        <p className="text-center text-sm text-muted-foreground">
          Failed attempts this visit: {passkeyFailures}/3
        </p>
      ) : null}

      {loading ? (
        <div className="flex justify-center py-2">
          <LoaderCircle className="size-6 animate-spin text-primary" />
        </div>
      ) : (
        <Button type="button" className="w-full" onClick={() => void onRetry()}>
          <Fingerprint className="size-4" />
          Retry Passkey
        </Button>
      )}

      <Button type="button" variant="ghost" className="w-full" onClick={onUsePasswordFallback}>
        Use email and password instead
      </Button>

      <p className="text-center text-sm text-muted-foreground">
        Don&apos;t have an account?{' '}
        <AuthLink to="/register">Sign up</AuthLink>
      </p>
    </div>
  );
}

interface CredentialsPanelProps extends ProviderPanelProps {
  email: string;
  ldapEnabled: boolean;
  loading: boolean;
  onEmailChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onReturnToPasskey: () => void;
  onSubmit: FormEventHandler<HTMLFormElement>;
  password: string;
  passkeySupported: boolean;
}

export function CredentialsPanel({
  authProviders,
  authProvidersLoading,
  email,
  ldapEnabled,
  loading,
  onEmailChange,
  onPasswordChange,
  onReturnToPasskey,
  onSubmit,
  password,
  passkeySupported,
}: CredentialsPanelProps) {
  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      <OAuthButtons
        mode="login"
        loading={authProvidersLoading}
        providers={authProviders}
      />

      <div className="space-y-2">
        <Label htmlFor="login-email">Email</Label>
        <Input
          id="login-email"
          autoComplete="username"
          autoFocus
          name="email"
          required
          type="email"
          value={email}
          onChange={(event) => onEmailChange(event.target.value)}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="login-password">Password</Label>
        <Input
          id="login-password"
          autoComplete="current-password"
          name="password"
          required
          type="password"
          value={password}
          onChange={(event) => onPasswordChange(event.target.value)}
        />
      </div>

      <div className="text-right">
        <AuthLink to="/forgot-password" className="text-sm">
          Forgot password?
        </AuthLink>
      </div>

      <Button type="submit" className="w-full" disabled={loading}>
        {loading ? 'Signing in...' : 'Sign In'}
      </Button>

      {ldapEnabled ? (
        <p className="text-center text-xs leading-5 text-muted-foreground">
          LDAP directory login is available. Use your directory credentials above.
        </p>
      ) : null}

      {passkeySupported ? (
        <Button type="button" variant="ghost" className="w-full" onClick={onReturnToPasskey}>
          Try passkey instead
        </Button>
      ) : null}

      <p className="text-center text-sm text-muted-foreground">
        Don&apos;t have an account?{' '}
        <AuthLink to="/register">Sign up</AuthLink>
      </p>
    </form>
  );
}

interface MethodChoicePanelProps {
  loading: boolean;
  methods: string[];
  onBack: () => void;
  onChooseMethod: (method: string) => void | Promise<void>;
  smsSending: boolean;
}

export function MethodChoicePanel({
  loading,
  methods,
  onBack,
  onChooseMethod,
  smsSending,
}: MethodChoicePanelProps) {
  return (
    <div className="space-y-3">
      {methods.includes('email') ? (
        <Button
          type="button"
          variant="outline"
          className="w-full"
          onClick={() => void onChooseMethod('email')}
        >
          Email Code
        </Button>
      ) : null}

      {methods.includes('totp') ? (
        <Button
          type="button"
          variant="outline"
          className="w-full"
          onClick={() => void onChooseMethod('totp')}
        >
          Authenticator App
        </Button>
      ) : null}

      {methods.includes('sms') ? (
        <Button
          type="button"
          variant="outline"
          className="w-full"
          disabled={smsSending}
          onClick={() => void onChooseMethod('sms')}
        >
          {smsSending ? 'Sending...' : 'SMS Code'}
        </Button>
      ) : null}

      {methods.includes('webauthn') ? (
        <Button
          type="button"
          variant="outline"
          className="w-full"
          disabled={loading}
          onClick={() => void onChooseMethod('webauthn')}
        >
          Security Key / Passkey
        </Button>
      ) : null}

      <Button type="button" variant="ghost" className="w-full" onClick={onBack}>
        Back
      </Button>
    </div>
  );
}

interface CodeVerificationPanelProps {
  backLabel?: string;
  infoMessage?: string;
  label: string;
  loading: boolean;
  onBack: () => void;
  onChange: (value: string) => void;
  onSubmit: FormEventHandler<HTMLFormElement>;
  submitLabel?: string;
  submittingLabel?: string;
  value: string;
}

export function CodeVerificationPanel({
  backLabel = 'Back',
  infoMessage,
  label,
  loading,
  onBack,
  onChange,
  onSubmit,
  submitLabel = 'Verify',
  submittingLabel = 'Verifying...',
  value,
}: CodeVerificationPanelProps) {
  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      {infoMessage ? (
        <Alert variant="info">
          <AlertDescription className="text-foreground">{infoMessage}</AlertDescription>
        </Alert>
      ) : null}

      <AuthCodeInput
        autoFocus
        label={label}
        value={value}
        onChange={onChange}
      />

      <Button type="submit" className="w-full" disabled={loading || value.length !== 6}>
        {loading ? submittingLabel : submitLabel}
      </Button>

      <Button type="button" variant="ghost" className="w-full" onClick={onBack}>
        {backLabel}
      </Button>
    </form>
  );
}

interface WebAuthnPanelProps {
  loading: boolean;
  onBack: () => void;
  onRetry: () => void | Promise<void>;
}

export function WebAuthnPanel({ loading, onBack, onRetry }: WebAuthnPanelProps) {
  return (
    <div className="space-y-4">
      <Alert variant="info">
        <AlertDescription className="text-foreground">
          {loading
            ? 'Please interact with your security key or approve the passkey prompt...'
            : 'Click below to authenticate with your security key or passkey.'}
        </AlertDescription>
      </Alert>

      {loading ? (
        <div className="flex justify-center py-2">
          <LoaderCircle className="size-6 animate-spin text-primary" />
        </div>
      ) : (
        <Button type="button" className="w-full" onClick={() => void onRetry()}>
          Retry Authentication
        </Button>
      )}

      <Button type="button" variant="ghost" className="w-full" onClick={onBack}>
        Back
      </Button>
    </div>
  );
}

interface MfaSetupPanelProps {
  loading: boolean;
  onBack: () => void;
  onChange: (value: string) => void;
  onSubmit: FormEventHandler<HTMLFormElement>;
  setupCode: string;
  setupData: Awaited<ReturnType<typeof mfaSetupInitApi>> | null;
}

export function MfaSetupPanel({
  loading,
  onBack,
  onChange,
  onSubmit,
  setupCode,
  setupData,
}: MfaSetupPanelProps) {
  if (!setupData) {
    return (
      <div className="flex justify-center py-6">
        <LoaderCircle className="size-6 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      <Alert variant="warning">
        <AlertTitle>Required by your organization</AlertTitle>
        <AlertDescription className="text-foreground">
          Your organization requires MFA. Set up an authenticator app to continue signing in.
        </AlertDescription>
      </Alert>

      <div className="space-y-3">
        <p className="text-sm text-foreground">
          1. Scan this QR code with your authenticator app:
        </p>
        <div className="flex justify-center rounded-xl border bg-white p-4 shadow-sm">
          <QRCodeSVG value={setupData.otpauthUri} size={180} />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="mfa-secret">2. Or enter this code manually:</Label>
        <Input
          id="mfa-secret"
          readOnly
          value={setupData.secret}
          className="font-mono text-sm"
        />
      </div>

      <AuthCodeInput
        autoFocus
        label="3. Enter the 6-digit code from your app:"
        value={setupCode}
        onChange={onChange}
      />

      <Button type="submit" className="w-full" disabled={loading || setupCode.length !== 6}>
        {loading ? 'Verifying...' : 'Enable MFA & Sign In'}
      </Button>

      <Button type="button" variant="ghost" className="w-full" onClick={onBack}>
        Back
      </Button>
    </form>
  );
}

interface TenantSelectionPanelProps {
  loading: boolean;
  memberships: TenantMembershipInfo[];
  onContinue: () => void | Promise<void>;
  onSelect: (tenantId: string) => void;
  selectedTenantId: string;
}

export function TenantSelectionPanel({
  loading,
  memberships,
  onContinue,
  onSelect,
  selectedTenantId,
}: TenantSelectionPanelProps) {
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        {memberships.map((membership) => {
          const selected = membership.tenantId === selectedTenantId;

          return (
            <button
              key={membership.tenantId}
              type="button"
              aria-pressed={selected}
              className={cn(
                'flex w-full items-center justify-between gap-4 rounded-xl border px-4 py-3 text-left transition-colors',
                selected
                  ? 'border-primary/40 bg-primary/10'
                  : 'border-border bg-card hover:border-border/80 hover:bg-accent/40',
              )}
              onClick={() => onSelect(membership.tenantId)}
            >
              <div className="flex items-start gap-3">
                <span className="inline-flex size-10 items-center justify-center rounded-full bg-muted text-muted-foreground">
                  <Building2 className="size-4" />
                </span>
                <span className="space-y-1">
                  <span className="block text-sm font-medium text-foreground">
                    {membership.name}
                  </span>
                  <span className="block text-sm text-muted-foreground">
                    {membership.role}
                  </span>
                </span>
              </div>
              {selected ? <CheckCircle2 className="size-5 text-primary" /> : null}
            </button>
          );
        })}
      </div>

      <Button
        type="button"
        className="w-full"
        disabled={loading || !selectedTenantId}
        onClick={() => void onContinue()}
      >
        {loading ? 'Selecting...' : 'Continue'}
      </Button>
    </div>
  );
}

type LoginCompletionResponse = AuthSuccessResponse | Awaited<ReturnType<typeof loginApi>>;

export type { LoginCompletionResponse };
