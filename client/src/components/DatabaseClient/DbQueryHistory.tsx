import { useState, useEffect, useCallback, useMemo } from 'react';
import {
  History, RefreshCw, Search, ChevronLeft, Clock, Ban, Bookmark, Trash2,
} from 'lucide-react';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog';
import { cn } from '@/lib/utils';
import { getQueryHistory, type QueryHistoryEntry } from '../../api/database.api';
import {
  deleteSavedQuery,
  deriveQueryLabel,
  loadSavedQueries,
  type SavedQuery,
} from './dbQueryHistoryUtils';

// ---------------------------------------------------------------------------
// Props & helpers
// ---------------------------------------------------------------------------

interface DbQueryHistoryProps {
  open: boolean;
  onClose: () => void;
  sessionId: string | null;
  connectionId: string;
  onSelectQuery: (sql: string) => void;
  refreshTrigger?: number;
  onSaveRequest?: () => void;
}

function formatRelativeTime(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const seconds = Math.floor(diff / 1000);
  if (seconds < 60) return 'just now';
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

const TYPE_COLORS: Record<string, string> = {
  SELECT: 'text-blue-400 border-blue-500/30',
  INSERT: 'text-green-400 border-green-500/30',
  UPDATE: 'text-yellow-400 border-yellow-500/30',
  DELETE: 'text-red-400 border-red-500/30',
  DDL: 'text-primary border-primary/30',
  OTHER: 'text-muted-foreground border-border',
};

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default function DbQueryHistory({
  open,
  onClose,
  sessionId,
  connectionId,
  onSelectQuery,
  refreshTrigger = 0,
}: DbQueryHistoryProps) {
  const [entries, setEntries] = useState<QueryHistoryEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [search, setSearch] = useState('');
  const [searchDebounced, setSearchDebounced] = useState('');
  const [savedQueries, setSavedQueries] = useState<SavedQuery[]>([]);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  const reloadSaved = useCallback(() => {
    if (connectionId) setSavedQueries(loadSavedQueries(connectionId));
  }, [connectionId]);

  const fetchHistory = useCallback(async (searchTerm?: string) => {
    if (!sessionId) return;
    setLoading(true);
    try {
      const result = await getQueryHistory(sessionId, 100, searchTerm || undefined);
      setEntries(result);
    } catch {
      // Silently ignore — history is best-effort
    } finally {
      setLoading(false);
    }
  }, [sessionId]);

  // Fetch on open, search change, or refresh trigger
  useEffect(() => {
    if (open && sessionId) {
      fetchHistory(searchDebounced);
      reloadSaved();
    }
  }, [open, sessionId, searchDebounced, fetchHistory, refreshTrigger, reloadSaved]);

  // Debounce search input
  useEffect(() => {
    const timer = setTimeout(() => setSearchDebounced(search), 300);
    return () => clearTimeout(timer);
  }, [search]);

  // Filter saved queries by search
  const filteredSaved = useMemo(() => {
    if (!search) return savedQueries;
    const term = search.toLowerCase();
    return savedQueries.filter(
      (q) => q.name.toLowerCase().includes(term) || q.sql.toLowerCase().includes(term),
    );
  }, [savedQueries, search]);

  const handleDeleteSaved = useCallback((id: string) => {
    deleteSavedQuery(connectionId, id);
    reloadSaved();
    setDeleteConfirm(null);
  }, [connectionId, reloadSaved]);

  if (!open) return null;

  return (
    <div className="w-80 border-l border-border flex flex-col overflow-hidden bg-card">
      {/* Header */}
      <div className="px-3 py-2 flex items-center gap-2 border-b border-border">
        <History className="size-4 text-primary" />
        <span className="text-sm font-semibold flex-1">Query History</span>
        <Button
          variant="ghost"
          size="icon"
          className="size-7"
          title="Refresh"
          onClick={() => fetchHistory(searchDebounced)}
          disabled={loading}
        >
          {loading ? <Loader2 className="size-4 animate-spin" /> : <RefreshCw className="size-4" />}
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="size-7"
          title="Close"
          onClick={onClose}
        >
          <ChevronLeft className="size-4" />
        </Button>
      </div>

      {/* Search */}
      <div className="px-3 py-2">
        <div className="relative">
          <Search className="absolute left-2 top-1/2 -translate-y-1/2 size-3.5 text-muted-foreground" />
          <Input
            className="h-8 text-xs pl-7"
            placeholder="Search queries..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
      </div>

      <Separator />

      {/* Saved queries section */}
      {filteredSaved.length > 0 && (
        <>
          <div className="px-3 py-1">
            <span className="text-xs font-bold text-primary">
              Saved
            </span>
          </div>
          <div>
            {filteredSaved.map((sq) => (
              <button
                key={sq.id}
                onClick={() => onSelectQuery(sq.sql)}
                className="w-full text-left py-2 px-3 border-b border-border flex items-center gap-2 hover:bg-accent/50 transition-colors"
              >
                <Bookmark className="size-4 text-primary shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold text-[0.8rem] overflow-hidden text-ellipsis whitespace-nowrap">
                    {sq.name}
                  </p>
                  <p className="text-xs text-muted-foreground font-mono text-[0.7rem] overflow-hidden text-ellipsis whitespace-nowrap">
                    {sq.sql.replace(/\s+/g, ' ').trim()}
                  </p>
                </div>
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-5 opacity-50 hover:opacity-100"
                  title="Remove"
                  onClick={(e) => { e.stopPropagation(); setDeleteConfirm(sq.id); }}
                >
                  <Trash2 className="size-3.5" />
                </Button>
              </button>
            ))}
          </div>
          <Separator />
        </>
      )}

      {/* Recent queries section */}
      {(entries.length > 0 || filteredSaved.length > 0) && (
        <div className="px-3 py-1">
          <span className="text-xs font-bold text-muted-foreground">
            Recent
          </span>
        </div>
      )}

      <div className="flex-1 overflow-auto">
        {entries.length === 0 && filteredSaved.length === 0 && !loading && (
          <div className="p-6 text-center">
            <p className="text-sm text-muted-foreground">
              {search ? 'No matching queries' : 'No query history yet'}
            </p>
            <p className="text-xs text-muted-foreground/60 mt-1">
              Press Ctrl+S to save a query
            </p>
          </div>
        )}

        {entries.map((entry) => {
          const label = deriveQueryLabel(entry.queryText);
          return (
            <button
              key={entry.id}
              onClick={() => onSelectQuery(entry.queryText)}
              className="w-full text-left py-2 px-3 border-b border-border flex flex-col items-start gap-0.5 min-h-[52px] max-h-[72px] hover:bg-accent/50 transition-colors"
            >
              {/* Row 1: type chip + label + timestamp */}
              <div className="flex items-center gap-1 w-full">
                <Badge
                  variant="outline"
                  className={cn('h-[18px] text-[0.65rem] font-bold px-1.5 py-0', TYPE_COLORS[entry.queryType] ?? TYPE_COLORS.OTHER)}
                >
                  {entry.queryType}
                </Badge>
                {entry.blocked && (
                  <Ban className="size-3.5 text-red-400" />
                )}
                <span className="flex-1 text-[0.78rem] font-semibold overflow-hidden text-ellipsis whitespace-nowrap">
                  {label}
                </span>
                <span className="text-[0.65rem] text-muted-foreground shrink-0">
                  {formatRelativeTime(entry.createdAt)}
                </span>
              </div>

              {/* Row 2: single-line SQL preview */}
              <span className="font-mono text-[0.7rem] text-muted-foreground overflow-hidden text-ellipsis whitespace-nowrap w-full">
                {entry.queryText.replace(/\s+/g, ' ').trim()}
              </span>

              {/* Row 3: metrics */}
              <div className="flex items-center gap-2">
                {entry.executionTimeMs != null && (
                  <span className="text-[0.65rem] text-muted-foreground/60 flex items-center gap-0.5">
                    <Clock className="size-[11px]" />
                    {entry.executionTimeMs}ms
                  </span>
                )}
                {entry.rowsAffected != null && (
                  <span className="text-[0.65rem] text-muted-foreground/60">
                    {entry.rowsAffected} rows
                  </span>
                )}
              </div>
            </button>
          );
        })}
      </div>

      {/* Delete confirmation */}
      <Dialog open={!!deleteConfirm} onOpenChange={(open) => !open && setDeleteConfirm(null)}>
        <DialogContent className="max-w-xs">
          <DialogHeader>
            <DialogTitle>Remove saved query?</DialogTitle>
          </DialogHeader>
          <p className="text-sm">This will remove the query from your saved list.</p>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeleteConfirm(null)}>Cancel</Button>
            <Button variant="destructive" onClick={() => deleteConfirm && handleDeleteSaved(deleteConfirm)}>
              Remove
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
