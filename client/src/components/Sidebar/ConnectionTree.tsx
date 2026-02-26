import { useState, useMemo } from 'react';
import {
  Box, Typography, List, ListItemButton, ListItemIcon, ListItemText,
  Collapse, Menu, MenuItem, Divider, IconButton, TextField, InputAdornment,
  Dialog, DialogTitle, DialogContent, DialogContentText, DialogActions, Button,
  FormControl, InputLabel, Select,
} from '@mui/material';
import type { SelectChangeEvent } from '@mui/material';
import {
  Computer as RdpIcon,
  Terminal as SshIcon,
  Folder as FolderIcon,
  FolderOpen as FolderOpenIcon,
  ExpandMore,
  ChevronRight,
  Share as ShareIcon,
  PlayArrow as ConnectIcon,
  OpenInNew as OpenInNewIcon,
  Edit as EditIcon,
  Delete as DeleteIcon,
  CreateNewFolder as CreateNewFolderIcon,
  Add as AddIcon,
  SwitchAccount as SwitchAccountIcon,
  DriveFileMove as MoveIcon,
  Search as SearchIcon,
  Clear as ClearIcon,
  Star as StarIcon,
  StarBorder as StarBorderIcon,
  AccessTime as RecentIcon,
  ViewList as ViewListIcon,
  ViewCompact as ViewCompactIcon,
} from '@mui/icons-material';
import { useConnectionsStore, Folder } from '../../store/connectionsStore';
import { useTabsStore } from '../../store/tabsStore';
import { useAuthStore } from '../../store/authStore';
import { useNotificationStore } from '../../store/notificationStore';
import { ConnectionData, deleteConnection, updateConnection } from '../../api/connections.api';
import { deleteFolder } from '../../api/folders.api';
import { openConnectionWindow } from '../../utils/openConnectionWindow';
import { getRecentConnectionIds } from '../../utils/recentConnections';

function getErrorMessage(err: unknown, fallback: string): string {
  return (err as { response?: { data?: { error?: string } } })?.response?.data?.error || fallback;
}

// Consistent indentation: base padding + depth * indent step
const BASE_PL = 2;
const INDENT = 2;
function depthPl(depth: number) { return BASE_PL + depth * INDENT; }

// --- Tree helpers ---

interface FolderNode {
  folder: Folder;
  children: FolderNode[];
}

function matchesSearch(conn: ConnectionData, query: string): boolean {
  const q = query.toLowerCase();
  return conn.name.toLowerCase().includes(q)
    || conn.host.toLowerCase().includes(q)
    || conn.type.toLowerCase().includes(q)
    || (conn.description?.toLowerCase().includes(q) ?? false);
}

function pruneFolderTree(nodes: FolderNode[], folderMap: Map<string, ConnectionData[]>): FolderNode[] {
  return nodes.reduce<FolderNode[]>((acc, node) => {
    const prunedChildren = pruneFolderTree(node.children, folderMap);
    const hasConnections = (folderMap.get(node.folder.id) || []).length > 0;
    if (hasConnections || prunedChildren.length > 0) {
      acc.push({ ...node, children: prunedChildren });
    }
    return acc;
  }, []);
}

function buildFolderTree(folders: Folder[]): FolderNode[] {
  const map = new Map<string, FolderNode>();
  for (const f of folders) {
    map.set(f.id, { folder: f, children: [] });
  }
  const roots: FolderNode[] = [];
  for (const node of map.values()) {
    const pid = node.folder.parentId;
    if (pid && map.has(pid)) {
      map.get(pid)!.children.push(node);
    } else {
      roots.push(node);
    }
  }
  return roots;
}

// --- ConnectionItem ---

interface ConnectionItemProps {
  conn: ConnectionData;
  depth: number;
  compact?: boolean;
  onEdit: (conn: ConnectionData) => void;
  onDelete: (conn: ConnectionData) => void;
  onMove: (conn: ConnectionData) => void;
  onShare: (conn: ConnectionData) => void;
  onConnectAs: (conn: ConnectionData) => void;
  onToggleFavorite?: (conn: ConnectionData) => void;
}

