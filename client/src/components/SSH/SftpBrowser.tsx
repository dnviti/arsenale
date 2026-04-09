import { useState, useEffect, useRef, useCallback } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Separator } from '@/components/ui/separator';
import {
  X,
  Download,
  Trash2,
  Upload,
  Folder,
  FileText,
  Link as LinkIcon,
  FolderPlus,
  RefreshCw,
  ChevronRight,
  Home,
  Loader2,
} from 'lucide-react';
import { useSftpTransfers, type SftpSocket } from '../../hooks/useSftpTransfers';
import SftpTransferQueue from './SftpTransferQueue';

interface SftpEntry {
  name: string;
  size: number;
  type: 'file' | 'directory' | 'symlink';
  modifiedAt: string;
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - d.getTime();
  if (diffMs < 86400000) {
    return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
  }
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: d.getFullYear() !== now.getFullYear() ? 'numeric' : undefined });
}

interface SftpBrowserProps {
  open: boolean;
  onClose: () => void;
  socket: SftpSocket | null;
  disableDownload?: boolean;
  disableUpload?: boolean;
}

export default function SftpBrowser({ open, onClose, socket, disableDownload, disableUpload }: SftpBrowserProps) {
  const [currentPath, setCurrentPath] = useState('/');
  const [entries, setEntries] = useState<SftpEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [dragOver, setDragOver] = useState(false);
  const [mkdirOpen, setMkdirOpen] = useState(false);
  const [mkdirName, setMkdirName] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<SftpEntry | null>(null);
  const [renameTarget, setRenameTarget] = useState<SftpEntry | null>(null);
  const [renameName, setRenameName] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);
  const refreshedTransfers = useRef<Set<string>>(new Set());

  const { transfers, uploadFile, downloadFile, cancelTransfer, clearCompleted } = useSftpTransfers(socket);

  const fetchEntries = useCallback((dirPath: string) => {
    if (!socket?.connected) return;
    setLoading(true);
    setError('');
    socket.emit('sftp:list', { path: dirPath }, (res: { entries?: SftpEntry[]; error?: string }) => {
      setLoading(false);
      if (res.error) {
        setError(res.error);
        return;
      }
      const sorted = (res.entries || []).sort((a, b) => {
        if (a.type === 'directory' && b.type !== 'directory') return -1;
        if (a.type !== 'directory' && b.type === 'directory') return 1;
        return a.name.localeCompare(b.name);
      });
      setEntries(sorted);
    });
  }, [socket]);

  /* eslint-disable react-hooks/set-state-in-effect -- triggers data fetch when drawer opens or path changes */
  useEffect(() => {
    if (open) fetchEntries(currentPath);
  }, [open, currentPath, fetchEntries]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const navigateTo = (newPath: string) => {
    setCurrentPath(newPath);
  };

  const handleEntryDoubleClick = (entry: SftpEntry) => {
    if (entry.type === 'directory' || entry.type === 'symlink') {
      const newPath = currentPath === '/'
        ? '/' + entry.name
        : currentPath + '/' + entry.name;
      navigateTo(newPath);
    }
  };

  const handleUpload = (files: FileList | null) => {
    if (!files) return;
    for (let i = 0; i < files.length; i++) {
      uploadFile(files[i], currentPath);
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    handleUpload(e.target.files);
    e.target.value = '';
  };

  const handleDownload = (entry: SftpEntry) => {
    const fullPath = currentPath === '/' ? '/' + entry.name : currentPath + '/' + entry.name;
    downloadFile(fullPath);
  };

  const handleMkdir = () => {
    if (!socket?.connected || !mkdirName.trim()) return;
    const newPath = currentPath === '/'
      ? '/' + mkdirName.trim()
      : currentPath + '/' + mkdirName.trim();
    socket.emit('sftp:mkdir', { path: newPath }, (res: { error?: string }) => {
      if (res.error) {
        setError(res.error);
      } else {
        fetchEntries(currentPath);
      }
      setMkdirOpen(false);
      setMkdirName('');
    });
  };

  const handleDelete = () => {
    if (!socket?.connected || !deleteTarget) return;
    const fullPath = currentPath === '/'
      ? '/' + deleteTarget.name
      : currentPath + '/' + deleteTarget.name;
    const event = deleteTarget.type === 'directory' ? 'sftp:rmdir' : 'sftp:delete';
    socket.emit(event, { path: fullPath }, (res: { error?: string }) => {
      if (res.error) {
        setError(res.error);
      } else {
        fetchEntries(currentPath);
      }
      setDeleteTarget(null);
    });
  };

  const handleRename = () => {
    if (!socket?.connected || !renameTarget || !renameName.trim()) return;
    const oldPath = currentPath === '/'
      ? '/' + renameTarget.name
      : currentPath + '/' + renameTarget.name;
    const newPath = currentPath === '/'
      ? '/' + renameName.trim()
      : currentPath + '/' + renameName.trim();
    socket.emit('sftp:rename', { oldPath, newPath }, (res: { error?: string }) => {
      if (res.error) {
        setError(res.error);
      } else {
        fetchEntries(currentPath);
      }
      setRenameTarget(null);
      setRenameName('');
    });
  };

  // Refresh listing when an upload completes
  /* eslint-disable react-hooks/set-state-in-effect -- triggers refresh when upload completes */
  useEffect(() => {
    let shouldRefresh = false;
    for (const t of transfers) {
      if (t.status === 'complete' && t.direction === 'upload' && !refreshedTransfers.current.has(t.transferId)) {
        refreshedTransfers.current.add(t.transferId);
        const dir = t.remotePath.substring(0, t.remotePath.lastIndexOf('/')) || '/';
        if (dir === currentPath) {
          shouldRefresh = true;
        }
      }
    }
    if (shouldRefresh) {
      fetchEntries(currentPath);
    }
  }, [transfers, currentPath, fetchEntries]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(true);
  };

  const handleDragLeave = () => setDragOver(false);

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    handleUpload(e.dataTransfer.files);
  };

  // Build breadcrumb segments
  const pathSegments = currentPath === '/' ? [] : currentPath.split('/').filter(Boolean);

  const entryIcon = (type: SftpEntry['type']) => {
    switch (type) {
      case 'directory': return <Folder className="h-4 w-4 text-primary" />;
      case 'symlink': return <LinkIcon className="h-4 w-4 text-muted-foreground" />;
      default: return <FileText className="h-4 w-4" />;
    }
  };

  if (!open) return null;

  return (
    <>
      <div
        className="absolute right-0 top-0 bottom-0 w-[360px] border-l bg-background flex flex-col z-10"
        onDragOver={disableUpload ? undefined : handleDragOver}
        onDragLeave={disableUpload ? undefined : handleDragLeave}
        onDrop={disableUpload ? undefined : handleDrop}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-3 pl-4">
          <span className="text-sm font-semibold">SFTP</span>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
        </div>

        <Separator />

        {/* Breadcrumb */}
        <div className="px-3 py-2 overflow-auto">
          <nav className="flex items-center gap-0.5 text-xs">
            <button
              className="flex items-center hover:underline text-muted-foreground"
              onClick={() => navigateTo('/')}
            >
              <Home className="h-3.5 w-3.5" />
            </button>
            {pathSegments.map((segment, idx) => {
              const segPath = '/' + pathSegments.slice(0, idx + 1).join('/');
              const isLast = idx === pathSegments.length - 1;
              return (
                <span key={segPath} className="flex items-center gap-0.5">
                  <ChevronRight className="h-3 w-3 text-muted-foreground" />
                  {isLast ? (
                    <span className="text-foreground">{segment}</span>
                  ) : (
                    <button
                      className="hover:underline text-muted-foreground"
                      onClick={() => navigateTo(segPath)}
                    >
                      {segment}
                    </button>
                  )}
                </span>
              );
            })}
          </nav>
        </div>

        <Separator />

        {/* Action bar */}
        <div className="flex gap-1 p-2 px-3">
          {!disableUpload && (
            <>
              <input
                type="file"
                multiple
                ref={fileInputRef}
                onChange={handleFileSelect}
                className="hidden"
              />
              <Button
                variant="outline"
                size="sm"
                className="flex-1 text-xs"
                onClick={() => fileInputRef.current?.click()}
              >
                <Upload className="h-3.5 w-3.5 mr-1" />
                Upload
              </Button>
            </>
          )}
          <Button
            variant="outline"
            size="sm"
            className="flex-1 text-xs"
            onClick={() => setMkdirOpen(true)}
          >
            <FolderPlus className="h-3.5 w-3.5 mr-1" />
            New Folder
          </Button>
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => fetchEntries(currentPath)} disabled={loading}>
            <RefreshCw className="h-3.5 w-3.5" />
          </Button>
        </div>

        {/* Drag overlay */}
        {dragOver && !disableUpload && (
          <div className="mx-3 p-3 border-2 border-dashed border-primary rounded text-center bg-muted/50">
            <p className="text-sm text-primary">Drop files here to upload</p>
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="mx-3 mb-2 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400 flex items-center justify-between">
            <span>{error}</span>
            <button onClick={() => setError('')} className="text-red-400 hover:text-red-300">
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        )}

        {/* File list */}
        <div className="flex-1 overflow-auto">
          {loading ? (
            <div className="flex justify-center p-6">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          ) : entries.length === 0 ? (
            <p className="text-sm text-muted-foreground p-4 text-center">
              This directory is empty
            </p>
          ) : (
            <div>
              {entries.map((entry) => (
                <div
                  key={entry.name}
                  onDoubleClick={() => handleEntryDoubleClick(entry)}
                  className={`flex items-center gap-2 px-3 py-1.5 hover:bg-muted/50 ${entry.type === 'directory' || entry.type === 'symlink' ? 'cursor-pointer' : ''}`}
                >
                  <div className="w-5 shrink-0">{entryIcon(entry.type)}</div>
                  <div className="flex-1 min-w-0">
                    <p className="text-[0.85rem] truncate">{entry.name}</p>
                    <p className="text-[0.7rem] text-muted-foreground">
                      {entry.type === 'directory'
                        ? formatDate(entry.modifiedAt)
                        : `${formatFileSize(entry.size)} - ${formatDate(entry.modifiedAt)}`}
                    </p>
                  </div>
                  <div className="flex items-center gap-0.5 shrink-0">
                    {entry.type === 'file' && !disableDownload && (
                      <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => handleDownload(entry)} title="Download">
                        <Download className="h-3.5 w-3.5" />
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 text-xs"
                      onClick={() => {
                        setRenameTarget(entry);
                        setRenameName(entry.name);
                      }}
                      title="Rename"
                    >
                      <span className="text-[0.7rem]">Aa</span>
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6"
                      onClick={() => setDeleteTarget(entry)}
                      title="Delete"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Transfer Queue */}
        <SftpTransferQueue
          transfers={transfers}
          onCancel={cancelTransfer}
          onClearCompleted={clearCompleted}
        />
      </div>

      {/* New Folder Dialog */}
      <Dialog open={mkdirOpen} onOpenChange={(v) => { if (!v) setMkdirOpen(false); }}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>New Folder</DialogTitle>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="folder-name">Folder name</Label>
            <Input
              id="folder-name"
              autoFocus
              value={mkdirName}
              onChange={(e) => setMkdirName(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') handleMkdir(); }}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setMkdirOpen(false)}>Cancel</Button>
            <Button onClick={handleMkdir} disabled={!mkdirName.trim()}>Create</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation */}
      <Dialog open={!!deleteTarget} onOpenChange={(v) => { if (!v) setDeleteTarget(null); }}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Delete {deleteTarget?.type === 'directory' ? 'Folder' : 'File'}</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deleteTarget?.name}&quot;?
              {deleteTarget?.type === 'directory' && ' The directory must be empty.'}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>Cancel</Button>
            <Button variant="destructive" onClick={handleDelete}>Delete</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Rename Dialog */}
      <Dialog open={!!renameTarget} onOpenChange={(v) => { if (!v) setRenameTarget(null); }}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>Rename</DialogTitle>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="rename-input">New name</Label>
            <Input
              id="rename-input"
              autoFocus
              value={renameName}
              onChange={(e) => setRenameName(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') handleRename(); }}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRenameTarget(null)}>Cancel</Button>
            <Button onClick={handleRename} disabled={!renameName.trim() || renameName === renameTarget?.name}>
              Rename
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
