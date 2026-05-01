import { useEffect, useRef, useState } from 'react';
import {
  ArrowRightLeft,
  DatabaseZap,
  ExternalLink,
  Folder,
  FolderOpen,
  FolderPlus,
  History,
  Monitor,
  Pencil,
  Play,
  Plus,
  Share2,
  Star,
  StarOff,
  TerminalSquare,
  Trash2,
  UserRound,
} from 'lucide-react';
import { useDraggable, useDroppable } from '@dnd-kit/core';
import { CSS } from '@dnd-kit/utilities';
import { downloadRdpFile } from '@/api/rdGateway.api';
import type { ConnectionData } from '@/api/connections.api';
import type { Folder as FolderData } from '@/store/connectionsStore';
import { useTabsStore } from '@/store/tabsStore';
import { openConnectionWindow } from '@/utils/openConnectionWindow';
import { cn } from '@/lib/utils';
import {
  SidebarContextMenu,
} from './sidebarUi';
import {
  buildFolderTree,
  collectFolderConnections,
  depthPl,
  folderHasSubfolders,
  matchesSearch,
  pruneFolderTree,
  type FolderNode,
} from './treeUtils';

export {
  buildFolderTree,
  collectFolderConnections,
  depthPl,
  folderHasSubfolders,
  matchesSearch,
  pruneFolderTree,
  type FolderNode,
};

function connectionIcon(type: ConnectionData['type']) {
  if (type === 'SSH') {
    return <TerminalSquare className="size-4" />;
  }
  if (type === 'DATABASE') {
    return <DatabaseZap className="size-4" />;
  }
  return <Monitor className="size-4" />;
}

type ContextMenuState = { x: number; y: number } | null;

export interface ConnectionItemProps {
  conn: ConnectionData;
  depth: number;
  compact?: boolean;
  draggable?: boolean;
  onEdit: (conn: ConnectionData) => void;
  onDelete: (conn: ConnectionData) => void;
  onMove: (conn: ConnectionData) => void;
  onShare: (conn: ConnectionData) => void;
  onConnectAs: (conn: ConnectionData) => void;
  onToggleFavorite?: (conn: ConnectionData) => void;
  onViewAuditLog?: (conn: ConnectionData) => void;
}

