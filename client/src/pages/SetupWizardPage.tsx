import { useEffect, useState, type ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  CheckCircle2,
  CircleX,
  Copy,
  Database,
  Download,
  Eye,
  EyeOff,
  LoaderCircle,
} from 'lucide-react';
import AuthLayout from '@/components/auth/AuthLayout';
import PasswordStrengthMeter from '@/components/common/PasswordStrengthMeter';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { cn } from '@/lib/utils';
import {
  completeSetup,
  getDbStatus,
  getSetupStatus,
  type DbStatusResponse,
  type SetupCompleteData,
} from '../api/setup.api';
import { useAuthStore } from '../store/authStore';
import { useVaultStore } from '../store/vaultStore';
import { extractApiError } from '../utils/apiError';
import { downloadTextFile } from '../utils/downloadFile';
import { useCopyToClipboard } from '../hooks/useCopyToClipboard';

const STEPS = ['Welcome', 'Database', 'Administrator', 'Organization', 'Settings', 'Complete'];

function SetupStepIndicator({ activeStep }: { activeStep: number }) {
  return (
    <ol className="grid gap-3 md:grid-cols-6">
      {STEPS.map((label, index) => {
        const isActive = index === activeStep;
        const isComplete = index < activeStep;

        return (
          <li
            key={label}
            className={cn(
              'rounded-xl border px-3 py-3 transition-colors',
              isActive
                ? 'border-primary/40 bg-primary/10'
                : isComplete
                  ? 'border-primary/20 bg-primary/5'
                  : 'border-border bg-muted/20',
            )}
          >
            <div className="flex items-center gap-3">
              <div
                className={cn(
                  'flex size-8 shrink-0 items-center justify-center rounded-full border text-xs font-semibold',
                  isActive || isComplete
                    ? 'border-primary bg-primary text-primary-foreground'
                    : 'border-border text-muted-foreground',
                )}
              >
                {index + 1}
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-foreground">{label}</p>
              </div>
            </div>
          </li>
        );
      })}
    </ol>
  );
}

function ReadOnlyField({ label, value }: { label: string; value: string }) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <Input readOnly value={value} />
    </div>
  );
}

function SettingSwitchCard({
  checked,
  children,
  description,
  label,
  onCheckedChange,
}: {
  checked: boolean;
  children?: ReactNode;
  description: string;
  label: string;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <div className="space-y-4 rounded-xl border bg-card p-4">
      <div className="flex items-start justify-between gap-4">
        <div className="space-y-1">
          <p className="text-sm font-medium text-foreground">{label}</p>
          <p className="text-sm leading-6 text-muted-foreground">{description}</p>
        </div>
        <Switch checked={checked} onCheckedChange={onCheckedChange} />
      </div>
      {checked ? children : null}
    </div>
  );
}

function CopyValueButton({ label, value }: { label: string; value: string }) {
  const { copied, copy } = useCopyToClipboard();

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      aria-label={`Copy ${label}`}
      title={copied ? 'Copied!' : `Copy ${label}`}
      onClick={() => void copy(value)}
    >
      <Copy className="size-4" />
    </Button>
  );
}

