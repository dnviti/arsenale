import { useState, useEffect, useRef, useMemo, useCallback } from 'react';
import {
  KeyRound, Key, ShieldCheck, Webhook, StickyNote,
  Plus, Search, Star, Pencil, Share2, Trash2, Copy,
  FolderInput, ShieldAlert,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from '@/components/ui/dialog';
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';
import { useDraggable } from '@dnd-kit/core';
import { CSS } from '@dnd-kit/utilities';
import { useSecretStore } from '../../store/secretStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import type { SecretListItem, SecretType, SecretScope } from '../../api/secrets.api';
import type { VaultFolderData } from '../../api/vault-folders.api';

const TYPE_ICONS: Record<SecretType, React.ReactNode> = {
  LOGIN: <KeyRound className="h-4 w-4" />,
  SSH_KEY: <Key className="h-4 w-4" />,
  CERTIFICATE: <ShieldCheck className="h-4 w-4" />,
  API_KEY: <Webhook className="h-4 w-4" />,
  SECURE_NOTE: <StickyNote className="h-4 w-4" />,
};

const TYPE_LABELS: Record<SecretType, string> = {
  LOGIN: 'Login',
  SSH_KEY: 'SSH Key',
  CERTIFICATE: 'Certificate',
  API_KEY: 'API Key',
  SECURE_NOTE: 'Secure Note',
};

const SCOPE_COLORS: Record<SecretScope, string> = {
  PERSONAL: 'secondary',
  TEAM: 'default',
  TENANT: 'outline',
};

// --- Draggable secret item ---

function DraggableSecretItem({
  secret,
  isSelected,
  daysUntilExpiry,
  onSelect,
  onContextMenu,
  onToggleFavorite,
}: {
  secret: SecretListItem;
  isSelected: boolean;
  daysUntilExpiry: number | null;
  onSelect: () => void;
  onContextMenu: (e: React.MouseEvent) => void;
  onToggleFavorite: () => void;
}) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    isDragging,
  } = useDraggable({
    id: `secret-${secret.id}`,
    data: { type: 'secret', secret },
  });

  return (
    <div
      ref={setNodeRef}
      onClick={onSelect}
      onContextMenu={onContextMenu}
      className={cn(
        'flex items-center gap-2 px-3 py-2 cursor-grab select-none rounded-md transition-colors',
        isSelected && 'bg-accent',
        !isSelected && 'hover:bg-accent/50',
        isDragging && 'opacity-40',
      )}
      style={transform ? { transform: CSS.Translate.toString(transform) } : undefined}
      {...listeners}
      {...attributes}
    >
      <div className="shrink-0">
        {TYPE_ICONS[secret.type]}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1">
          <span className="text-sm truncate flex-1">{secret.name}</span>
          <Badge
            variant={SCOPE_COLORS[secret.scope] as 'default' | 'secondary' | 'outline'}
            className="text-[0.65rem] px-1.5 py-0 h-[18px] shrink-0"
          >
            {secret.scope === 'PERSONAL' ? 'Me' : secret.scope === 'TEAM' ? 'Team' : 'Org'}
          </Badge>
        </div>
        <div className="flex items-center gap-1">
          <span className="text-xs text-muted-foreground truncate">
            {TYPE_LABELS[secret.type]}
          </span>
          {secret.pwnedCount > 0 && (
            <Badge variant="destructive" className="text-[0.6rem] px-1 py-0 h-4 gap-0.5">
              <ShieldAlert className="h-2.5 w-2.5" />
              Breached
            </Badge>
          )}
          {daysUntilExpiry !== null && daysUntilExpiry <= 30 && (
            <Badge
              variant={daysUntilExpiry <= 7 ? 'destructive' : 'secondary'}
              className="text-[0.6rem] px-1 py-0 h-4"
            >
              {daysUntilExpiry <= 0 ? 'Expired' : `${daysUntilExpiry}d left`}
            </Badge>
          )}
        </div>
      </div>
      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 shrink-0"
        onClick={(e) => { e.stopPropagation(); onToggleFavorite(); }}
      >
        <Star className={cn('h-4 w-4', secret.isFavorite && 'fill-yellow-500 text-yellow-500')} />
      </Button>
    </div>
  );
}

// --- Main panel ---