export function ConnectionItem({
  conn,
  depth,
  compact,
  draggable = false,
  onEdit,
  onDelete,
  onMove,
  onShare,
  onConnectAs,
  onToggleFavorite,
  onViewAuditLog,
}: ConnectionItemProps) {
  const openTab = useTabsStore((state) => state.openTab);
  const [contextMenu, setContextMenu] = useState<ContextMenuState>(null);
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    isDragging,
  } = useDraggable({
    id: `connection-${conn.id}`,
    data: { type: 'connection', connection: conn },
    disabled: !draggable,
  });

  const quickConnect = () => {
    if (conn.defaultCredentialMode === 'domain') {
      openTab(conn, { username: '', password: '', credentialMode: 'domain' });
      return;
    }
    if (conn.defaultCredentialMode === 'prompt') {
      onConnectAs(conn);
      return;
    }
    openTab(conn);
  };

  const contextMenuActions = [
    {
      label: 'Connect',
      icon: <Play className="size-4" />,
      onSelect: quickConnect,
    },
    {
      label: 'Connect As...',
      icon: <UserRound className="size-4" />,
      onSelect: () => onConnectAs(conn),
    },
    {
      label: 'Open in New Window',
      icon: <ExternalLink className="size-4" />,
      onSelect: () => openConnectionWindow(conn.id),
    },
    ...(conn.type === 'RDP'
      ? [{
          label: 'Open with Native Client',
          icon: <ExternalLink className="size-4" />,
          onSelect: () => {
            void downloadRdpFile(conn.id, conn.name).catch(() => {});
          },
        }]
      : []),
    ...(conn.isOwner && onToggleFavorite
      ? [{
          label: conn.isFavorite ? 'Remove from Favorites' : 'Add to Favorites',
          icon: conn.isFavorite ? <StarOff className="size-4" /> : <Star className="size-4" />,
          onSelect: () => onToggleFavorite(conn),
        }]
      : []),
    {
      label: 'Move to Folder',
      icon: <ArrowRightLeft className="size-4" />,
      disabled: !conn.isOwner,
      separatorBefore: true,
      onSelect: () => onMove(conn),
    },
    {
      label: 'Edit',
      icon: <Pencil className="size-4" />,
      disabled: !conn.isOwner,
      onSelect: () => onEdit(conn),
    },
    {
      label: 'Share',
      icon: <Share2 className="size-4" />,
      disabled: !conn.isOwner,
      onSelect: () => onShare(conn),
    },
    ...(onViewAuditLog
      ? [{
          label: 'Activity Log',
          icon: <History className="size-4" />,
          onSelect: () => onViewAuditLog(conn),
        }]
      : []),
    {
      label: 'Delete',
      icon: <Trash2 className="size-4" />,
      disabled: !conn.isOwner,
      destructive: true,
      onSelect: () => onDelete(conn),
    },
  ];

  const dragHandleProps = draggable ? { ...listeners, ...attributes } : undefined;

  return (
    <>
      <div
        ref={setNodeRef}
        className={cn(
          'group flex w-full items-center gap-1 border-l-2 border-transparent pr-2 transition-colors hover:bg-accent/70',
          compact ? 'py-1' : 'py-1.5',
          draggable && 'cursor-grab active:cursor-grabbing',
          isDragging && 'opacity-40',
        )}
        style={{
          paddingLeft: depthPl(depth),
          transform: transform ? CSS.Translate.toString(transform) : undefined,
        }}
      >
        <button
          type="button"
          onDoubleClick={quickConnect}
          onContextMenu={(event) => {
            event.preventDefault();
            event.stopPropagation();
            setContextMenu({ x: event.clientX, y: event.clientY });
          }}
          className="flex min-w-0 flex-1 items-center gap-3 rounded-md py-1 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
          {...dragHandleProps}
        >
          <span className="shrink-0 text-muted-foreground">
            {connectionIcon(conn.type)}
          </span>
          <span className="min-w-0 flex-1">
            <span className="block truncate text-sm text-foreground">{conn.name}</span>
            {!compact ? (
              <span className="block truncate text-xs text-muted-foreground">
                {conn.host}:{conn.port}
              </span>
            ) : null}
          </span>
        </button>
        {conn.isOwner && onToggleFavorite ? (
          <button
            type="button"
            aria-label={conn.isFavorite ? 'Remove from favorites' : 'Add to favorites'}
            className="inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-background hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/60 focus-visible:ring-offset-2 focus-visible:ring-offset-background"
            onClick={() => onToggleFavorite(conn)}
          >
            {conn.isFavorite ? (
              <Star className="size-4 fill-current text-primary" />
            ) : (
              <Star className="size-4" />
            )}
          </button>
        ) : null}
      </div>

      <SidebarContextMenu
        open={contextMenu !== null}
        position={contextMenu}
        onClose={() => setContextMenu(null)}
        label={conn.name}
        actions={contextMenuActions}
      />
    </>
  );
}

export interface FolderItemProps {
  node: FolderNode;
  connections: ConnectionData[];
  folderMap: Map<string, ConnectionData[]>;
  depth: number;
  compact?: boolean;
  isDndEnabled?: boolean;
  teamId?: string;
  onEditConnection: (conn: ConnectionData) => void;
  onDeleteConnection: (conn: ConnectionData) => void;
  onMoveConnection: (conn: ConnectionData) => void;
  onShareConnection: (conn: ConnectionData) => void;
  onConnectAsConnection: (conn: ConnectionData) => void;
  onToggleFavorite: (conn: ConnectionData) => void;
  onViewAuditLog?: (conn: ConnectionData) => void;
  onCreateConnection: (folderId: string, teamId?: string) => void;
  onCreateFolder: (parentId?: string, teamId?: string) => void;
  onEditFolder: (folder: FolderData) => void;
  onDeleteFolder: (folder: FolderData) => void;
  onBulkOpen?: (folderId: string) => void;
  onShareFolder?: (folderId: string, folderName: string) => void;
}

