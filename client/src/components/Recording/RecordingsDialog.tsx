import { useState, useEffect, useCallback } from 'react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import {
  ChevronLeft,
  ChevronRight,
  Download,
  Film,
  Loader2,
  Monitor,
  Play,
  Terminal,
  Trash2,
  Video,
  X,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  listRecordings,
  deleteRecording,
  exportRecordingVideo,
} from '../../api/recordings.api';
import type { Recording } from '../../api/recordings.api';
import api from '../../api/client';
import RecordingPlayerDialog from './RecordingPlayerDialog';

interface RecordingsDialogProps {
  open: boolean;
  onClose: () => void;
}

const PROTOCOL_BADGE: Record<string, string> = {
  SSH: 'bg-green-500/15 text-green-400 border-green-500/30',
  RDP: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
  VNC: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/30',
};

const PROTOCOL_ICON: Record<string, React.ElementType> = {
  SSH: Terminal,
  RDP: Monitor,
  VNC: Monitor,
};

function formatDuration(seconds: number | null) {
  if (seconds === null) return '\u2014';
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, '0')}`;
}

function formatSize(bytes: number | null) {
  if (bytes === null) return '\u2014';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

function formatDate(iso: string) {
  const d = new Date(iso);
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

function formatTime(iso: string) {
  const d = new Date(iso);
  return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
}

export default function RecordingsDialog({ open, onClose }: RecordingsDialogProps) {
  const [recordings, setRecordings] = useState<Recording[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [protocolFilter, setProtocolFilter] = useState<string>('all');
  const [page, setPage] = useState(0);
  const [playingRecording, setPlayingRecording] = useState<Recording | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Recording | null>(null);
  const [convertingIds, setConvertingIds] = useState<Set<string>>(new Set());
  const limit = 25;

  const fetchRecordings = useCallback(async () => {
    setLoading(true);
    try {
      const result = await listRecordings({
        protocol: protocolFilter === 'all' ? undefined : protocolFilter,
        status: 'COMPLETE',
        limit,
        offset: page * limit,
      });
      setRecordings(result.recordings);
      setTotal(result.total);
    } catch {
      // silently handle
    } finally {
      setLoading(false);
    }
  }, [protocolFilter, page]);

  useEffect(() => {
    if (open) fetchRecordings();
  }, [open, fetchRecordings]);

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await deleteRecording(deleteTarget.id);
      setDeleteTarget(null);
      fetchRecordings();
    } catch {
      // silently handle
    }
  };

  const handleDownload = async (rec: Recording) => {
    const { data } = await api.get(`/recordings/${rec.id}/stream`, { responseType: 'blob' });
    const url = URL.createObjectURL(data);
    const a = document.createElement('a');
    a.href = url;
    a.download = `recording-${rec.id}.${rec.format === 'asciicast' ? 'cast' : rec.format}`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleExportVideo = async (rec: Recording) => {
    setConvertingIds((prev) => new Set(prev).add(rec.id));
    try {
      const blob = await exportRecordingVideo(rec.id);
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `recording-${rec.id}.m4v`;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      // silently handle
    } finally {
      setConvertingIds((prev) => {
        const next = new Set(prev);
        next.delete(rec.id);
        return next;
      });
    }
  };

  const totalPages = Math.ceil(total / limit);

  return (
    <>
      <Dialog open={open} onOpenChange={(v) => { if (!v) onClose(); }}>
        <DialogContent
          showCloseButton={false}
          className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
        >
          {/* Header — compact single-line bar */}
          <div className="flex h-8 shrink-0 items-center gap-2 border-b px-3">
            <Video className="size-3.5 text-muted-foreground" />
            <span className="text-xs font-medium">Recordings</span>
            <span className="text-[10px] tabular-nums text-muted-foreground">({total})</span>

            <div className="ml-auto flex items-center gap-1.5">
              <Select value={protocolFilter} onValueChange={(v) => { setProtocolFilter(v); setPage(0); }}>
                <SelectTrigger className="h-6 w-[90px] text-[11px] px-2">
                  <SelectValue placeholder="Protocol" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All</SelectItem>
                  <SelectItem value="SSH">SSH</SelectItem>
                  <SelectItem value="RDP">RDP</SelectItem>
                  <SelectItem value="VNC">VNC</SelectItem>
                </SelectContent>
              </Select>
              <Button variant="ghost" size="icon-xs" onClick={onClose}>
                <X className="size-3.5" />
              </Button>
            </div>
          </div>

          {/* Content */}
          <ScrollArea className="flex-1">
            {loading ? (
              <div className="flex items-center justify-center py-16">
                <Loader2 className="size-5 animate-spin text-muted-foreground" />
              </div>
            ) : recordings.length === 0 ? (
              <div className="flex flex-col items-center gap-3 py-16 text-center">
                <Video className="size-8 text-muted-foreground/40" />
                <div className="space-y-1">
                  <p className="text-sm font-medium text-muted-foreground">No recordings found</p>
                  <p className="text-xs text-muted-foreground/70">
                    Enable session recording in your environment configuration.
                  </p>
                </div>
              </div>
            ) : (
              <div className="px-1">
                {recordings.map((rec) => {
                  const Icon = PROTOCOL_ICON[rec.protocol] ?? Monitor;
                  const converting = convertingIds.has(rec.id);

                  return (
                    <div
                      key={rec.id}
                      className="group flex items-center gap-3 border-b border-border/40 px-3 py-2 transition-colors hover:bg-muted/50"
                    >
                      {/* Icon + connection name */}
                      <div className="flex min-w-0 flex-1 items-center gap-3">
                        <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted/60">
                          <Icon className="size-4 text-muted-foreground" />
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <span className="truncate text-sm font-medium">{rec.connection.name}</span>
                            <Badge className={cn('shrink-0 px-1.5 py-0 text-[10px] leading-tight', PROTOCOL_BADGE[rec.protocol])}>
                              {rec.protocol}
                            </Badge>
                          </div>
                          <div className="flex items-center gap-3 text-xs text-muted-foreground">
                            <span>{rec.user?.username || rec.user?.email || '\u2014'}</span>
                            <span className="text-border">\u00b7</span>
                            <span>{formatDate(rec.createdAt)} {formatTime(rec.createdAt)}</span>
                            <span className="text-border">\u00b7</span>
                            <span className="tabular-nums">{formatDuration(rec.duration)}</span>
                            <span className="text-border">\u00b7</span>
                            <span className="tabular-nums">{formatSize(rec.fileSize)}</span>
                          </div>
                        </div>
                      </div>

                      {/* Actions */}
                      <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button variant="ghost" size="icon-sm" onClick={() => setPlayingRecording(rec)}>
                              <Play className="size-3.5" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent side="bottom">Play</TooltipContent>
                        </Tooltip>

                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button variant="ghost" size="icon-sm" onClick={() => handleDownload(rec)}>
                              <Download className="size-3.5" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent side="bottom">Download raw</TooltipContent>
                        </Tooltip>

                        {(rec.format === 'guac' || rec.format === 'asciicast') ? (
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button variant="ghost" size="icon-sm" onClick={() => handleExportVideo(rec)} disabled={converting}>
                                {converting ? <Loader2 className="size-3.5 animate-spin" /> : <Film className="size-3.5" />}
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent side="bottom">Export MP4</TooltipContent>
                          </Tooltip>
                        ) : null}

                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button variant="ghost" size="icon-sm" onClick={() => setDeleteTarget(rec)} className="text-muted-foreground hover:text-destructive">
                              <Trash2 className="size-3.5" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent side="bottom">Delete</TooltipContent>
                        </Tooltip>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </ScrollArea>

          {/* Pagination */}
          {totalPages > 1 ? (
            <div className="flex items-center justify-between border-t px-4 py-2">
              <Button variant="ghost" size="sm" disabled={page === 0} onClick={() => setPage((p) => p - 1)} className="gap-1 text-xs">
                <ChevronLeft className="size-3.5" /> Previous
              </Button>
              <span className="text-xs tabular-nums text-muted-foreground">
                {page + 1} / {totalPages}
              </span>
              <Button variant="ghost" size="sm" disabled={(page + 1) * limit >= total} onClick={() => setPage((p) => p + 1)} className="gap-1 text-xs">
                Next <ChevronRight className="size-3.5" />
              </Button>
            </div>
          ) : null}
        </DialogContent>
      </Dialog>

      {/* Player */}
      <RecordingPlayerDialog
        open={!!playingRecording}
        onClose={() => setPlayingRecording(null)}
        recording={playingRecording}
      />

      {/* Delete confirmation */}
      <Dialog open={!!deleteTarget} onOpenChange={(v) => { if (!v) setDeleteTarget(null); }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Delete Recording</DialogTitle>
            <DialogDescription>
              Delete the recording for &quot;{deleteTarget?.connection.name}&quot;? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setDeleteTarget(null)}>Cancel</Button>
            <Button variant="destructive" size="sm" onClick={handleDelete}>Delete</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
