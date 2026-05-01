import type { RefObject } from 'react';
import { Button } from '@/components/ui/button';
import {
  ChevronRight,
  Download,
  FileText,
  Folder,
  FolderPlus,
  Link as LinkIcon,
  Loader2,
  Pencil,
  RefreshCw,
  Trash2,
  Upload,
} from 'lucide-react';
import type { SshFileEntry } from '../../api/sshFiles.api';
import { formatDate, formatFileSize } from './sftpBrowserUtils';

interface SftpWorkspacePaneProps {
  currentPath: string;
  entries: SshFileEntry[];
  loading: boolean;
  disableUpload?: boolean;
  disableDownload?: boolean;
  dragOver: boolean;
  fileInputRef: RefObject<HTMLInputElement | null>;
  onNavigateTo: (path: string) => void;
  onRefresh: () => void;
  onFileSelect: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onOpenCreateFolder: () => void;
  onDownload: (entry: SshFileEntry) => void;
  onRename: (entry: SshFileEntry) => void;
  onDelete: (entry: SshFileEntry) => void;
  onDragOver: (event: React.DragEvent) => void;
  onDragLeave: () => void;
  onDrop: (event: React.DragEvent) => void;
}

export default function SftpWorkspacePane({
  currentPath,
  entries,
  loading,
  disableUpload,
  disableDownload,
  dragOver,
  fileInputRef,
  onNavigateTo,
  onRefresh,
  onFileSelect,
  onOpenCreateFolder,
  onDownload,
  onRename,
  onDelete,
  onDragOver,
  onDragLeave,
  onDrop,
}: SftpWorkspacePaneProps) {
  const pathSegments = currentPath.split('/').filter(Boolean);

  const entryIcon = (type: SshFileEntry['type']) => {
    switch (type) {
      case 'directory':
        return <Folder className="h-4 w-4 text-primary" />;
      case 'symlink':
        return <LinkIcon className="h-4 w-4 text-muted-foreground" />;
      default:
        return <FileText className="h-4 w-4" />;
    }
  };

  return (
    <div
      className="flex min-h-0 flex-1 flex-col"
      onDragOver={disableUpload ? undefined : onDragOver}
      onDragLeave={disableUpload ? undefined : onDragLeave}
      onDrop={disableUpload ? undefined : onDrop}
    >
      <div className="px-3 py-2">
        <nav className="flex items-center gap-0.5 text-xs">
          <button
            className="text-muted-foreground hover:underline"
            onClick={() => onNavigateTo('')}
            type="button"
          >
            Workspace
          </button>
          {pathSegments.map((segment, index) => {
            const segmentPath = pathSegments.slice(0, index + 1).join('/');
            const isLast = index === pathSegments.length - 1;
            return (
              <span key={segmentPath} className="flex items-center gap-0.5">
                <ChevronRight className="h-3 w-3 text-muted-foreground" />
                {isLast ? (
                  <span className="text-foreground">{segment}</span>
                ) : (
                  <button
                    className="text-muted-foreground hover:underline"
                    onClick={() => onNavigateTo(segmentPath)}
                    type="button"
                  >
                    {segment}
                  </button>
                )}
              </span>
            );
          })}
        </nav>
      </div>

      <div className="flex gap-1 px-3 pb-2">
        {!disableUpload && (
          <>
            <input
              type="file"
              multiple
              ref={fileInputRef}
              onChange={onFileSelect}
              className="hidden"
            />
            <Button
              variant="outline"
              size="sm"
              className="flex-1 text-xs"
              onClick={() => fileInputRef.current?.click()}
            >
              <Upload className="mr-1 h-3.5 w-3.5" />
              Upload
            </Button>
          </>
        )}
        <Button
          variant="outline"
          size="sm"
          className="flex-1 text-xs"
          onClick={onOpenCreateFolder}
        >
          <FolderPlus className="mr-1 h-3.5 w-3.5" />
          New Folder
        </Button>
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onRefresh} disabled={loading}>
          <RefreshCw className="h-3.5 w-3.5" />
        </Button>
      </div>

      {dragOver && !disableUpload && (
        <div className="mx-3 rounded border-2 border-dashed border-primary bg-muted/50 p-3 text-center">
          <p className="text-sm text-primary">Drop files here to upload</p>
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-auto">
        {loading ? (
          <div className="flex justify-center p-6">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          </div>
        ) : entries.length === 0 ? (
          <p className="p-4 text-center text-sm text-muted-foreground">
            This workspace is empty.
          </p>
        ) : (
          <div>
            {entries.map((entry) => (
              <div
                key={entry.name}
                className="group flex cursor-default items-center gap-2 px-3 py-1.5 hover:bg-muted/50"
                onDoubleClick={() => {
                  if (entry.type === 'directory' || entry.type === 'symlink') {
                    onNavigateTo(currentPath ? `${currentPath}/${entry.name}` : entry.name);
                  }
                }}
              >
                {entryIcon(entry.type)}
                <div className="min-w-0 flex-1">
                  <p className="truncate text-[0.85rem]">{entry.name}</p>
                  <p className="text-[0.75rem] text-muted-foreground">
                    {entry.type === 'directory' ? 'Folder' : formatFileSize(entry.size)} • {formatDate(entry.modifiedAt)}
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                  {entry.type === 'file' && !disableDownload && (
                    <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => onDownload(entry)} title="Download">
                      <Download className="h-3.5 w-3.5" />
                    </Button>
                  )}
                  <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => onRename(entry)} title="Rename">
                    <Pencil className="h-3.5 w-3.5" />
                  </Button>
                  <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => onDelete(entry)} title="Delete">
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
