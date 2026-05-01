import { useState, useEffect } from 'react';
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
import { createFolder, updateFolder, FolderData } from '../../api/folders.api';
import { useConnectionsStore } from '../../store/connectionsStore';
import { useAsyncAction } from '../../hooks/useAsyncAction';

interface FolderDialogProps {
  open: boolean;
  onClose: () => void;
  folder?: FolderData | null;
  parentId?: string | null;
  teamId?: string | null;
}

function getDescendantIds(folderId: string, folders: FolderData[]): Set<string> {
  const ids = new Set<string>();
  const queue = [folderId];
  while (queue.length > 0) {
    const current = queue.pop() as string;
    for (const f of folders) {
      if (f.parentId === current && !ids.has(f.id)) {
        ids.add(f.id);
        queue.push(f.id);
      }
    }
  }
  return ids;
}

export default function FolderDialog({ open, onClose, folder, parentId, teamId }: FolderDialogProps) {
  const [name, setName] = useState('');
  const [selectedParentId, setSelectedParentId] = useState('');
  const { loading, error, setError, clearError, run } = useAsyncAction();
  const folders = useConnectionsStore((s) => s.folders);
  const fetchConnections = useConnectionsStore((s) => s.fetchConnections);

  const isEditMode = Boolean(folder);

  useEffect(() => {
    if (open && folder) {
      setName(folder.name);
      setSelectedParentId(folder.parentId || '');
    } else if (open) {
      setName('');
      setSelectedParentId(parentId || '');
    }
    clearError();
  // eslint-disable-next-line react-hooks/exhaustive-deps -- clearError is stable (useCallback with [])
  }, [open, folder, parentId]);

  const excludedIds = folder
    ? new Set([folder.id, ...getDescendantIds(folder.id, folders)])
    : new Set<string>();

  const availableParents = folders.filter((f) => !excludedIds.has(f.id));

  const handleSubmit = async () => {
    if (!name.trim()) {
      setError('Folder name is required');
      return;
    }

    const ok = await run(async () => {
      if (isEditMode && folder) {
        const data: { name?: string; parentId?: string | null } = {};
        if (name !== folder.name) data.name = name.trim();
        if (selectedParentId !== (folder.parentId || '')) {
          data.parentId = selectedParentId || null;
        }
        if (Object.keys(data).length > 0) {
          await updateFolder(folder.id, data);
        }
      } else {
        await createFolder({
          name: name.trim(),
          ...(selectedParentId ? { parentId: selectedParentId } : {}),
          ...(teamId ? { teamId } : {}),
        });
      }
      await fetchConnections();
    }, isEditMode ? 'Failed to update folder' : 'Failed to create folder');
    if (ok) handleClose();
  };

  const handleClose = () => {
    setName('');
    setSelectedParentId('');
    clearError();
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) handleClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        <DialogTitle className="sr-only">{isEditMode ? 'Rename Folder' : 'New Folder'}</DialogTitle>
        <DialogDescription className="sr-only">
          {isEditMode ? 'Rename an existing folder' : 'Create a new folder'}
        </DialogDescription>

        {/* Compact header */}
        <div className="flex h-8 shrink-0 items-center gap-2 border-b px-3">
          <span className="text-xs font-medium">{isEditMode ? 'Rename Folder' : 'New Folder'}</span>
          <div className="ml-auto">
            <Button variant="ghost" size="icon-xs" onClick={handleClose}>
              <X className="size-3.5" />
            </Button>
          </div>
        </div>

        <ScrollArea className="flex-1">
          <div className="mx-auto max-w-lg px-6 py-4">
            {error && (
              <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
                {error}
              </div>
            )}

            <div className="flex flex-col gap-4">
              <div className="space-y-2">
                <Label htmlFor="folder-name">Folder Name</Label>
                <Input
                  id="folder-name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                  autoFocus
                />
              </div>

              <div className="space-y-2">
                <Label>Parent Folder</Label>
                <Select value={selectedParentId || '__root__'} onValueChange={(v) => setSelectedParentId(v === '__root__' ? '' : v)}>
                  <SelectTrigger>
                    <SelectValue placeholder="None (root level)" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__root__">None (root level)</SelectItem>
                    {availableParents.map((f) => (
                      <SelectItem key={f.id} value={f.id}>{f.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
          </div>
        </ScrollArea>

        <div className="flex shrink-0 items-center justify-end gap-2 border-t px-4 py-2">
          <Button variant="outline" onClick={handleClose}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading
              ? (isEditMode ? 'Saving...' : 'Creating...')
              : (isEditMode ? 'Save' : 'Create')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
