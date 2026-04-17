import { Button } from '@/components/ui/button';
import { Download, History, RotateCcw, Trash2, Loader2 } from 'lucide-react';
import type { ManagedHistoryEntry } from '../../api/managedHistory.api';
import { formatManagedFileSize, formatManagedTimestamp } from './managedSandboxUi';

interface ManagedHistoryListProps {
  items: ManagedHistoryEntry[];
  loading: boolean;
  emptyMessage: string;
  disableDownload?: boolean;
  disableRestore?: boolean;
  onDownload: (item: ManagedHistoryEntry) => void;
  onRestore: (item: ManagedHistoryEntry) => void;
  onDelete: (item: ManagedHistoryEntry) => void;
}

export default function ManagedHistoryList({
  items,
  loading,
  emptyMessage,
  disableDownload = false,
  disableRestore = false,
  onDownload,
  onRestore,
  onDelete,
}: ManagedHistoryListProps) {
  if (loading) {
    return (
      <div className="flex justify-center p-6">
        <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 p-6 text-center">
        <History className="h-5 w-5 text-muted-foreground" />
        <p className="text-sm text-muted-foreground">{emptyMessage}</p>
      </div>
    );
  }

  return (
    <div>
      {items.map((item) => (
        <div
          key={item.id}
          className="group flex items-center gap-2 px-3 py-2 hover:bg-muted/50"
        >
          <History className="h-4 w-4 shrink-0 text-muted-foreground" />
          <div className="min-w-0 flex-1">
            <p className="truncate text-[0.85rem]">{item.fileName}</p>
            <p className="text-[0.75rem] text-muted-foreground">
              {formatManagedFileSize(item.size)} • {formatManagedTimestamp(item.transferAt)}
              {item.restoredName ? ` • Restored as ${item.restoredName}` : ''}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
            {!disableDownload && (
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => onDownload(item)}
                title="Download"
              >
                <Download className="h-3.5 w-3.5" />
              </Button>
            )}
            {!disableRestore && (
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => onRestore(item)}
                title="Restore"
              >
                <RotateCcw className="h-3.5 w-3.5" />
              </Button>
            )}
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={() => onDelete(item)}
              title="Delete"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}
