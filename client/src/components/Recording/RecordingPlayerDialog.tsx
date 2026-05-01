import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import {
  Activity,
  BarChart3,
  ChevronDown,
  ExternalLink,
  Loader2,
  Maximize,
  Minimize,
  X,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import type { Recording } from '../../api/recordings.api';
import { analyzeRecording, type RecordingAnalysis } from '../../api/recordings.api';
import { openRecordingWindow } from '../../utils/openRecordingWindow';
import { extractApiError } from '../../utils/apiError';
import { getRecordingAuditTrail } from '../../api/audit.api';
import type { AuditLogEntry } from '../../api/audit.api';
import { ACTION_LABELS, getActionColor } from '../Audit/auditConstants';
import GuacPlayer from './GuacPlayer';
import SshPlayer from './SshPlayer';

interface RecordingPlayerDialogProps {
  open: boolean;
  onClose: () => void;
  recording: Recording | null;
  initialPanel?: RecordingPlayerInitialPanel;
}

export type RecordingPlayerInitialPanel = 'player' | 'analysis' | 'audit';

const PROTOCOL_BADGE: Record<string, string> = {
  SSH: 'bg-green-500/15 text-green-400 border-green-500/30',
  RDP: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
  VNC: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/30',
};

const ACTION_BADGE: Record<string, string> = {
  success: 'bg-green-500/15 text-green-400 border-green-500/30',
  error: 'bg-red-500/15 text-red-400 border-red-500/30',
  warning: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/30',
  info: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
};

export default function RecordingPlayerDialog({
  open,
  onClose,
  recording,
  initialPanel = 'player',
}: RecordingPlayerDialogProps) {
  const [fullScreen, setFullScreen] = useState(false);
  const [analysis, setAnalysis] = useState<RecordingAnalysis | null>(null);
  const [analyzing, setAnalyzing] = useState(false);
  const [analysisError, setAnalysisError] = useState('');
  const [showAnalysis, setShowAnalysis] = useState(false);
  const [auditTrail, setAuditTrail] = useState<AuditLogEntry[]>([]);
  const [auditLoading, setAuditLoading] = useState(false);
  const [showAuditTrail, setShowAuditTrail] = useState(false);

  useEffect(() => {
    setAuditTrail([]);
    setShowAuditTrail(false);
    setAuditLoading(false);
    setAnalysis(null);
    setShowAnalysis(false);
    setAnalysisError('');
  }, [recording?.id]);

  useEffect(() => {
    if (!recording) {
      return;
    }

    if (initialPanel === 'analysis') {
      setShowAnalysis(true);
      setShowAuditTrail(false);
      if (!analysis && !analyzing && recording.format !== 'asciicast') {
        void handleAnalyze();
      }
      return;
    }

    if (initialPanel === 'audit') {
      setShowAuditTrail(true);
      setShowAnalysis(false);
      if (auditTrail.length === 0 && !auditLoading) {
        void handleAuditTrail();
      }
      return;
    }

    setShowAnalysis(false);
    setShowAuditTrail(false);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [recording?.id, initialPanel]);

  if (!recording) return null;

  const isTextRecording = recording.format === 'asciicast';

  const handleAnalyze = async () => {
    if (!recording) return;
    if (analysis) { setShowAnalysis((v) => !v); return; }
    setAnalyzing(true);
    setAnalysisError('');
    try {
      const result = await analyzeRecording(recording.id);
      setAnalysis(result);
      setShowAnalysis(true);
    } catch (err: unknown) {
      setAnalysisError(extractApiError(err, 'Failed to analyze recording'));
    } finally {
      setAnalyzing(false);
    }
  };

  const handleOpenInNewWindow = () => {
    openRecordingWindow(recording.id, recording.width, recording.height);
    onClose();
  };

  const handleAuditTrail = async () => {
    if (!recording) return;
    if (auditTrail.length > 0) { setShowAuditTrail((v) => !v); return; }
    setAuditLoading(true);
    try {
      const result = await getRecordingAuditTrail(recording.id);
      setAuditTrail(result.data);
      setShowAuditTrail(true);
    } catch {
      // Audit trail not available
    } finally {
      setAuditLoading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose(); }}>
      <DialogContent
        showCloseButton={false}
        className={cn(
          'flex flex-col overflow-hidden gap-0 p-0',
          fullScreen
            ? 'h-[100dvh] w-screen max-w-none rounded-none border-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border'
            : 'h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl',
        )}
      >
        {/* Header bar */}
        <DialogHeader className="flex shrink-0 flex-row items-center gap-2 border-b bg-background/80 px-3 py-2 backdrop-blur">
          <DialogTitle className="flex min-w-0 flex-1 items-center gap-2 text-sm font-medium">
            <span className="truncate">{recording.connection.name}</span>
            <Badge className={cn('shrink-0 px-1.5 py-0 text-[10px] leading-tight', PROTOCOL_BADGE[recording.protocol])}>
              {recording.protocol}
            </Badge>
          </DialogTitle>

          <div className="flex items-center gap-0.5">
            {!isTextRecording ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon-sm" onClick={handleAnalyze} disabled={analyzing}>
                    {analyzing ? <Loader2 className="size-3.5 animate-spin" /> : <BarChart3 className="size-3.5" />}
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="bottom">{analysis ? (showAnalysis ? 'Hide analysis' : 'Show analysis') : 'Analyze'}</TooltipContent>
              </Tooltip>
            ) : null}

            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-sm" onClick={handleAuditTrail} disabled={auditLoading}>
                  {auditLoading ? <Loader2 className="size-3.5 animate-spin" /> : <Activity className="size-3.5" />}
                </Button>
              </TooltipTrigger>
              <TooltipContent side="bottom">{auditTrail.length > 0 ? (showAuditTrail ? 'Hide audit trail' : 'Show audit trail') : 'Load audit trail'}</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-sm" onClick={handleOpenInNewWindow}>
                  <ExternalLink className="size-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="bottom">Open in new window</TooltipContent>
            </Tooltip>

            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-sm" onClick={() => setFullScreen((v) => !v)}>
                  {fullScreen ? <Minimize className="size-3.5" /> : <Maximize className="size-3.5" />}
                </Button>
              </TooltipTrigger>
              <TooltipContent side="bottom">{fullScreen ? 'Exit fullscreen' : 'Fullscreen'}</TooltipContent>
            </Tooltip>

            <Button variant="ghost" size="icon-sm" onClick={onClose}>
              <X className="size-3.5" />
            </Button>
          </div>
        </DialogHeader>

        {/* Panels + Player */}
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          {/* Error banner */}
          {analysisError ? (
            <div className="mx-3 mt-2 rounded-md border border-destructive/30 bg-destructive/10 px-3 py-1.5 text-xs text-destructive">
              {analysisError}
            </div>
          ) : null}

          {/* Analysis panel */}
          {analysis ? (
            <Collapsible open={showAnalysis} onOpenChange={setShowAnalysis}>
              <CollapsibleTrigger asChild>
                <button type="button" className="flex w-full items-center gap-2 border-b bg-muted/30 px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted/50">
                  <ChevronDown className={cn('size-3 transition-transform', showAnalysis && 'rotate-180')} />
                  Recording Analysis
                </button>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <ScrollArea className="max-h-[180px] border-b">
                  <div className="grid grid-cols-2 gap-x-6 gap-y-1 p-3 text-xs sm:grid-cols-3">
                    <div>
                      <span className="text-muted-foreground">File Size</span>
                      <p className="font-medium tabular-nums">{(analysis.fileSize / 1024 / 1024).toFixed(2)} MB</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Display</span>
                      <p className="font-medium tabular-nums">{analysis.displayWidth} x {analysis.displayHeight}</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Sync Frames</span>
                      <p className="font-medium tabular-nums">{analysis.syncCount.toLocaleString()}</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Display Data</span>
                      <p className="font-medium">{analysis.hasLayer0Image ? 'Yes' : 'No'}</p>
                    </div>
                    <div>
                      <span className="text-muted-foreground">Truncated</span>
                      <p className="font-medium">{analysis.truncated ? 'Yes (>10 MB)' : 'No'}</p>
                    </div>
                    {Object.entries(analysis.instructions)
                      .sort((a, b) => b[1] - a[1])
                      .slice(0, 6)
                      .map(([op, count]) => (
                        <div key={op}>
                          <span className="text-muted-foreground">{op}</span>
                          <p className="font-medium tabular-nums">{count.toLocaleString()}</p>
                        </div>
                      ))}
                  </div>
                </ScrollArea>
              </CollapsibleContent>
            </Collapsible>
          ) : null}

          {/* Audit trail panel */}
          {auditTrail.length > 0 ? (
            <Collapsible open={showAuditTrail} onOpenChange={setShowAuditTrail}>
              <CollapsibleTrigger asChild>
                <button type="button" className="flex w-full items-center gap-2 border-b bg-muted/30 px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted/50">
                  <ChevronDown className={cn('size-3 transition-transform', showAuditTrail && 'rotate-180')} />
                  Audit Trail ({auditTrail.length})
                </button>
              </CollapsibleTrigger>
              <CollapsibleContent>
                <ScrollArea className="max-h-[180px] border-b">
                  <div className="divide-y divide-border/40">
                    {auditTrail.map((entry) => (
                      <div key={entry.id} className="flex items-center gap-3 px-3 py-1.5 text-xs">
                        <span className="w-[130px] shrink-0 tabular-nums text-muted-foreground">
                          {new Date(entry.createdAt).toLocaleString()}
                        </span>
                        <Badge className={cn('shrink-0 px-1.5 py-0 text-[10px] leading-tight', ACTION_BADGE[getActionColor(entry.action)] || '')}>
                          {ACTION_LABELS[entry.action] || entry.action}
                        </Badge>
                        <span className="min-w-0 truncate text-muted-foreground">
                          {entry.details
                            ? Object.entries(entry.details).map(([k, v]) => `${k}: ${v}`).join(' \u00b7 ')
                            : ''}
                        </span>
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              </CollapsibleContent>
            </Collapsible>
          ) : null}

          {/* Player */}
          <div className="flex min-h-0 flex-1 flex-col">
            {isTextRecording ? (
              <SshPlayer recordingId={recording.id} />
            ) : (
              <GuacPlayer recordingId={recording.id} />
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
