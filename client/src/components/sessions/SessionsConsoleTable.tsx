import {
  Activity,
  BarChart3,
  Download,
  Eye,
  Film,
  Loader2,
  Pause,
  Play,
  Square,
  Trash2,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import type { SessionConsoleSession } from '@/api/sessions.api';
import { cn } from '@/lib/utils';
import {
  formatRecordingDuration,
  formatRecordingSize,
  formatSessionTimestamp,
  getSessionStatusBadgeClass,
} from './sessionConsoleUtils';

interface SessionsConsoleTableProps {
  sessions: SessionConsoleSession[];
  loading: boolean;
  canObserveSessions: boolean;
  canControlSessions: boolean;
  canDeleteRecording: boolean;
  busySessionIds: Record<string, 'pause' | 'resume' | 'terminate'>;
  busyRecordingIds: Record<string, 'download' | 'export' | 'delete'>;
  onObserve: (session: SessionConsoleSession) => void;
  onPauseResume: (session: SessionConsoleSession) => void;
  onTerminate: (session: SessionConsoleSession) => void;
  onPlayback: (session: SessionConsoleSession) => void;
  onDownload: (session: SessionConsoleSession) => void;
  onExport: (session: SessionConsoleSession) => void;
  onAnalyze: (session: SessionConsoleSession) => void;
  onAudit: (session: SessionConsoleSession) => void;
  onDeleteRecording: (session: SessionConsoleSession) => void;
}

function ActionIconButton({
  label,
  disabled,
  onClick,
  children,
  className,
}: {
  label: string;
  disabled?: boolean;
  onClick: () => void;
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={label}
          title={label}
          disabled={disabled}
          className={className}
          onClick={onClick}
        >
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent side="bottom">{label}</TooltipContent>
    </Tooltip>
  );
}

export default function SessionsConsoleTable({
  sessions,
  loading,
  canObserveSessions,
  canControlSessions,
  canDeleteRecording,
  busySessionIds,
  busyRecordingIds,
  onObserve,
  onPauseResume,
  onTerminate,
  onPlayback,
  onDownload,
  onExport,
  onAnalyze,
  onAudit,
  onDeleteRecording,
}: SessionsConsoleTableProps) {
  return (
    <div className="rounded-xl border border-border/70 bg-card/70">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead>User</TableHead>
            <TableHead>Connection</TableHead>
            <TableHead>Protocol</TableHead>
            <TableHead>Gateway</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Started</TableHead>
            <TableHead>Last Activity</TableHead>
            <TableHead>Recording</TableHead>
            <TableHead className="w-[15rem] text-right">Actions</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {loading ? (
            <TableRow>
              <TableCell colSpan={9} className="py-12 text-center text-muted-foreground">
                <div className="inline-flex items-center gap-2">
                  <Loader2 className="size-4 animate-spin" />
                  Loading sessions
                </div>
              </TableCell>
            </TableRow>
          ) : sessions.length === 0 ? (
            <TableRow>
              <TableCell colSpan={9} className="py-12 text-center text-muted-foreground">
                No sessions match these filters.
              </TableCell>
            </TableRow>
          ) : (
            sessions.map((session) => {
              const recordingId = session.recording.id ?? null;
              const recordingBusy = recordingId ? busyRecordingIds[recordingId] : undefined;
              const liveAction = busySessionIds[session.id];
              const isLive = session.status !== 'CLOSED';
              const canPlayRecording = session.status === 'CLOSED' && session.recording.exists && recordingId;

              return (
                <TableRow key={session.id}>
                  <TableCell>
                    <div className="space-y-1">
                      <div className="font-medium">{session.username || session.email}</div>
                      {session.username ? (
                        <div className="text-xs text-muted-foreground">{session.email}</div>
                      ) : null}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <div className="font-medium">{session.connectionName}</div>
                      <div className="text-xs text-muted-foreground">
                        {session.connectionHost}:{session.connectionPort}
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{session.protocol}</Badge>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <div>{session.gatewayName || 'Direct'}</div>
                      {session.instanceName ? (
                        <div className="text-xs text-muted-foreground">{session.instanceName}</div>
                      ) : null}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge className={cn('px-1.5 py-0 text-[10px] leading-tight', getSessionStatusBadgeClass(session.status))}>
                      {session.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    <div>{formatSessionTimestamp(session.startedAt)}</div>
                    <div className="mt-1 font-mono text-[10px] uppercase tracking-[0.16em] text-muted-foreground/70">
                      {session.durationFormatted}
                    </div>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatSessionTimestamp(session.lastActivityAt)}
                  </TableCell>
                  <TableCell>
                    {session.recording.exists ? (
                      <div className="space-y-1 text-xs text-muted-foreground">
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className="px-1.5 py-0 text-[10px] leading-tight">
                            {session.recording.format || 'recording'}
                          </Badge>
                          <span>{session.recording.status || 'READY'}</span>
                        </div>
                        <div>
                          {formatRecordingDuration(session.recording.duration)} · {formatRecordingSize(session.recording.fileSize)}
                        </div>
                      </div>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex flex-wrap items-center justify-end gap-1">
                      {isLive && canObserveSessions ? (
                        <ActionIconButton label="Observe session" onClick={() => onObserve(session)}>
                          <Eye className="size-3.5" />
                        </ActionIconButton>
                      ) : null}
                      {isLive && canControlSessions ? (
                        <>
                          <ActionIconButton
                            label={session.status === 'PAUSED' ? 'Resume session' : 'Pause session'}
                            disabled={Boolean(liveAction)}
                            onClick={() => onPauseResume(session)}
                            className="text-sky-300 hover:text-sky-200"
                          >
                            {liveAction === 'pause' || liveAction === 'resume' ? (
                              <Loader2 className="size-3.5 animate-spin" />
                            ) : session.status === 'PAUSED' ? (
                              <Play className="size-3.5" />
                            ) : (
                              <Pause className="size-3.5" />
                            )}
                          </ActionIconButton>
                          <ActionIconButton
                            label="Stop session"
                            disabled={Boolean(liveAction)}
                            onClick={() => onTerminate(session)}
                            className="text-destructive hover:text-destructive"
                          >
                            {liveAction === 'terminate' ? (
                              <Loader2 className="size-3.5 animate-spin" />
                            ) : (
                              <Square className="size-3.5" />
                            )}
                          </ActionIconButton>
                        </>
                      ) : null}
                      {canPlayRecording ? (
                        <>
                          <ActionIconButton label="Playback recording" onClick={() => onPlayback(session)}>
                            <Play className="size-3.5" />
                          </ActionIconButton>
                          <ActionIconButton
                            label="Download raw recording"
                            disabled={recordingBusy === 'download'}
                            onClick={() => onDownload(session)}
                          >
                            {recordingBusy === 'download' ? <Loader2 className="size-3.5 animate-spin" /> : <Download className="size-3.5" />}
                          </ActionIconButton>
                          <ActionIconButton
                            label="Export MP4"
                            disabled={recordingBusy === 'export'}
                            onClick={() => onExport(session)}
                          >
                            {recordingBusy === 'export' ? <Loader2 className="size-3.5 animate-spin" /> : <Film className="size-3.5" />}
                          </ActionIconButton>
                          <ActionIconButton label="Analyze recording" onClick={() => onAnalyze(session)}>
                            <BarChart3 className="size-3.5" />
                          </ActionIconButton>
                          <ActionIconButton label="Open recording audit trail" onClick={() => onAudit(session)}>
                            <Activity className="size-3.5" />
                          </ActionIconButton>
                          {canDeleteRecording ? (
                            <ActionIconButton
                              label="Delete recording"
                              disabled={recordingBusy === 'delete'}
                              onClick={() => onDeleteRecording(session)}
                              className="text-destructive hover:text-destructive"
                            >
                              {recordingBusy === 'delete' ? <Loader2 className="size-3.5 animate-spin" /> : <Trash2 className="size-3.5" />}
                            </ActionIconButton>
                          ) : null}
                        </>
                      ) : null}
                    </div>
                  </TableCell>
                </TableRow>
              );
            })
          )}
        </TableBody>
      </Table>
    </div>
  );
}
