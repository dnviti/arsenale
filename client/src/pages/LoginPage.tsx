import { useEffect, useRef, useState, type FormEvent } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import AuthLayout from '@/components/auth/AuthLayout';
import {
  loginApi,
  mfaSetupInitApi,
  mfaSetupVerifyApi,
  requestEmailCodeApi,
  requestPasskeyOptionsApi,
  requestSmsCodeApi,
  requestWebAuthnOptionsApi,
  verifyEmailCodeApi,
  verifyPasskeyApi,
  verifySmsApi,
  verifyTotpApi,
  verifyWebAuthnApi,
  type AuthSuccessResponse,
  type TenantMembershipInfo,
} from '@/api/auth.api';
import { resendVerificationEmail } from '@/api/email.api';
import { getOAuthProviders, type OAuthProviders } from '@/api/oauth.api';
import { switchTenant as switchTenantApi } from '@/api/tenant.api';
import {
  browserSupportsWebAuthn,
  startAuthentication,
} from '@simplewebauthn/browser';
import { useAuthStore } from '@/store/authStore';
import { useUiPreferencesStore } from '@/store/uiPreferencesStore';
import { extractApiError } from '@/utils/apiError';
import {
  CodeVerificationPanel,
  CredentialsPanel,
  MethodChoicePanel,
  MfaSetupPanel,
  PasskeyPanel,
  TenantSelectionPanel,
  WebAuthnPanel,
  type LoginCompletionResponse,
} from './login/loginPageStepPanels';
import { LOGIN_STEP_SUBTITLES, type LoginStep } from './login/loginPageUtils';

const LDAP_PROVIDER_NAME = 'LDAP';

