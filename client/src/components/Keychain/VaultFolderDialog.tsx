import { useState, useEffect } from 'react';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Alert } from '@/components/ui/alert';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { createVaultFolder, updateVaultFolder } from '../../api/vault-folders.api';
import type { VaultFolderData } from '../../api/vault-folders.api';
import { useSecretStore } from '../../store/secretStore';
import { useAuthStore } from '../../store/authStore';
import { useTeamStore } from '../../store/teamStore';
import { extractApiError } from '../../utils/apiError';
import { isAdminOrAbove } from '../../utils/roles';

interface VaultFolderDialogProps {
  open: boolean;
  onClose: () => void;
  folder?: VaultFolderData | null;
  parentId?: string | null;
  scope?: 'PERSONAL' | 'TEAM' | 'TENANT';
  teamId?: string | null;
}

function getDescendantIds(folderId: string, folders: VaultFolderData[]): Set<string> {
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

export default function VaultFolderDialog({
  open, onClose, folder, parentId, scope: propScope = 'PERSONAL', teamId: propTeamId,
}: VaultFolderDialogProps) {
  const [name, setName] = useState('');
  const [selectedParentId, setSelectedParentId] = useState('');
  const [selectedScope, setSelectedScope] = useState<'PERSONAL' | 'TEAM' | 'TENANT'>('PERSONAL');
  const [selectedTeamId, setSelectedTeamId] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const vaultFolders = useSecretStore((s) => s.vaultFolders);
  const vaultTeamFolders = useSecretStore((s) => s.vaultTeamFolders);
  const vaultTenantFolders = useSecretStore((s) => s.vaultTenantFolders);
  const tenantVaultStatus = useSecretStore((s) => s.tenantVaultStatus);
  const fetchVaultFolders = useSecretStore((s) => s.fetchVaultFolders);
  const fetchSecrets = useSecretStore((s) => s.fetchSecrets);

  const user = useAuthStore((s) => s.user);
  const teams = useTeamStore((s) => s.teams);
  const fetchTeams = useTeamStore((s) => s.fetchTeams);

  const isEditMode = Boolean(folder);

  const canSelectTeam = user?.tenantId && teams.length > 0;
  const canSelectTenant = user?.tenantId && isAdminOrAbove(user.tenantRole);
  const tenantVaultReady = tenantVaultStatus?.initialized && tenantVaultStatus?.hasAccess;

  // Determine active scope and teamId
  const activeScope = isEditMode ? (folder?.scope ?? 'PERSONAL') : selectedScope;
  const activeTeamId = isEditMode ? (folder?.teamId ?? null) : (selectedTeamId || null);

  // Get folders for the current scope
  const scopeFolders = activeScope === 'TEAM'
    ? vaultTeamFolders.filter((f) => f.teamId === activeTeamId)
    : activeScope === 'TENANT'
      ? vaultTenantFolders
      : vaultFolders;

  useEffect(() => {
    if (open) {
      fetchTeams();
      if (folder) {
        setName(folder.name);
        setSelectedParentId(folder.parentId || '');
        setSelectedScope(folder.scope);
        setSelectedTeamId(folder.teamId || '');
      } else {
        setName('');
        setSelectedParentId(parentId || '');
        setSelectedScope(propScope);
        setSelectedTeamId(propTeamId || '');
      }
      setError('');
    }
  }, [open, folder, parentId, propScope, propTeamId, fetchTeams]);

  // Reset parent when scope/team changes (parent folders are scope-specific)
  useEffect(() => {
    if (!isEditMode) {
      setSelectedParentId('');
    }
  }, [selectedScope, selectedTeamId, isEditMode]);

  const excludedIds = folder
    ? new Set([folder.id, ...getDescendantIds(folder.id, scopeFolders)])
    : new Set<string>();

  const availableParents = scopeFolders.filter((f) => !excludedIds.has(f.id));

  const handleSubmit = async () => {
    setError('');
    if (!name.trim()) {
      setError('Folder name is required');
      return;
    }
    if (!isEditMode && activeScope === 'TEAM' && !activeTeamId) {
      setError('Please select a team');
      return;
    }

    setLoading(true);
    try {
      if (isEditMode && folder) {
        const data: { name?: string; parentId?: string | null } = {};
        if (name !== folder.name) data.name = name.trim();
        if (selectedParentId !== (folder.parentId || '')) {
          data.parentId = selectedParentId || null;
        }
        if (Object.keys(data).length > 0) {
          await updateVaultFolder(folder.id, data);
        }
      } else {
        await createVaultFolder({
          name: name.trim(),
          scope: activeScope,
          ...(selectedParentId ? { parentId: selectedParentId } : {}),
          ...(activeScope === 'TEAM' && activeTeamId ? { teamId: activeTeamId } : {}),
        });
      }
      await fetchVaultFolders();
      await fetchSecrets();
      handleClose();
    } catch (err: unknown) {
      setError(extractApiError(err, isEditMode ? 'Failed to update folder' : 'Failed to create folder'));
    } finally {
      setLoading(false);
    }
  };

  const handleClose = () => {
    setName('');
    setSelectedParentId('');
    setSelectedScope('PERSONAL');
    setSelectedTeamId('');
    setError('');
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) handleClose(); }}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>{isEditMode ? 'Rename Folder' : 'New Folder'}</DialogTitle>
          <DialogDescription className="sr-only">
            {isEditMode ? 'Rename an existing folder' : 'Create a new vault folder'}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          {error && <Alert variant="destructive">{error}</Alert>}

          <div className="space-y-1.5">
            <Label>Folder Name *</Label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
          </div>

          {/* Scope selector -- only on create */}
          {!isEditMode && (
            <div className="space-y-1.5">
              <Label>Scope</Label>
              <Select value={selectedScope} onValueChange={(v) => setSelectedScope(v as 'PERSONAL' | 'TEAM' | 'TENANT')}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="PERSONAL">Personal</SelectItem>
                  {canSelectTeam && <SelectItem value="TEAM">Team</SelectItem>}
                  {canSelectTenant && (
                    <SelectItem value="TENANT" disabled={!tenantVaultReady}>
                      Organization{!tenantVaultReady ? ' (vault not initialized)' : ''}
                    </SelectItem>
                  )}
                </SelectContent>
              </Select>
            </div>
          )}

          {/* Team selector -- only on create + TEAM scope */}
          {!isEditMode && selectedScope === 'TEAM' && (
            <div className="space-y-1.5">
              <Label>Team</Label>
              <Select value={selectedTeamId} onValueChange={setSelectedTeamId}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {teams.map((t) => (
                    <SelectItem key={t.id} value={t.id}>{t.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <div className="space-y-1.5">
            <Label>Parent Folder</Label>
            <Select value={selectedParentId || '__root__'} onValueChange={(v) => setSelectedParentId(v === '__root__' ? '' : v)}>
              <SelectTrigger><SelectValue placeholder="None (root level)" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="__root__">None (root level)</SelectItem>
                {availableParents.map((f) => (
                  <SelectItem key={f.id} value={f.id}>{f.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading
              ? (isEditMode ? 'Saving...' : 'Creating...')
              : (isEditMode ? 'Save' : 'Create')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
