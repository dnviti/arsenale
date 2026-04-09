import { useMemo, useState } from 'react';
import {
  DatabaseZap,
  FolderPlus,
  Monitor,
  Plus,
  SearchX,
  Share2,
  Star,
  TerminalSquare,
  TimerReset,
} from 'lucide-react';
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  pointerWithin,
  useDroppable,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import type { DragEndEvent, DragStartEvent } from '@dnd-kit/core';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import type { ConnectionData } from '@/api/connections.api';
import { deleteConnection, updateConnection } from '@/api/connections.api';
import { deleteFolder } from '@/api/folders.api';
import { useAuthStore } from '@/store/authStore';
import { useConnectionsStore, type Folder } from '@/store/connectionsStore';
import { useNotificationStore } from '@/store/notificationStore';
import { useTabsStore } from '@/store/tabsStore';
import { useUiPreferencesStore } from '@/store/uiPreferencesStore';
import { extractApiError } from '@/utils/apiError';
import { getRecentConnectionIds } from '@/utils/recentConnections';
import TeamConnectionSection from './TeamConnectionSection';
import {
  buildFolderTree,
  collectFolderConnections,
  ConnectionItem,
  FolderItem,
  folderHasSubfolders,
  matchesSearch,
  pruneFolderTree,
} from './treeHelpers';
import {
  SidebarConfirmDialog,
  SidebarIconButton,
  SidebarSearchInput,
  SidebarSectionHeader,
} from './sidebarUi';

interface ConnectionTreeProps {
  typeFilter?: ConnectionData['type'][];
  onEditConnection: (conn: ConnectionData) => void;
  onShareConnection: (conn: ConnectionData) => void;
  onConnectAsConnection: (conn: ConnectionData) => void;
  onCreateConnection: (folderId?: string, teamId?: string) => void;
  onCreateFolder: (parentId?: string, teamId?: string) => void;
  onEditFolder: (folder: Folder) => void;
  onShareFolder: (folderId: string, folderName: string) => void;
  onViewAuditLog?: (conn: ConnectionData) => void;
}

function dragIcon(type: ConnectionData['type']) {
  if (type === 'SSH') {
    return <TerminalSquare className="size-4 text-muted-foreground" />;
  }
  if (type === 'DATABASE') {
    return <DatabaseZap className="size-4 text-muted-foreground" />;
  }
  return <Monitor className="size-4 text-muted-foreground" />;
}

