import { useState, useEffect } from 'react';
import { Trash2 } from 'lucide-react';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Alert } from '@/components/ui/alert';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { cn } from '@/lib/utils';
import {
  shareSecret, unshareSecret, listShares,
} from '../../api/secrets.api';
import type { SecretShare } from '../../api/secrets.api';
import { useAuthStore } from '../../store/authStore';
import { UserSearchResult } from '../../api/user.api';
import UserPicker from '../UserPicker';
import { extractApiError } from '../../utils/apiError';

interface ShareSecretDialogProps {
  open: boolean;
  onClose: () => void;
  secretId: string;
  secretName: string;
  teamId?: string | null;
}

export default function ShareSecretDialog({
  open,
  onClose,
  secretId,
  secretName,
  teamId,
}: ShareSecretDialogProps) {
  const hasTenant = !!useAuthStore((s) => s.user?.tenantId);
  const [email, setEmail] = useState('');
  const [selectedUser, setSelectedUser] = useState<UserSearchResult | null>(null);
  const [scope, setScope] = useState<'tenant' | 'team'>('tenant');
  const [permission, setPermission] = useState<'READ_ONLY' | 'FULL_ACCESS'>('READ_ONLY');
  const [shares, setShares] = useState<SecretShare[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (open && secretId) {
      loadShares();
      setSelectedUser(null);
      setEmail('');
      setError('');
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- only trigger on dialog open
  }, [open, secretId]);

  const loadShares = async () => {
    try {
      const data = await listShares(secretId);
      setShares(data);
    } catch {
      // silently fail
    }
  };

  const sharedUserIds = shares.map((s) => s.userId);

  const handleShare = async () => {
    setError('');
    if (hasTenant) {
      if (!selectedUser) {
        setError('Select a user to share with');
        return;
      }
    } else {
      if (!email) {
        setError('Email is required');
        return;
      }
    }
    setLoading(true);
    try {
      const target = selectedUser
        ? { userId: selectedUser.id }
        : { email };
      await shareSecret(secretId, target, permission);
      setSelectedUser(null);
      setEmail('');
      await loadShares();
    } catch (err: unknown) {
      setError(extractApiError(err, 'Failed to share secret'));
    } finally {
      setLoading(false);
    }
  };

  const handleUnshare = async (userId: string) => {
    try {
      await unshareSecret(secretId, userId);
      await loadShares();
    } catch {
      // silently fail
    }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose(); }}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Share: {secretName}</DialogTitle>
          <DialogDescription className="sr-only">Share this secret with other users</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          {error && <Alert variant="destructive">{error}</Alert>}

          {hasTenant && teamId && (
            <ToggleGroup
              type="single"
              value={scope}
              onValueChange={(val) => { if (val) setScope(val as 'tenant' | 'team'); }}
            >
              <ToggleGroupItem value="tenant" size="sm">Organization</ToggleGroupItem>
              <ToggleGroupItem value="team" size="sm">My Team</ToggleGroupItem>
            </ToggleGroup>
          )}

          <div className="flex gap-2">
            {hasTenant ? (
              <div className="flex-1">
                <UserPicker
                  value={selectedUser}
                  onSelect={setSelectedUser}
                  scope={scope}
                  teamId={scope === 'team' && teamId ? teamId : undefined}
                  placeholder="Search users by name or email..."
                  excludeUserIds={sharedUserIds}
                />
              </div>
            ) : (
              <div className="flex-1 space-y-1.5">
                <Label>User email</Label>
                <Input
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>
            )}
            <div className="space-y-1.5 min-w-[140px]">
              <Label>Permission</Label>
              <Select
                value={permission}
                onValueChange={(v) => setPermission(v as 'READ_ONLY' | 'FULL_ACCESS')}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="READ_ONLY">Read Only</SelectItem>
                  <SelectItem value="FULL_ACCESS">Full Access</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-end">
              <Button onClick={handleShare} disabled={loading} className="whitespace-nowrap">
                Share
              </Button>
            </div>
          </div>

          {shares.length > 0 ? (
            <div className="space-y-1">
              {shares.map((share) => (
                <div key={share.id} className="flex items-center justify-between py-2 px-1">
                  <div>
                    <p className="text-sm">{share.email}</p>
                    <Badge
                      variant={share.permission === 'FULL_ACCESS' ? 'default' : 'secondary'}
                      className="mt-0.5"
                    >
                      {share.permission === 'READ_ONLY' ? 'Read Only' : 'Full Access'}
                    </Badge>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => handleUnshare(share.userId)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-muted-foreground text-center py-4">
              Not shared with anyone yet
            </p>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
