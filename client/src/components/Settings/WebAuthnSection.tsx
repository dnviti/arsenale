import { useCallback, useEffect, useState } from 'react';
import { browserSupportsWebAuthn, startRegistration } from '@simplewebauthn/browser';
import { KeyRound, LoaderCircle, Pencil, Trash2 } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
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
import { extractApiError } from '../../utils/apiError';
import {
  getWebAuthnCredentials,
  getWebAuthnRegistrationOptions,
  getWebAuthnStatus,
  registerWebAuthnCredential,
  removeWebAuthnCredential,
  renameWebAuthnCredential,
  type WebAuthnCredentialInfo,
} from '../../api/webauthn.api';
import {
  SettingsButtonRow,
  SettingsPanel,
  SettingsStatusBadge,
} from './settings-ui';

function credentialMetadata(credential: WebAuthnCredentialInfo) {
  const parts = [`Added ${new Date(credential.createdAt).toLocaleDateString()}`];

  if (credential.deviceType) {
    parts.unshift(credential.deviceType);
  }

  if (credential.lastUsedAt) {
    parts.push(`Last used ${new Date(credential.lastUsedAt).toLocaleDateString()}`);
  }

  return parts.join(' · ');
}

export default function WebAuthnSection() {
  const notify = useNotificationStore((state) => state.notify);
  const [enabled, setEnabled] = useState(false);
  const [credentials, setCredentials] = useState<WebAuthnCredentialInfo[]>([]);
  const [statusLoading, setStatusLoading] = useState(true);
  const [browserSupported, setBrowserSupported] = useState(true);
  const [registering, setRegistering] = useState(false);
  const [saving, setSaving] = useState(false);
  const [pendingCredential, setPendingCredential] = useState<unknown>(null);
  const [pendingChallenge, setPendingChallenge] = useState<string | null>(null);
  const [friendlyName, setFriendlyName] = useState('');
  const [renameTarget, setRenameTarget] = useState<WebAuthnCredentialInfo | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<WebAuthnCredentialInfo | null>(null);
  const [error, setError] = useState('');

  const loadData = useCallback(async () => {
    try {
      const [status, nextCredentials] = await Promise.all([
        getWebAuthnStatus(),
        getWebAuthnCredentials(),
      ]);
      setEnabled(status.enabled);
      setCredentials(nextCredentials);
    } catch {
      setError('Failed to load passkeys and security keys.');
    } finally {
      setStatusLoading(false);
    }
  }, []);

  useEffect(() => {
    setBrowserSupported(browserSupportsWebAuthn());
    void loadData();
  }, [loadData]);

  const resetRegistrationState = () => {
    setPendingCredential(null);
    setPendingChallenge(null);
    setFriendlyName('');
  };

  const handleStartRegistration = async () => {
    setError('');
    setRegistering(true);

    try {
      const options = await getWebAuthnRegistrationOptions();
      const credential = await startRegistration({ optionsJSON: options });
      setPendingCredential(credential);
      setPendingChallenge(options.challenge);
      setFriendlyName('');
    } catch (err: unknown) {
      if ((err as Error)?.name === 'NotAllowedError') {
        setError('Registration was cancelled or timed out.');
      } else {
        setError(extractApiError(err, 'Failed to start registration.'));
      }
    } finally {
      setRegistering(false);
    }
  };

  const handleCompleteRegistration = async () => {
    if (!pendingCredential) return;

    setError('');
    setSaving(true);

    try {
      await registerWebAuthnCredential(
        pendingCredential,
        friendlyName.trim() || undefined,
        pendingChallenge || undefined,
      );
      notify('Security key registered successfully.', 'success');
      resetRegistrationState();
      await loadData();
    } catch (err: unknown) {
      setError(extractApiError(err, 'Registration verification failed.'));
    } finally {
      setSaving(false);
    }
  };

  const handleRename = async () => {
    if (!renameTarget || !friendlyName.trim()) return;

    setError('');
    setSaving(true);

    try {
      await renameWebAuthnCredential(renameTarget.id, friendlyName.trim());
      notify('Security key renamed.', 'success');
      setRenameTarget(null);
      setFriendlyName('');
      await loadData();
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to rename credential.'));
    } finally {
      setSaving(false);
    }
  };

  const handleRemove = async () => {
    if (!deleteTarget) return;

    setError('');
    setSaving(true);

    try {
      await removeWebAuthnCredential(deleteTarget.id);
      notify('Security key removed.', 'success');
      setDeleteTarget(null);
      await loadData();
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to remove credential.'));
    } finally {
      setSaving(false);
    }
  };

  if (statusLoading) {
    return (
      <SettingsPanel
        title="Passkeys & Security Keys"
        description="Use passkeys or hardware-backed keys for phishing-resistant sign-in."
      >
        <p className="text-sm text-muted-foreground">Loading passkey settings...</p>
      </SettingsPanel>
    );
  }

  return (
    <>
      <SettingsPanel
        title="Passkeys & Security Keys"
        description="Use passkeys or hardware security keys for passwordless sign-in and stronger verification."
        heading={
          <div className="flex flex-wrap items-center gap-2">
            <SettingsStatusBadge tone={enabled ? 'success' : 'neutral'}>
              {enabled ? 'Enabled' : 'Disabled'}
            </SettingsStatusBadge>
            {credentials.length > 0 && (
              <Badge variant="outline">
                {credentials.length} key{credentials.length === 1 ? '' : 's'}
              </Badge>
            )}
          </div>
        }
      >
        <div className="space-y-4">
          {!browserSupported && (
            <Alert variant="warning">
              <AlertDescription>
                Your browser does not support WebAuthn. Use a recent version of Chrome,
                Firefox, Safari, or Edge.
              </AlertDescription>
            </Alert>
          )}

          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {credentials.length > 0 ? (
            <div className="space-y-3">
              {credentials.map((credential) => (
                <div
                  key={credential.id}
                  className="flex flex-wrap items-start justify-between gap-4 rounded-xl border border-border/70 bg-background/60 p-4"
                >
                  <div className="min-w-0 space-y-2">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="text-sm font-medium text-foreground">
                        {credential.friendlyName}
                      </p>
                      {credential.backedUp && <Badge variant="outline">Synced</Badge>}
                    </div>
                    <p className="text-sm text-muted-foreground">
                      {credentialMetadata(credential)}
                    </p>
                  </div>

                  <SettingsButtonRow>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      disabled={saving}
                      onClick={() => {
                        setRenameTarget(credential);
                        setFriendlyName(credential.friendlyName);
                      }}
                    >
                      <Pencil className="size-4" />
                      Rename
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      disabled={saving}
                      onClick={() => setDeleteTarget(credential)}
                    >
                      <Trash2 className="size-4" />
                      Remove
                    </Button>
                  </SettingsButtonRow>
                </div>
              ))}
            </div>
          ) : (
            <div className="rounded-xl border border-dashed border-border/70 bg-background/40 p-4">
              <p className="text-sm text-muted-foreground">
                No passkeys or hardware security keys are registered yet.
              </p>
            </div>
          )}

          {browserSupported && (
            <Button
              type="button"
              disabled={registering || saving}
              onClick={() => void handleStartRegistration()}
            >
              {registering ? (
                <LoaderCircle className="size-4 animate-spin" />
              ) : (
                <KeyRound className="size-4" />
              )}
              {registering ? 'Waiting for device...' : 'Add Security Key'}
            </Button>
          )}
        </div>
      </SettingsPanel>

      <Dialog
        open={Boolean(pendingCredential)}
        onOpenChange={(open) => {
          if (!open && !saving) {
            resetRegistrationState();
          }
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Name Your Security Key</DialogTitle>
            <DialogDescription>
              Give this device a recognisable name so you can manage it later.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <Label htmlFor="webauthn-friendly-name">Key name</Label>
            <Input
              id="webauthn-friendly-name"
              value={friendlyName}
              onChange={(event) => setFriendlyName(event.target.value)}
              placeholder="YubiKey 5, MacBook Touch ID, iPhone Passkey"
              maxLength={64}
              autoFocus
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  void handleCompleteRegistration();
                }
              }}
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={saving}
              onClick={resetRegistrationState}
            >
              Cancel
            </Button>
            <Button
              type="button"
              disabled={saving}
              onClick={() => void handleCompleteRegistration()}
            >
              {saving ? 'Saving...' : 'Save'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={Boolean(renameTarget)}
        onOpenChange={(open) => {
          if (!open && !saving) {
            setRenameTarget(null);
            setFriendlyName('');
          }
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Rename Security Key</DialogTitle>
            <DialogDescription>
              Update the label shown for this credential in your account.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <Label htmlFor="webauthn-rename">Key name</Label>
            <Input
              id="webauthn-rename"
              value={friendlyName}
              onChange={(event) => setFriendlyName(event.target.value)}
              maxLength={64}
              autoFocus
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  void handleRename();
                }
              }}
            />
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={saving}
              onClick={() => {
                setRenameTarget(null);
                setFriendlyName('');
              }}
            >
              Cancel
            </Button>
            <Button
              type="button"
              disabled={saving || !friendlyName.trim()}
              onClick={() => void handleRename()}
            >
              {saving ? 'Saving...' : 'Save'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={Boolean(deleteTarget)}
        onOpenChange={(open) => {
          if (!open && !saving) {
            setDeleteTarget(null);
          }
        }}
      >
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>Remove Security Key?</DialogTitle>
            <DialogDescription>
              This credential will no longer be available for sign-in.
              {credentials.length === 1
                ? ' Removing your last key disables WebAuthn until you add another one.'
                : ''}
            </DialogDescription>
          </DialogHeader>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={saving}
              onClick={() => setDeleteTarget(null)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={saving}
              onClick={() => void handleRemove()}
            >
              {saving ? 'Removing...' : 'Remove'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
