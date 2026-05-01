import { useMemo } from 'react';
import { ChevronDown, ChevronRight, FolderPlus, Plus, Users } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useUiPreferencesStore } from '@/store/uiPreferencesStore';
import type { ConnectionData } from '@/api/connections.api';
import type { Folder } from '@/store/connectionsStore';
import {
  buildFolderTree,
  ConnectionItem,
  FolderItem,
  matchesSearch,
  pruneFolderTree,
} from './treeHelpers';
import { SidebarIconButton } from './sidebarUi';

interface TeamConnectionSectionProps {
  teamId: string;
  teamName: string;
  teamRole: string;
  connections: ConnectionData[];
  folders: Folder[];
  compact: boolean;
  searchQuery: string;
  onEditConnection: (conn: ConnectionData) => void;
  onDeleteConnection: (conn: ConnectionData) => void;
  onMoveConnection: (conn: ConnectionData) => void;
  onShareConnection: (conn: ConnectionData) => void;
  onConnectAsConnection: (conn: ConnectionData) => void;
  onToggleFavorite: (conn: ConnectionData) => void;
  onViewAuditLog?: (conn: ConnectionData) => void;
  onCreateConnection: (folderId?: string, teamId?: string) => void;
  onCreateFolder: (parentId?: string, teamId?: string) => void;
  onEditFolder: (folder: Folder) => void;
  onDeleteFolder: (folder: Folder) => void;
  onBulkOpen?: (folderId: string) => void;
  onShareFolder?: (folderId: string, folderName: string) => void;
}

export default function TeamConnectionSection({
  teamId,
  teamName,
  teamRole,
  connections,
  folders,
  compact,
  searchQuery,
  onEditConnection,
  onDeleteConnection,
  onMoveConnection,
  onShareConnection,
  onConnectAsConnection,
  onToggleFavorite,
  onViewAuditLog,
  onCreateConnection,
  onCreateFolder,
  onEditFolder,
  onDeleteFolder,
  onBulkOpen,
  onShareFolder,
}: TeamConnectionSectionProps) {
  const sidebarTeamSections = useUiPreferencesStore((state) => state.sidebarTeamSections);
  const toggleTeamSection = useUiPreferencesStore((state) => state.toggleTeamSection);
  const isOpen = sidebarTeamSections[teamId] ?? true;
  const isSearching = searchQuery.trim().length > 0;
  const canCreate = teamRole === 'TEAM_ADMIN' || teamRole === 'TEAM_EDITOR';

  const { filteredRootConnections, filteredFolderMap, filteredFolderTree } = useMemo(() => {
    const filteredConnections = isSearching
      ? connections.filter((connection) => matchesSearch(connection, searchQuery))
      : connections;

    const rootConnections = filteredConnections.filter((connection) => !connection.folderId);
    const folderMap = new Map<string, ConnectionData[]>();
    filteredConnections.forEach((connection) => {
      if (!connection.folderId) {
        return;
      }
      const group = folderMap.get(connection.folderId) || [];
      group.push(connection);
      folderMap.set(connection.folderId, group);
    });

    const fullTree = buildFolderTree(folders);
    return {
      filteredRootConnections: rootConnections,
      filteredFolderMap: folderMap,
      filteredFolderTree: isSearching ? pruneFolderTree(fullTree, folderMap) : fullTree,
    };
  }, [connections, folders, isSearching, searchQuery]);

  if (isSearching && filteredRootConnections.length === 0 && filteredFolderTree.length === 0) {
    return null;
  }

  return (
    <section className="space-y-1">
      <div className="flex items-center gap-1 px-2">
        <button
          type="button"
          onClick={() => toggleTeamSection(teamId)}
          className="flex min-w-0 flex-1 items-center gap-2 rounded-lg px-2 py-1.5 text-left transition-colors hover:bg-accent/70"
        >
          {isOpen ? (
            <ChevronDown className="size-4 shrink-0 text-muted-foreground" />
          ) : (
            <ChevronRight className="size-4 shrink-0 text-muted-foreground" />
          )}
          <Users className="size-4 shrink-0 text-muted-foreground" />
          <span className="truncate text-sm font-medium text-foreground">{teamName}</span>
        </button>

        {canCreate ? (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <SidebarIconButton aria-label={`Add to ${teamName}`}>
                <Plus className="size-4" />
              </SidebarIconButton>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => onCreateConnection(undefined, teamId)}>
                <Plus className="size-4" />
                New Connection
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => onCreateFolder(undefined, teamId)}>
                <FolderPlus className="size-4" />
                New Folder
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : null}
      </div>

      {isOpen ? (
        <div>
          {filteredFolderTree.map((node) => (
            <FolderItem
              key={node.folder.id}
              node={node}
              connections={filteredFolderMap.get(node.folder.id) || []}
              folderMap={filteredFolderMap}
              depth={0}
              compact={compact}
              teamId={teamId}
              onEditConnection={onEditConnection}
              onDeleteConnection={onDeleteConnection}
              onMoveConnection={onMoveConnection}
              onShareConnection={onShareConnection}
              onConnectAsConnection={onConnectAsConnection}
              onToggleFavorite={onToggleFavorite}
              onViewAuditLog={onViewAuditLog}
              onCreateConnection={onCreateConnection}
              onCreateFolder={onCreateFolder}
              onEditFolder={onEditFolder}
              onDeleteFolder={onDeleteFolder}
              onBulkOpen={onBulkOpen}
              onShareFolder={onShareFolder}
            />
          ))}
          {filteredRootConnections.map((connection) => (
            <ConnectionItem
              key={connection.id}
              conn={connection}
              depth={0}
              compact={compact}
              onEdit={onEditConnection}
              onDelete={onDeleteConnection}
              onMove={onMoveConnection}
              onShare={onShareConnection}
              onConnectAs={onConnectAsConnection}
              onToggleFavorite={onToggleFavorite}
              onViewAuditLog={onViewAuditLog}
            />
          ))}
        </div>
      ) : null}
    </section>
  );
}