interface SecretListPanelProps {
  onCreateSecret: () => void;
  onEditSecret: (secret: SecretListItem) => void;
  onShareSecret: (secret: SecretListItem) => void;
  onDeleteSecret: (secret: SecretListItem) => void;
}

export default function SecretListPanel({
  onCreateSecret,
  onEditSecret,
  onShareSecret,
  onDeleteSecret,
}: SecretListPanelProps) {
  const secrets = useSecretStore((s) => s.secrets);
  const selectedSecret = useSecretStore((s) => s.selectedSecret);
  const fetchSecret = useSecretStore((s) => s.fetchSecret);
  const fetchSecrets = useSecretStore((s) => s.fetchSecrets);
  const toggleFavorite = useSecretStore((s) => s.toggleFavorite);
  const setFilters = useSecretStore((s) => s.setFilters);
  const selectedFolderId = useSecretStore((s) => s.selectedFolderId);
  const moveSecret = useSecretStore((s) => s.moveSecret);
  const vaultFolders = useSecretStore((s) => s.vaultFolders);
  const vaultTeamFolders = useSecretStore((s) => s.vaultTeamFolders);
  const vaultTenantFolders = useSecretStore((s) => s.vaultTenantFolders);

  const scopeFilter = useUiPreferencesStore((s) => s.keychainScopeFilter);
  const typeFilter = useUiPreferencesStore((s) => s.keychainTypeFilter);
  const setPref = useUiPreferencesStore((s) => s.set);

  const [search, setSearch] = useState('');
  const searchTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Stable time reference for expiry calculations (avoids Date.now() in render)
  // eslint-disable-next-line react-hooks/exhaustive-deps -- secrets used intentionally to refresh timestamp when list changes
  const now = useMemo(() => new Date().getTime(), [secrets.length]);

  // Context menu
  const [contextMenu, setContextMenu] = useState<{
    mouseX: number;
    mouseY: number;
    secret: SecretListItem;
  } | null>(null);

  // Move to folder dialog
  const [moveTarget, setMoveTarget] = useState<SecretListItem | null>(null);
  const [moveDestination, setMoveDestination] = useState('');

  const allFolders = useMemo<VaultFolderData[]>(() => [
    ...vaultFolders,
    ...vaultTeamFolders,
    ...vaultTenantFolders,
  ], [vaultFolders, vaultTeamFolders, vaultTenantFolders]);

  useEffect(() => {
    fetchSecrets();
  }, [fetchSecrets]);

  useEffect(() => {
    if (searchTimer.current) clearTimeout(searchTimer.current);
    searchTimer.current = setTimeout(() => {
      setFilters({ search: search || undefined });
    }, 300);
    return () => { if (searchTimer.current) clearTimeout(searchTimer.current); };
  }, [search, setFilters]);

  const handleScopeChange = (value: string) => {
    setPref('keychainScopeFilter', value);
    setFilters({ scope: value === 'ALL' ? undefined : value as SecretScope });
  };

  const handleTypeChange = (value: string) => {
    setPref('keychainTypeFilter', value);
    setFilters({ type: value === 'ALL' ? undefined : value as SecretType });
  };

  const handleContextMenu = (e: React.MouseEvent, secret: SecretListItem) => {
    e.preventDefault();
    setContextMenu({ mouseX: e.clientX, mouseY: e.clientY, secret });
  };

  const getDaysUntilExpiry = useCallback((expiresAt: string): number => {
    const diff = new Date(expiresAt).getTime() - now;
    return Math.ceil(diff / (1000 * 60 * 60 * 24));
  }, [now]);

  const handleMoveToFolder = (secret: SecretListItem) => {
    setMoveTarget(secret);
    setMoveDestination(secret.folderId || '');
  };

  const handleConfirmMove = async () => {
    if (!moveTarget) return;
    const newFolderId = moveDestination || null;
    if (newFolderId === (moveTarget.folderId ?? null)) {
      setMoveTarget(null);
      return;
    }
    await moveSecret(moveTarget.id, newFolderId);
    setMoveTarget(null);
  };

  // Current folder name for breadcrumb
  const currentFolderName = useMemo(() => {
    if (!selectedFolderId) return null;
    const folder = allFolders.find((f) => f.id === selectedFolderId);
    return folder?.name ?? null;
  }, [selectedFolderId, allFolders]);

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between p-3 pb-2">
        <h3 className="text-lg font-semibold truncate">
          {currentFolderName ? currentFolderName : 'Secrets'}
        </h3>
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onCreateSecret} title="New Secret">
          <Plus className="h-4 w-4" />
        </Button>
      </div>

      {/* Search */}
      <div className="px-3 pb-2">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search secrets..."
            className="pl-8"
          />
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-2 px-3 pb-2">
        <Select value={scopeFilter} onValueChange={handleScopeChange}>
          <SelectTrigger className="flex-1 h-9"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="ALL">All</SelectItem>
            <SelectItem value="PERSONAL">Personal</SelectItem>
            <SelectItem value="TEAM">Team</SelectItem>
            <SelectItem value="TENANT">Organization</SelectItem>
          </SelectContent>
        </Select>
        <Select value={typeFilter} onValueChange={handleTypeChange}>
          <SelectTrigger className="flex-1 h-9"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="ALL">All</SelectItem>
            <SelectItem value="LOGIN">Login</SelectItem>
            <SelectItem value="SSH_KEY">SSH Key</SelectItem>
            <SelectItem value="CERTIFICATE">Certificate</SelectItem>
            <SelectItem value="API_KEY">API Key</SelectItem>
            <SelectItem value="SECURE_NOTE">Secure Note</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Secret list */}
      <div className="flex-1 overflow-auto">
        {secrets.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8">
            No secrets found
          </p>
        ) : (
          <div className="space-y-0.5 px-1">
            {secrets.map((secret) => {
              const daysUntilExpiry = secret.expiresAt ? getDaysUntilExpiry(secret.expiresAt) : null;
              return (
                <DraggableSecretItem
                  key={secret.id}
                  secret={secret}
                  isSelected={selectedSecret?.id === secret.id}
                  daysUntilExpiry={daysUntilExpiry}
                  onSelect={() => fetchSecret(secret.id)}
                  onContextMenu={(e) => handleContextMenu(e, secret)}
                  onToggleFavorite={() => toggleFavorite(secret.id)}
                />
              );
            })}
          </div>
        )}
      </div>

      {/* Context menu */}
      {contextMenu && (
        <DropdownMenu open onOpenChange={() => setContextMenu(null)}>
          <DropdownMenuTrigger asChild>
            <div
              className="fixed"
              style={{ left: contextMenu.mouseX, top: contextMenu.mouseY, width: 0, height: 0 }}
            />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuItem onClick={() => { onEditSecret(contextMenu.secret); setContextMenu(null); }}>
              <Pencil className="h-4 w-4 mr-2" /> Edit
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => { onShareSecret(contextMenu.secret); setContextMenu(null); }}>
              <Share2 className="h-4 w-4 mr-2" /> Share
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => { handleMoveToFolder(contextMenu.secret); setContextMenu(null); }}>
              <FolderInput className="h-4 w-4 mr-2" /> Move to Folder
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => { toggleFavorite(contextMenu.secret.id); setContextMenu(null); }}>
              <Star className="h-4 w-4 mr-2" /> Toggle Favorite
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => { navigator.clipboard.writeText(contextMenu.secret.name); setContextMenu(null); }}>
              <Copy className="h-4 w-4 mr-2" /> Copy Name
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-destructive" onClick={() => { onDeleteSecret(contextMenu.secret); setContextMenu(null); }}>
              <Trash2 className="h-4 w-4 mr-2" /> Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}

      {/* Move to Folder dialog */}
      <Dialog open={!!moveTarget} onOpenChange={(v) => { if (!v) setMoveTarget(null); }}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>Move &quot;{moveTarget?.name}&quot;</DialogTitle>
            <DialogDescription className="sr-only">Select a destination folder</DialogDescription>
          </DialogHeader>
          <div className="space-y-1.5">
            <Label>Destination Folder</Label>
            <Select value={moveDestination || '__root__'} onValueChange={(v) => setMoveDestination(v === '__root__' ? '' : v)}>
              <SelectTrigger><SelectValue placeholder="Root (no folder)" /></SelectTrigger>
              <SelectContent>
                <SelectItem value="__root__">Root (no folder)</SelectItem>
                {allFolders.map((f) => (
                  <SelectItem key={f.id} value={f.id}>{f.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setMoveTarget(null)}>Cancel</Button>
            <Button onClick={handleConfirmMove}>Move</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
