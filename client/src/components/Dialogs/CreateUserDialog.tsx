import { useState, useEffect } from 'react';
import {
  Dialog, DialogContent, DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem,
} from '@/components/ui/select';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Copy, RefreshCw, X } from 'lucide-react';
import { useTenantStore } from '../../store/tenantStore';
import { getEmailStatus } from '../../api/admin.api';
import { useAsyncAction } from '../../hooks/useAsyncAction';
import type { CreateUserResult } from '../../api/tenant.api';
import { ASSIGNABLE_ROLES, ROLE_LABELS, type TenantRole } from '../../utils/roles';

interface CreateUserDialogProps {
  open: boolean;
  onClose: () => void;
}

function generatePassword(): string {
  const chars = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%&*';
  const arr = new Uint8Array(16);
  crypto.getRandomValues(arr);
  return Array.from(arr, (b) => chars[b % chars.length]).join('');
}

export default function CreateUserDialog({ open, onClose }: CreateUserDialogProps) {
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [role, setRole] = useState<TenantRole>('MEMBER');
  const [expiresAt, setExpiresAt] = useState('');
  const [sendWelcomeEmail, setSendWelcomeEmail] = useState(false);
  const [emailConfigured, setEmailConfigured] = useState(false);
  const { loading, error, setError, clearError, run } = useAsyncAction();
  const [result, setResult] = useState<CreateUserResult | null>(null);
  const [copied, setCopied] = useState('');
  const createUser = useTenantStore((s) => s.createUser);

  useEffect(() => {
    if (open) {
      getEmailStatus()
        .then((s) => setEmailConfigured(s.configured))
        .catch(() => setEmailConfigured(false));
    }
  }, [open]);

  const handleGenerate = () => {
    const pw = generatePassword();
    setPassword(pw);
    setConfirmPassword(pw);
  };

  const handleCopy = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    setCopied(label);
    setTimeout(() => setCopied(''), 2000);
  };

  const handleSubmit = async () => {
    if (!email.trim()) { setError('Email is required'); return; }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) { setError('Please enter a valid email address'); return; }
    if (password.length < 8) { setError('Password must be at least 8 characters'); return; }
    // eslint-disable-next-line security/detect-possible-timing-attacks -- client-side UI validation, not a security comparison
    if (password !== confirmPassword) { setError('Passwords do not match'); return; }

    await run(async () => {
      const res = await createUser({
        email: email.trim(),
        username: username.trim() || undefined,
        password,
        role,
        sendWelcomeEmail: emailConfigured ? sendWelcomeEmail : false,
        expiresAt: expiresAt ? new Date(expiresAt).toISOString() : undefined,
      });
      setResult(res);
    }, 'Failed to create user');
  };

  const handleClose = () => {
    setEmail('');
    setUsername('');
    setPassword('');
    setConfirmPassword('');
    setRole('MEMBER');
    setExpiresAt('');
    setSendWelcomeEmail(false);
    clearError();
    setResult(null);
    setCopied('');
    onClose();
  };

  if (result) {
    return (
      <Dialog open={open} onOpenChange={(next) => { if (!next) handleClose(); }}>
        <DialogContent
          showCloseButton={false}
          className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
        >
          <DialogTitle className="sr-only">User Created Successfully</DialogTitle>
          <DialogDescription className="sr-only">User account details</DialogDescription>

          {/* Compact header */}
          <div className="flex h-8 shrink-0 items-center gap-2 border-b px-3">
            <span className="text-xs font-medium">User Created Successfully</span>
            <div className="ml-auto">
              <Button variant="ghost" size="icon-xs" onClick={handleClose}>
                <X className="size-3.5" />
              </Button>
            </div>
          </div>

          <ScrollArea className="flex-1">
            <div className="mx-auto max-w-2xl px-6 py-4">
              <div className="rounded-md border border-emerald-600/50 bg-emerald-600/10 px-4 py-3 text-sm text-emerald-400">
                Account created for {result.user.email}
              </div>

              <div className="mt-4 rounded-lg border p-4 space-y-3">
                <div>
                  <p className="text-xs text-muted-foreground">Email</p>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-mono flex-1">{result.user.email}</span>
                    <Button variant="ghost" size="icon" className="size-7" onClick={() => handleCopy(result.user.email, 'email')}>
                      <Copy className="size-3.5" />
                    </Button>
                  </div>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Password</p>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-mono flex-1">{password}</span>
                    <Button variant="ghost" size="icon" className="size-7" onClick={() => handleCopy(password, 'password')}>
                      <Copy className="size-3.5" />
                    </Button>
                  </div>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Recovery Key (show once)</p>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-mono flex-1 break-all">{result.recoveryKey}</span>
                    <Button variant="ghost" size="icon" className="size-7" onClick={() => handleCopy(result.recoveryKey, 'recovery')}>
                      <Copy className="size-3.5" />
                    </Button>
                  </div>
                </div>
                {copied && (
                  <p className="text-xs text-emerald-400">Copied {copied}!</p>
                )}
              </div>

              <div className="mt-4 rounded-md border border-yellow-600/50 bg-yellow-600/10 px-4 py-3 text-sm text-yellow-500">
                Save these credentials now. The recovery key will not be shown again.
              </div>
            </div>
          </ScrollArea>

          <div className="flex shrink-0 items-center justify-end gap-2 border-t px-4 py-2">
            <Button onClick={handleClose}>Done</Button>
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) handleClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        <DialogTitle className="sr-only">Create User</DialogTitle>
        <DialogDescription className="sr-only">Create a new user account</DialogDescription>

        {/* Compact header */}
        <div className="flex h-8 shrink-0 items-center gap-2 border-b px-3">
          <span className="text-xs font-medium">Create User</span>
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
                <Label htmlFor="create-email">Email Address</Label>
                <Input
                  id="create-email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  autoFocus
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="create-username">Username (optional)</Label>
                <Input
                  id="create-username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="create-password">Password</Label>
                <div className="flex gap-2">
                  <Input
                    id="create-password"
                    type="text"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    required
                    className="flex-1"
                  />
                  <Button
                    variant="ghost"
                    size="icon"
                    onClick={handleGenerate}
                    title="Generate password"
                    type="button"
                  >
                    <RefreshCw className="size-4" />
                  </Button>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="create-confirm">Confirm Password</Label>
                <Input
                  id="create-confirm"
                  type="text"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  className={confirmPassword && password !== confirmPassword ? 'border-destructive' : ''}
                />
                {confirmPassword && password !== confirmPassword && (
                  <p className="text-xs text-destructive">Passwords do not match</p>
                )}
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
                <Label htmlFor="create-expires">Access Expires At</Label>
                <Input
                  id="create-expires"
                  type="datetime-local"
                  value={expiresAt}
                  onChange={(e) => setExpiresAt(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">Leave empty for permanent access</p>
              </div>

              {emailConfigured && (
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="send-welcome"
                    checked={sendWelcomeEmail}
                    onCheckedChange={(v) => setSendWelcomeEmail(v === true)}
                  />
                  <Label htmlFor="send-welcome" className="font-normal">
                    Send welcome email with credentials
                  </Label>
                </div>
              )}
            </div>
          </div>
        </ScrollArea>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t px-4 py-2">
          <Button variant="outline" onClick={handleClose}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? 'Creating...' : 'Create User'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