export default function SetupWizardPage() {
  const navigate = useNavigate();
  const setAuth = useAuthStore((state) => state.setAuth);
  const setVaultUnlocked = useVaultStore((state) => state.setUnlocked);

  const [statusChecking, setStatusChecking] = useState(true);
  const [activeStep, setActiveStep] = useState(0);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [dbStatus, setDbStatus] = useState<DbStatusResponse | null>(null);
  const [dbLoading, setDbLoading] = useState(false);
  const [adminEmail, setAdminEmail] = useState('');
  const [adminUsername, setAdminUsername] = useState('');
  const [adminPassword, setAdminPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [tenantName, setTenantName] = useState('');
  const [selfSignupEnabled, setSelfSignupEnabled] = useState(false);
  const [configureSmtp, setConfigureSmtp] = useState(false);
  const [smtpHost, setSmtpHost] = useState('');
  const [smtpPort, setSmtpPort] = useState('587');
  const [smtpUser, setSmtpUser] = useState('');
  const [smtpPass, setSmtpPass] = useState('');
  const [smtpFrom, setSmtpFrom] = useState('');
  const [smtpSecure, setSmtpSecure] = useState(false);
  const [recoveryKey, setRecoveryKey] = useState('');
  const [systemSecrets, setSystemSecrets] = useState<Array<{
    description: string;
    name: string;
    value: string;
  }>>([]);

  useEffect(() => {
    getSetupStatus()
      .then((status) => {
        if (!status.required) {
          navigate('/', { replace: true });
          return;
        }
        setStatusChecking(false);
      })
      .catch(() => {
        setStatusChecking(false);
      });
  }, [navigate]);

  const testDbConnection = async () => {
    setDbLoading(true);
    try {
      const status = await getDbStatus();
      setDbStatus(status);
    } catch {
      setDbStatus({ host: '', port: 0, database: '', connected: false, version: null });
    } finally {
      setDbLoading(false);
    }
  };

  const canProceed = () => {
    switch (activeStep) {
      case 0:
        return true;
      case 1:
        return dbStatus?.connected === true;
      case 2:
        return adminEmail.length > 0
          && adminPassword.length >= 10
          && adminPassword === confirmPassword;
      case 3:
        return tenantName.length > 0;
      case 4:
        return true;
      default:
        return false;
    }
  };

  const handleNext = async () => {
    setError('');

    if (activeStep === 4) {
      setLoading(true);
      try {
        const body: SetupCompleteData = {
          admin: {
            email: adminEmail,
            password: adminPassword,
            ...(adminUsername ? { username: adminUsername } : {}),
          },
          tenant: { name: tenantName },
          settings: {
            selfSignupEnabled,
            ...(configureSmtp && smtpHost
              ? {
                  smtp: {
                    host: smtpHost,
                    port: parseInt(smtpPort, 10) || 587,
                    ...(smtpUser ? { user: smtpUser } : {}),
                    ...(smtpPass ? { pass: smtpPass } : {}),
                    ...(smtpFrom ? { from: smtpFrom } : {}),
                    secure: smtpSecure,
                  },
                }
              : {}),
          },
        };

        const result = await completeSetup(body);

        setRecoveryKey(result.recoveryKey);
        setSystemSecrets(result.systemSecrets || []);
        setAuth(result.accessToken, result.csrfToken ?? '', {
          id: result.user.id,
          email: result.user.email,
          username: result.user.username,
          avatarData: null,
          tenantId: result.tenant.id,
          tenantRole: 'OWNER',
          vaultSetupComplete: true,
        });
        setVaultUnlocked(true);
        setActiveStep(5);
      } catch (err: unknown) {
        setError(extractApiError(err, 'Setup failed. Please try again.'));
      } finally {
        setLoading(false);
      }
      return;
    }

    setActiveStep((previous) => previous + 1);
  };

  const handleDownloadRecoveryKey = () => {
    const content = [
      'Arsenale Recovery Key',
      '='.repeat(40),
      '',
      recoveryKey,
      '',
      'Store this key in a safe place. It is the only way to recover your vault if you forget your password.',
    ].join('\n');
    downloadTextFile(content, 'arsenale-recovery-key.txt');
  };

  const handleDownloadSecrets = () => {
    const content = systemSecrets.map((secret) => `${secret.name}=${secret.value}`).join('\n');
    downloadTextFile(content, 'arsenale-system-secrets.env');
  };

  const passwordsMismatch = confirmPassword.length > 0 && adminPassword !== confirmPassword;

  let stepContent: ReactNode = null;

  if (activeStep === 0) {
    stepContent = (
      <div className="space-y-4">
        <div className="space-y-2">
          <h2 className="text-xl font-semibold text-foreground">Welcome to Arsenale</h2>
          <p className="text-sm leading-6 text-muted-foreground">
            Arsenale is a secure remote access and privileged access management platform.
            This wizard will guide you through the initial setup to get your platform ready.
          </p>
        </div>
        <div className="rounded-xl border bg-card p-4">
          <p className="mb-3 text-sm font-medium text-foreground">Here&apos;s what we&apos;ll do:</p>
          <ul className="space-y-2 text-sm leading-6 text-muted-foreground">
            <li><strong className="text-foreground">Verify database connection</strong> so the control plane can store state.</li>
            <li><strong className="text-foreground">Create an administrator account</strong> with full platform control.</li>
            <li><strong className="text-foreground">Create an organization</strong> for your users, teams, and policies.</li>
            <li><strong className="text-foreground">Configure basic settings</strong> like sign-up and email delivery.</li>
          </ul>
        </div>
        <p className="text-sm leading-6 text-muted-foreground">
          This wizard runs only once. After completion, you&apos;ll be logged in and ready to go.
        </p>
      </div>
    );
  } else if (activeStep === 1) {
    stepContent = (
      <div className="space-y-4">
        <div className="space-y-2">
          <h2 className="flex items-center gap-2 text-xl font-semibold text-foreground">
            <Database className="size-5" />
            Database Connection
          </h2>
          <p className="text-sm leading-6 text-muted-foreground">
            Verify the PostgreSQL database connection. These values come from `DATABASE_URL`.
            To change them, update your environment and restart the server.
          </p>
        </div>

        {dbStatus ? (
          <div className="space-y-4 rounded-xl border bg-card p-4">
            <div className="grid gap-4 md:grid-cols-3">
              <ReadOnlyField label="Host" value={dbStatus.host || '(not set)'} />
              <ReadOnlyField label="Port" value={String(dbStatus.port)} />
              <ReadOnlyField label="Database" value={dbStatus.database || '(not set)'} />
            </div>
            <div className="flex items-center gap-2 text-sm">
              {dbStatus.connected ? (
                <>
                  <CheckCircle2 className="size-4 text-primary" />
                  <span className="font-medium text-primary">Connected</span>
                </>
              ) : (
                <>
                  <CircleX className="size-4 text-destructive" />
                  <span className="font-medium text-destructive">Connection failed</span>
                </>
              )}
            </div>
            {dbStatus.version ? (
              <p className="text-xs text-muted-foreground">{dbStatus.version}</p>
            ) : null}
          </div>
        ) : null}

        <Button type="button" variant="outline" disabled={dbLoading} onClick={() => void testDbConnection()}>
          {dbLoading ? <LoaderCircle className="size-4 animate-spin" /> : <Database className="size-4" />}
          {dbStatus ? 'Retest Connection' : 'Test Connection'}
        </Button>
      </div>
    );
  } else if (activeStep === 2) {
    stepContent = (
      <div className="space-y-4">
        <div className="space-y-2">
          <h2 className="text-xl font-semibold text-foreground">Create Administrator Account</h2>
          <p className="text-sm leading-6 text-muted-foreground">
            This will be the first user with full platform control. All connection credentials
            will be encrypted with a key derived from this password.
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="setup-admin-email">Email</Label>
          <Input
            id="setup-admin-email"
            autoFocus
            required
            type="email"
            value={adminEmail}
            onChange={(event) => setAdminEmail(event.target.value)}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="setup-admin-username">Username</Label>
          <Input
            id="setup-admin-username"
            placeholder="Optional display name"
            value={adminUsername}
            onChange={(event) => setAdminUsername(event.target.value)}
          />
          <p className="text-xs text-muted-foreground">A display name for your profile.</p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="setup-admin-password">Password</Label>
          <div className="relative">
            <Input
              id="setup-admin-password"
              required
              type={showPassword ? 'text' : 'password'}
              value={adminPassword}
              onChange={(event) => setAdminPassword(event.target.value)}
            />
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="absolute right-1 top-1 size-8"
              aria-label={showPassword ? 'Hide password' : 'Show password'}
              onClick={() => setShowPassword((previous) => !previous)}
            >
              {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
            </Button>
          </div>
          <p className="text-xs text-muted-foreground">Minimum 10 characters.</p>
        </div>

        <PasswordStrengthMeter password={adminPassword} />

        <div className="space-y-2">
          <Label htmlFor="setup-admin-confirm-password">Confirm Password</Label>
          <Input
            id="setup-admin-confirm-password"
            required
            type="password"
            value={confirmPassword}
            onChange={(event) => setConfirmPassword(event.target.value)}
          />
          {passwordsMismatch ? (
            <p className="text-xs text-destructive">Passwords do not match</p>
          ) : null}
        </div>
      </div>
    );
  } else if (activeStep === 3) {
    stepContent = (
      <div className="space-y-4">
        <div className="space-y-2">
          <h2 className="text-xl font-semibold text-foreground">Create Your Organization</h2>
          <p className="text-sm leading-6 text-muted-foreground">
            An organization groups your users, teams, connections, and security policies together.
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="setup-tenant-name">Organization Name</Label>
          <Input
            id="setup-tenant-name"
            autoFocus
            placeholder="Acme Corp"
            required
            value={tenantName}
            onChange={(event) => setTenantName(event.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            For example: Acme Corp, IT Department, Home Lab.
          </p>
        </div>
      </div>
    );
  } else if (activeStep === 4) {
    stepContent = (
      <div className="space-y-4">
        <div className="space-y-2">
          <h2 className="text-xl font-semibold text-foreground">Platform Settings</h2>
          <p className="text-sm leading-6 text-muted-foreground">
            Configure how your platform handles new users and notifications.
            You can change all of these later in Settings.
          </p>
        </div>

        <SettingSwitchCard
          checked={selfSignupEnabled}
          label="Allow self-registration"
          description={
            selfSignupEnabled
              ? 'Anyone can create an account on the login page. You can assign them later.'
              : 'Only administrators can create accounts. This is recommended for most deployments.'
          }
          onCheckedChange={setSelfSignupEnabled}
        />

        <SettingSwitchCard
          checked={configureSmtp}
          label="Configure email notifications"
          description={
            configureSmtp
              ? 'Enter your SMTP server details to enable email verification and notifications.'
              : 'Skip this for now. You can configure it later in Settings.'
          }
          onCheckedChange={setConfigureSmtp}
        >
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="setup-smtp-host">SMTP Host</Label>
              <Input
                id="setup-smtp-host"
                placeholder="smtp.example.com"
                value={smtpHost}
                onChange={(event) => setSmtpHost(event.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="setup-smtp-port">Port</Label>
              <Input
                id="setup-smtp-port"
                type="number"
                value={smtpPort}
                onChange={(event) => setSmtpPort(event.target.value)}
              />
            </div>

            <div className="flex items-center justify-between rounded-xl border bg-background/60 px-4 py-3">
              <div className="space-y-1">
                <p className="text-sm font-medium text-foreground">Use TLS</p>
                <p className="text-xs text-muted-foreground">Enable secure SMTP transport.</p>
              </div>
              <Switch checked={smtpSecure} onCheckedChange={setSmtpSecure} />
            </div>

            <div className="space-y-2">
              <Label htmlFor="setup-smtp-user">Username</Label>
              <Input
                id="setup-smtp-user"
                value={smtpUser}
                onChange={(event) => setSmtpUser(event.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="setup-smtp-pass">Password</Label>
              <Input
                id="setup-smtp-pass"
                type="password"
                value={smtpPass}
                onChange={(event) => setSmtpPass(event.target.value)}
              />
            </div>

            <div className="space-y-2 md:col-span-2">
              <Label htmlFor="setup-smtp-from">From Address</Label>
              <Input
                id="setup-smtp-from"
                placeholder="noreply@example.com"
                type="email"
                value={smtpFrom}
                onChange={(event) => setSmtpFrom(event.target.value)}
              />
            </div>
          </div>
        </SettingSwitchCard>
      </div>
    );
  } else if (activeStep === 5) {
    stepContent = (
      <div className="space-y-4">
        <Alert variant="success">
          <AlertTitle>Setup complete</AlertTitle>
          <AlertDescription className="text-foreground">
            Your Arsenale platform is ready. You can enter the app as soon as you save the recovery material below.
          </AlertDescription>
        </Alert>

        <div className="rounded-xl border bg-card p-4">
          <ul className="space-y-2 text-sm text-muted-foreground">
            <li>Administrator account: <strong className="text-foreground">{adminEmail}</strong></li>
            <li>Organization: <strong className="text-foreground">{tenantName}</strong></li>
            <li>Self-registration: <strong className="text-foreground">{selfSignupEnabled ? 'enabled' : 'disabled'}</strong></li>
            {configureSmtp && smtpHost ? (
              <li>Email delivery: <strong className="text-foreground">{smtpHost}:{smtpPort}</strong></li>
            ) : null}
          </ul>
        </div>

        <Alert variant="warning">
          <AlertTitle>Save your recovery key</AlertTitle>
          <AlertDescription className="text-foreground">
            This key is the only way to recover your encrypted vault if you forget your password.
            It will not be shown again.
          </AlertDescription>
        </Alert>

        <div className="space-y-3 rounded-xl border bg-muted/30 p-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <p className="min-w-0 flex-1 break-all font-mono text-sm leading-6 text-foreground">
              {recoveryKey}
            </p>
            <div className="flex shrink-0 items-center gap-1">
              <CopyValueButton label="recovery key" value={recoveryKey} />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label="Download recovery key as file"
                title="Download recovery key"
                onClick={handleDownloadRecoveryKey}
              >
                <Download className="size-4" />
              </Button>
            </div>
          </div>
        </div>

        {systemSecrets.length > 0 ? (
          <div className="space-y-4">
            <Alert variant="info">
              <AlertTitle>System secrets</AlertTitle>
              <AlertDescription className="text-foreground">
                These secrets are auto-generated, stored encrypted, and managed automatically.
                Save a backup now because they will not be shown again.
              </AlertDescription>
            </Alert>

            <div className="space-y-3">
              {systemSecrets.map((secret) => (
                <div key={secret.name} className="space-y-2 rounded-xl border bg-card p-4">
                  <div className="space-y-1">
                    <p className="text-sm font-medium text-foreground">{secret.name}</p>
                    <p className="text-xs leading-5 text-muted-foreground">{secret.description}</p>
                  </div>
                  <div className="flex items-start gap-2 rounded-lg border bg-background/80 p-3">
                    <p className="min-w-0 flex-1 break-all font-mono text-xs leading-5 text-foreground">
                      {secret.value}
                    </p>
                    <CopyValueButton label={secret.name} value={secret.value} />
                  </div>
                </div>
              ))}
            </div>

            <Button type="button" variant="outline" onClick={handleDownloadSecrets}>
              <Download className="size-4" />
              Download Secrets
            </Button>
          </div>
        ) : null}
      </div>
    );
  }

  if (statusChecking) {
    return (
      <AuthLayout
        cardClassName="max-w-5xl"
        title="Arsenale Setup"
        description="Checking whether initial setup is required."
      >
        <div className="flex justify-center py-8">
          <LoaderCircle className="size-6 animate-spin text-primary" />
        </div>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout
      cardClassName="max-w-5xl"
      title="Arsenale Setup"
      description="Initial platform configuration wizard."
    >
      <SetupStepIndicator activeStep={activeStep} />

      {error ? (
        <Alert variant="destructive">
          <AlertDescription className="text-foreground">{error}</AlertDescription>
        </Alert>
      ) : null}

      {stepContent}

      <div className="flex flex-col gap-3 border-t pt-4 sm:flex-row sm:items-center sm:justify-between">
        {activeStep > 0 && activeStep < 5 ? (
          <Button type="button" variant="ghost" disabled={loading} onClick={() => setActiveStep((previous) => previous - 1)}>
            Back
          </Button>
        ) : (
          <div />
        )}

        {activeStep < 5 ? (
          <Button type="button" disabled={!canProceed() || loading} onClick={() => void handleNext()}>
            {loading ? <LoaderCircle className="size-4 animate-spin" /> : null}
            {activeStep === 4 ? 'Complete Setup' : 'Next'}
          </Button>
        ) : (
          <Button type="button" onClick={() => navigate('/', { replace: true })}>
            Get Started
          </Button>
        )}
      </div>
    </AuthLayout>
  );
}
