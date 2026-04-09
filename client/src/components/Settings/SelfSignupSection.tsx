import { useState, useEffect } from 'react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { getAppConfig, setSelfSignup } from '../../api/admin.api';
import { extractApiError } from '../../utils/apiError';
import {
  SettingsLoadingState,
  SettingsPanel,
  SettingsSwitchRow,
} from './settings-ui';

export default function SelfSignupSection() {
  const [enabled, setEnabled] = useState(true);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [envLocked, setEnvLocked] = useState(false);

  useEffect(() => {
    getAppConfig()
      .then((cfg) => { setEnabled(cfg.selfSignupEnabled); setEnvLocked(cfg.selfSignupEnvLocked); setLoading(false); })
      .catch(() => setLoading(false));
  }, []);

  const handleToggle = async (nextEnabled: boolean) => {
    setSaving(true);
    setError('');
    try {
      const updated = await setSelfSignup(nextEnabled);
      setEnabled(updated.selfSignupEnabled);
      setEnvLocked(updated.selfSignupEnvLocked);
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to update setting'));
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <SettingsPanel
        title="Self Signup"
        description="Public registration policy and onboarding control."
      >
        <SettingsLoadingState message="Loading self-signup policy..." />
      </SettingsPanel>
    );
  }

  return (
    <SettingsPanel
      title="Self Signup"
      description="Public registration policy and onboarding control."
      contentClassName="space-y-4"
    >
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <SettingsSwitchRow
        title="Allow new users to register themselves"
        description="When disabled, only organization admins can create user accounts."
        checked={enabled}
        disabled={saving || envLocked}
        onCheckedChange={handleToggle}
      />

      {envLocked && (
        <Alert variant="info">
          <AlertDescription>
            Self-registration is locked at the environment level. Update
            {' '}
            <code className="rounded bg-background/80 px-1.5 py-0.5 text-xs text-foreground">SELF_SIGNUP_ENABLED</code>
            {' '}
            and restart the server to change it.
          </AlertDescription>
        </Alert>
      )}
    </SettingsPanel>
  );
}