function ConnectionItem({ conn, depth, compact, onEdit, onDelete, onMove, onShare, onConnectAs, onToggleFavorite }: ConnectionItemProps) {
  const openTab = useTabsStore((s) => s.openTab);
  const [contextMenu, setContextMenu] = useState<{ mouseX: number; mouseY: number } | null>(null);

  const handleContextMenu = (event: React.MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
    setContextMenu({ mouseX: event.clientX - 2, mouseY: event.clientY - 4 });
  };

  const handleCloseMenu = () => setContextMenu(null);

  const handleConnect = () => {
    handleCloseMenu();
    openTab(conn);
  };

  const handleOpenInNewWindow = () => {
    handleCloseMenu();
    openConnectionWindow(conn.id);
  };

  const handleEdit = () => {
    handleCloseMenu();
    onEdit(conn);
  };

  const handleDelete = () => {
    handleCloseMenu();
    onDelete(conn);
  };

  const handleMove = () => {
    handleCloseMenu();
    onMove(conn);
  };

  const handleShare = () => {
    handleCloseMenu();
    onShare(conn);
  };

  const handleConnectAs = () => {
    handleCloseMenu();
    onConnectAs(conn);
  };

  return (
    <>
      <ListItemButton
        dense
        onDoubleClick={() => openTab(conn)}
        onContextMenu={handleContextMenu}
        sx={{ pl: depthPl(depth), ...(compact && { py: 0.125 }) }}
      >
        <ListItemIcon sx={{ minWidth: compact ? 24 : 32 }}>
          {conn.type === 'RDP' ? (
            <RdpIcon fontSize="small" color="primary" />
          ) : (
            <SshIcon fontSize="small" color="secondary" />
          )}
        </ListItemIcon>
        <ListItemText
          primary={conn.name}
          secondary={compact ? undefined : `${conn.host}:${conn.port}`}
          primaryTypographyProps={{ variant: 'body2', noWrap: true }}
          secondaryTypographyProps={{ variant: 'caption', noWrap: true }}
        />
        {conn.isOwner && onToggleFavorite && (
          <IconButton
            size="small"
            onClick={(e) => { e.stopPropagation(); onToggleFavorite(conn); }}
            sx={{ p: 0.25 }}
          >
            {conn.isFavorite
              ? <StarIcon fontSize="small" color="warning" />
              : <StarBorderIcon fontSize="small" sx={{ opacity: 0.3 }} />}
          </IconButton>
        )}
      </ListItemButton>

      <Menu
        open={contextMenu !== null}
        onClose={handleCloseMenu}
        anchorReference="anchorPosition"
        anchorPosition={
          contextMenu !== null
            ? { top: contextMenu.mouseY, left: contextMenu.mouseX }
            : undefined
        }
      >
        <MenuItem onClick={handleConnect}>
          <ListItemIcon><ConnectIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Connect</ListItemText>
        </MenuItem>
        <MenuItem onClick={handleConnectAs}>
          <ListItemIcon><SwitchAccountIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Connect As...</ListItemText>
        </MenuItem>
        <MenuItem onClick={handleOpenInNewWindow}>
          <ListItemIcon><OpenInNewIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Open in New Window</ListItemText>
        </MenuItem>
        {conn.isOwner && onToggleFavorite && (
          <MenuItem onClick={() => { handleCloseMenu(); onToggleFavorite(conn); }}>
            <ListItemIcon>
              {conn.isFavorite
                ? <StarBorderIcon fontSize="small" />
                : <StarIcon fontSize="small" color="warning" />}
            </ListItemIcon>
            <ListItemText>{conn.isFavorite ? 'Remove from Favorites' : 'Add to Favorites'}</ListItemText>
          </MenuItem>
        )}
        <Divider />
        <MenuItem onClick={handleMove} disabled={!conn.isOwner}>
          <ListItemIcon><MoveIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Move to Folder</ListItemText>
        </MenuItem>
        <MenuItem onClick={handleEdit} disabled={!conn.isOwner}>
          <ListItemIcon><EditIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Edit</ListItemText>
        </MenuItem>
        <MenuItem onClick={handleShare} disabled={!conn.isOwner}>
          <ListItemIcon><ShareIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Share</ListItemText>
        </MenuItem>
        <MenuItem onClick={handleDelete} disabled={!conn.isOwner}>
          <ListItemIcon><DeleteIcon fontSize="small" color={conn.isOwner ? 'error' : undefined} /></ListItemIcon>
          <ListItemText>Delete</ListItemText>
        </MenuItem>
      </Menu>
    </>
  );
}

// --- FolderItem ---

