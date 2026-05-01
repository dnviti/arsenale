import { useCallback, useEffect, useRef, useState } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select';
import {
  X,
  RefreshCw,
  Play,
  Pause,
  Maximize,
  Minimize,
  Loader2,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { ContainerLogStreamSnapshot } from '../../api/live.api';
import { connectSSE } from '../../api/sse';
import { getInstanceLogs, type ManagedInstanceData } from '../../api/gateway.api';
import { useAuthStore } from '../../store/authStore';

const TAIL_OPTIONS = [100, 200, 500, 1000] as const;
const MIN_HEIGHT = 200;
const DEFAULT_HEIGHT = 500;

interface ContainerLogDialogProps {
  open: boolean;
  onClose: () => void;
  gatewayId: string;
  instance: ManagedInstanceData | null;
}

export default function ContainerLogDialog({
  open, onClose, gatewayId, instance,
}: ContainerLogDialogProps) {
  const accessToken = useAuthStore((s) => s.accessToken);
  const [logs, setLogs] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tail, setTail] = useState<number>(200);
  const [live, setLive] = useState(true);
  const [fullScreen, setFullScreen] = useState(false);
  const [height, setHeight] = useState(DEFAULT_HEIGHT);
  const logsEndRef = useRef<HTMLDivElement>(null);
  const dragRef = useRef<{ startY: number; startH: number } | null>(null);

  const fetchLogs = useCallback(async () => {
    if (!instance) return;
    setLoading(true);
    setError(null);
    try {
      const data = await getInstanceLogs(gatewayId, instance.id, tail);
      setLogs(data.logs);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch logs');
      setLive(false);
    } finally {
      setLoading(false);
    }
  }, [gatewayId, instance, tail]);

  useEffect(() => {
    if (!open) {
      setLogs('');
      setError(null);
      setLive(true);
      setLoading(false);
      return;
    }
    if (open && instance && !live) {
      void fetchLogs();
    }
  }, [open, instance, live, fetchLogs]);

  useEffect(() => {
    if (!live || !open || !instance || !accessToken) return undefined;

    setLoading(true);
    setError(null);

    const params = new URLSearchParams({ tail: String(tail) });
    return connectSSE({
      url: `/api/gateways/${gatewayId}/instances/${instance.id}/logs/stream?${params.toString()}`,
      accessToken,
      onEvent: ({ event, data }) => {
        if (event !== 'snapshot') return;
        const snapshot = data as ContainerLogStreamSnapshot;
        setLogs(snapshot.logs);
        setLoading(false);
        setError(null);
      },
      onError: (streamError) => {
        const status = (streamError as Error & { status?: number }).status;
        setError(streamError.message);
        setLoading(false);
        if (status != null && !(status === 408 || status === 429 || (status >= 500 && status !== 501))) {
          setLive(false);
        }
      },
    });
  }, [live, open, instance, accessToken, tail, gatewayId]);

  useEffect(() => {
    if (logs && logsEndRef.current) {
      logsEndRef.current.scrollIntoView();
    }
  }, [logs]);

  const handleTailChange = (value: string) => {
    setTail(Number(value));
  };

  const handleDragStart = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    dragRef.current = { startY: e.clientY, startH: height };

    const onMove = (ev: MouseEvent) => {
      if (!dragRef.current) return;
      const delta = ev.clientY - dragRef.current.startY;
      setHeight(Math.max(MIN_HEIGHT, dragRef.current.startH + delta));
    };
    const onUp = () => {
      dragRef.current = null;
      document.removeEventListener('mousemove', onMove);
      document.removeEventListener('mouseup', onUp);
    };
    document.addEventListener('mousemove', onMove);
    document.addEventListener('mouseup', onUp);
  }, [height]);

  const handleClose = () => {
    setLive(false);
    onClose();
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) handleClose(); }}>
      <DialogContent
        showCloseButton={false}
        className={cn(
          "flex flex-col p-0 gap-0",
          fullScreen
            ? "h-[100dvh] w-screen max-w-none rounded-none border-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
            : "sm:max-w-3xl",
        )}
      >
        <DialogHeader className="flex flex-row items-center gap-2 px-4 py-2 border-b shrink-0">
          <DialogTitle className="flex-1 font-mono text-sm truncate">
            {instance?.containerName ?? 'Container Logs'}
          </DialogTitle>
          <Select value={String(tail)} onValueChange={handleTailChange}>
            <SelectTrigger className="h-7 w-[100px] text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {TAIL_OPTIONS.map((n) => (
                <SelectItem key={n} value={String(n)}>{n} lines</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button
            variant={live ? 'default' : 'ghost'}
            size="icon"
            className="h-7 w-7"
            onClick={() => setLive((v) => !v)}
            title={live ? 'Pause live' : 'Live view (auto-refresh)'}
          >
            {live ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={fetchLogs}
            disabled={loading}
            title="Refresh"
          >
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7"
            onClick={() => setFullScreen((v) => !v)}
            title={fullScreen ? 'Exit full screen' : 'Full screen'}
          >
            {fullScreen ? <Minimize className="h-4 w-4" /> : <Maximize className="h-4 w-4" />}
          </Button>
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={handleClose}>
            <X className="h-4 w-4" />
          </Button>
        </DialogHeader>

        <div className="flex flex-col flex-1 min-h-0">
          {loading && !logs ? (
            <div className="flex justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : error ? (
            <div className="p-4">
              <p className="text-red-400 text-sm">{error}</p>
            </div>
          ) : !logs ? (
            <div className="p-4">
              <p className="text-muted-foreground text-sm">No logs available</p>
            </div>
          ) : (
            <div
              className="bg-gray-900 text-gray-100 font-mono text-[0.8125rem] leading-relaxed whitespace-pre overflow-x-auto p-4 overflow-y-auto"
              style={{
                height: fullScreen ? '100%' : height,
                flex: fullScreen ? 1 : undefined,
              }}
            >
              {logs}
              <div ref={logsEndRef} />
            </div>
          )}
          {/* Resize drag handle */}
          {!fullScreen && logs && (
            <div
              onMouseDown={handleDragStart}
              className="h-1.5 cursor-ns-resize bg-border hover:bg-primary shrink-0"
            />
          )}
        </div>
        {live && (
          <div className="px-4 py-1 flex items-center gap-1.5">
            <div className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
            <span className="text-xs text-muted-foreground">
              Live stream connected
            </span>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
