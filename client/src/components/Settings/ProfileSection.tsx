import { useEffect, useRef, useState } from 'react';
import { Mail, PencilLine, Upload } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import {
  confirmEmailChange,
  getProfile,
  initiateEmailChange,
  type EmailChangeInitResult,
  type VerificationMethod,
  updateProfile,
  uploadAvatar,
} from '../../api/user.api';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import { useAuthStore } from '../../store/authStore';
import { useNotificationStore } from '../../store/notificationStore';
import IdentityVerification from '../common/IdentityVerification';
import { SettingsButtonRow, SettingsPanel } from './settings-ui';

interface ProfileSectionProps {
  onHasPasswordResolved: (hasPassword: boolean) => void;
  linkedProvider?: string | null;
}

type EmailChangePhase =
  | 'idle'
  | 'entering-email'
  | 'dual-otp'
  | 'identity-verifying';

function avatarFallbackLabel(username: string, email: string) {
  const source = username.trim() || email.trim();
  return source.slice(0, 2).toUpperCase() || 'AR';
}

export default function ProfileSection({
  onHasPasswordResolved,
  linkedProvider,
}: ProfileSectionProps) {
  const updateUser = useAuthStore((state) => state.updateUser);
  const notify = useNotificationStore((state) => state.notify);
  const { loading, error, setError, run } = useAsyncAction();
  const {
    loading: emailChangeLoading,
    error: emailChangeError,
    setError: setEmailChangeError,
    run: runEmailChange,
  } = useAsyncAction();

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null);
  const [emailChangePhase, setEmailChangePhase] =
    useState<EmailChangePhase>('idle');
  const [newEmail, setNewEmail] = useState('');
  const [codeOld, setCodeOld] = useState('');
  const [codeNew, setCodeNew] = useState('');
  const [verificationId, setVerificationId] = useState('');
  const [verificationMethod, setVerificationMethod] =
    useState<VerificationMethod>('password');
  const [verificationMetadata, setVerificationMetadata] = useState<
    Record<string, unknown> | undefined
  >();

  useEffect(() => {
    getProfile()
      .then((profile) => {
        setUsername(profile.username ?? '');
        setEmail(profile.email);
        setAvatarPreview(profile.avatarData);
        onHasPasswordResolved(profile.hasPassword);
      })
      .catch(() => {
        setError('Failed to load profile');
      });
  }, [onHasPasswordResolved, setError]);

  useEffect(() => {
    if (!linkedProvider) return;
    notify(
      `${linkedProvider.charAt(0).toUpperCase() + linkedProvider.slice(1)} account linked successfully`,
      'success',
    );
  }, [linkedProvider, notify]);

  const resetEmailChange = () => {
    setEmailChangePhase('idle');
    setNewEmail('');
    setCodeOld('');
    setCodeNew('');
    setVerificationId('');
    setVerificationMetadata(undefined);
    setEmailChangeError('');
  };

  const handleAvatarChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    if (file.size > 200 * 1024) {
      setError('Avatar must be under 200KB');
      event.target.value = '';
      return;
    }

    const reader = new FileReader();
    reader.onload = () => {
      const dataUrl = reader.result as string;
      setAvatarPreview(dataUrl);
      setError('');
      uploadAvatar(dataUrl)
        .then((result) => {
          updateUser({ avatarData: result.avatarData });
          notify('Avatar updated', 'success');
        })
        .catch(() => {
          setError('Failed to upload avatar');
        });
    };
    reader.readAsDataURL(file);
    event.target.value = '';
  };

  const handleProfileSave = async (event: React.FormEvent) => {
    event.preventDefault();
    const ok = await run(async () => {
      const result = await updateProfile({ username: username || undefined });
      updateUser({ username: result.username });
    }, 'Failed to update profile');

    if (ok) {
      notify('Profile updated successfully', 'success');
    }
  };

  const handleInitiateEmailChange = async () => {
    if (!newEmail || newEmail === email) {
      setEmailChangeError('Please enter a different email address.');
      return;
    }

    await runEmailChange(async () => {
      const result: EmailChangeInitResult = await initiateEmailChange(newEmail);
      if (result.flow === 'dual-otp') {
        setEmailChangePhase('dual-otp');
        return;
      }

      setVerificationId(result.verificationId ?? '');
      setVerificationMethod(result.method ?? 'password');
      setVerificationMetadata(result.metadata);
      setEmailChangePhase('identity-verifying');
    }, 'Failed to initiate email change');
  };

  const handleConfirmDualOtp = async () => {
    if (codeOld.length !== 6 || codeNew.length !== 6) {
      setEmailChangeError('Please enter both 6-digit codes.');
      return;
    }

    const ok = await runEmailChange(async () => {
      const result = await confirmEmailChange({ codeOld, codeNew });
      setEmail(result.email);
      updateUser({ email: result.email });
    }, 'Failed to confirm email change');

    if (ok) {
      notify('Email changed successfully', 'success');
      resetEmailChange();
    }
  };

  const handleIdentityVerified = async (nextVerificationId: string) => {
    const ok = await runEmailChange(async () => {
      const result = await confirmEmailChange({
        verificationId: nextVerificationId,
      });
      setEmail(result.email);
      updateUser({ email: result.email });
    }, 'Failed to confirm email change');

    if (ok) {
      notify('Email changed successfully', 'success');
      resetEmailChange();
      return;
    }

    setEmailChangePhase('entering-email');
  };

  return (
    <SettingsPanel
      title="Profile"
      description="Identity, avatar, and account email for your workspace access."
    >
      <div className="space-y-6">
        <div className="flex flex-wrap items-center gap-4 rounded-xl border border-border/70 bg-background/60 p-4">
          <Avatar className="size-16 border border-border/70">
            <AvatarImage src={avatarPreview ?? undefined} alt={email} />
            <AvatarFallback className="text-sm font-semibold">
              {avatarFallbackLabel(username, email)}
            </AvatarFallback>
          </Avatar>

          <div className="space-y-1">
            <p className="text-sm font-medium text-foreground">
              {username || email}
            </p>
            <p className="text-sm text-muted-foreground">
              Upload a small avatar to personalise shared workspaces and reviews.
            </p>
          </div>

          <div className="ml-auto">
            <Button
              type="button"
              variant="outline"
              onClick={() => fileInputRef.current?.click()}
            >
              <Upload className="size-4" />
              Change Avatar
            </Button>
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              hidden
              onChange={handleAvatarChange}
            />
          </div>
        </div>

        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <form className="space-y-4" onSubmit={handleProfileSave}>
          <div className="space-y-2">
            <Label htmlFor="profile-username">Username</Label>
            <Input
              id="profile-username"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              placeholder="Optional display name"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="profile-email">Email</Label>
            <Input id="profile-email" value={email} readOnly />
            <p className="text-xs leading-5 text-muted-foreground">
              Email changes use the dedicated verification flow below.
            </p>
          </div>

          <Button type="submit" disabled={loading}>
            <PencilLine className="size-4" />
            {loading ? 'Saving...' : 'Save Profile'}
          </Button>
        </form>

        <Separator />

        <div className="space-y-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <Mail className="size-4 text-primary" />
              <h4 className="text-sm font-semibold text-foreground">
                Change Email
              </h4>
            </div>
            <p className="text-sm text-muted-foreground">
              Move your sign-in email without leaving the settings flow.
            </p>
          </div>

          {emailChangeError && (
            <Alert variant="destructive">
              <AlertDescription>{emailChangeError}</AlertDescription>
            </Alert>
          )}

          {emailChangePhase === 'idle' && (
            <Button
              type="button"
              variant="outline"
              onClick={() => setEmailChangePhase('entering-email')}
            >
              Change Email
            </Button>
          )}

          {emailChangePhase === 'entering-email' && (
            <div className="space-y-4 rounded-xl border border-border/70 bg-background/60 p-4">
              <div className="space-y-2">
                <Label htmlFor="profile-new-email">New email</Label>
                <Input
                  id="profile-new-email"
                  type="email"
                  value={newEmail}
                  onChange={(event) => setNewEmail(event.target.value)}
                  autoFocus
                />
              </div>

              <SettingsButtonRow>
                <Button
                  type="button"
                  disabled={emailChangeLoading}
                  onClick={() => void handleInitiateEmailChange()}
                >
                  {emailChangeLoading ? 'Sending...' : 'Continue'}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={resetEmailChange}
                >
                  Cancel
                </Button>
              </SettingsButtonRow>
            </div>
          )}

          {emailChangePhase === 'dual-otp' && (
            <div className="space-y-4 rounded-xl border border-border/70 bg-background/60 p-4">
              <p className="text-sm text-muted-foreground">
                Verification codes were sent to both your current and new email
                addresses.
              </p>

              <div className="space-y-2">
                <Label htmlFor="profile-code-old">Code from current email</Label>
                <Input
                  id="profile-code-old"
                  value={codeOld}
                  onChange={(event) =>
                    setCodeOld(
                      event.target.value.replace(/\D/g, '').slice(0, 6),
                    )
                  }
                  maxLength={6}
                  inputMode="numeric"
                  autoFocus
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="profile-code-new">Code from new email</Label>
                <Input
                  id="profile-code-new"
                  value={codeNew}
                  onChange={(event) =>
                    setCodeNew(
                      event.target.value.replace(/\D/g, '').slice(0, 6),
                    )
                  }
                  maxLength={6}
                  inputMode="numeric"
                />
              </div>

              <SettingsButtonRow>
                <Button
                  type="button"
                  disabled={emailChangeLoading}
                  onClick={() => void handleConfirmDualOtp()}
                >
                  {emailChangeLoading ? 'Verifying...' : 'Confirm'}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={resetEmailChange}
                >
                  Cancel
                </Button>
              </SettingsButtonRow>
            </div>
          )}

          {emailChangePhase === 'identity-verifying' && (
            <IdentityVerification
              verificationId={verificationId}
              method={verificationMethod}
              metadata={verificationMetadata}
              onVerified={handleIdentityVerified}
              onCancel={resetEmailChange}
            />
          )}
        </div>
      </div>
    </SettingsPanel>
  );
}