export default function LoginPage() {
  const [authProviders, setAuthProviders] = useState<OAuthProviders | null>(null);
  const [authProvidersLoading, setAuthProvidersLoading] = useState(true);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [passkeySupported, setPasskeySupported] = useState(() => browserSupportsWebAuthn());
  const [passkeyAttempted, setPasskeyAttempted] = useState(false);
  const [passkeyFailures, setPasskeyFailures] = useState(0);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [loading, setLoading] = useState(false);
  const [step, setStep] = useState<LoginStep>(() => {
    const last = useUiPreferencesStore.getState().lastLoginMethod;
    if (last === 'passkey' && browserSupportsWebAuthn()) {
      return 'passkey';
    }
    return 'credentials';
  });
  const [tempToken, setTempToken] = useState('');
  const [emailCode, setEmailCode] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [smsCode, setSmsCode] = useState('');
  const [mfaMethods, setMfaMethods] = useState<string[]>([]);
  const [smsSending, setSmsSending] = useState(false);
  const [mfaSetupData, setMfaSetupData] = useState<Awaited<ReturnType<typeof mfaSetupInitApi>> | null>(null);
  const [mfaSetupCode, setMfaSetupCode] = useState('');
  const [showResend, setShowResend] = useState(false);
  const [resendCountdown, setResendCountdown] = useState(0);
  const [pendingLoginData, setPendingLoginData] = useState<AuthSuccessResponse | null>(null);
  const [tenantMemberships, setTenantMemberships] = useState<TenantMembershipInfo[]>([]);
  const [selectedTenantId, setSelectedTenantId] = useState('');
  const countdownRef = useRef<ReturnType<typeof setInterval>>(undefined);
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const setAuth = useAuthStore((state) => state.setAuth);

  useEffect(() => {
    let active = true;

    setPasskeySupported(browserSupportsWebAuthn());
    getOAuthProviders()
      .then((providers) => {
        if (!active) {
          return;
        }
        setAuthProviders(providers);
      })
      .catch(() => {
        if (!active) {
          return;
        }
        setAuthProviders(null);
      })
      .finally(() => {
        if (active) {
          setAuthProvidersLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    const nextParams = new URLSearchParams(searchParams);
    let changed = false;

    if (nextParams.get('verified') === 'true') {
      setSuccess('Email verified successfully! You can now sign in.');
      nextParams.delete('verified');
      changed = true;
    }

    if (nextParams.get('registered') === 'true') {
      setSuccess('Registration successful! You can now sign in.');
      nextParams.delete('registered');
      changed = true;
    }

    if (nextParams.get('passwordReset') === 'true') {
      setSuccess('Password reset successful! You can now sign in with your new password.');
      nextParams.delete('passwordReset');
      changed = true;
    }

    const verifyError = nextParams.get('verifyError');
    if (verifyError) {
      setError(verifyError);
      nextParams.delete('verifyError');
      changed = true;
    }

    const oauthError = nextParams.get('error');
    if (oauthError) {
      const errorCode = nextParams.get('code');
      if (errorCode === 'registration_disabled') {
        setError(
          'Public registration is currently disabled. Contact your organization administrator to get an account.',
        );
      } else if (errorCode === 'account_disabled') {
        setError('Your account has been disabled. Contact your organization administrator.');
      } else {
        setError(decodeURIComponent(oauthError));
      }
      nextParams.delete('error');
      nextParams.delete('code');
      changed = true;
    }

    if (changed) {
      setSearchParams(nextParams, { replace: true });
    }
  }, [searchParams, setSearchParams]);

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

  const buildRedirect = () => {
    const params = new URLSearchParams();
    const autoconnect = searchParams.get('autoconnect');
    const action = searchParams.get('action');

    if (autoconnect) {
      params.set('autoconnect', autoconnect);
    }
    if (action) {
      params.set('action', action);
    }

    const query = params.toString();
    return query ? `/?${query}` : '/';
  };

  const completeLogin = (data: AuthSuccessResponse) => {
    const memberships = data.tenantMemberships ?? [];
    const acceptedMemberships = memberships.filter((membership) => !membership.pending);

    if (acceptedMemberships.length >= 2) {
      setPendingLoginData(data);
      setTenantMemberships(acceptedMemberships);

      const lastActiveTenantId = useUiPreferencesStore.getState().lastActiveTenantId;
      const preselectedMembership =
        acceptedMemberships.find((membership) => membership.tenantId === lastActiveTenantId)
        ?? acceptedMemberships.find((membership) => membership.isActive)
        ?? acceptedMemberships[0];

      setSelectedTenantId(preselectedMembership.tenantId);
      setStep('tenant-select');
      return;
    }

    setAuth(data.accessToken, data.csrfToken, data.user);
    const activeMembership =
      memberships.find((membership) => membership.isActive) ?? acceptedMemberships[0];
    if (activeMembership) {
      useUiPreferencesStore.getState().set('lastActiveTenantId', activeMembership.tenantId);
    }
    navigate(buildRedirect());
  };

  const handleTenantConfirm = async () => {
    if (!pendingLoginData || !selectedTenantId) {
      return;
    }

    setError('');
    setLoading(true);
    try {
      setAuth(
        pendingLoginData.accessToken,
        pendingLoginData.csrfToken,
        pendingLoginData.user,
      );

      const activeMembership = tenantMemberships.find((membership) => membership.isActive);
      if (!activeMembership || activeMembership.tenantId !== selectedTenantId) {
        const result = await switchTenantApi(selectedTenantId);
        setAuth(result.accessToken, result.csrfToken, result.user);
      }

      useUiPreferencesStore.getState().set('lastActiveTenantId', selectedTenantId);
      navigate(buildRedirect());
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to select organization'));
    } finally {
      setLoading(false);
    }
  };

  const handleWebAuthnAuth = async (token?: string) => {
    const resolvedToken = token || tempToken;
    setError('');
    setLoading(true);
    try {
      const options = await requestWebAuthnOptionsApi(resolvedToken);
      const credential = await startAuthentication({ optionsJSON: options });
      const data = await verifyWebAuthnApi(resolvedToken, credential, options.challenge);
      completeLogin(data);
    } catch (err: unknown) {
      if ((err as Error)?.name === 'NotAllowedError') {
        setError('Authentication was cancelled or timed out.');
      } else {
        setError(extractApiError(err, 'WebAuthn authentication failed.'));
      }
    } finally {
      setLoading(false);
    }
  };

  const applyAuthResponse = async (data: LoginCompletionResponse) => {
    if ('requiresMFA' in data && data.requiresMFA) {
      setTempToken(data.tempToken);
      setMfaMethods(data.methods);

      if (data.methods.length === 1) {
        if (data.methods[0] === 'email') {
          await requestEmailCodeApi(data.tempToken);
          setStep('email');
          return;
        }
        if (data.methods[0] === 'totp') {
          setStep('totp');
          return;
        }
        if (data.methods[0] === 'webauthn') {
          setStep('webauthn');
          void handleWebAuthnAuth(data.tempToken);
          return;
        }
        await requestSmsCodeApi(data.tempToken);
        setStep('sms');
        return;
      }

      setStep('mfa-choice');
      return;
    }

    if ('mfaSetupRequired' in data && data.mfaSetupRequired) {
      setTempToken(data.tempToken);
      setStep('mfa-setup');
      try {
        const setupData = await mfaSetupInitApi(data.tempToken);
        setMfaSetupData(setupData);
      } catch {
        setError('Failed to initialize MFA setup');
      }
      return;
    }

    if ('requiresTOTP' in data && data.requiresTOTP) {
      setTempToken(data.tempToken);
      setStep('totp');
      return;
    }

    if ('accessToken' in data) {
      completeLogin(data);
    }
  };

  const registerPasskeyFailure = (message: string) => {
    setPasskeyFailures((previous) => {
      const next = previous + 1;
      if (next >= 3) {
        setStep('credentials');
      }
      return next;
    });
    setError(message);
  };

  const handlePasskeyAuth = async () => {
    if (!passkeySupported) {
      setStep('credentials');
      return;
    }

    setPasskeyAttempted(true);
    setError('');
    setLoading(true);
    try {
      const response = await requestPasskeyOptionsApi();
      const credential = await startAuthentication({
        optionsJSON: response.options,
      });
      const data = await verifyPasskeyApi(
        response.tempToken,
        credential,
        String(response.options.challenge ?? ''),
      );
      setPasskeyFailures(0);
      useUiPreferencesStore.getState().set('lastLoginMethod', 'passkey');
      await applyAuthResponse(data);
    } catch (err: unknown) {
      if ((err as Error)?.name === 'NotAllowedError') {
        registerPasskeyFailure('Passkey authentication was cancelled or timed out.');
      } else {
        registerPasskeyFailure(extractApiError(err, 'Passkey authentication failed.'));
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!passkeySupported || passkeyAttempted || step !== 'passkey') {
      return;
    }
    void handlePasskeyAuth();
  }, [passkeyAttempted, passkeySupported, step]);

  useEffect(() => {
    if (!passkeySupported && step === 'passkey') {
      setStep('credentials');
    }
  }, [passkeySupported, step]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setSuccess('');
    setShowResend(false);
    setLoading(true);
    try {
      const data = await loginApi(email, password);
      useUiPreferencesStore.getState().set('lastLoginMethod', 'credentials');
      await applyAuthResponse(data);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Login failed'));
      if ((err as { response?: { status?: number } })?.response?.status === 403) {
        setShowResend(true);
      }
    } finally {
      setLoading(false);
    }
  };

  const handleEmailSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = await verifyEmailCodeApi(tempToken, emailCode);
      completeLogin(data);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Invalid code'));
    } finally {
      setLoading(false);
    }
  };

  const handleTotpSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = await verifyTotpApi(tempToken, totpCode);
      completeLogin(data);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Invalid code'));
    } finally {
      setLoading(false);
    }
  };

  const handleSmsSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = await verifySmsApi(tempToken, smsCode);
      completeLogin(data);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Invalid code'));
    } finally {
      setLoading(false);
    }
  };

  const handleChooseMethod = async (method: string) => {
    setError('');
    if (method === 'email') {
      try {
        await requestEmailCodeApi(tempToken);
        setStep('email');
      } catch (err: unknown) {
        setError(extractApiError(err, 'Failed to send email code'));
      }
      return;
    }

    if (method === 'totp') {
      setStep('totp');
      return;
    }

    if (method === 'webauthn') {
      setStep('webauthn');
      void handleWebAuthnAuth();
      return;
    }

    setSmsSending(true);
    try {
      await requestSmsCodeApi(tempToken);
      setStep('sms');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to send SMS code'));
    } finally {
      setSmsSending(false);
    }
  };

  const handleBackToCredentials = () => {
    setStep('credentials');
    setEmailCode('');
    setTotpCode('');
    setSmsCode('');
    setTempToken('');
    setMfaMethods([]);
    setError('');
  };

  const handleBackToChoice = () => {
    setStep('mfa-choice');
    setEmailCode('');
    setTotpCode('');
    setSmsCode('');
    setError('');
  };

  const handleUsePasswordFallback = () => {
    setError('');
    setStep('credentials');
    setPasskeyAttempted(true);
    useUiPreferencesStore.getState().set('lastLoginMethod', 'credentials');
  };

  const handleReturnToPasskey = () => {
    setError('');
    setPasskeyAttempted(false);
    setPasskeyFailures(0);
    setStep('passkey');
    useUiPreferencesStore.getState().set('lastLoginMethod', 'passkey');
  };

  const handleMfaSetupSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = await mfaSetupVerifyApi(tempToken, mfaSetupCode);
      completeLogin(data);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Invalid code'));
    } finally {
      setLoading(false);
    }
  };

  const handleResend = async () => {
    try {
      await resendVerificationEmail(email);
      setResendCountdown(60);
      setSuccess('Verification email sent! Check your inbox.');
      setError('');
      setShowResend(false);
    } catch {
      // Server returns 200 for valid email format regardless of account status.
    }
  };

  const canGoBackToChoice = mfaMethods.length > 1;
  const ldapEnabled = Boolean(authProviders?.ldap);

  const renderStepPanel = () => {
    switch (step) {
      case 'passkey':
        return (
          <PasskeyPanel
            authProviders={authProviders}
            authProvidersLoading={authProvidersLoading}
            loading={loading}
            onRetry={handlePasskeyAuth}
            onUsePasswordFallback={handleUsePasswordFallback}
            passkeyFailures={passkeyFailures}
          />
        );
      case 'credentials':
        return (
          <CredentialsPanel
            authProviders={authProviders}
            authProvidersLoading={authProvidersLoading}
            email={email}
            ldapEnabled={ldapEnabled}
            loading={loading}
            onEmailChange={setEmail}
            onPasswordChange={setPassword}
            onReturnToPasskey={handleReturnToPasskey}
            onSubmit={handleSubmit}
            password={password}
            passkeySupported={passkeySupported}
          />
        );
      case 'mfa-choice':
        return (
          <MethodChoicePanel
            loading={loading}
            methods={mfaMethods}
            onBack={handleBackToCredentials}
            onChooseMethod={handleChooseMethod}
            smsSending={smsSending}
          />
        );
      case 'email':
        return (
          <CodeVerificationPanel
            infoMessage="A verification code has been sent to your email address."
            label="Email Code"
            loading={loading}
            onBack={canGoBackToChoice ? handleBackToChoice : handleBackToCredentials}
            onChange={setEmailCode}
            onSubmit={handleEmailSubmit}
            value={emailCode}
          />
        );
      case 'totp':
        return (
          <CodeVerificationPanel
            label="Authenticator Code"
            loading={loading}
            onBack={canGoBackToChoice ? handleBackToChoice : handleBackToCredentials}
            onChange={setTotpCode}
            onSubmit={handleTotpSubmit}
            value={totpCode}
          />
        );
      case 'sms':
        return (
          <CodeVerificationPanel
            infoMessage="A verification code has been sent to your phone."
            label="SMS Code"
            loading={loading}
            onBack={canGoBackToChoice ? handleBackToChoice : handleBackToCredentials}
            onChange={setSmsCode}
            onSubmit={handleSmsSubmit}
            value={smsCode}
          />
        );
      case 'webauthn':
        return (
          <WebAuthnPanel
            loading={loading}
            onBack={canGoBackToChoice ? handleBackToChoice : handleBackToCredentials}
            onRetry={handleWebAuthnAuth}
          />
        );
      case 'mfa-setup':
        return (
          <MfaSetupPanel
            loading={loading}
            onBack={handleBackToCredentials}
            onChange={setMfaSetupCode}
            onSubmit={handleMfaSetupSubmit}
            setupCode={mfaSetupCode}
            setupData={mfaSetupData}
          />
        );
      case 'tenant-select':
        return (
          <TenantSelectionPanel
            loading={loading}
            memberships={tenantMemberships}
            onContinue={handleTenantConfirm}
            onSelect={setSelectedTenantId}
            selectedTenantId={selectedTenantId}
          />
        );
      default:
        return null;
    }
  };

  return (
    <AuthLayout
      cardClassName="max-w-md"
      title="Arsenale"
      description={LOGIN_STEP_SUBTITLES[step]}
      titleClassName="text-4xl font-normal"
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

      {showResend ? (
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
      ) : null}

      {renderStepPanel()}

      {ldapEnabled && step === 'passkey' ? (
        <p className="sr-only">{LDAP_PROVIDER_NAME} directory login is available.</p>
      ) : null}
    </AuthLayout>
  );
}
