import { useState, useEffect } from 'react';
import {
  Dialog, DialogContent, DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import {
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem,
} from '@/components/ui/select';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Trash2, X } from 'lucide-react';
import {
  shareConnection, unshareConnection, listShares, ShareData,
} from '../../api/sharing.api';
import { useAuthStore } from '../../store/authStore';
import { UserSearchResult } from '../../api/user.api';
import UserPicker from '../UserPicker';
import { useAsyncAction } from '../../hooks/useAsyncAction';

interface ShareDialogProps {
  open: boolean;
  onClose: () => void;
  connectionId: string;
  connectionName: string;
  teamId?: string | null;
}

export default function ShareDialog({
  open,
  onClose,
  connectionId,
  connectionName,
  teamId,
}: ShareDialogProps) {
  const hasTenant = !!useAuthStore((s) => s.user?.tenantId);
  const [email, setEmail] = useState('');
  const [selectedUser, setSelectedUser] = useState<UserSearchResult | null>(null);
  const [scope, setScope] = useState<'tenant' | 'team'>('tenant');
  const [permission, setPermission] = useState<'READ_ONLY' | 'FULL_ACCESS'>('READ_ONLY');
  const [shares, setShares] = useState<ShareData[]>([]);
  const { loading, error, setError, clearError, run } = useAsyncAction();

  useEffect(() => {
    if (open) {
      loadShares();
      setSelectedUser(null);
      setEmail('');
      clearError();
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- loadShares is defined inline; clearError is stable (useCallback with [])
  }, [open, connectionId]);

  const loadShares = async () => {
    try {
      const data = await listShares(connectionId);
      setShares(data);
    } catch {}
  };

  const sharedUserIds = shares.map((s) => s.userId);

  const handleShare = async () => {
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
    await run(async () => {
      const target = selectedUser
        ? { userId: selectedUser.id }
        : { email };
      await shareConnection(connectionId, target, permission);
      setSelectedUser(null);
      setEmail('');
      await loadShares();
    }, 'Failed to share connection');
  };

  const handleUnshare = async (userId: string) => {
    try {
      await unshareConnection(connectionId, userId);
      await loadShares();
    } catch {}
  };

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        <DialogTitle className="sr-only">Share: {connectionName}</DialogTitle>
        <DialogDescription className="sr-only">Share connection with other users</DialogDescription>

        {/* Compact header */}
        <div className="flex h-8 shrink-0 items-center gap-2 border-b px-3">
          <span className="text-xs font-medium">Share: {connectionName}</span>
          <div className="ml-auto">
            <Button variant="ghost" size="icon-xs" onClick={onClose}>
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

            {hasTenant && teamId && (
              <ToggleGroup
                type="single"
                value={scope}
                onValueChange={(val) => { if (val) setScope(val as 'tenant' | 'team'); }}
                className="mb-4 self-start"
              >
                <ToggleGroupItem value="tenant" size="sm">Organization</ToggleGroupItem>
                <ToggleGroupItem value="team" size="sm">My Team</ToggleGroupItem>
              </ToggleGroup>
            )}

            <div className="flex gap-2 items-end">
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
                <div className="flex-1 space-y-2">
                  <Label htmlFor="share-email">User email</Label>
                  <Input
                    id="share-email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                  />
                </div>
              )}
              <div className="space-y-2">
                <Label className="sr-only">Permission</Label>
                <Select value={permission} onValueChange={(v) => setPermission(v as 'READ_ONLY' | 'FULL_ACCESS')}>
                  <SelectTrigger className="w-[140px]">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="READ_ONLY">Read Only</SelectItem>
                    <SelectItem value="FULL_ACCESS">Full Access</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <Button onClick={handleShare} disabled={loading} className="whitespace-nowrap">
                Share
              </Button>
            </div>

            {shares.length > 0 ? (
              <div className="mt-4 space-y-1">
                {shares.map((share) => (
                  <div key={share.id} className="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-accent/50">
                    <div className="flex flex-col gap-1">
                      <span className="text-sm">{share.email}</span>
                      <Badge
                        variant={share.permission === 'FULL_ACCESS' ? 'default' : 'secondary'}
                        className="w-fit"
                      >
                        {share.permission === 'READ_ONLY' ? 'Read Only' : 'Full Access'}
                      </Badge>
                    </div>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="size-8"
                      onClick={() => handleUnshare(share.userId)}
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                ))}
              </div>
            ) : (
              <p className="mt-4 text-sm text-muted-foreground text-center py-4">
                Not shared with anyone yet
              </p>
            )}
          </div>
        </ScrollArea>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t px-4 py-2">
          <Button variant="outline" onClick={onClose}>Close</Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
