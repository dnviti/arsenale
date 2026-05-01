import { useState, useMemo, useEffect, useRef } from 'react';
import {
  Folder, FolderOpen, ChevronDown, ChevronRight, Star,
  FolderPlus, Pencil, Trash2, Inbox,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from '@/components/ui/dialog';
import {
  DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';
import { useDroppable } from '@dnd-kit/core';
import { useSecretStore } from '../../store/secretStore';
import { useAuthStore } from '../../store/authStore';
import { useTeamStore } from '../../store/teamStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import { deleteVaultFolder } from '../../api/vault-folders.api';
import type { VaultFolderData } from '../../api/vault-folders.api';
import { extractApiError } from '../../utils/apiError';

// --- Tree building ---

interface FolderNode {
  folder: VaultFolderData;
  children: FolderNode[];
}

function buildFolderTree(folders: VaultFolderData[]): FolderNode[] {
  const map = new Map<string, FolderNode>();
  for (const f of folders) {
    map.set(f.id, { folder: f, children: [] });
  }
  const roots: FolderNode[] = [];
  for (const node of map.values()) {
    const pid = node.folder.parentId;
    if (pid && map.has(pid)) {
      map.get(pid)?.children.push(node);
    } else {
      roots.push(node);
    }
  }
  return roots;
}

// --- Props ---

interface SecretTreeProps {
  onCreateFolder: (scope: 'PERSONAL' | 'TEAM' | 'TENANT', parentId?: string, teamId?: string) => void;
  onEditFolder: (folder: VaultFolderData) => void;
}

// --- Droppable folder item ---

function FolderTreeItem({
  node,
  depth,
  selectedFolderId,
  onSelect,
  onContextMenu,
}: {
  node: FolderNode;
  depth: number;
  selectedFolderId: string | null;
  onSelect: (folderId: string) => void;
  onContextMenu: (e: React.MouseEvent, folder: VaultFolderData) => void;
}) {
  const expandState = useUiPreferencesStore((s) => s.keychainFolderExpandState);
  const toggleFolder = useUiPreferencesStore((s) => s.toggleKeychainFolder);
  const isOpen = expandState[node.folder.id] ?? true;

  const { setNodeRef, isOver } = useDroppable({
    id: `vault-folder-${node.folder.id}`,
    data: { type: 'vault-folder', folderId: node.folder.id },
  });

  // Auto-expand collapsed folders on drag hover after 500ms
  const dragOverTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(() => {
    if (isOver && !isOpen) {
      dragOverTimerRef.current = setTimeout(() => toggleFolder(node.folder.id), 500);
    }
    return () => {
      if (dragOverTimerRef.current) clearTimeout(dragOverTimerRef.current);
    };
  }, [isOver, isOpen, node.folder.id, toggleFolder]);

  const isSelected = selectedFolderId === node.folder.id;

  return (
    <>
      <div
        ref={setNodeRef}
        onClick={() => onSelect(node.folder.id)}
        onContextMenu={(e) => onContextMenu(e, node.folder)}
        className={cn(
          'flex items-center py-1 cursor-pointer rounded-md transition-colors text-sm',
          isSelected && 'bg-accent',
          !isSelected && 'hover:bg-accent/50',
          isOver && 'bg-accent/70 border-l-2 border-primary',
        )}
        style={{ paddingLeft: `${6 + depth * 16}px` }}
      >
        <div className="w-5 shrink-0 flex items-center justify-center">
          {node.children.length > 0 ? (
            <button
              className="p-0"
              onClick={(e) => { e.stopPropagation(); toggleFolder(node.folder.id); }}
            >
              {isOpen ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
            </button>
          ) : null}
        </div>
        <div className="w-5 shrink-0 flex items-center justify-center mr-1">
          {isOpen && node.children.length > 0 ? (
            <FolderOpen className="h-4 w-4" />
          ) : (
            <Folder className="h-4 w-4" />
          )}
        </div>
        <span className="truncate text-xs">{node.folder.name}</span>
      </div>

      {node.children.length > 0 && isOpen && (
        <div>
          {node.children.map((child) => (
            <FolderTreeItem
              key={child.folder.id}
              node={child}
              depth={depth + 1}
              selectedFolderId={selectedFolderId}
              onSelect={onSelect}
              onContextMenu={onContextMenu}
            />
          ))}
        </div>
      )}
    </>
  );
}

// --- Main SecretTree ---

export default function SecretTree({ onCreateFolder, onEditFolder }: SecretTreeProps) {
  const vaultFolders = useSecretStore((s) => s.vaultFolders);
  const vaultTeamFolders = useSecretStore((s) => s.vaultTeamFolders);
  const vaultTenantFolders = useSecretStore((s) => s.vaultTenantFolders);
  const selectedFolderId = useSecretStore((s) => s.selectedFolderId);
  const setSelectedFolderId = useSecretStore((s) => s.setSelectedFolderId);
  const secrets = useSecretStore((s) => s.secrets);
  const fetchVaultFolders = useSecretStore((s) => s.fetchVaultFolders);
  const fetchSecrets = useSecretStore((s) => s.fetchSecrets);

  const user = useAuthStore((s) => s.user);
  const teams = useTeamStore((s) => s.teams);
  const hasTenant = !!user?.tenantId;

  const [deleteFolderTarget, setDeleteFolderTarget] = useState<VaultFolderData | null>(null);
  const [contextMenu, setContextMenu] = useState<{
    mouseX: number; mouseY: number; folder: VaultFolderData;
  } | null>(null);

  // Special filter states
  const [favoritesSelected, setFavoritesSelected] = useState(false);
  const [activeScopeFilter, setActiveScopeFilter] = useState<string | null>(null);

  useEffect(() => {
    fetchVaultFolders();
  }, [fetchVaultFolders]);

  const personalTree = useMemo(() => buildFolderTree(vaultFolders), [vaultFolders]);

  // Group team folders by teamId, ensuring all user teams appear even with 0 folders
  const teamGroups = useMemo(() => {
    const groups = new Map<string, { teamName: string; folders: VaultFolderData[] }>();
    // Seed with all teams the user belongs to
    for (const t of teams) {
      groups.set(t.id, { teamName: t.name, folders: [] });
    }
    // Merge in actual folder data
    for (const f of vaultTeamFolders) {
      if (!f.teamId) continue;
      if (!groups.has(f.teamId)) {
        groups.set(f.teamId, { teamName: f.teamName || 'Unknown Team', folders: [] });
      }
      groups.get(f.teamId)?.folders.push(f);
    }
    return Array.from(groups.entries()).map(([teamId, g]) => ({
      teamId,
      teamName: g.teamName,
      tree: buildFolderTree(g.folders),
    }));
  }, [vaultTeamFolders, teams]);

  const tenantTree = useMemo(() => buildFolderTree(vaultTenantFolders), [vaultTenantFolders]);

  // Root droppable (move secret to no folder)
  const { setNodeRef: rootDropRef, isOver: isOverRoot } = useDroppable({
    id: 'vault-root-drop-zone',
    data: { type: 'vault-root', folderId: null },
  });

  const handleSelectFolder = (folderId: string) => {
    setFavoritesSelected(false);
    setActiveScopeFilter(null);
    useSecretStore.getState().setFilters({ scope: undefined, isFavorite: undefined });
    setSelectedFolderId(folderId === selectedFolderId ? null : folderId);
  };

  const handleSelectScope = (scope: 'PERSONAL' | 'TEAM' | 'TENANT') => {
    setFavoritesSelected(false);
    setSelectedFolderId(null);
    setActiveScopeFilter(scope);
    useSecretStore.getState().setFilters({ scope, isFavorite: undefined });
  };

  const handleSelectAll = () => {
    setFavoritesSelected(false);
    setActiveScopeFilter(null);
    setSelectedFolderId(null);
    // Clear isFavorite and scope filters
    useSecretStore.getState().setFilters({ isFavorite: undefined, scope: undefined });
  };

  const handleSelectFavorites = () => {
    setFavoritesSelected(true);
    setActiveScopeFilter(null);
    setSelectedFolderId(null);
    useSecretStore.getState().setFilters({ isFavorite: true, scope: undefined });
  };

  // Reset isFavorite filter when not in favorites mode
  useEffect(() => {
    if (!favoritesSelected) {
      const { filters } = useSecretStore.getState();
      if (filters.isFavorite !== undefined) {
        useSecretStore.getState().setFilters({ isFavorite: undefined });
      }
    }
  }, [favoritesSelected]);

  const handleContextMenu = (e: React.MouseEvent, folder: VaultFolderData) => {
    e.preventDefault();
    e.stopPropagation();
    setContextMenu({ mouseX: e.clientX, mouseY: e.clientY, folder });
  };

  const handleDeleteFolder = async () => {
    if (!deleteFolderTarget) return;
    try {
      await deleteVaultFolder(deleteFolderTarget.id);
      if (selectedFolderId === deleteFolderTarget.id) {
        setSelectedFolderId(null);
      }
      await fetchVaultFolders();
      await fetchSecrets();
    } catch (err) {
      console.error(extractApiError(err, 'Failed to delete folder'));
    }
    setDeleteFolderTarget(null);
  };

  const hasFavorites = secrets.some((s) => s.isFavorite);

  return (
    <div className="flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-2">
        <span className="text-sm font-medium">Folders</span>
        <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => onCreateFolder('PERSONAL')} title="New Folder">
          <FolderPlus className="h-4 w-4" />
        </Button>
      </div>

      {/* Tree content */}
      <div className="flex-1 overflow-auto">
        {/* All Secrets */}
        <div
          ref={rootDropRef}
          onClick={handleSelectAll}
          className={cn(
            'flex items-center py-1 px-3 cursor-pointer rounded-md transition-colors text-sm',
            selectedFolderId === null && !favoritesSelected && !activeScopeFilter && 'bg-accent',
            (selectedFolderId !== null || favoritesSelected || activeScopeFilter) && 'hover:bg-accent/50',
            isOverRoot && 'bg-accent/70 border-l-2 border-primary',
          )}
        >
          <div className="w-5 shrink-0" />
          <Inbox className="h-4 w-4 mr-1 shrink-0" />
          <span className="text-xs">All Secrets</span>
        </div>

        {/* Favorites */}
        {hasFavorites && (
          <div
            onClick={handleSelectFavorites}
            className={cn(
              'flex items-center py-1 px-3 cursor-pointer rounded-md transition-colors text-sm',
              favoritesSelected && 'bg-accent',
              !favoritesSelected && 'hover:bg-accent/50',
            )}
          >
            <div className="w-5 shrink-0" />
            <Star className="h-4 w-4 mr-1 shrink-0 text-yellow-500" />
            <span className="text-xs">Favorites</span>
          </div>
        )}

        {/* Personal folders */}
        <Separator className="my-1" />
        <span
          className={cn(
            'px-3 py-0.5 text-xs block cursor-pointer transition-colors hover:text-primary',
            activeScopeFilter === 'PERSONAL' ? 'text-primary font-bold' : 'text-muted-foreground',
          )}
          onClick={() => handleSelectScope('PERSONAL')}
        >
          Personal
        </span>
        {personalTree.length > 0 && (
          <div>
            {personalTree.map((node) => (
              <FolderTreeItem
                key={node.folder.id}
                node={node}
                depth={0}
                selectedFolderId={selectedFolderId}
                onSelect={handleSelectFolder}
                onContextMenu={handleContextMenu}
              />
            ))}
          </div>
        )}

        {/* Tenant folders */}
        {hasTenant && (
          <>
            <Separator className="my-1" />
            <span
              className={cn(
                'px-3 py-0.5 text-xs block cursor-pointer transition-colors hover:text-primary',
                activeScopeFilter === 'TENANT' ? 'text-primary font-bold' : 'text-muted-foreground',
              )}
              onClick={() => handleSelectScope('TENANT')}
            >
              Organization
            </span>
            {tenantTree.length > 0 && (
              <div>
                {tenantTree.map((node) => (
                  <FolderTreeItem
                    key={node.folder.id}
                    node={node}
                    depth={0}
                    selectedFolderId={selectedFolderId}
                    onSelect={handleSelectFolder}
                    onContextMenu={handleContextMenu}
                  />
                ))}
              </div>
            )}
          </>
        )}

        {/* Team folders */}
        {teamGroups.map((group) => (
          <div key={group.teamId}>
            <Separator className="my-1" />
            <span
              className={cn(
                'px-3 py-0.5 text-xs block cursor-pointer transition-colors hover:text-primary',
                activeScopeFilter === group.teamId ? 'text-primary font-bold' : 'text-muted-foreground',
              )}
              onClick={() => {
                setFavoritesSelected(false);
                setSelectedFolderId(null);
                setActiveScopeFilter(group.teamId);
                useSecretStore.getState().setFilters({ scope: 'TEAM', isFavorite: undefined });
              }}
            >
              {group.teamName}
            </span>
            {group.tree.length > 0 && (
              <div>
                {group.tree.map((node) => (
                  <FolderTreeItem
                    key={node.folder.id}
                    node={node}
                    depth={0}
                    selectedFolderId={selectedFolderId}
                    onSelect={handleSelectFolder}
                    onContextMenu={handleContextMenu}
                  />
                ))}
              </div>
            )}
          </div>
        ))}
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
            <DropdownMenuItem onClick={() => {
              onCreateFolder(contextMenu.folder.scope, contextMenu.folder.id, contextMenu.folder.teamId ?? undefined);
              setContextMenu(null);
            }}>
              <FolderPlus className="h-4 w-4 mr-2" /> New Subfolder
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={() => {
              onEditFolder(contextMenu.folder);
              setContextMenu(null);
            }}>
              <Pencil className="h-4 w-4 mr-2" /> Rename
            </DropdownMenuItem>
            <DropdownMenuItem className="text-destructive" onClick={() => {
              setDeleteFolderTarget(contextMenu.folder);
              setContextMenu(null);
            }}>
              <Trash2 className="h-4 w-4 mr-2" /> Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      )}

      {/* Delete folder confirmation */}
      <Dialog open={!!deleteFolderTarget} onOpenChange={(v) => { if (!v) setDeleteFolderTarget(null); }}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>Delete Folder</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteFolderTarget?.name}&quot;?
              Secrets in this folder will be moved to the parent folder.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteFolderTarget(null)}>Cancel</Button>
            <Button variant="destructive" onClick={handleDeleteFolder}>Delete</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
