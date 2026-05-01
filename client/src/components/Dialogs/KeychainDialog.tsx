import { useState, useEffect, useCallback } from 'react';
import {
  DndContext,
  DragOverlay,
  pointerWithin,
  useSensor,
  useSensors,
  PointerSensor,
} from '@dnd-kit/core';
import type { DragStartEvent, DragEndEvent } from '@dnd-kit/core';
import {
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
  Info,
  KeyRound,
  X,
} from 'lucide-react';
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Separator } from '@/components/ui/separator';
import SecretTree from '../Keychain/SecretTree';
import SecretListPanel from '../Keychain/SecretListPanel';
import SecretDetailView from '../Keychain/SecretDetailView';
import SecretDialog from '../Keychain/SecretDialog';
import ShareSecretDialog from '../Keychain/ShareSecretDialog';
import ExternalShareDialog from '../Keychain/ExternalShareDialog';
import VaultFolderDialog from '../Keychain/VaultFolderDialog';
import { useSecretStore } from '../../store/secretStore';
import { useAuthStore } from '../../store/authStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import type { SecretListItem, SecretDetail } from '../../api/secrets.api';
import type { VaultFolderData, VaultFolderScope } from '../../api/vault-folders.api';
import { getSecret } from '../../api/secrets.api';
import RecoveryKeyConfirmDialog from '../common/RecoveryKeyConfirmDialog';
import { isAdminOrAbove } from '../../utils/roles';
import { getVaultRecoveryStatus, recoverVaultWithKey, explicitVaultReset } from '../../api/vault.api';
import { useAsyncAction } from '../../hooks/useAsyncAction';

interface KeychainDialogProps {
  open: boolean;
  onClose: () => void;
}

