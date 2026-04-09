import { useState } from 'react';
import {
  Dialog, DialogContent, DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem,
} from '@/components/ui/select';
import { ScrollArea } from '@/components/ui/scroll-area';
import { X } from 'lucide-react';
import { useTenantStore } from '../../store/tenantStore';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import { ASSIGNABLE_ROLES, ROLE_LABELS, type TenantRole } from '../../utils/roles';

interface InviteDialogProps {
  open: boolean;
  onClose: () => void;
}

export default function InviteDialog({ open, onClose }: InviteDialogProps) {
  const [email, setEmail] = useState('');
  const [role, setRole] = useState<TenantRole>('MEMBER');
  const [expiresAt, setExpiresAt] = useState('');
  const { loading, error, setError, run } = useAsyncAction();
  const inviteUser = useTenantStore((s) => s.inviteUser);

  const handleSubmit = async () => {
    if (!email.trim()) {
      setError('Email is required');
      return;
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
      setError('Please enter a valid email address');
      return;
    }

    const ok = await run(async () => {
      await inviteUser(email.trim(), role, expiresAt ? new Date(expiresAt).toISOString() : undefined);
    }, 'Failed to invite user');
    if (ok) handleClose();
  };

  const handleClose = () => {
    setEmail('');
    setRole('MEMBER');
    setExpiresAt('');
    setError('');
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) handleClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        <DialogTitle className="sr-only">Invite Member</DialogTitle>
        <DialogDescription className="sr-only">Invite a new member to the organization</DialogDescription>

        {/* Compact header */}
        <div className="flex h-8 shrink-0 items-center gap-2 border-b px-3">
          <span className="text-xs font-medium">Invite Member</span>
          <div className="ml-auto">
            <Button variant="ghost" size="icon-xs" onClick={handleClose}>
              <X className="size-3.5" />
            </Button>
          </div>
        </div>

        <ScrollArea className="flex-1">
          <div className="mx-auto max-w-2xl px-6 py-4">
            {error && (
              <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
                {error}
              </div>
            )}

            <div className="flex flex-col gap-4">
              <div className="space-y-2">
                <Label htmlFor="invite-email">Email Address</Label>
                <Input
                  id="invite-email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  autoFocus
                />
              </div>

              <div className="space-y-2">
                <Label>Role</Label>
                <Select value={role} onValueChange={(v) => setRole(v as TenantRole)}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {ASSIGNABLE_ROLES.map((r) => (
                      <SelectItem key={r} value={r}>{ROLE_LABELS[r]}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="invite-expires">Access Expires At</Label>
                <Input
                  id="invite-expires"
                  type="datetime-local"
                  value={expiresAt}
                  onChange={(e) => setExpiresAt(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">Leave empty for permanent access</p>
              </div>
            </div>
          </div>
        </ScrollArea>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t px-4 py-2">
          <Button variant="outline" onClick={handleClose}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? 'Inviting...' : 'Invite'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