interface FolderItemProps {
  node: FolderNode;
  connections: ConnectionData[];
  folderMap: Map<string, ConnectionData[]>;
  depth: number;
  compact?: boolean;
  onEditConnection: (conn: ConnectionData) => void;
  onDeleteConnection: (conn: ConnectionData) => void;
  onMoveConnection: (conn: ConnectionData) => void;
  onShareConnection: (conn: ConnectionData) => void;
  onConnectAsConnection: (conn: ConnectionData) => void;
  onToggleFavorite: (conn: ConnectionData) => void;
  onCreateConnection: (folderId: string) => void;
  onCreateFolder: (parentId?: string) => void;
  onEditFolder: (folder: Folder) => void;
  onDeleteFolder: (folder: Folder) => void;
}

function FolderItem({
  node, connections, folderMap, depth, compact,
  onEditConnection, onDeleteConnection, onMoveConnection, onShareConnection, onConnectAsConnection, onToggleFavorite,
  onCreateConnection, onCreateFolder, onEditFolder, onDeleteFolder,
}: FolderItemProps) {
  const [open, setOpen] = useState(true);
  const [contextMenu, setContextMenu] = useState<{ mouseX: number; mouseY: number } | null>(null);

  const handleContextMenu = (event: React.MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
    setContextMenu({ mouseX: event.clientX - 2, mouseY: event.clientY - 4 });
  };

  const handleCloseMenu = () => setContextMenu(null);

  return (
    <>
      <ListItemButton
        dense
        onClick={() => setOpen(!open)}
        onContextMenu={handleContextMenu}
        sx={{ pl: depthPl(depth), ...(compact && { py: 0.125 }) }}
      >
        <ListItemIcon sx={{ minWidth: compact ? 24 : 32 }}>
          {open ? <FolderOpenIcon fontSize="small" /> : <FolderIcon fontSize="small" />}
        </ListItemIcon>
        <ListItemText
          primary={node.folder.name}
          primaryTypographyProps={{ variant: 'body2' }}
        />
        {open ? <ExpandMore fontSize="small" /> : <ChevronRight fontSize="small" />}
      </ListItemButton>

      <Menu
        open={contextMenu !== null}
        onClose={handleCloseMenu}
        anchorReference="anchorPosition"
        anchorPosition={
          contextMenu !== null
            ? { top: contextMenu.mouseY, left: contextMenu.mouseX }
            : undefined
        }
      >
        <MenuItem onClick={() => { handleCloseMenu(); onCreateConnection(node.folder.id); }}>
          <ListItemIcon><AddIcon fontSize="small" /></ListItemIcon>
          <ListItemText>New Connection</ListItemText>
        </MenuItem>
        <MenuItem onClick={() => { handleCloseMenu(); onCreateFolder(node.folder.id); }}>
          <ListItemIcon><CreateNewFolderIcon fontSize="small" /></ListItemIcon>
          <ListItemText>New Subfolder</ListItemText>
        </MenuItem>
        <Divider />
        <MenuItem onClick={() => { handleCloseMenu(); onEditFolder(node.folder); }}>
          <ListItemIcon><EditIcon fontSize="small" /></ListItemIcon>
          <ListItemText>Rename</ListItemText>
        </MenuItem>
        <MenuItem onClick={() => { handleCloseMenu(); onDeleteFolder(node.folder); }}>
          <ListItemIcon><DeleteIcon fontSize="small" color="error" /></ListItemIcon>
          <ListItemText>Delete</ListItemText>
        </MenuItem>
      </Menu>

      <Collapse in={open}>
        <List disablePadding>
          {node.children.map((child) => (
            <FolderItem
              key={child.folder.id}
              node={child}
              connections={folderMap.get(child.folder.id) || []}
              folderMap={folderMap}
              depth={depth + 1}
              compact={compact}
              onEditConnection={onEditConnection}
              onDeleteConnection={onDeleteConnection}
              onMoveConnection={onMoveConnection}
              onShareConnection={onShareConnection}
              onConnectAsConnection={onConnectAsConnection}
              onToggleFavorite={onToggleFavorite}
              onCreateConnection={onCreateConnection}
              onCreateFolder={onCreateFolder}
              onEditFolder={onEditFolder}
              onDeleteFolder={onDeleteFolder}
            />
          ))}
          {connections.map((conn) => (
            <ConnectionItem
              key={conn.id}
              conn={conn}
              depth={depth + 1}
              compact={compact}
              onEdit={onEditConnection}
              onDelete={onDeleteConnection}
              onMove={onMoveConnection}
              onShare={onShareConnection}
              onConnectAs={onConnectAsConnection}
              onToggleFavorite={onToggleFavorite}
            />
          ))}
        </List>
      </Collapse>
    </>
  );
}