export default function KeychainDialog({ open, onClose }: KeychainDialogProps) {
  const selectedSecret = useSecretStore((s) => s.selectedSecret);
  const fetchSecret = useSecretStore((s) => s.fetchSecret);
  const deleteSecretAction = useSecretStore((s) => s.deleteSecret);
  const toggleFavorite = useSecretStore((s) => s.toggleFavorite);
  const tenantVaultStatus = useSecretStore((s) => s.tenantVaultStatus);
  const fetchTenantVaultStatus = useSecretStore((s) => s.fetchTenantVaultStatus);
  const initTenantVault = useSecretStore((s) => s.initTenantVault);
  const checkSecretBreach = useSecretStore((s) => s.checkSecretBreach);
  const user = useAuthStore((s) => s.user);

  const treeOpen = useUiPreferencesStore((s) => s.keychainTreeOpen);
  const togglePref = useUiPreferencesStore((s) => s.toggle);

  const moveSecret = useSecretStore((s) => s.moveSecret);

  const isAdmin = isAdminOrAbove(user?.tenantRole);
  const hasTenant = !!user?.tenantId;

  const [initializingVault, setInitializingVault] = useState(false);
  const [activeSecretDrag, setActiveSecretDrag] = useState<SecretListItem | null>(null);

  // Vault recovery state
  const [vaultNeedsRecovery, setVaultNeedsRecovery] = useState(false);
  const [vaultHasRecoveryKey, setVaultHasRecoveryKey] = useState(false);
  const [recoveryKeyInput, setRecoveryKeyInput] = useState('');
  const [recoveryPasswordInput, setRecoveryPasswordInput] = useState('');
  const [newRecoveryKey, setNewRecoveryKey] = useState('');
  const [showRecoveryKeyDialog, setShowRecoveryKeyDialog] = useState(false);
  const [resetConfirmOpen, setResetConfirmOpen] = useState(false);
  const [resetConfirmText, setResetConfirmText] = useState('');
  const [resetPasswordInput, setResetPasswordInput] = useState('');
  const recoverAction = useAsyncAction();
  const resetAction = useAsyncAction();

  // DnD sensors
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
  );

  const handleDragStart = (event: DragStartEvent) => {
    const secret = event.active.data.current?.secret as SecretListItem | undefined;
    if (secret) setActiveSecretDrag(secret);
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    setActiveSecretDrag(null);
    const { active, over } = event;
    if (!over) return;

    const secret = active.data.current?.secret as SecretListItem | undefined;
    if (!secret) return;

    const targetFolderId = (over.data.current?.folderId as string | null) ?? null;
    if (targetFolderId === (secret.folderId ?? null)) return;

    await moveSecret(secret.id, targetFolderId);
  };

  // Fetch vault recovery status on open
  const fetchRecoveryStatus = useCallback(async () => {
    try {
      const status = await getVaultRecoveryStatus();
      setVaultNeedsRecovery(status.needsRecovery);
      setVaultHasRecoveryKey(status.hasRecoveryKey);
    } catch {
      // silently fail — non-critical
    }
  }, []);

  useEffect(() => {
    if (open) fetchRecoveryStatus();
  }, [open, fetchRecoveryStatus]);

  useEffect(() => {
    if (open && hasTenant) fetchTenantVaultStatus();
  }, [open, hasTenant, fetchTenantVaultStatus]);

  const handleRecoverVault = async () => {
    await recoverAction.run(async () => {
      const result = await recoverVaultWithKey(recoveryKeyInput, recoveryPasswordInput);
      setNewRecoveryKey(result.newRecoveryKey);
      setShowRecoveryKeyDialog(true);
      setVaultNeedsRecovery(false);
      setRecoveryKeyInput('');
      setRecoveryPasswordInput('');
    }, 'Vault recovery failed');
  };

  const handleExplicitReset = async () => {
    await resetAction.run(async () => {
      const result = await explicitVaultReset(resetPasswordInput);
      setNewRecoveryKey(result.newRecoveryKey);
      setShowRecoveryKeyDialog(true);
      setVaultNeedsRecovery(false);
      setResetConfirmOpen(false);
      setResetConfirmText('');
      setResetPasswordInput('');
    }, 'Vault reset failed');
  };

  const handleInitTenantVault = async () => {
    setInitializingVault(true);
    try {
      await initTenantVault();
    } catch {
      // error handled by store
    } finally {
      setInitializingVault(false);
    }
  };

  // Dialog state
  const [secretDialogOpen, setSecretDialogOpen] = useState(false);
  const [editingSecret, setEditingSecret] = useState<SecretDetail | null>(null);
  const [shareTarget, setShareTarget] = useState<{ id: string; name: string; teamId?: string | null } | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<SecretListItem | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [externalShareTarget, setExternalShareTarget] = useState<{ id: string; name: string } | null>(null);

  // Folder dialog state
  const [folderDialogOpen, setFolderDialogOpen] = useState(false);
  const [editingFolder, setEditingFolder] = useState<VaultFolderData | null>(null);
  const [folderDialogScope, setFolderDialogScope] = useState<VaultFolderScope>('PERSONAL');
  const [folderDialogParentId, setFolderDialogParentId] = useState<string | null>(null);
  const [folderDialogTeamId, setFolderDialogTeamId] = useState<string | null>(null);

  const handleCreateSecret = () => {
    setEditingSecret(null);
    setSecretDialogOpen(true);
  };

  const handleEditSecret = async (secret: SecretListItem) => {
    try {
      const detail = await getSecret(secret.id);
      setEditingSecret(detail);
      setSecretDialogOpen(true);
    } catch {
      setEditingSecret(null);
      setSecretDialogOpen(true);
    }
  };

  const handleShareSecret = (secret: SecretListItem) => {
    setShareTarget({ id: secret.id, name: secret.name, teamId: secret.teamId });
  };

  const handleDeleteSecret = (secret: SecretListItem) => {
    setDeleteTarget(secret);
  };

  const confirmDelete = async () => {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteSecretAction(deleteTarget.id);
      setDeleteTarget(null);
    } catch {
      // silently fail
    } finally {
      setDeleting(false);
    }
  };

  const handleRestore = () => {
    if (selectedSecret) {
      fetchSecret(selectedSecret.id);
    }
  };

  const handleCreateFolder = (scope: VaultFolderScope, parentId?: string, teamId?: string) => {
    setEditingFolder(null);
    setFolderDialogScope(scope);
    setFolderDialogParentId(parentId || null);
    setFolderDialogTeamId(teamId || null);
    setFolderDialogOpen(true);
  };

  const handleEditFolder = (folder: VaultFolderData) => {
    setEditingFolder(folder);
    setFolderDialogScope(folder.scope);
    setFolderDialogTeamId(folder.teamId);
    setFolderDialogOpen(true);
  };

  const closeResetDialog = () => {
    setResetConfirmOpen(false);
    setResetConfirmText('');
    setResetPasswordInput('');
    resetAction.clearError();
  };

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="h-[100dvh] w-screen max-w-none gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        <div className="flex h-full min-h-0 flex-col bg-background">
          {/* ── Header ── */}
          <div className="flex items-center justify-between border-b px-4 py-3">
            <div className="flex items-center gap-2">
              <div className="flex size-7 items-center justify-center rounded-md bg-primary/10 text-primary">
                <KeyRound className="size-3.5" />
              </div>
              <DialogHeader className="gap-0 space-y-0 p-0">
                <DialogTitle className="text-sm font-semibold tracking-tight">
                  Keychain
                </DialogTitle>
                <DialogDescription className="sr-only">
                  Manage secrets, credentials, and vault folders.
                </DialogDescription>
              </DialogHeader>
            </div>
            <DialogClose asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 text-muted-foreground hover:text-foreground"
                aria-label="Close keychain"
              >
                <X className="size-3.5" />
              </Button>
            </DialogClose>
          </div>

          {/* ── Vault recovery banner ── */}
          {vaultNeedsRecovery && (
            <div className="border-b bg-card/40 px-4 py-3">
              <div className="mb-3 flex items-start gap-2 rounded-lg border border-yellow-500/30 bg-yellow-500/5 px-3 py-2.5">
                <AlertTriangle className="mt-0.5 size-4 shrink-0 text-yellow-500" />
                <div>
                  <p className="text-xs font-semibold text-foreground">
                    Vault Recovery Required
                  </p>
                  <p className="mt-0.5 text-xs text-muted-foreground">
                    Your vault is locked after a password reset. Enter your recovery key to restore access.
                  </p>
                </div>
              </div>

              {recoverAction.error && (
                <div className="mb-3 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs text-destructive">
                  {recoverAction.error}
                </div>
              )}
              {resetAction.error && (
                <div className="mb-3 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs text-destructive">
                  {resetAction.error}
                </div>
              )}

              {vaultHasRecoveryKey && (
                <div className="mb-3 flex flex-col gap-2">
                  <Input
                    value={recoveryKeyInput}
                    onChange={(e) => setRecoveryKeyInput(e.target.value.trim())}
                    placeholder="Enter your vault recovery key"
                    className="h-8 text-xs"
                  />
                  <Input
                    type="password"
                    value={recoveryPasswordInput}
                    onChange={(e) => setRecoveryPasswordInput(e.target.value)}
                    placeholder="Enter your current password"
                    className="h-8 text-xs"
                  />
                  <Button
                    size="sm"
                    onClick={handleRecoverVault}
                    disabled={recoverAction.loading || !recoveryKeyInput || !recoveryPasswordInput}
                  >
                    {recoverAction.loading ? 'Recovering...' : 'Recover Vault'}
                  </Button>
                </div>
              )}

              <Separator className="my-3" />

              <Button
                size="sm"
                variant="destructive"
                onClick={() => setResetConfirmOpen(true)}
              >
                Reset Vault (lose all data)
              </Button>
            </div>
          )}

          {/* ── Tenant vault banners ── */}
          {hasTenant && tenantVaultStatus && !tenantVaultStatus.initialized && isAdmin && (
            <div className="flex items-center gap-2 border-b bg-card/40 px-4 py-2.5">
              <Info className="size-4 shrink-0 text-primary" />
              <p className="flex-1 text-xs text-muted-foreground">
                Organization vault is not initialized. Initialize it to create and share organization-scoped secrets.
              </p>
              <Button
                size="sm"
                variant="outline"
                onClick={handleInitTenantVault}
                disabled={initializingVault}
              >
                {initializingVault ? 'Initializing...' : 'Initialize Now'}
              </Button>
            </div>
          )}
          {hasTenant && tenantVaultStatus && tenantVaultStatus.initialized && !tenantVaultStatus.hasAccess && (
            <div className="flex items-center gap-2 border-b bg-yellow-500/5 px-4 py-2.5">
              <AlertTriangle className="size-4 shrink-0 text-yellow-500" />
              <p className="flex-1 text-xs text-muted-foreground">
                You don&apos;t have access to the organization vault yet. Ask an admin to distribute the key to you.
              </p>
            </div>
          )}

          {/* ── Main content — 3-column layout with DnD ── */}
          <DndContext
            sensors={sensors}
            collisionDetection={pointerWithin}
            onDragStart={handleDragStart}
            onDragEnd={handleDragEnd}
          >
            <div className="flex min-h-0 flex-1 overflow-hidden">
              {/* Folder tree panel */}
              {treeOpen && (
                <div className="flex w-[200px] min-w-[200px] flex-col overflow-hidden border-r bg-card/30">
                  <SecretTree
                    onCreateFolder={handleCreateFolder}
                    onEditFolder={handleEditFolder}
                  />
                </div>
              )}

              {/* Tree toggle */}
              <button
                type="button"
                onClick={() => togglePref('keychainTreeOpen')}
                title={treeOpen ? 'Hide folders' : 'Show folders'}
                className="flex w-5 items-center justify-center border-r text-muted-foreground transition-colors hover:bg-accent/40 hover:text-foreground"
              >
                {treeOpen ? (
                  <ChevronLeft className="size-3.5" />
                ) : (
                  <ChevronRight className="size-3.5" />
                )}
              </button>

              {/* Secret list panel */}
              <div className="flex w-[320px] min-w-[320px] flex-col overflow-hidden border-r bg-card/30">
                <SecretListPanel
                  onCreateSecret={handleCreateSecret}
                  onEditSecret={handleEditSecret}
                  onShareSecret={handleShareSecret}
                  onDeleteSecret={handleDeleteSecret}
                />
              </div>

              {/* Detail panel */}
              <div className="flex flex-1 flex-col overflow-auto bg-background">
                {selectedSecret ? (
                  <SecretDetailView
                    secret={selectedSecret}
                    onEdit={() => {
                      setEditingSecret(selectedSecret);
                      setSecretDialogOpen(true);
                    }}
                    onShare={() => setShareTarget({ id: selectedSecret.id, name: selectedSecret.name, teamId: selectedSecret.teamId })}
                    onExternalShare={() => setExternalShareTarget({ id: selectedSecret.id, name: selectedSecret.name })}
                    onDelete={() => setDeleteTarget(selectedSecret)}
                    onToggleFavorite={() => toggleFavorite(selectedSecret.id)}
                    onRestore={handleRestore}
                    onCheckBreach={checkSecretBreach}
                  />
                ) : (
                  <div className="flex flex-1 items-center justify-center">
                    <p className="text-sm text-muted-foreground">
                      Select a secret to view its details
                    </p>
                  </div>
                )}
              </div>
            </div>

            {/* Drag overlay */}
            <DragOverlay dropAnimation={null}>
              {activeSecretDrag && (
                <div className="pointer-events-none flex max-w-[220px] items-center gap-2 rounded-lg border bg-card px-3 py-1.5 opacity-90 shadow-lg">
                  <span className="truncate text-xs">{activeSecretDrag.name}</span>
                </div>
              )}
            </DragOverlay>
          </DndContext>
        </div>

        {/* ── Sub-dialogs ── */}
        <SecretDialog
          open={secretDialogOpen}
          onClose={() => { setSecretDialogOpen(false); setEditingSecret(null); }}
          secret={editingSecret}
        />

        <ShareSecretDialog
          open={!!shareTarget}
          onClose={() => setShareTarget(null)}
          secretId={shareTarget?.id ?? ''}
          secretName={shareTarget?.name ?? ''}
          teamId={shareTarget?.teamId}
        />

        <ExternalShareDialog
          open={!!externalShareTarget}
          onClose={() => setExternalShareTarget(null)}
          secretId={externalShareTarget?.id ?? ''}
          secretName={externalShareTarget?.name ?? ''}
        />

        <VaultFolderDialog
          open={folderDialogOpen}
          onClose={() => { setFolderDialogOpen(false); setEditingFolder(null); }}
          folder={editingFolder}
          parentId={folderDialogParentId}
          scope={folderDialogScope}
          teamId={folderDialogTeamId}
        />

        {/* Delete confirmation — shadcn */}
        <Dialog open={!!deleteTarget} onOpenChange={(next) => { if (!next) setDeleteTarget(null); }}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Delete Secret</DialogTitle>
              <DialogDescription>
                Are you sure you want to delete &quot;{deleteTarget?.name}&quot;? This action cannot be undone.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter className="gap-2 sm:gap-0">
              <Button variant="outline" onClick={() => setDeleteTarget(null)}>
                Cancel
              </Button>
              <Button variant="destructive" onClick={confirmDelete} disabled={deleting}>
                {deleting ? 'Deleting...' : 'Delete'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Vault explicit reset confirmation — shadcn */}
        <Dialog open={resetConfirmOpen} onOpenChange={(next) => { if (!next) closeResetDialog(); }}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Reset Vault</DialogTitle>
              <DialogDescription>
                This will permanently delete all your saved credentials, secrets, and encrypted data. This action cannot be undone.
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-3 py-2">
              {resetAction.error && (
                <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs text-destructive">
                  {resetAction.error}
                </div>
              )}
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-foreground">Current Password</label>
                <Input
                  type="password"
                  value={resetPasswordInput}
                  onChange={(e) => setResetPasswordInput(e.target.value)}
                  className="h-8 text-xs"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-foreground">
                  Type <strong className="text-destructive">RESET</strong> to confirm
                </label>
                <Input
                  value={resetConfirmText}
                  onChange={(e) => setResetConfirmText(e.target.value)}
                  placeholder="RESET"
                  className="h-8 text-xs"
                />
              </div>
            </div>
            <DialogFooter className="gap-2 sm:gap-0">
              <Button variant="outline" onClick={closeResetDialog}>
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={handleExplicitReset}
                disabled={resetConfirmText !== 'RESET' || !resetPasswordInput || resetAction.loading}
              >
                {resetAction.loading ? 'Resetting...' : 'Reset Vault'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Recovery key display after successful recovery/reset */}
        <RecoveryKeyConfirmDialog
          open={showRecoveryKeyDialog}
          recoveryKey={newRecoveryKey}
          onConfirmed={() => { setShowRecoveryKeyDialog(false); setNewRecoveryKey(''); }}
        />
      </DialogContent>
    </Dialog>
  );
}