export function FolderItem({
  node,
  connections,
  folderMap,
  depth,
  compact,
  isDndEnabled = false,
  teamId,
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
}: FolderItemProps) {
  const [open, setOpen] = useState(true);
  const [contextMenu, setContextMenu] = useState<ContextMenuState>(null);
  const { setNodeRef, isOver } = useDroppable({
    id: `folder-${node.folder.id}`,
    data: { type: 'folder', folderId: node.folder.id },
    disabled: !isDndEnabled,
  });
  const dragOverTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (isOver && !open) {
      dragOverTimerRef.current = setTimeout(() => setOpen(true), 500);
    }
    return () => {
      if (dragOverTimerRef.current) {
        clearTimeout(dragOverTimerRef.current);
      }
    };
  }, [isOver, open]);

  const contextMenuActions = [
    {
      label: 'New Connection',
      icon: <Plus className="size-4" />,
      onSelect: () => onCreateConnection(node.folder.id, teamId),
    },
    {
      label: 'New Subfolder',
      icon: <FolderPlus className="size-4" />,
      onSelect: () => onCreateFolder(node.folder.id, teamId),
    },
    ...(onBulkOpen
      ? [{
          label: 'Open All',
          icon: <Play className="size-4" />,
          onSelect: () => onBulkOpen(node.folder.id),
        }]
      : []),
    ...(onShareFolder
      ? [{
          label: 'Share Folder',
          icon: <Share2 className="size-4" />,
          onSelect: () => onShareFolder(node.folder.id, node.folder.name),
        }]
      : []),
    {
      label: 'Rename',
      icon: <Pencil className="size-4" />,
      separatorBefore: true,
      onSelect: () => onEditFolder(node.folder),
    },
    {
      label: 'Delete',
      icon: <Trash2 className="size-4" />,
      destructive: true,
      onSelect: () => onDeleteFolder(node.folder),
    },
  ];

  return (
    <>
      <button
        ref={setNodeRef}
        type="button"
        onClick={() => setOpen((current) => !current)}
        onContextMenu={(event) => {
          event.preventDefault();
          event.stopPropagation();
          setContextMenu({ x: event.clientX, y: event.clientY });
        }}
        className={cn(
          'flex w-full items-center gap-3 border-l-2 border-transparent py-2 pr-2 text-left transition-colors hover:bg-accent/70',
          compact ? 'py-1.5' : 'py-2',
          isOver && 'border-primary bg-primary/10',
        )}
        style={{
          paddingLeft: depthPl(depth),
        }}
      >
        <span className="shrink-0 text-muted-foreground">
          {open ? <FolderOpen className="size-4" /> : <Folder className="size-4" />}
        </span>
        <span className="min-w-0 flex-1 truncate text-sm text-foreground">
          {node.folder.name}
        </span>
        {open ? (
          <ChevronDownIcon className="size-4 text-muted-foreground" />
        ) : (
          <ChevronRightIcon className="size-4 text-muted-foreground" />
        )}
      </button>

      <SidebarContextMenu
        open={contextMenu !== null}
        position={contextMenu}
        onClose={() => setContextMenu(null)}
        label={node.folder.name}
        actions={contextMenuActions}
      />

      {open ? (
        <div>
          {node.children.map((child) => (
            <FolderItem
              key={child.folder.id}
              node={child}
              connections={folderMap.get(child.folder.id) || []}
              folderMap={folderMap}
              depth={depth + 1}
              compact={compact}
              isDndEnabled={isDndEnabled}
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
          {connections.map((conn) => (
            <ConnectionItem
              key={conn.id}
              conn={conn}
              depth={depth + 1}
              compact={compact}
              draggable={isDndEnabled && conn.isOwner}
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
    </>
  );
}

function ChevronDownIcon({ className }: { className?: string }) {
  return <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true"><path d="m6 9 6 6 6-6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}

function ChevronRightIcon({ className }: { className?: string }) {
  return <svg viewBox="0 0 24 24" fill="none" className={className} aria-hidden="true"><path d="m9 6 6 6-6 6" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" /></svg>;
}
