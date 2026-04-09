import { useState, useEffect, useCallback, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from '@/components/ui/select';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog';
import {
  RefreshCw,
  Square,
  Monitor,
  Server,
  Terminal,
} from 'lucide-react';
import type { ActiveSessionStreamSnapshot } from '../../api/live.api';
import { connectSSE } from '../../api/sse';
import { useAuthStore } from '../../store/authStore';
import { useGatewayStore } from '../../store/gatewayStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import { isGatewayGroup } from '../../utils/gatewayMode';

const statusBadgeClass: Record<string, string> = {
  ACTIVE: 'bg-green-500/15 text-green-400 border-green-500/30',
  IDLE: 'bg-yellow-500/15 text-yellow-400 border-yellow-500/30',
  CLOSED: '',
};

export default function SessionDashboard() {
  const activeSessions = useGatewayStore((s) => s.activeSessions);
  const sessionCount = useGatewayStore((s) => s.sessionCount);
  const sessionsLoading = useGatewayStore((s) => s.sessionsLoading);
  const gateways = useGatewayStore((s) => s.gateways);
  const fetchActiveSessions = useGatewayStore((s) => s.fetchActiveSessions);
  const fetchSessionCount = useGatewayStore((s) => s.fetchSessionCount);
  const applyActiveSessionStreamSnapshot = useGatewayStore((s) => s.applyActiveSessionStreamSnapshot);
  const terminateSessionAction = useGatewayStore((s) => s.terminateSession);
  const accessToken = useAuthStore((s) => s.accessToken);

  const autoRefresh = useUiPreferencesStore((s) => s.orchestrationAutoRefresh);
  const toggleAutoRefresh = useUiPreferencesStore((s) => s.toggle);

  const [protocolFilter, setProtocolFilter] = useState<string>('all');
  const [gatewayFilter, setGatewayFilter] = useState<string>('all');
  const [terminateTarget, setTerminateTarget] = useState<{ id: string; label: string } | null>(null);

  const filters = useMemo(() => {
    const f: { protocol?: 'SSH' | 'RDP'; gatewayId?: string } = {};
    if (protocolFilter !== 'all') f.protocol = protocolFilter as 'SSH' | 'RDP';
    if (gatewayFilter !== 'all') f.gatewayId = gatewayFilter;
    return f;
  }, [protocolFilter, gatewayFilter]);

  const refresh = useCallback(() => {
    void fetchActiveSessions(filters);
    void fetchSessionCount();
  }, [filters, fetchActiveSessions, fetchSessionCount]);

  useEffect(() => {
    if (autoRefresh || !accessToken) return undefined;
    refresh();
    return undefined;
  }, [refresh, autoRefresh, accessToken]);

  useEffect(() => {
    if (!autoRefresh || !accessToken) return undefined;

    const params = new URLSearchParams();
    if (filters.protocol) params.set('protocol', filters.protocol);
    if (filters.gatewayId) params.set('gatewayId', filters.gatewayId);
    const query = params.toString();

    return connectSSE({
      url: query ? `/api/sessions/active/stream?${query}` : '/api/sessions/active/stream',
      accessToken,
      onEvent: ({ event, data }) => {
        if (event !== 'snapshot') return;
        applyActiveSessionStreamSnapshot(data as ActiveSessionStreamSnapshot);
      },
    });
  }, [autoRefresh, accessToken, filters.protocol, filters.gatewayId, applyActiveSessionStreamSnapshot]);

  const sshCount = activeSessions.filter((s) => s.protocol === 'SSH').length;
  const rdpCount = activeSessions.filter((s) => s.protocol === 'RDP').length;
  const managedGateways = gateways.filter((g) => isGatewayGroup(g)).length;

  const handleTerminate = async () => {
    if (!terminateTarget) return;
    try {
      await terminateSessionAction(terminateTarget.id);
    } finally {
      setTerminateTarget(null);
    }
  };

  return (
    <div>
      {/* Metric cards */}
      <div className="flex gap-3 mb-4 flex-wrap">
        <MetricCard label="Total Active" value={sessionCount} icon={<Monitor className="h-4 w-4" />} />
        <MetricCard label="SSH Sessions" value={sshCount} icon={<Terminal className="h-4 w-4" />} />
        <MetricCard label="RDP Sessions" value={rdpCount} icon={<Server className="h-4 w-4" />} />
        <MetricCard label="Managed Gateways" value={managedGateways} icon={<Server className="h-4 w-4" />} />
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3 mb-3">
        <Select value={protocolFilter} onValueChange={setProtocolFilter}>
          <SelectTrigger className="w-[130px] h-8 text-sm">
            <SelectValue placeholder="Protocol" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="SSH">SSH</SelectItem>
            <SelectItem value="RDP">RDP</SelectItem>
          </SelectContent>
        </Select>
        <Select value={gatewayFilter} onValueChange={setGatewayFilter}>
          <SelectTrigger className="w-[180px] h-8 text-sm">
            <SelectValue placeholder="Gateway" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            {gateways.map((gw) => (
              <SelectItem key={gw.id} value={gw.id}>{gw.name}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button
          variant="outline"
          size="sm"
          onClick={refresh}
          disabled={sessionsLoading}
        >
          <RefreshCw className="h-3.5 w-3.5 mr-1" />
          Refresh
        </Button>
        <div className="flex items-center gap-2">
          <Switch
            checked={autoRefresh}
            onCheckedChange={() => toggleAutoRefresh('orchestrationAutoRefresh')}
          />
          <Label className="text-sm">Live updates</Label>
        </div>
      </div>

      {/* Sessions table */}
      <div className="rounded-lg border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b">
              <th className="text-left py-2 px-3 font-medium">User</th>
              <th className="text-left py-2 px-3 font-medium">Connection</th>
              <th className="text-left py-2 px-3 font-medium">Protocol</th>
              <th className="text-left py-2 px-3 font-medium">Gateway</th>
              <th className="text-left py-2 px-3 font-medium">Status</th>
              <th className="text-left py-2 px-3 font-medium">Started</th>
              <th className="text-left py-2 px-3 font-medium">Last Activity</th>
              <th className="text-left py-2 px-3 font-medium">Duration</th>
              <th className="text-right py-2 px-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {activeSessions.length === 0 ? (
              <tr>
                <td colSpan={9} className="text-center py-8 text-muted-foreground">
                  No active sessions
                </td>
              </tr>
            ) : (
              activeSessions.map((session) => (
                <tr key={session.id} className="border-b border-border/50">
                  <td className="py-2 px-3">{session.username || session.email}</td>
                  <td className="py-2 px-3">
                    <p className="text-sm">{session.connectionName}</p>
                    <p className="text-xs text-muted-foreground">
                      {session.connectionHost}:{session.connectionPort}
                    </p>
                  </td>
                  <td className="py-2 px-3">
                    <Badge variant="outline">{session.protocol}</Badge>
                  </td>
                  <td className="py-2 px-3">{session.gatewayName || 'Direct'}</td>
                  <td className="py-2 px-3">
                    <Badge className={statusBadgeClass[session.status] ?? ''}>
                      {session.status}
                    </Badge>
                  </td>
                  <td className="py-2 px-3">
                    <span className="text-xs">
                      {new Date(session.startedAt).toLocaleString()}
                    </span>
                  </td>
                  <td className="py-2 px-3">
                    <span className="text-xs">
                      {new Date(session.lastActivityAt).toLocaleString()}
                    </span>
                  </td>
                  <td className="py-2 px-3">{session.durationFormatted}</td>
                  <td className="py-2 px-3 text-right">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="h-7 w-7 text-red-400 hover:text-red-300"
                      onClick={() =>
                        setTerminateTarget({
                          id: session.id,
                          label: `${session.username || session.email} - ${session.connectionName}`,
                        })
                      }
                      title="Terminate session"
                    >
                      <Square className="h-4 w-4" />
                    </Button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Terminate confirmation */}
      <Dialog open={Boolean(terminateTarget)} onOpenChange={(v) => { if (!v) setTerminateTarget(null); }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Terminate Session</DialogTitle>
            <DialogDescription>
              Are you sure you want to terminate the session for <strong>{terminateTarget?.label}</strong>?
              The user&apos;s connection will be dropped immediately.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setTerminateTarget(null)}>Cancel</Button>
            <Button variant="destructive" onClick={handleTerminate}>
              Terminate
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function MetricCard({ label, value, icon }: { label: string; value: number; icon: React.ReactElement }) {
  return (
    <div className="flex-1 min-w-[160px] rounded-lg border p-4">
      <div className="flex items-center gap-2 mb-1">
        {icon}
        <span className="text-xs text-muted-foreground">{label}</span>
      </div>
      <p className="text-3xl font-bold">{value}</p>
    </div>
  );
}
