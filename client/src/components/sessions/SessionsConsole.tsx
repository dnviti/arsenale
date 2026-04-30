import { useCallback, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { AlertCircle, ChevronLeft, ChevronRight, RefreshCw } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import {
  deleteRecording,
  downloadRecordingRaw,
  exportRecordingVideo,
} from '@/api/recordings.api';
import {
  getSessionConsole,
  pauseSession,
  resumeSession,
  terminateSession,
  type SessionConsoleStatus,
  type SessionConsoleResponse,
  type SessionConsoleSession,
} from '@/api/sessions.api';
import { useAuthStore } from '@/store/authStore';
import { useGatewayStore } from '@/store/gatewayStore';
import { cn } from '@/lib/utils';
import { extractApiError } from '@/utils/apiError';
import RecordingPlayerLauncher from './RecordingPlayerLauncher';
import SessionsConsoleTable from './SessionsConsoleTable';
import {
  buildSessionsRoute,
  readSessionsRouteState,
  type SessionsRouteState,
} from './sessionConsoleRoute';
import {
  formatStatusFilterLabel,
  getConsoleServerStatuses,
  matchesStatusFilter,
  matchesTextFilter,
  recordingExtension,
  SESSION_STATUS_FILTER_OPTIONS,
  SESSION_PAGE_SIZE,
} from './sessionConsoleUtils';

interface SessionsConsoleViewProps {
  routeState: SessionsRouteState;
  onRouteStateChange: (nextState: SessionsRouteState) => void;
  layout?: 'page' | 'dialog';
}

export function ControlledSessionsConsole({
  routeState,
  onRouteStateChange,
  layout = 'page',
}: SessionsConsoleViewProps) {
  const canObserveSessions = useAuthStore((state) => state.permissions.canObserveSessions);
  const canControlSessions = useAuthStore((state) => state.permissions.canControlSessions);
  const gateways = useGatewayStore((state) => state.gateways);
  const fetchSessionCount = useGatewayStore((state) => state.fetchSessionCount);

  const [result, setResult] = useState<SessionConsoleResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [actionError, setActionError] = useState('');
  const [busySessionIds, setBusySessionIds] = useState<Record<string, 'pause' | 'resume' | 'terminate'>>({});
  const [busyRecordingIds, setBusyRecordingIds] = useState<Record<string, 'download' | 'export' | 'delete'>>({});
  const [playerRequest, setPlayerRequest] = useState<{ recordingId: string; initialPanel?: 'analysis' | 'audit' } | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<SessionConsoleSession | null>(null);

  const scope = result?.scope ?? null;
  const serverStatuses = useMemo(
    () => getConsoleServerStatuses(routeState.status, scope),
    [routeState.status, scope],
  );

  const updateRoute = useCallback((nextState: Partial<typeof routeState>) => {
    onRouteStateChange({ ...routeState, ...nextState });
  }, [onRouteStateChange, routeState]);

  const loadSessions = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
        const next = await getSessionConsole({
          protocol: routeState.protocol === 'all' ? undefined : routeState.protocol,
          status: serverStatuses,
          gatewayId: routeState.gatewayId === 'all' ? undefined : routeState.gatewayId,
          limit: SESSION_PAGE_SIZE,
          offset: routeState.page * SESSION_PAGE_SIZE,
      });
      setResult(next);
    } catch (loadError: unknown) {
      setError(extractApiError(loadError, 'Failed to load the sessions console'));
    } finally {
      setLoading(false);
    }
  }, [routeState.gatewayId, routeState.page, routeState.protocol, serverStatuses]);

  useEffect(() => {
    void loadSessions();
  }, [loadSessions]);

  const sessions: SessionConsoleSession[] = result?.sessions ?? [];
  const visibleSessions = useMemo(() => sessions.filter((session: SessionConsoleSession) => {
    if (!matchesTextFilter(session, routeState.q)) {
      return false;
    }
    if (!matchesStatusFilter(session, routeState.status)) {
      return false;
    }
    if (routeState.recorded && !session.recording.exists) {
      return false;
    }
    return true;
  }), [routeState.q, routeState.recorded, routeState.status, sessions]);

  const gatewayOptions = useMemo(() => {
    const byId = new Map<string, { id: string; label: string }>();
    for (const gateway of gateways) {
      byId.set(gateway.id, { id: gateway.id, label: gateway.name });
    }
    for (const session of sessions) {
      if (session.gatewayId) {
        byId.set(session.gatewayId, { id: session.gatewayId, label: session.gatewayName || 'Gateway' });
      }
    }
    return Array.from(byId.values()).sort((a, b) => a.label.localeCompare(b.label));
  }, [gateways, sessions]);

  const totals = useMemo(() => ({
    visible: result?.total ?? 0,
    loaded: visibleSessions.length,
    active: sessions.filter((session: SessionConsoleSession) => session.status === 'ACTIVE' || session.status === 'IDLE').length,
    closed: sessions.filter((session: SessionConsoleSession) => session.status === 'CLOSED').length,
    recorded: sessions.filter((session: SessionConsoleSession) => session.recording.exists).length,
  }), [result?.total, sessions, visibleSessions.length]);

  const totalPages = Math.max(1, Math.ceil((result?.total ?? 0) / SESSION_PAGE_SIZE));
  const canDeleteRecording = canControlSessions || scope === 'own';
  const ownScopeRestriction = scope === 'own' && (routeState.status.includes('CLOSED') || routeState.recorded);

  const statusFilterLabel = useMemo(
    () => formatStatusFilterLabel(routeState.status),
    [routeState.status],
  );

  const toggleStatus = useCallback((status: SessionConsoleStatus, checked: boolean) => {
    const nextStatuses = checked
      ? [...routeState.status, status]
      : routeState.status.filter((value: SessionConsoleStatus) => value !== status);
    updateRoute({ status: nextStatuses, page: 0 });
  }, [routeState.status, updateRoute]);

  const handleObserve = useCallback((session: SessionConsoleSession) => {
    const route = `/session-observer/${session.protocol.toLowerCase()}/${session.id}`;
    window.open(route, '_blank', 'noopener,noreferrer,width=1600,height=960');
  }, []);

  const runSessionAction = useCallback(async (
    session: SessionConsoleSession,
    action: 'pause' | 'resume' | 'terminate',
  ) => {
    setActionError('');
    setBusySessionIds((current) => ({ ...current, [session.id]: action }));
    try {
      if (action === 'pause') {
        await pauseSession(session.id);
      } else if (action === 'resume') {
        await resumeSession(session.id);
      } else {
        await terminateSession(session.id);
      }
      await Promise.all([loadSessions(), fetchSessionCount()]);
    } catch (sessionError: unknown) {
      setActionError(extractApiError(sessionError, `Failed to ${action} session`));
    } finally {
      setBusySessionIds((current) => {
        const { [session.id]: _ignored, ...rest } = current;
        void _ignored;
        return rest;
      });
    }
  }, [fetchSessionCount, loadSessions]);

  const runRecordingAction = useCallback(async (
    session: SessionConsoleSession,
    action: 'download' | 'export' | 'delete',
  ) => {
    const recordingId = session.recording.id;
    if (!recordingId) {
      return;
    }

    setActionError('');
    setBusyRecordingIds((current) => ({ ...current, [recordingId]: action }));
    try {
      if (action === 'download') {
        const blob = await downloadRecordingRaw(recordingId);
        const url = URL.createObjectURL(blob);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = `recording-${recordingId}.${recordingExtension(session.recording.format)}`;
        anchor.click();
        URL.revokeObjectURL(url);
      } else if (action === 'export') {
        const blob = await exportRecordingVideo(recordingId);
        const url = URL.createObjectURL(blob);
        const anchor = document.createElement('a');
        anchor.href = url;
        anchor.download = `recording-${recordingId}.m4v`;
        anchor.click();
        URL.revokeObjectURL(url);
      } else {
        await deleteRecording(recordingId);
        setDeleteTarget(null);
        await loadSessions();
      }
    } catch (recordingError: unknown) {
      setActionError(extractApiError(recordingError, `Failed to ${action} recording`));
    } finally {
      setBusyRecordingIds((current) => {
        const { [recordingId]: _ignored, ...rest } = current;
        void _ignored;
        return rest;
      });
    }
  }, [loadSessions]);

  return (
    <div className={cn(
      'flex min-h-0 flex-1 overflow-auto',
      layout === 'page' ? 'p-6' : 'bg-background',
    )}>
      <div className={cn(
        'mx-auto flex w-full flex-col gap-6',
        layout === 'page' ? 'max-w-[96rem]' : 'p-4 sm:p-6',
      )}>
        <Card className="border-border/60 bg-card/80">
          <CardHeader className="gap-4 md:flex-row md:items-end md:justify-between">
            <div>
              <div className="flex items-center gap-2">
                <CardTitle>Sessions</CardTitle>
                {scope ? (
                  <Badge variant="outline" className="text-[10px] uppercase tracking-[0.18em] text-muted-foreground">
                    {scope === 'tenant' ? 'Tenant scope' : 'Own scope'}
                  </Badge>
                ) : null}
              </div>
              <CardDescription>
                {scope === 'tenant'
                  ? 'Live sessions and recording history for every session you are allowed to review.'
                  : 'Only your live sessions are available here. Closed history is not exposed in own scope.'}
              </CardDescription>
            </div>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span>{totals.visible} visible</span>
              <span>·</span>
              <span>{totals.recorded} recorded</span>
              <Button type="button" variant="outline" size="sm" onClick={() => void loadSessions()} disabled={loading}>
                <RefreshCw className="size-3.5" />
                Refresh
              </Button>
            </div>
          </CardHeader>
          <CardContent className="grid gap-3 md:grid-cols-4">
            <MetricCard label="Loaded rows" value={String(totals.loaded)} />
            <MetricCard label="Active" value={String(totals.active)} />
            <MetricCard label="Closed" value={String(totals.closed)} />
            <MetricCard label="Recorded" value={String(totals.recorded)} />
          </CardContent>
        </Card>

        {error ? (
          <Alert variant="destructive">
            <AlertCircle className="size-4" />
            <AlertTitle>Sessions console unavailable</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        ) : null}

        {actionError ? (
          <Alert variant="destructive">
            <AlertCircle className="size-4" />
            <AlertTitle>Session action failed</AlertTitle>
            <AlertDescription>{actionError}</AlertDescription>
          </Alert>
        ) : null}

        {ownScopeRestriction ? (
          <Alert variant="warning">
            <AlertCircle className="size-4" />
            <AlertTitle>Closed history is not available in own scope</AlertTitle>
            <AlertDescription>
              This account can only review its own live sessions. Recorded and closed presets are shown against the rows the backend exposes.
            </AlertDescription>
          </Alert>
        ) : null}

        <Card className="border-border/60 bg-card/80">
          <CardContent className="flex flex-col gap-3 p-4 md:flex-row md:flex-wrap md:items-center">
            <Input
              value={routeState.q}
              onChange={(event) => updateRoute({ q: event.target.value, page: 0 })}
              placeholder="Search loaded rows by user or connection"
              className="md:max-w-xs"
            />
            <Select value={routeState.protocol} onValueChange={(value) => updateRoute({ protocol: value as typeof routeState.protocol, page: 0 })}>
              <SelectTrigger className="w-full md:w-[10rem]">
                <SelectValue placeholder="Protocol" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All protocols</SelectItem>
                <SelectItem value="SSH">SSH</SelectItem>
                <SelectItem value="RDP">RDP</SelectItem>
                <SelectItem value="VNC">VNC</SelectItem>
              </SelectContent>
            </Select>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button type="button" variant="outline" className="w-full justify-start md:w-[16rem]">
                  {statusFilterLabel}
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-[16rem]">
                <DropdownMenuLabel>Status filters</DropdownMenuLabel>
                <DropdownMenuSeparator />
                {SESSION_STATUS_FILTER_OPTIONS.map((option) => (
                  <DropdownMenuCheckboxItem
                    key={option.value}
                    checked={routeState.status.includes(option.value)}
                    onCheckedChange={(checked) => toggleStatus(option.value, checked === true)}
                  >
                    {option.label}
                  </DropdownMenuCheckboxItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
            <Select value={routeState.gatewayId} onValueChange={(value) => updateRoute({ gatewayId: value, page: 0 })}>
              <SelectTrigger className="w-full md:w-[14rem]">
                <SelectValue placeholder="Gateway" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All gateways</SelectItem>
                {gatewayOptions.map((gateway) => (
                  <SelectItem key={gateway.id} value={gateway.id}>{gateway.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              type="button"
              variant={routeState.recorded ? 'secondary' : 'outline'}
              size="sm"
              onClick={() => updateRoute({ recorded: !routeState.recorded, page: 0 })}
            >
              Recorded only
            </Button>
          </CardContent>
        </Card>

        <SessionsConsoleTable
          sessions={visibleSessions}
          loading={loading}
          canObserveSessions={canObserveSessions}
          canControlSessions={canControlSessions}
          canDeleteRecording={canDeleteRecording}
          busySessionIds={busySessionIds}
          busyRecordingIds={busyRecordingIds}
          onObserve={handleObserve}
          onPauseResume={(session) => void runSessionAction(session, session.status === 'PAUSED' ? 'resume' : 'pause')}
          onTerminate={(session) => void runSessionAction(session, 'terminate')}
          onPlayback={(session) => {
            if (session.recording.id) {
              setPlayerRequest({ recordingId: session.recording.id });
            }
          }}
          onDownload={(session) => void runRecordingAction(session, 'download')}
          onExport={(session) => void runRecordingAction(session, 'export')}
          onAnalyze={(session) => {
            if (session.recording.id) {
              setPlayerRequest({ recordingId: session.recording.id, initialPanel: 'analysis' });
            }
          }}
          onAudit={(session) => {
            if (session.recording.id) {
              setPlayerRequest({ recordingId: session.recording.id, initialPanel: 'audit' });
            }
          }}
          onDeleteRecording={setDeleteTarget}
        />

        <div className="flex items-center justify-between rounded-xl border border-border/70 bg-card/60 px-4 py-3 text-sm text-muted-foreground">
          <span>Page {routeState.page + 1} of {totalPages}</span>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={routeState.page === 0}
              onClick={() => updateRoute({ page: Math.max(0, routeState.page - 1) })}
            >
              <ChevronLeft className="size-3.5" />
              Previous
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={routeState.page + 1 >= totalPages}
              onClick={() => updateRoute({ page: routeState.page + 1 })}
            >
              Next
              <ChevronRight className="size-3.5" />
            </Button>
          </div>
        </div>

        <RecordingPlayerLauncher request={playerRequest} onClose={() => setPlayerRequest(null)} />

        {deleteTarget ? (
          <Card className="border-destructive/30 bg-destructive/5">
            <CardHeader>
              <CardTitle className="text-base">Delete recording</CardTitle>
              <CardDescription>
                Remove the recording attached to <strong>{deleteTarget.connectionName}</strong>? This cannot be undone.
              </CardDescription>
            </CardHeader>
            <CardContent className="flex items-center justify-end gap-2">
              <Button type="button" variant="outline" size="sm" onClick={() => setDeleteTarget(null)}>
                Cancel
              </Button>
              <Button
                type="button"
                variant="destructive"
                size="sm"
                onClick={() => void runRecordingAction(deleteTarget, 'delete')}
              >
                Delete
              </Button>
            </CardContent>
          </Card>
        ) : null}
      </div>
    </div>
  );
}

export default function SessionsConsole() {
  const [searchParams, setSearchParams] = useSearchParams();
  const routeState = readSessionsRouteState(searchParams);

  const handleRouteStateChange = useCallback((nextState: SessionsRouteState) => {
    const nextUrl = buildSessionsRoute(nextState);
    const query = nextUrl.includes('?') ? nextUrl.slice(nextUrl.indexOf('?') + 1) : '';
    setSearchParams(new URLSearchParams(query), { replace: true });
  }, [setSearchParams]);

  return (
    <ControlledSessionsConsole
      routeState={routeState}
      onRouteStateChange={handleRouteStateChange}
      layout="page"
    />
  );
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border/70 bg-muted/20 px-4 py-3">
      <div className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">{label}</div>
      <div className="mt-2 text-2xl font-semibold tabular-nums text-foreground">{value}</div>
    </div>
  );
}
