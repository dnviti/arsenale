import { useMemo } from 'react';
import {
  Activity,
  Command,
  DatabaseZap,
  KeyRound,
  Monitor,
  Network,
  Plus,
  TerminalSquare,
  Video,
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import { useConnectionsStore } from '@/store/connectionsStore';
import { useTabsStore } from '@/store/tabsStore';
import { useGatewayStore } from '@/store/gatewayStore';
import { useVaultStore } from '@/store/vaultStore';
import { useAuthStore } from '@/store/authStore';
import { useFeatureFlagsStore } from '@/store/featureFlagsStore';
import { useUiPreferencesStore } from '@/store/uiPreferencesStore';
import { useCommandPaletteStore } from '@/store/commandPaletteStore';
import { summarizeGatewayStatuses } from '@/utils/gatewayStatus';
import { getRecentConnectionIds } from '@/utils/recentConnections';
import type { ConnectionData } from '@/api/connections.api';
import type { ConnectionFilter } from './AppSidebar';

function connectionIcon(type: string) {
  switch (type) {
    case 'SSH': return <TerminalSquare className="size-4" />;
    case 'DATABASE': return <DatabaseZap className="size-4" />;
    default: return <Monitor className="size-4" />;
  }
}

interface DashboardPanelProps {
  onCreateConnection: () => void;
  onOpenKeychain: () => void;
}

export default function DashboardPanel({ onCreateConnection, onOpenKeychain }: DashboardPanelProps) {
  const ownConnections = useConnectionsStore((s) => s.ownConnections);
  const sharedConnections = useConnectionsStore((s) => s.sharedConnections);
  const teamConnections = useConnectionsStore((s) => s.teamConnections);
  const openTab = useTabsStore((s) => s.openTab);
  const tabs = useTabsStore((s) => s.tabs);
  const gateways = useGatewayStore((s) => s.gateways);
  const vaultUnlocked = useVaultStore((s) => s.unlocked);
  const vaultInitialized = useVaultStore((s) => s.initialized);
  const keychainEnabled = useFeatureFlagsStore((s) => s.keychainEnabled);
  const recordingsEnabled = useFeatureFlagsStore((s) => s.recordingsEnabled);
  const userId = useAuthStore((s) => s.user?.id);
  const setPreference = useUiPreferencesStore((s) => s.set);
  const togglePalette = useCommandPaletteStore((s) => s.toggle);

  const allConnections = useMemo(
    () => [...ownConnections, ...sharedConnections, ...teamConnections],
    [ownConnections, sharedConnections, teamConnections],
  );

  const recentConnections = useMemo(() => {
    if (!userId) return [];
    const recentIds = getRecentConnectionIds(userId);
    const connMap = new Map(allConnections.map((c) => [c.id, c]));
    return recentIds
      .map((id) => connMap.get(id))
      .filter((c): c is ConnectionData => c !== undefined)
      .slice(0, 8);
  }, [allConnections, userId]);

  const gatewaySummary = summarizeGatewayStatuses(gateways);

  const setConnectionFilter = (filter: ConnectionFilter) => {
    setPreference('workspaceActiveView', filter);
  };

  return (
    <div className="flex flex-1 items-start justify-center overflow-auto p-6">
      <div className="w-full max-w-4xl space-y-6">
        {/* Quick actions */}
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          <button
            type="button"
            className="group flex flex-col items-center gap-2 rounded-xl border bg-card p-4 text-center transition-colors hover:border-primary/30 hover:bg-primary/5"
            onClick={onCreateConnection}
          >
            <Plus className="size-5 text-muted-foreground group-hover:text-primary" />
            <span className="text-xs font-medium">New Connection</span>
          </button>
          <button
            type="button"
            className="group flex flex-col items-center gap-2 rounded-xl border bg-card p-4 text-center transition-colors hover:border-primary/30 hover:bg-primary/5"
            onClick={togglePalette}
          >
            <Command className="size-5 text-muted-foreground group-hover:text-primary" />
            <span className="text-xs font-medium">Quick Search</span>
          </button>
          {keychainEnabled ? (
            <button
              type="button"
              className="group flex flex-col items-center gap-2 rounded-xl border bg-card p-4 text-center transition-colors hover:border-primary/30 hover:bg-primary/5"
              onClick={onOpenKeychain}
            >
              <KeyRound className="size-5 text-muted-foreground group-hover:text-primary" />
              <span className="text-xs font-medium">Keychain</span>
            </button>
          ) : null}
          {recordingsEnabled ? (
            <button
              type="button"
              className="group flex flex-col items-center gap-2 rounded-xl border bg-card p-4 text-center transition-colors hover:border-primary/30 hover:bg-primary/5"
              onClick={() => setConnectionFilter('remote')}
            >
              <Video className="size-5 text-muted-foreground group-hover:text-primary" />
              <span className="text-xs font-medium">Recordings</span>
            </button>
          ) : null}
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-3 gap-3">
          <Card className="border-border/50">
            <CardHeader className="pb-2">
              <CardDescription className="flex items-center gap-1.5 text-xs">
                <Activity className="size-3" /> Sessions
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-semibold tabular-nums">{tabs.length}</div>
            </CardContent>
          </Card>
          <Card className="border-border/50">
            <CardHeader className="pb-2">
              <CardDescription className="flex items-center gap-1.5 text-xs">
                <Network className="size-3" /> Gateways
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center gap-2">
                <span className="text-2xl font-semibold tabular-nums">{gateways.length}</span>
                {gatewaySummary.total > 0 ? (
                  <Badge variant="outline" className={cn(
                    'text-[10px]',
                    gatewaySummary.healthy === gatewaySummary.total
                      ? 'border-emerald-500/30 text-emerald-400'
                      : gatewaySummary.healthy > 0 || gatewaySummary.degraded > 0
                        ? 'border-amber-500/30 text-amber-400'
                        : gatewaySummary.unknown > 0
                          ? 'border-zinc-500/30 text-zinc-400'
                          : 'border-red-500/30 text-red-400',
                  )}>
                    {gatewaySummary.healthy} healthy
                  </Badge>
                ) : null}
              </div>
            </CardContent>
          </Card>
          {keychainEnabled && vaultInitialized ? (
            <Card className="border-border/50">
              <CardHeader className="pb-2">
                <CardDescription className="flex items-center gap-1.5 text-xs">
                  <KeyRound className="size-3" /> Vault
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Badge variant="outline" className={cn(
                  'text-xs',
                  vaultUnlocked ? 'border-primary/30 text-primary' : 'border-destructive/30 text-destructive',
                )}>
                  {vaultUnlocked ? 'Unlocked' : 'Locked'}
                </Badge>
              </CardContent>
            </Card>
          ) : (
            <Card className="border-border/50">
              <CardHeader className="pb-2">
                <CardDescription className="flex items-center gap-1.5 text-xs">
                  <TerminalSquare className="size-3" /> Connections
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-semibold tabular-nums">{allConnections.length}</div>
              </CardContent>
            </Card>
          )}
        </div>

        {/* Recent connections */}
        {recentConnections.length > 0 ? (
          <Card className="border-border/50">
            <CardHeader>
              <CardTitle className="text-sm">Recent Connections</CardTitle>
              <CardDescription className="text-xs">Double-click to connect</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 gap-1 sm:grid-cols-2">
                {recentConnections.map((conn) => (
                  <button
                    key={conn.id}
                    type="button"
                    className="flex items-center gap-3 rounded-lg p-2.5 text-left transition-colors hover:bg-accent"
                    onDoubleClick={() => openTab(conn)}
                  >
                    <span className="text-muted-foreground">
                      {connectionIcon(conn.type)}
                    </span>
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-medium">{conn.name}</div>
                      <div className="truncate text-xs text-muted-foreground">
                        {conn.host || conn.type}
                      </div>
                    </div>
                    <Badge variant="outline" className="shrink-0 text-[10px]">
                      {conn.type}
                    </Badge>
                  </button>
                ))}
              </div>
            </CardContent>
          </Card>
        ) : null}

        {/* Keyboard shortcuts hint */}
        <div className="text-center text-xs text-muted-foreground/60">
          Press <kbd className="rounded border bg-muted px-1.5 py-0.5 text-[10px] font-mono">Cmd+K</kbd> to search &middot; <kbd className="rounded border bg-muted px-1.5 py-0.5 text-[10px] font-mono">Cmd+B</kbd> to toggle sidebar
        </div>
      </div>
    </div>
  );
}