export default function ConnectionTree({
  typeFilter,
  onEditConnection,
  onShareConnection,
  onConnectAsConnection,
  onCreateConnection,
  onCreateFolder,
  onEditFolder,
  onShareFolder,
  onViewAuditLog,
}: ConnectionTreeProps) {
  const rawOwnConnections = useConnectionsStore((state) => state.ownConnections);
  const rawSharedConnections = useConnectionsStore((state) => state.sharedConnections);
  const rawTeamConnections = useConnectionsStore((state) => state.teamConnections);

  // Apply type filter when provided
  const typeSet = useMemo(() => typeFilter ? new Set(typeFilter) : null, [typeFilter]);
  const ownConnections = useMemo(() => typeSet ? rawOwnConnections.filter((c) => typeSet.has(c.type)) : rawOwnConnections, [rawOwnConnections, typeSet]);
  const sharedConnections = useMemo(() => typeSet ? rawSharedConnections.filter((c) => typeSet.has(c.type)) : rawSharedConnections, [rawSharedConnections, typeSet]);
  const teamConnections = useMemo(() => typeSet ? rawTeamConnections.filter((c) => typeSet.has(c.type)) : rawTeamConnections, [rawTeamConnections, typeSet]);
  const folders = useConnectionsStore((state) => state.folders);
  const teamFolders = useConnectionsStore((state) => state.teamFolders);
  const fetchConnections = useConnectionsStore((state) => state.fetchConnections);
  const toggleFavorite = useConnectionsStore((state) => state.toggleFavorite);
  const moveConnection = useConnectionsStore((state) => state.moveConnection);
  const userId = useAuthStore((state) => state.user?.id);
  const recentTick = useTabsStore((state) => state.recentTick);
  const notify = useNotificationStore((state) => state.notify);
  const openTab = useTabsStore((state) => state.openTab);
  const favoritesOpen = useUiPreferencesStore((state) => state.sidebarFavoritesOpen);
  const recentsOpen = useUiPreferencesStore((state) => state.sidebarRecentsOpen);
  const sharedOpen = useUiPreferencesStore((state) => state.sidebarSharedOpen);
  const compact = useUiPreferencesStore((state) => state.sidebarCompact);
  const togglePreference = useUiPreferencesStore((state) => state.toggle);

  const [searchQuery, setSearchQuery] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<ConnectionData | null>(null);
  const [deleteFolderTarget, setDeleteFolderTarget] = useState<Folder | null>(null);
  const [moveTarget, setMoveTarget] = useState<ConnectionData | null>(null);
  const [moveDestination, setMoveDestination] = useState('');
  const [activeConnection, setActiveConnection] = useState<ConnectionData | null>(null);
  const [bulkOpenTarget, setBulkOpenTarget] = useState<{
    folderId: string;
    connections: ConnectionData[];
  } | null>(null);
  const [bulkOpenSubfolderPrompt, setBulkOpenSubfolderPrompt] = useState<{
    folderId: string;
    thisOnly: number;
    withSubs: number;
  } | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
  );

  const { setNodeRef: rootDropRef, isOver: isOverRoot } = useDroppable({
    id: 'root-drop-zone',
    data: { type: 'root', folderId: null },
  });

  const allFolders = useMemo(() => [...folders, ...teamFolders], [folders, teamFolders]);

  const { filteredRootConnections, filteredFolderMap, filteredFolderTree, filteredSharedConnections } =
    useMemo(() => {
      const isSearching = searchQuery.trim().length > 0;
      const filteredOwn = isSearching
        ? ownConnections.filter((connection) => matchesSearch(connection, searchQuery))
        : ownConnections;
      const filteredShared = isSearching
        ? sharedConnections.filter((connection) => matchesSearch(connection, searchQuery))
        : sharedConnections;

      const rootConnections = filteredOwn.filter((connection) => !connection.folderId);
      const folderMap = new Map<string, ConnectionData[]>();
      filteredOwn.forEach((connection) => {
        if (!connection.folderId) {
          return;
        }
        const group = folderMap.get(connection.folderId) || [];
        group.push(connection);
        folderMap.set(connection.folderId, group);
      });

      const fullTree = buildFolderTree(folders);
      const filteredTree = isSearching ? pruneFolderTree(fullTree, folderMap) : fullTree;

      return {
        filteredRootConnections: rootConnections,
        filteredFolderMap: folderMap,
        filteredFolderTree: filteredTree,
        filteredSharedConnections: filteredShared,
      };
    }, [folders, ownConnections, searchQuery, sharedConnections]);

  const favoriteConnections = useMemo(
    () => ownConnections.filter((connection) => connection.isFavorite),
    [ownConnections],
  );

  const recentConnections = useMemo(() => {
    if (!userId) {
      return [];
    }
    const recentIds = getRecentConnectionIds(userId);
    const availableConnections = [...ownConnections, ...sharedConnections];
    const connectionMap = new Map(availableConnections.map((connection) => [connection.id, connection]));
    return recentIds
      .map((id) => connectionMap.get(id))
      .filter((connection): connection is ConnectionData => connection !== undefined)
      .slice(0, 5);
  }, [ownConnections, recentTick, sharedConnections, userId]);

  const teamGroups = useMemo(() => {
    const groups = new Map<string, {
      teamId: string;
      teamName: string;
      teamRole: string;
      connections: ConnectionData[];
      folders: Folder[];
    }>();

    for (const connection of teamConnections) {
      if (!connection.teamId) {
        continue;
      }
      if (!groups.has(connection.teamId)) {
        groups.set(connection.teamId, {
          teamId: connection.teamId,
          teamName: connection.teamName || 'Unknown Team',
          teamRole: connection.teamRole || 'TEAM_VIEWER',
          connections: [],
          folders: [],
        });
      }
      groups.get(connection.teamId)?.connections.push(connection);
    }

    for (const folder of teamFolders) {
      if (!folder.teamId) {
        continue;
      }
      const group = groups.get(folder.teamId);
      if (group) {
        group.folders.push(folder);
      } else {
        groups.set(folder.teamId, {
          teamId: folder.teamId,
          teamName: folder.teamName || 'Unknown Team',
          teamRole: 'TEAM_VIEWER',
          connections: [],
          folders: [folder],
        });
      }
    }

    return Array.from(groups.values()).sort((left, right) => left.teamName.localeCompare(right.teamName));
  }, [teamConnections, teamFolders]);

  const isSearching = searchQuery.trim().length > 0;
  const isDndEnabled = !isSearching;

  const handleDragStart = (event: DragStartEvent) => {
    const connection = event.active.data.current?.connection as ConnectionData | undefined;
    if (connection) {
      setActiveConnection(connection);
    }
  };

  const handleDragEnd = async (event: DragEndEvent) => {
    setActiveConnection(null);
    const connection = event.active.data.current?.connection as ConnectionData | undefined;
    const targetFolderId = (event.over?.data.current?.folderId as string | null) ?? null;

    if (!connection || !event.over || targetFolderId === (connection.folderId ?? null)) {
      return;
    }

    try {
      await moveConnection(connection.id, targetFolderId);
    } catch (error) {
      notify(extractApiError(error, 'Failed to move connection'));
    }
  };

  const handleToggleFavorite = async (connection: ConnectionData) => {
    await toggleFavorite(connection.id);
  };

  const handleOpenMoveDialog = (connection: ConnectionData) => {
    setMoveTarget(connection);
    setMoveDestination(connection.folderId || '__root__');
  };

  const handleConfirmMove = async () => {
    if (!moveTarget) {
      return;
    }

    const destinationFolderId = moveDestination === '__root__' ? null : moveDestination;
    if (destinationFolderId === moveTarget.folderId) {
      setMoveTarget(null);
      return;
    }

    try {
      await updateConnection(moveTarget.id, { folderId: destinationFolderId });
      await fetchConnections();
      setMoveTarget(null);
    } catch (error) {
      notify(extractApiError(error, 'Failed to move connection'));
    }
  };

  const handleConfirmDelete = async () => {
    if (!deleteTarget) {
      return;
    }

    try {
      await deleteConnection(deleteTarget.id);
      await fetchConnections();
      setDeleteTarget(null);
    } catch (error) {
      notify(extractApiError(error, 'Failed to delete connection'));
    }
  };

  const handleConfirmDeleteFolder = async () => {
    if (!deleteFolderTarget) {
      return;
    }

    try {
      await deleteFolder(deleteFolderTarget.id);
      await fetchConnections();
      setDeleteFolderTarget(null);
    } catch (error) {
      notify(extractApiError(error, 'Failed to delete folder'));
    }
  };

  const bulkOpenOne = (connection: ConnectionData) => {
    if (connection.defaultCredentialMode === 'domain') {
      openTab(connection, { username: '', password: '', credentialMode: 'domain' });
      return;
    }
    openTab(connection);
  };

  const handleBulkOpen = (folderId: string) => {
    const directConnections = collectFolderConnections(folderId, filteredFolderMap, allFolders, false);
    const hasSubfolders = folderHasSubfolders(folderId, allFolders);

    if (hasSubfolders) {
      const recursiveConnections = collectFolderConnections(folderId, filteredFolderMap, allFolders, true);
      setBulkOpenSubfolderPrompt({
        folderId,
        thisOnly: directConnections.length,
        withSubs: recursiveConnections.length,
      });
      return;
    }

    if (directConnections.length > 5) {
      setBulkOpenTarget({ folderId, connections: directConnections });
      return;
    }

    directConnections.forEach(bulkOpenOne);
  };

  const handleBulkOpenChoice = (recursive: boolean) => {
    if (!bulkOpenSubfolderPrompt) {
      return;
    }

    const connections = collectFolderConnections(
      bulkOpenSubfolderPrompt.folderId,
      filteredFolderMap,
      allFolders,
      recursive,
    );
    setBulkOpenSubfolderPrompt(null);

    if (connections.length > 5) {
      setBulkOpenTarget({ folderId: bulkOpenSubfolderPrompt.folderId, connections });
      return;
    }

    connections.forEach(bulkOpenOne);
  };

  const showNoResults = searchQuery.trim().length > 0
    && filteredRootConnections.length === 0
    && filteredFolderTree.length === 0
    && filteredSharedConnections.length === 0
    && teamGroups.every((group) => group.connections.every((connection) => !matchesSearch(connection, searchQuery)));

  return (
    <div className="space-y-3 py-3">
      <div className="flex items-center gap-1 px-2">
        <div className="min-w-0 flex-1 px-2">
          <div className="truncate text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
            My Connections
          </div>
        </div>
        <SidebarIconButton
          active={compact}
          title={compact ? 'Normal view' : 'Compact view'}
          onClick={() => togglePreference('sidebarCompact')}
        >
          <TimerReset className="size-4" />
        </SidebarIconButton>
        <SidebarIconButton title="New Connection" onClick={() => onCreateConnection()}>
          <Plus className="size-4" />
        </SidebarIconButton>
        <SidebarIconButton title="New Folder" onClick={() => onCreateFolder()}>
          <FolderPlus className="size-4" />
        </SidebarIconButton>
      </div>

      <SidebarSearchInput value={searchQuery} onChange={setSearchQuery} />

      {!isSearching && favoriteConnections.length > 0 ? (
        <section className="space-y-1">
          <SidebarSectionHeader
            open={favoritesOpen}
            label="Favorites"
            icon={<Star className="size-4" />}
            onToggle={() => togglePreference('sidebarFavoritesOpen')}
          />
          {favoritesOpen ? (
            <div>
              {favoriteConnections.map((connection) => (
                <ConnectionItem
                  key={`fav-${connection.id}`}
                  conn={connection}
                  depth={0}
                  compact={compact}
                  onEdit={onEditConnection}
                  onDelete={setDeleteTarget}
                  onMove={handleOpenMoveDialog}
                  onShare={onShareConnection}
                  onConnectAs={onConnectAsConnection}
                  onToggleFavorite={handleToggleFavorite}
                  onViewAuditLog={onViewAuditLog}
                />
              ))}
            </div>
          ) : null}
        </section>
      ) : null}

      {!isSearching && recentConnections.length > 0 ? (
        <section className="space-y-1">
          <SidebarSectionHeader
            open={recentsOpen}
            label="Recent"
            icon={<TimerReset className="size-4" />}
            onToggle={() => togglePreference('sidebarRecentsOpen')}
          />
          {recentsOpen ? (
            <div>
              {recentConnections.map((connection) => (
                <ConnectionItem
                  key={`recent-${connection.id}`}
                  conn={connection}
                  depth={0}
                  compact={compact}
                  onEdit={onEditConnection}
                  onDelete={setDeleteTarget}
                  onMove={handleOpenMoveDialog}
                  onShare={onShareConnection}
                  onConnectAs={onConnectAsConnection}
                  onToggleFavorite={connection.isOwner ? handleToggleFavorite : undefined}
                  onViewAuditLog={onViewAuditLog}
                />
              ))}
            </div>
          ) : null}
        </section>
      ) : null}

      {!isSearching && (favoriteConnections.length > 0 || recentConnections.length > 0) ? (
        <Separator />
      ) : null}

      <DndContext
        sensors={sensors}
        collisionDetection={pointerWithin}
        onDragStart={handleDragStart}
        onDragEnd={handleDragEnd}
      >
        <div
          ref={rootDropRef}
          className={isOverRoot ? 'rounded-xl bg-primary/5' : undefined}
        >
          {filteredFolderTree.map((node) => (
            <FolderItem
              key={node.folder.id}
              node={node}
              connections={filteredFolderMap.get(node.folder.id) || []}
              folderMap={filteredFolderMap}
              depth={0}
              compact={compact}
              isDndEnabled={isDndEnabled}
              onEditConnection={onEditConnection}
              onDeleteConnection={setDeleteTarget}
              onMoveConnection={handleOpenMoveDialog}
              onShareConnection={onShareConnection}
              onConnectAsConnection={onConnectAsConnection}
              onToggleFavorite={handleToggleFavorite}
              onViewAuditLog={onViewAuditLog}
              onCreateConnection={onCreateConnection}
              onCreateFolder={onCreateFolder}
              onEditFolder={onEditFolder}
              onDeleteFolder={setDeleteFolderTarget}
              onBulkOpen={handleBulkOpen}
              onShareFolder={onShareFolder}
            />
          ))}
          {filteredRootConnections.map((connection) => (
            <ConnectionItem
              key={connection.id}
              conn={connection}
              depth={0}
              compact={compact}
              draggable={isDndEnabled && connection.isOwner}
              onEdit={onEditConnection}
              onDelete={setDeleteTarget}
              onMove={handleOpenMoveDialog}
              onShare={onShareConnection}
              onConnectAs={onConnectAsConnection}
              onToggleFavorite={handleToggleFavorite}
              onViewAuditLog={onViewAuditLog}
            />
          ))}
        </div>

        <DragOverlay dropAnimation={null}>
          {activeConnection ? (
            <div className="flex max-w-[14rem] items-center gap-2 rounded-xl border bg-popover px-3 py-2 shadow-lg">
              {dragIcon(activeConnection.type)}
              <span className="truncate text-sm text-foreground">{activeConnection.name}</span>
            </div>
          ) : null}
        </DragOverlay>
      </DndContext>

      {teamGroups.length > 0 ? <Separator /> : null}
      {teamGroups.map((group) => (
        <TeamConnectionSection
          key={group.teamId}
          teamId={group.teamId}
          teamName={group.teamName}
          teamRole={group.teamRole}
          connections={group.connections}
          folders={group.folders}
          compact={compact}
          searchQuery={searchQuery}
          onEditConnection={onEditConnection}
          onDeleteConnection={setDeleteTarget}
          onMoveConnection={handleOpenMoveDialog}
          onShareConnection={onShareConnection}
          onConnectAsConnection={onConnectAsConnection}
          onToggleFavorite={handleToggleFavorite}
          onViewAuditLog={onViewAuditLog}
          onCreateConnection={onCreateConnection}
          onCreateFolder={onCreateFolder}
          onEditFolder={onEditFolder}
          onDeleteFolder={setDeleteFolderTarget}
          onBulkOpen={handleBulkOpen}
          onShareFolder={onShareFolder}
        />
      ))}

      {showNoResults ? (
        <div className="flex flex-col items-center gap-2 px-4 py-8 text-center">
          <SearchX className="size-5 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">No connections match your search.</p>
        </div>
      ) : null}

      {filteredSharedConnections.length > 0 ? (
        <>
          <Separator />
          <section className="space-y-1">
            <SidebarSectionHeader
              open={sharedOpen}
              label="Shared With Me"
              icon={<Share2 className="size-4" />}
              onToggle={() => togglePreference('sidebarSharedOpen')}
            />
            {sharedOpen ? (
              <div>
                {filteredSharedConnections.map((connection) => (
                  <ConnectionItem
                    key={connection.id}
                    conn={connection}
                    depth={0}
                    compact={compact}
                    onEdit={onEditConnection}
                    onDelete={setDeleteTarget}
                    onMove={handleOpenMoveDialog}
                    onShare={onShareConnection}
                    onConnectAs={onConnectAsConnection}
                    onToggleFavorite={handleToggleFavorite}
                    onViewAuditLog={onViewAuditLog}
                  />
                ))}
              </div>
            ) : null}
          </section>
        </>
      ) : null}

      <Dialog open={moveTarget !== null} onOpenChange={(open) => { if (!open) setMoveTarget(null); }}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Move &quot;{moveTarget?.name}&quot;</DialogTitle>
            <DialogDescription>
              Choose where this connection should live in your tree.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground" htmlFor="move-destination">
              Destination Folder
            </label>
            <Select value={moveDestination} onValueChange={setMoveDestination}>
              <SelectTrigger id="move-destination">
                <SelectValue placeholder="Select a folder" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__root__">Root (no folder)</SelectItem>
                {folders.map((folder) => (
                  <SelectItem key={folder.id} value={folder.id}>
                    {folder.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setMoveTarget(null)}>
              Cancel
            </Button>
            <Button type="button" onClick={() => void handleConfirmMove()}>
              Move
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <SidebarConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
        title="Delete Connection"
        description={
          <p className="leading-6">
            Are you sure you want to delete &quot;{deleteTarget?.name}&quot;? This action cannot be undone.
          </p>
        }
        confirmLabel="Delete"
        destructive
        onConfirm={() => void handleConfirmDelete()}
      />

      <SidebarConfirmDialog
        open={deleteFolderTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteFolderTarget(null); }}
        title="Delete Folder"
        description={
          <p className="leading-6">
            Are you sure you want to delete &quot;{deleteFolderTarget?.name}&quot;?
            Connections in this folder will be moved to the root level.
          </p>
        }
        confirmLabel="Delete"
        destructive
        onConfirm={() => void handleConfirmDeleteFolder()}
      />

      <Dialog open={bulkOpenSubfolderPrompt !== null} onOpenChange={(open) => { if (!open) setBulkOpenSubfolderPrompt(null); }}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Open All Connections</DialogTitle>
            <DialogDescription>
              This folder contains subfolders. Decide which connections should open.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="justify-start sm:justify-between">
            <Button type="button" variant="outline" onClick={() => setBulkOpenSubfolderPrompt(null)}>
              Cancel
            </Button>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Button type="button" variant="outline" onClick={() => handleBulkOpenChoice(false)}>
                This folder only ({bulkOpenSubfolderPrompt?.thisOnly ?? 0})
              </Button>
              <Button type="button" onClick={() => handleBulkOpenChoice(true)}>
                Include subfolders ({bulkOpenSubfolderPrompt?.withSubs ?? 0})
              </Button>
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <SidebarConfirmDialog
        open={bulkOpenTarget !== null}
        onOpenChange={(open) => { if (!open) setBulkOpenTarget(null); }}
        title={`Open ${bulkOpenTarget?.connections.length ?? 0} Connections?`}
        description={
          <p className="leading-6">
            This will create {bulkOpenTarget?.connections.length ?? 0} new tabs.
          </p>
        }
        confirmLabel="Open All"
        onConfirm={() => {
          bulkOpenTarget?.connections.forEach(bulkOpenOne);
          setBulkOpenTarget(null);
        }}
      />
    </div>
  );
}
