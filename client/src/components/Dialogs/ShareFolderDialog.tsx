import { useState, useEffect, useMemo } from 'react';
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
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Terminal, Monitor, Loader2, X } from 'lucide-react';
import { batchShareConnections, BatchShareResult } from '../../api/sharing.api';
import { useAuthStore } from '../../store/authStore';
import { useConnectionsStore } from '../../store/connectionsStore';
import { ConnectionData } from '../../api/connections.api';
import { UserSearchResult } from '../../api/user.api';
import { collectFolderConnections } from '../Sidebar/treeHelpers';
import UserPicker from '../UserPicker';
import { useAsyncAction } from '../../hooks/useAsyncAction';

interface ShareFolderDialogProps {
  open: boolean;
  onClose: () => void;
  folderId: string;
  folderName: string;
}

export default function ShareFolderDialog({
  open,
  onClose,
  folderId,
  folderName,
}: ShareFolderDialogProps) {
  const hasTenant = !!useAuthStore((s) => s.user?.tenantId);
  const ownConnections = useConnectionsStore((s) => s.ownConnections);
  const folders = useConnectionsStore((s) => s.folders);

  const [email, setEmail] = useState('');
  const [selectedUser, setSelectedUser] = useState<UserSearchResult | null>(null);
  const [scope, setScope] = useState<'tenant' | 'team'>('tenant');
  const [permission, setPermission] = useState<'READ_ONLY' | 'FULL_ACCESS'>('READ_ONLY');
  const { loading, error, setError, clearError, run } = useAsyncAction();
  const [result, setResult] = useState<BatchShareResult | null>(null);

  // Collect owned connections in this folder (recursively)
  const folderConnections = useMemo(() => {
    if (!open) return [];
    const folderMap = new Map<string, ConnectionData[]>();
    ownConnections.forEach((c) => {
      if (c.folderId) {
        const list = folderMap.get(c.folderId) || [];
        list.push(c);
        folderMap.set(c.folderId, list);
      }
    });
    return collectFolderConnections(folderId, folderMap, folders, true)
      .filter((c) => c.isOwner);
  }, [open, folderId, ownConnections, folders]);

  useEffect(() => {
    if (open) {
      setSelectedUser(null);
      setEmail('');
      clearError();
      setResult(null);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- clearError is stable (useCallback with [])
  }, [open, folderId]);

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

    if (folderConnections.length === 0) {
      setError('No owned connections found in this folder');
      return;
    }

    await run(async () => {
      const target = selectedUser
        ? { userId: selectedUser.id }
        : { email };
      const connectionIds = folderConnections.map((c) => c.id);
      const res = await batchShareConnections(connectionIds, target, permission, folderName);
      setResult(res);
      setSelectedUser(null);
      setEmail('');
    }, 'Failed to share connections');
  };

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        <DialogTitle className="sr-only">Share Folder: {folderName}</DialogTitle>
        <DialogDescription className="sr-only">Share all connections in this folder</DialogDescription>

        {/* Compact header */}
        <div className="flex h-8 shrink-0 items-center gap-2 border-b px-3">
          <span className="text-xs font-medium">Share Folder: {folderName}</span>
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

            {result && (
              <div className={`mb-4 rounded-md border px-4 py-3 text-sm ${
                result.failed > 0
                  ? 'border-yellow-600/50 bg-yellow-600/10 text-yellow-500'
                  : 'border-emerald-600/50 bg-emerald-600/10 text-emerald-400'
              }`}>
                {result.shared} of {result.shared + result.failed + result.alreadyShared} connection{result.shared + result.failed + result.alreadyShared !== 1 ? 's' : ''} shared successfully
                {result.alreadyShared > 0 && ` (${result.alreadyShared} already shared)`}
                {result.failed > 0 && ` (${result.failed} failed)`}
              </div>
            )}

            <p className="text-sm text-muted-foreground">
              {folderConnections.length} connection{folderConnections.length !== 1 ? 's' : ''} will be shared (including subfolders)
            </p>

            <div className="mt-4 max-h-40 overflow-auto rounded-lg border">
              <div className="divide-y">
                {folderConnections.map((conn) => (
                  <div key={conn.id} className="flex items-center gap-2 px-3 py-1.5">
                    {conn.type === 'SSH'
                      ? <Terminal className="size-4 text-muted-foreground" />
                      : conn.type === 'VNC'
                      ? <Monitor className="size-4 text-blue-400" />
                      : <Monitor className="size-4 text-primary" />}
                    <span className="text-sm truncate">{conn.name}</span>
                  </div>
                ))}
                {folderConnections.length === 0 && (
                  <div className="px-3 py-3">
                    <span className="text-sm text-muted-foreground">No owned connections in this folder</span>
                  </div>
                )}
              </div>
            </div>

            {hasTenant && (
              <ToggleGroup
                type="single"
                value={scope}
                onValueChange={(val) => { if (val) setScope(val as 'tenant' | 'team'); }}
                className="mt-4 self-start"
              >
                <ToggleGroupItem value="tenant" size="sm">Organization</ToggleGroupItem>
                <ToggleGroupItem value="team" size="sm">My Team</ToggleGroupItem>
              </ToggleGroup>
            )}

            <div className="mt-4 flex gap-2 items-end">
              {hasTenant ? (
                <div className="flex-1">
                  <UserPicker
                    value={selectedUser}
                    onSelect={setSelectedUser}
                    scope={scope}
                    placeholder="Search users by name or email..."
                  />
                </div>
              ) : (
                <div className="flex-1 space-y-2">
                  <Label htmlFor="share-folder-email">User email</Label>
                  <Input
                    id="share-folder-email"
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
              <Button
                onClick={handleShare}
                disabled={loading || folderConnections.length === 0}
                className="whitespace-nowrap"
              >
                {loading ? <Loader2 className="size-4 animate-spin" /> : 'Share'}
              </Button>
            </div>
          </div>
        </ScrollArea>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t px-4 py-2">
          <Button variant="outline" onClick={onClose}>Close</Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