// --- ConnectionTree ---

interface ConnectionTreeProps {
  onEditConnection: (conn: ConnectionData) => void;
  onShareConnection: (conn: ConnectionData) => void;
  onConnectAsConnection: (conn: ConnectionData) => void;
  onCreateConnection: (folderId?: string) => void;
  onCreateFolder: (parentId?: string) => void;
  onEditFolder: (folder: Folder) => void;
}

export default function ConnectionTree({ onEditConnection, onShareConnection, onConnectAsConnection, onCreateConnection, onCreateFolder, onEditFolder }: ConnectionTreeProps) {
  const ownConnections = useConnectionsStore((s) => s.ownConnections);
  const sharedConnections = useConnectionsStore((s) => s.sharedConnections);
  const folders = useConnectionsStore((s) => s.folders);
  const fetchConnections = useConnectionsStore((s) => s.fetchConnections);
  const toggleFav = useConnectionsStore((s) => s.toggleFavorite);
  const userId = useAuthStore((s) => s.user?.id);
  const recentTick = useTabsStore((s) => s.recentTick);
  const notify = useNotificationStore((s) => s.notify);
  const [searchQuery, setSearchQuery] = useState('');
  const [favoritesOpen, setFavoritesOpen] = useState(true);
  const [recentsOpen, setRecentsOpen] = useState(true);
  const [compact, setCompact] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<ConnectionData | null>(null);
  const [deleteFolderTarget, setDeleteFolderTarget] = useState<Folder | null>(null);
  const [moveTarget, setMoveTarget] = useState<ConnectionData | null>(null);
  const [moveDestination, setMoveDestination] = useState('');

  const handleToggleFavorite = async (conn: ConnectionData) => {
    await toggleFav(conn.id);
  };

  const handleOpenMoveDialog = (conn: ConnectionData) => {
    setMoveTarget(conn);
    setMoveDestination(conn.folderId || '');
  };

  const handleConfirmMove = async () => {
    if (!moveTarget) return;
    const newFolderId = moveDestination || null;
    if (newFolderId === moveTarget.folderId) {
      setMoveTarget(null);
      return;
    }
    try {
      await updateConnection(moveTarget.id, { folderId: newFolderId });
      await fetchConnections();
    } catch (err) {
      notify(getErrorMessage(err, 'Failed to move connection'));
    }
    setMoveTarget(null);
  };

  const handleConfirmDelete = async () => {
    if (!deleteTarget) return;
    try {
      await deleteConnection(deleteTarget.id);
      await fetchConnections();
    } catch (err) {
      notify(getErrorMessage(err, 'Failed to delete connection'));
    }
    setDeleteTarget(null);
  };

  const handleConfirmDeleteFolder = async () => {
    if (!deleteFolderTarget) return;
    try {
      await deleteFolder(deleteFolderTarget.id);
      await fetchConnections();
    } catch (err) {
      notify(getErrorMessage(err, 'Failed to delete folder'));
    }
    setDeleteFolderTarget(null);
  };

  // Filter and group connections by folder
  const { filteredRootConnections, filteredFolderMap, filteredFolderTree, filteredSharedConnections } = useMemo(() => {
    const isSearching = searchQuery.trim().length > 0;
    const filteredOwn = isSearching ? ownConnections.filter((c) => matchesSearch(c, searchQuery)) : ownConnections;
    const filteredShared = isSearching ? sharedConnections.filter((c) => matchesSearch(c, searchQuery)) : sharedConnections;

    const rootConns = filteredOwn.filter((c) => !c.folderId);
    const fMap = new Map<string, ConnectionData[]>();
    filteredOwn.forEach((c) => {
      if (c.folderId) {
        const list = fMap.get(c.folderId) || [];
        list.push(c);
        fMap.set(c.folderId, list);
      }
    });

    const fullTree = buildFolderTree(folders);
    const prunedTree = isSearching ? pruneFolderTree(fullTree, fMap) : fullTree;

    return {
      filteredRootConnections: rootConns,
      filteredFolderMap: fMap,
      filteredFolderTree: prunedTree,
      filteredSharedConnections: filteredShared,
    };
  }, [ownConnections, sharedConnections, folders, searchQuery]);

  const favoriteConnections = useMemo(() => {
    return ownConnections.filter((c) => c.isFavorite);
  }, [ownConnections]);

  const recentConnections = useMemo(() => {
    if (!userId) return [];
    const recentIds = getRecentConnectionIds(userId);
    const allConnections = [...ownConnections, ...sharedConnections];
    const connectionMap = new Map(allConnections.map((c) => [c.id, c]));
    return recentIds
      .map((id) => connectionMap.get(id))
      .filter((c): c is ConnectionData => c !== undefined)
      .slice(0, 5);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ownConnections, sharedConnections, userId, recentTick]);

  const isSearching = searchQuery.trim().length > 0;

  return (
    <Box sx={{ py: 1 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', px: 2, mb: 1 }}>
        <Typography variant="subtitle2" sx={{ flexGrow: 1 }}>
          My Connections
        </Typography>
        <IconButton size="small" onClick={() => setCompact((v) => !v)} title={compact ? 'Normal view' : 'Compact view'}>
          {compact ? <ViewListIcon fontSize="small" /> : <ViewCompactIcon fontSize="small" />}
        </IconButton>
        <IconButton size="small" onClick={() => onCreateConnection()} title="New Connection">
          <AddIcon fontSize="small" />
        </IconButton>
        <IconButton size="small" onClick={() => onCreateFolder()} title="New Folder">
          <CreateNewFolderIcon fontSize="small" />
        </IconButton>
      </Box>
      <Box sx={{ px: 2, mb: 1 }}>
        <TextField
          size="small"
          fullWidth
          placeholder="Search connections..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Escape') setSearchQuery(''); }}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon fontSize="small" color="action" />
                </InputAdornment>
              ),
              endAdornment: searchQuery ? (
                <InputAdornment position="end">
                  <IconButton size="small" onClick={() => setSearchQuery('')} edge="end">
                    <ClearIcon fontSize="small" />
                  </IconButton>
                </InputAdornment>
              ) : null,
            },
          }}
        />
      </Box>

      {/* Favorites section */}
      {!isSearching && favoriteConnections.length > 0 && (
        <>
          <Box
            sx={{ display: 'flex', alignItems: 'center', px: 2, mt: 1, mb: 0.5, cursor: 'pointer', userSelect: 'none' }}
            onClick={() => setFavoritesOpen((prev) => !prev)}
          >
            {favoritesOpen ? <ExpandMore sx={{ fontSize: 18, mr: 0.5 }} /> : <ChevronRight sx={{ fontSize: 18, mr: 0.5 }} />}
            <StarIcon fontSize="small" color="warning" sx={{ mr: 1 }} />
            <Typography variant="subtitle2">Favorites</Typography>
          </Box>
          <Collapse in={favoritesOpen}>
            <List disablePadding>
              {favoriteConnections.map((conn) => (
                <ConnectionItem
                  key={`fav-${conn.id}`}
                  conn={conn}
                  depth={0}
                  compact={compact}
                  onEdit={onEditConnection}
                  onDelete={setDeleteTarget}
                  onMove={handleOpenMoveDialog}
                  onShare={onShareConnection}
                  onConnectAs={onConnectAsConnection}
                  onToggleFavorite={handleToggleFavorite}
                />
              ))}
            </List>
          </Collapse>
        </>
      )}

      {/* Recent section */}
      {!isSearching && recentConnections.length > 0 && (
        <>
          <Box
            sx={{ display: 'flex', alignItems: 'center', px: 2, mt: 1, mb: 0.5, cursor: 'pointer', userSelect: 'none' }}
            onClick={() => setRecentsOpen((prev) => !prev)}
          >
            {recentsOpen ? <ExpandMore sx={{ fontSize: 18, mr: 0.5 }} /> : <ChevronRight sx={{ fontSize: 18, mr: 0.5 }} />}
            <RecentIcon fontSize="small" sx={{ mr: 1 }} />
            <Typography variant="subtitle2">Recent</Typography>
          </Box>
          <Collapse in={recentsOpen}>
            <List disablePadding>
              {recentConnections.map((conn) => (
                <ConnectionItem
                  key={`recent-${conn.id}`}
                  conn={conn}
                  depth={0}
                  compact={compact}
                  onEdit={onEditConnection}
                  onDelete={setDeleteTarget}
                  onMove={handleOpenMoveDialog}
                  onShare={onShareConnection}
                  onConnectAs={onConnectAsConnection}
                  onToggleFavorite={conn.isOwner ? handleToggleFavorite : undefined}
                />
              ))}
            </List>
          </Collapse>
        </>
      )}

      {/* Divider between quick-access sections and main tree */}
      {!isSearching && (favoriteConnections.length > 0 || recentConnections.length > 0) && (
        <Divider sx={{ my: 1 }} />
      )}

      <List disablePadding>
        {filteredFolderTree.map((node) => (
          <FolderItem
            key={node.folder.id}
            node={node}
            connections={filteredFolderMap.get(node.folder.id) || []}
            folderMap={filteredFolderMap}
            depth={0}
            compact={compact}
            onEditConnection={onEditConnection}
            onDeleteConnection={setDeleteTarget}
            onMoveConnection={handleOpenMoveDialog}
            onShareConnection={onShareConnection}
            onConnectAsConnection={onConnectAsConnection}
            onToggleFavorite={handleToggleFavorite}
            onCreateConnection={onCreateConnection}
            onCreateFolder={onCreateFolder}
            onEditFolder={onEditFolder}
            onDeleteFolder={setDeleteFolderTarget}
          />
        ))}
        {filteredRootConnections.map((conn) => (
          <ConnectionItem
            key={conn.id}
            conn={conn}
            depth={0}
            compact={compact}
            onEdit={onEditConnection}
            onDelete={setDeleteTarget}
            onMove={handleOpenMoveDialog}
            onShare={onShareConnection}
            onConnectAs={onConnectAsConnection}
            onToggleFavorite={handleToggleFavorite}
          />
        ))}
      </List>

      {searchQuery.trim() && filteredRootConnections.length === 0 && filteredFolderTree.length === 0 && filteredSharedConnections.length === 0 && (
        <Typography variant="body2" color="text.secondary" sx={{ px: 2, py: 2, textAlign: 'center' }}>
          No connections match your search.
        </Typography>
      )}

      {filteredSharedConnections.length > 0 && (
        <>
          <Box sx={{ display: 'flex', alignItems: 'center', px: 2, mt: 2, mb: 1 }}>
            <ShareIcon fontSize="small" sx={{ mr: 1 }} />
            <Typography variant="subtitle2">Shared with me</Typography>
          </Box>
          <List disablePadding>
            {filteredSharedConnections.map((conn) => (
              <ConnectionItem
                key={conn.id}
                conn={conn}
                depth={0}
                compact={compact}
                onEdit={onEditConnection}
                onDelete={setDeleteTarget}
                onMove={handleOpenMoveDialog}
                onShare={onShareConnection}
                onConnectAs={onConnectAsConnection}
                onToggleFavorite={handleToggleFavorite}
              />
            ))}
          </List>
        </>
      )}

      {/* Move to Folder dialog */}
      <Dialog open={moveTarget !== null} onClose={() => setMoveTarget(null)} maxWidth="xs" fullWidth>
        <DialogTitle>Move &quot;{moveTarget?.name}&quot;</DialogTitle>
        <DialogContent>
          <FormControl fullWidth sx={{ mt: 1 }}>
            <InputLabel>Destination Folder</InputLabel>
            <Select
              value={moveDestination}
              label="Destination Folder"
              onChange={(e: SelectChangeEvent) => setMoveDestination(e.target.value)}
            >
              <MenuItem value="">Root (no folder)</MenuItem>
              {folders.map((f) => (
                <MenuItem key={f.id} value={f.id}>{f.name}</MenuItem>
              ))}
            </Select>
          </FormControl>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setMoveTarget(null)}>Cancel</Button>
          <Button onClick={handleConfirmMove} variant="contained">Move</Button>
        </DialogActions>
      </Dialog>

      {/* Delete connection confirmation */}
      <Dialog open={deleteTarget !== null} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Delete Connection</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete &quot;{deleteTarget?.name}&quot;? This action cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button onClick={handleConfirmDelete} color="error" variant="contained">Delete</Button>
        </DialogActions>
      </Dialog>

      {/* Delete folder confirmation */}
      <Dialog open={deleteFolderTarget !== null} onClose={() => setDeleteFolderTarget(null)}>
        <DialogTitle>Delete Folder</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete &quot;{deleteFolderTarget?.name}&quot;?
            Connections in this folder will be moved to the root level.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteFolderTarget(null)}>Cancel</Button>
          <Button onClick={handleConfirmDeleteFolder} color="error" variant="contained">Delete</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
