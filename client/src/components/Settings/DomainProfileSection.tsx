import { useEffect, useState } from 'react';
import { Building2 } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useNotificationStore } from '../../store/notificationStore';
import { useVaultStore } from '../../store/vaultStore';
import { extractApiError } from '../../utils/apiError';
import {
  clearDomainProfile,
  getDomainProfile,
  updateDomainProfile,
  type DomainProfile,
} from '../../api/user.api';
import {
  SettingsButtonRow,
  SettingsPanel,
  SettingsStatusBadge,
} from './settings-ui';

export default function DomainProfileSection() {
  const notify = useNotificationStore((state) => state.notify);
  const vaultUnlocked = useVaultStore((state) => state.unlocked);
  const [profile, setProfile] = useState<DomainProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState(false);
  const [confirmClearOpen, setConfirmClearOpen] = useState(false);
  const [error, setError] = useState('');
  const [domainName, setDomainName] = useState('');
  const [domainUsername, setDomainUsername] = useState('');
  const [domainPassword, setDomainPassword] = useState('');
  const [clearPassword, setClearPassword] = useState(false);

  useEffect(() => {
    getDomainProfile()
      .then((data) => {
        setProfile(data);
        setDomainName(data.domainName ?? '');
        setDomainUsername(data.domainUsername ?? '');
      })
      .catch(() => setError('Failed to load domain profile'))
      .finally(() => setLoading(false));
  }, []);

  const hasProfile = Boolean(profile?.domainName || profile?.domainUsername);

  const resetFormFromProfile = () => {
    setDomainName(profile?.domainName ?? '');
    setDomainUsername(profile?.domainUsername ?? '');
    setDomainPassword('');
    setClearPassword(false);
    setError('');
  };

  const handleSave = async () => {
    setError('');
    setSaving(true);

    try {
      const payload: Record<string, string | null | undefined> = {};

      if (domainName !== (profile?.domainName ?? '')) {
        payload.domainName = domainName;
      }
      if (domainUsername !== (profile?.domainUsername ?? '')) {
        payload.domainUsername = domainUsername;
      }
      if (domainPassword) {
        payload.domainPassword = domainPassword;
      } else if (clearPassword) {
        payload.domainPassword = null;
      }

      const result = await updateDomainProfile(payload);
      setProfile(result);
      setDomainPassword('');
      setClearPassword(false);
      setEditing(false);
      notify('Domain profile updated', 'success');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to update domain profile'));
    } finally {
      setSaving(false);
    }
  };

  const handleClear = async () => {
    setConfirmClearOpen(false);
    setError('');
    setSaving(true);

    try {
      await clearDomainProfile();
      const emptyProfile: DomainProfile = {
        domainName: null,
        domainUsername: null,
        hasDomainPassword: false,
      };
      setProfile(emptyProfile);
      setDomainName('');
      setDomainUsername('');
      setDomainPassword('');
      setClearPassword(false);
      setEditing(false);
      notify('Domain profile cleared', 'success');
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to clear domain profile'));
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <SettingsPanel
        title="Domain Identity"
        description="Configure your Windows or Active Directory identity for connection defaults."
      >
        <p className="text-sm text-muted-foreground">Loading domain profile...</p>
      </SettingsPanel>
    );
  }

  return (
    <>
      <SettingsPanel
        title="Domain Identity"
        description="Configure your Windows or Active Directory credentials for RDP and SSH sign-in defaults."
        heading={
          <SettingsStatusBadge tone={hasProfile ? 'success' : 'neutral'}>
            {hasProfile ? 'Configured' : 'Not configured'}
          </SettingsStatusBadge>
        }
      >
        <div className="space-y-4">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {!editing ? (
            <div className="space-y-4">
              {hasProfile ? (
                <div className="space-y-3 rounded-xl border border-border/70 bg-background/60 p-4">
                  <div className="flex items-center gap-2">
                    <Building2 className="size-4 text-primary" />
                    <p className="text-sm font-medium text-foreground">
                      Active domain identity
                    </p>
                  </div>

                  <div className="grid gap-3 sm:grid-cols-2">
                    <div className="space-y-1">
                      <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                        Domain
                      </p>
                      <p className="text-sm text-foreground">
                        {profile?.domainName ?? '—'}
                      </p>
                    </div>
                    <div className="space-y-1">
                      <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                        Username
                      </p>
                      <p className="text-sm text-foreground">
                        {profile?.domainUsername ?? '—'}
                      </p>
                    </div>
                  </div>

                  <div className="space-y-1">
                    <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">
                      Password
                    </p>
                    <SettingsStatusBadge
                      tone={profile?.hasDomainPassword ? 'success' : 'neutral'}
                    >
                      {profile?.hasDomainPassword ? 'Stored' : 'Not set'}
                    </SettingsStatusBadge>
                  </div>
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No domain identity is configured yet.
                </p>
              )}

              <SettingsButtonRow>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    resetFormFromProfile();
                    setEditing(true);
                  }}
                >
                  {hasProfile ? 'Edit' : 'Configure'}
                </Button>
                {hasProfile && (
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setConfirmClearOpen(true)}
                  >
                    Clear
                  </Button>
                )}
              </SettingsButtonRow>
            </div>
          ) : (
            <div className="space-y-4 rounded-xl border border-border/70 bg-background/60 p-4">
              <div className="space-y-2">
                <Label htmlFor="domain-name">Domain name</Label>
                <Input
                  id="domain-name"
                  value={domainName}
                  onChange={(event) => setDomainName(event.target.value)}
                  placeholder="CONTOSO or contoso.com"
                />
                <p className="text-xs leading-5 text-muted-foreground">
                  Use either the NetBIOS name or the fully-qualified domain name.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="domain-username">Domain username</Label>
                <Input
                  id="domain-username"
                  value={domainUsername}
                  onChange={(event) => setDomainUsername(event.target.value)}
                  placeholder="john.doe"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="domain-password">
                  {profile?.hasDomainPassword && !clearPassword
                    ? 'New password'
                    : 'Domain password'}
                </Label>
                <Input
                  id="domain-password"
                  type="password"
                  value={domainPassword}
                  onChange={(event) => {
                    setDomainPassword(event.target.value);
                    if (event.target.value) {
                      setClearPassword(false);
                    }
                  }}
                  disabled={!vaultUnlocked && !profile?.hasDomainPassword}
                  placeholder={
                    profile?.hasDomainPassword && !clearPassword
                      ? 'Leave blank to keep the existing password'
                      : 'Optional'
                  }
                />
                <p className="text-xs leading-5 text-muted-foreground">
                  {!vaultUnlocked
                    ? 'Unlock your vault to set or change the stored password.'
                    : 'The password is encrypted with your vault key.'}
                </p>
              </div>

              {profile?.hasDomainPassword && !domainPassword && (
                <Button
                  type="button"
                  variant="ghost"
                  className="justify-start px-0"
                  onClick={() => setClearPassword((current) => !current)}
                >
                  {clearPassword ? 'Keep existing password' : 'Remove saved password'}
                </Button>
              )}

              <SettingsButtonRow>
                <Button
                  type="button"
                  disabled={saving}
                  onClick={() => void handleSave()}
                >
                  {saving ? 'Saving...' : 'Save'}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  disabled={saving}
                  onClick={() => {
                    resetFormFromProfile();
                    setEditing(false);
                  }}
                >
                  Cancel
                </Button>
              </SettingsButtonRow>
            </div>
          )}
        </div>
      </SettingsPanel>

      <Dialog open={confirmClearOpen} onOpenChange={setConfirmClearOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Clear Domain Identity</DialogTitle>
            <DialogDescription>
              This removes the saved domain name, username, and stored password.
            </DialogDescription>
          </DialogHeader>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={saving}
              onClick={() => setConfirmClearOpen(false)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={saving}
              onClick={() => void handleClear()}
            >
              {saving ? 'Clearing...' : 'Clear'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
