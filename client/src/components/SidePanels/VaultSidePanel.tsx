import { useState, useEffect, useCallback } from 'react';
import { KeyRound, Lock, Plus } from 'lucide-react';
import {
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
} from '@/components/ui/sidebar';
import { Button } from '@/components/ui/button';
import { useVaultStore } from '@/store/vaultStore';
import { useSecretStore } from '@/store/secretStore';
import SecretTree from '../Keychain/SecretTree';
import SecretListPanel from '../Keychain/SecretListPanel';
import SecretDialog from '../Keychain/SecretDialog';
import VaultFolderDialog from '../Keychain/VaultFolderDialog';
import ShareSecretDialog from '../Keychain/ShareSecretDialog';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import type { SecretDetail, SecretListItem } from '@/api/secrets.api';
import type { VaultFolderData, VaultFolderScope } from '@/api/vault-folders.api';
import { getSecret } from '@/api/secrets.api';

export default function VaultSidePanel() {
  const vaultUnlocked = useVaultStore((s) => s.unlocked);
  const checkVaultStatus = useVaultStore((s) => s.checkStatus);

  const fetchSecrets = useSecretStore((s) => s.fetchSecrets);
  const fetchVaultFolders = useSecretStore((s) => s.fetchVaultFolders);
  const deleteSecretAction = useSecretStore((s) => s.deleteSecret);

  // Check vault status on mount
  useEffect(() => {
    checkVaultStatus();
  }, [checkVaultStatus]);

  // Load data when vault is unlocked
  useEffect(() => {
    if (vaultUnlocked) {
      fetchVaultFolders();
      fetchSecrets();
    }
  }, [vaultUnlocked, fetchVaultFolders, fetchSecrets]);

  // --- Secret dialog state ---
  const [secretDialogOpen, setSecretDialogOpen] = useState(false);
  const [editingSecret, setEditingSecret] = useState<SecretDetail | null>(null);

  // --- Folder dialog state ---
  const [folderDialogOpen, setFolderDialogOpen] = useState(false);
  const [editingFolder, setEditingFolder] = useState<VaultFolderData | null>(null);
  const [folderDialogScope, setFolderDialogScope] = useState<VaultFolderScope>('PERSONAL');
  const [folderDialogParentId, setFolderDialogParentId] = useState<string | null>(null);
  const [folderDialogTeamId, setFolderDialogTeamId] = useState<string | null>(null);

  // --- Share dialog state ---
  const [shareTarget, setShareTarget] = useState<{
    id: string;
    name: string;
    teamId?: string | null;
  } | null>(null);

  // --- Delete confirmation state ---
  const [deleteTarget, setDeleteTarget] = useState<SecretListItem | null>(null);
  const [deleting, setDeleting] = useState(false);

  // --- Handlers ---

  const handleCreateSecret = useCallback(() => {
    setEditingSecret(null);
    setSecretDialogOpen(true);
  }, []);

  const handleEditSecret = useCallback(async (secret: SecretListItem) => {
    try {
      const detail = await getSecret(secret.id);
      setEditingSecret(detail);
    } catch {
      setEditingSecret(null);
    }
    setSecretDialogOpen(true);
  }, []);

  const handleShareSecret = useCallback((secret: SecretListItem) => {
    setShareTarget({ id: secret.id, name: secret.name, teamId: secret.teamId });
  }, []);

  const handleDeleteSecret = useCallback((secret: SecretListItem) => {
    setDeleteTarget(secret);
  }, []);

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteSecretAction(deleteTarget.id);
      setDeleteTarget(null);
    } catch {
      // handled by store
    } finally {
      setDeleting(false);
    }
  };

  const handleCreateFolder = useCallback(
    (scope: VaultFolderScope, parentId?: string, teamId?: string) => {
      setEditingFolder(null);
      setFolderDialogScope(scope);
      setFolderDialogParentId(parentId || null);
      setFolderDialogTeamId(teamId || null);
      setFolderDialogOpen(true);
    },
    [],
  );

  const handleEditFolder = useCallback((folder: VaultFolderData) => {
    setEditingFolder(folder);
    setFolderDialogScope(folder.scope);
    setFolderDialogTeamId(folder.teamId);
    setFolderDialogOpen(true);
  }, []);

  // --- Locked state ---

  if (!vaultUnlocked) {
    return (
      <SidebarGroup>
        <SidebarGroupLabel>
          <KeyRound className="size-4" />
          Vault
        </SidebarGroupLabel>
        <SidebarGroupContent>
          <div className="flex flex-col items-center gap-2 px-2 py-8 text-center">
            <Lock className="size-8 text-muted-foreground/50" />
            <p className="text-xs font-medium text-muted-foreground">
              Vault Locked
            </p>
            <p className="text-[0.7rem] text-muted-foreground/70">
              Unlock your vault to browse secrets
            </p>
          </div>
        </SidebarGroupContent>
      </SidebarGroup>
    );
  }

  // --- Unlocked state ---

  return (
    <>
      {/* Folder tree */}
      <SidebarGroup>
        <SidebarGroupLabel>
          <KeyRound className="size-4" />
          Vault
        </SidebarGroupLabel>
        <SidebarGroupAction asChild>
          <button
            type="button"
            onClick={handleCreateSecret}
            title="New Secret"
          >
            <Plus className="size-4" />
            <span className="sr-only">New Secret</span>
          </button>
        </SidebarGroupAction>
        <SidebarGroupContent>
          <div className="flex flex-col overflow-hidden">
            <SecretTree
              onCreateFolder={handleCreateFolder}
              onEditFolder={handleEditFolder}
            />
          </div>
        </SidebarGroupContent>
      </SidebarGroup>

      {/* Secret list */}
      <SidebarGroup>
        <SidebarGroupContent>
          <SecretListPanel
            onCreateSecret={handleCreateSecret}
            onEditSecret={handleEditSecret}
            onShareSecret={handleShareSecret}
            onDeleteSecret={handleDeleteSecret}
          />
        </SidebarGroupContent>
      </SidebarGroup>

      {/* --- Sub-dialogs --- */}
      <SecretDialog
        open={secretDialogOpen}
        onClose={() => {
          setSecretDialogOpen(false);
          setEditingSecret(null);
        }}
        secret={editingSecret}
      />

      <ShareSecretDialog
        open={!!shareTarget}
        onClose={() => setShareTarget(null)}
        secretId={shareTarget?.id ?? ''}
        secretName={shareTarget?.name ?? ''}
        teamId={shareTarget?.teamId}
      />

      <VaultFolderDialog
        open={folderDialogOpen}
        onClose={() => {
          setFolderDialogOpen(false);
          setEditingFolder(null);
        }}
        folder={editingFolder}
        parentId={folderDialogParentId}
        scope={folderDialogScope}
        teamId={folderDialogTeamId}
      />

      {/* Delete confirmation */}
      <Dialog
        open={!!deleteTarget}
        onOpenChange={(next) => {
          if (!next) setDeleteTarget(null);
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Delete Secret</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.name}&quot;?
              This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDelete}
              disabled={deleting}
            >
              {deleting ? 'Deleting...' : 'Delete'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
