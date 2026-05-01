import { Copy, Loader2 } from 'lucide-react';
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
import IdentityVerification from '../common/IdentityVerification';
import type { VerificationMethod } from '../../api/user.api';

export interface TenantDialogTarget {
  id: string;
  name: string;
}

export interface TenantExpiryTarget extends TenantDialogTarget {
  expiresAt: string | null;
}

export function DeleteTenantDialog({
  confirmName,
  deleting,
  onClose,
  onConfirm,
  onConfirmNameChange,
  open,
  tenantName,
}: {
  confirmName: string;
  deleting: boolean;
  onClose: () => void;
  onConfirm: () => void;
  onConfirmNameChange: (value: string) => void;
  open: boolean;
  tenantName: string;
}) {
  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Delete Organization</DialogTitle>
          <DialogDescription>
            This permanently deletes the organization, every team, and every membership. Type
            {' '}
            <strong>{tenantName}</strong>
            {' '}
            to confirm.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2">
          <Label htmlFor="delete-org-name">Organization name</Label>
          <Input
            id="delete-org-name"
            value={confirmName}
            placeholder={tenantName}
            onChange={(event) => onConfirmNameChange(event.target.value)}
          />
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            disabled={deleting || confirmName !== tenantName}
            onClick={onConfirm}
          >
            {deleting ? <Loader2 className="animate-spin" /> : null}
            {deleting ? 'Deleting...' : 'Delete Organization'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function RemoveMemberDialog({
  onClose,
  onConfirm,
  open,
  target,
}: {
  onClose: () => void;
  onConfirm: () => void;
  open: boolean;
  target: TenantDialogTarget | null;
}) {
  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Remove Member</DialogTitle>
          <DialogDescription>
            Remove
            {' '}
            <strong>{target?.name}</strong>
            {' '}
            from the organization and every team they belong to.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button type="button" variant="destructive" onClick={onConfirm}>
            Remove Member
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function MembershipExpiryDialog({
  onClose,
  onRemove,
  onSave,
  onValueChange,
  open,
  saving,
  target,
  value,
}: {
  onClose: () => void;
  onRemove: () => void;
  onSave: () => void;
  onValueChange: (value: string) => void;
  open: boolean;
  saving: boolean;
  target: TenantExpiryTarget | null;
  value: string;
}) {
  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Membership Expiration</DialogTitle>
          <DialogDescription>
            Set or remove the organization access expiry for
            {' '}
            <strong>{target?.name}</strong>
            .
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2">
          <Label htmlFor="tenant-member-expiry">Expires at</Label>
          <Input
            id="tenant-member-expiry"
            type="datetime-local"
            value={value}
            onChange={(event) => onValueChange(event.target.value)}
          />
          <p className="text-sm text-muted-foreground">Clear the field and remove expiration to keep access open-ended.</p>
        </div>

        <DialogFooter className="sm:justify-between">
          <div>
            {target?.expiresAt ? (
              <Button type="button" variant="outline" onClick={onRemove} disabled={saving}>
                Remove Expiration
              </Button>
            ) : null}
          </div>
          <div className="flex items-center gap-2">
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="button" onClick={onSave} disabled={saving || !value}>
              {saving ? <Loader2 className="animate-spin" /> : null}
              {saving ? 'Saving...' : 'Save'}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function ChangeUserEmailDialog({
  error,
  loading,
  metadata,
  method,
  newEmail,
  onClose,
  onEmailChange,
  onSubmit,
  onVerified,
  open,
  phase,
  target,
  verificationId,
}: {
  error: string;
  loading: boolean;
  metadata?: Record<string, unknown>;
  method: VerificationMethod;
  newEmail: string;
  onClose: () => void;
  onEmailChange: (value: string) => void;
  onSubmit: () => void;
  onVerified: (verificationId: string) => void;
  open: boolean;
  phase: 'input' | 'verifying' | 'done';
  target: TenantDialogTarget | null;
  verificationId: string;
}) {
  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Change Email</DialogTitle>
          <DialogDescription>
            Update the sign-in email for
            {' '}
            <strong>{target?.name}</strong>
            .
          </DialogDescription>
        </DialogHeader>

        {error ? (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        {phase === 'input' ? (
          <div className="space-y-2">
            <Label htmlFor="tenant-change-email">New email</Label>
            <Input
              id="tenant-change-email"
              type="email"
              autoFocus
              value={newEmail}
              onChange={(event) => onEmailChange(event.target.value)}
            />
            <p className="text-sm text-muted-foreground">
              The updated email is applied immediately after your identity verification succeeds.
            </p>
          </div>
        ) : (
          <IdentityVerification
            verificationId={verificationId}
            method={method}
            metadata={metadata}
            onVerified={onVerified}
            onCancel={onClose}
          />
        )}

        <DialogFooter>
          {phase === 'input' ? (
            <>
              <Button type="button" variant="outline" onClick={onClose}>
                Cancel
              </Button>
              <Button type="button" onClick={onSubmit} disabled={loading || !newEmail.trim()}>
                {loading ? <Loader2 className="animate-spin" /> : null}
                {loading ? 'Verifying...' : 'Continue'}
              </Button>
            </>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function ChangeUserPasswordDialog({
  confirmPassword,
  error,
  loading,
  metadata,
  method,
  newPassword,
  onClose,
  onConfirmPasswordChange,
  onCopyRecoveryKey,
  onNewPasswordChange,
  onSubmit,
  onVerified,
  open,
  phase,
  recoveryKey,
  target,
  verificationId,
}: {
  confirmPassword: string;
  error: string;
  loading: boolean;
  metadata?: Record<string, unknown>;
  method: VerificationMethod;
  newPassword: string;
  onClose: () => void;
  onConfirmPasswordChange: (value: string) => void;
  onCopyRecoveryKey: () => void;
  onNewPasswordChange: (value: string) => void;
  onSubmit: () => void;
  onVerified: (verificationId: string) => void;
  open: boolean;
  phase: 'input' | 'verifying' | 'done';
  recoveryKey: string;
  target: TenantDialogTarget | null;
  verificationId: string;
}) {
  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen && phase !== 'verifying') onClose(); }}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Change Password</DialogTitle>
          <DialogDescription>
            Reset the password for
            {' '}
            <strong>{target?.name}</strong>
            .
          </DialogDescription>
        </DialogHeader>

        {error ? (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        {phase === 'input' ? (
          <div className="space-y-4">
            <Alert>
              <AlertDescription>
                Resetting the password also resets the user&apos;s vault. Stored secrets become inaccessible until the recovery key is used.
              </AlertDescription>
            </Alert>
            <div className="space-y-2">
              <Label htmlFor="tenant-new-password">New password</Label>
              <Input
                id="tenant-new-password"
                type="password"
                autoFocus
                value={newPassword}
                onChange={(event) => onNewPasswordChange(event.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="tenant-confirm-password">Confirm password</Label>
              <Input
                id="tenant-confirm-password"
                type="password"
                value={confirmPassword}
                onChange={(event) => onConfirmPasswordChange(event.target.value)}
              />
            </div>
          </div>
        ) : null}

        {phase === 'verifying' ? (
          <IdentityVerification
            verificationId={verificationId}
            method={method}
            metadata={metadata}
            onVerified={onVerified}
            onCancel={onClose}
          />
        ) : null}

        {phase === 'done' ? (
          <div className="space-y-4">
            <Alert>
              <AlertDescription>
                Password updated. Save the recovery key now. It will not be shown again.
              </AlertDescription>
            </Alert>
            <div className="space-y-2">
              <Label htmlFor="tenant-recovery-key">Recovery key</Label>
              <Input
                id="tenant-recovery-key"
                readOnly
                value={recoveryKey}
                className="font-mono text-xs"
              />
            </div>
            <Button type="button" variant="outline" onClick={onCopyRecoveryKey}>
              <Copy className="size-4" />
              Copy Recovery Key
            </Button>
          </div>
        ) : null}

        <DialogFooter>
          {phase === 'input' ? (
            <>
              <Button type="button" variant="outline" onClick={onClose}>
                Cancel
              </Button>
              <Button type="button" onClick={onSubmit} disabled={loading || !newPassword || !confirmPassword}>
                {loading ? <Loader2 className="animate-spin" /> : null}
                {loading ? 'Verifying...' : 'Continue'}
              </Button>
            </>
          ) : null}
          {phase === 'done' ? (
            <Button type="button" onClick={onClose}>
              Close
            </Button>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function MandatoryMfaDialog({
  onClose,
  onConfirm,
  open,
  stats,
}: {
  onClose: () => void;
  onConfirm: () => void;
  open: boolean;
  stats: { total: number; withoutMfa: number } | null;
}) {
  return (
    <Dialog open={open} onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Enable Mandatory MFA</DialogTitle>
          <DialogDescription>
            {stats && stats.withoutMfa > 0
              ? (
                <>
                  <strong>{stats.withoutMfa}</strong>
                  {' '}
                  of
                  {' '}
                  <strong>{stats.total}</strong>
                  {' '}
                  members still do not have MFA configured.
                </>
              )
              : 'All current members already have MFA configured.'}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button type="button" onClick={onConfirm}>
            Enable Mandatory MFA
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
