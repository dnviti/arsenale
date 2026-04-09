import { useState, useEffect, useCallback } from 'react';
import {
  History,
  Search,
  Loader2,
  LogIn,
  LogOut,
  ShieldAlert,
  FolderPlus,
  Share2,
  KeyRound,
  UserCog,
  Server,
  Database,
  FileText,
  AlertTriangle,
  RefreshCw,
  Lock,
  Unlock,
  Trash2,
  Eye,
  Link,
  Video,
  type LucideIcon,
} from 'lucide-react';
import {
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarInput,
} from '@/components/ui/sidebar';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';
import {
  getAuditLogs,
  type AuditLogEntry,
  type AuditAction,
  type AuditLogParams,
} from '../../api/audit.api';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import {
  ACTION_LABELS,
  getActionColor,
  ALL_ACTIONS,
} from '../Audit/auditConstants';

/* ------------------------------------------------------------------ */
/*  Helpers                                                           */
/* ------------------------------------------------------------------ */

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

/** Map semantic color from `getActionColor` to a Tailwind dot color. */
const DOT_COLOR: Record<string, string> = {
  default: 'bg-muted-foreground',
  primary: 'bg-primary',
  secondary: 'bg-muted-foreground',
  error: 'bg-destructive',
  warning: 'bg-yellow-500',
  success: 'bg-emerald-400',
  info: 'bg-blue-400',
};

const BADGE_COLOR: Record<string, string> = {
  default: '',
  primary: 'bg-primary/15 text-primary border-primary/30',
  secondary: 'bg-muted text-muted-foreground',
  error: 'bg-destructive/15 text-destructive border-destructive/30',
  warning: 'bg-yellow-600/15 text-yellow-500 border-yellow-600/30',
  success: 'bg-emerald-600/15 text-emerald-400 border-emerald-600/30',
  info: 'bg-blue-600/15 text-blue-400 border-blue-600/30',
};

/** Pick a compact icon for an audit action. */
function getActionIcon(action: AuditAction): LucideIcon {
  if (action.startsWith('LOGIN') || action === 'REGISTER') return LogIn;
  if (action === 'LOGOUT') return LogOut;
  if (action.startsWith('VAULT')) return Lock;
  if (action === 'VAULT_UNLOCK') return Unlock;
  if (action.startsWith('CREATE_') || action === 'SFTP_MKDIR') return FolderPlus;
  if (action.startsWith('DELETE_') || action.startsWith('SFTP_DELETE')) return Trash2;
  if (action.startsWith('SHARE_') || action.startsWith('UNSHARE_') || action === 'BATCH_SHARE') return Share2;
  if (action.startsWith('SECRET_')) return KeyRound;
  if (action.startsWith('GATEWAY_')) return Server;
  if (action.startsWith('SESSION_')) return Link;
  if (action.startsWith('TEAM_') || action.startsWith('TENANT_')) return UserCog;
  if (action.startsWith('SSH_KEY_')) return KeyRound;
  if (action.startsWith('TOTP_') || action.startsWith('SMS_') || action.startsWith('OAUTH_') || action === 'PASSWORD_CHANGE') return ShieldAlert;
  if (action === 'PASSWORD_REVEAL') return Eye;
  if (action.startsWith('DB_')) return Database;
  if (action.startsWith('SFTP_')) return FileText;
  if (action.startsWith('RECORDING_')) return Video;
  if (action === 'IMPOSSIBLE_TRAVEL_DETECTED' || action === 'ANOMALOUS_LATERAL_MOVEMENT' || action === 'TOKEN_HIJACK_ATTEMPT' || action === 'REFRESH_TOKEN_REUSE') return AlertTriangle;
  if (action === 'PROFILE_UPDATE') return UserCog;
  return History;
}

/** Grouped action categories for the filter dropdown (avoids 80+ items). */
const ACTION_GROUPS: { label: string; actions: AuditAction[] }[] = [
  {
    label: 'Authentication',
    actions: ['LOGIN', 'LOGIN_OAUTH', 'LOGIN_TOTP', 'LOGIN_SMS', 'LOGIN_FAILURE', 'LOGOUT', 'REGISTER'],
  },
  {
    label: 'Vault',
    actions: ['VAULT_UNLOCK', 'VAULT_LOCK', 'VAULT_SETUP', 'VAULT_AUTO_LOCK'],
  },
  {
    label: 'Connections',
    actions: ['CREATE_CONNECTION', 'UPDATE_CONNECTION', 'DELETE_CONNECTION', 'SHARE_CONNECTION', 'UNSHARE_CONNECTION', 'UPDATE_SHARE_PERMISSION', 'BATCH_SHARE', 'CONNECTION_FAVORITE'],
  },
  {
    label: 'Secrets',
    actions: ['SECRET_CREATE', 'SECRET_READ', 'SECRET_UPDATE', 'SECRET_DELETE', 'SECRET_SHARE', 'SECRET_UNSHARE', 'SECRET_EXTERNAL_SHARE', 'SECRET_EXTERNAL_ACCESS', 'SECRET_EXTERNAL_REVOKE', 'SECRET_SHARE_UPDATE', 'SECRET_VERSION_RESTORE'],
  },
  {
    label: 'Sessions',
    actions: ['SESSION_START', 'SESSION_END', 'SESSION_TIMEOUT', 'SESSION_ERROR', 'SESSION_TERMINATE', 'SESSION_TERMINATED_POLICY_VIOLATION'],
  },
  {
    label: 'Gateways',
    actions: ['GATEWAY_CREATE', 'GATEWAY_UPDATE', 'GATEWAY_DELETE', 'GATEWAY_DEPLOY', 'GATEWAY_UNDEPLOY', 'GATEWAY_SCALE', 'GATEWAY_SCALE_UP', 'GATEWAY_SCALE_DOWN', 'GATEWAY_RESTART', 'GATEWAY_HEALTH_CHECK', 'GATEWAY_VIEW_LOGS', 'GATEWAY_RECONCILE', 'GATEWAY_TEMPLATE_CREATE', 'GATEWAY_TEMPLATE_UPDATE', 'GATEWAY_TEMPLATE_DELETE', 'GATEWAY_TEMPLATE_DEPLOY'],
  },
  {
    label: 'Security',
    actions: ['PASSWORD_CHANGE', 'PASSWORD_REVEAL', 'TOTP_ENABLE', 'TOTP_DISABLE', 'SMS_MFA_ENABLE', 'SMS_MFA_DISABLE', 'REFRESH_TOKEN_REUSE', 'IMPOSSIBLE_TRAVEL_DETECTED', 'ANOMALOUS_LATERAL_MOVEMENT', 'TOKEN_HIJACK_ATTEMPT'],
  },
  {
    label: 'Database',
    actions: ['DB_QUERY_EXECUTED', 'DB_QUERY_BLOCKED', 'DB_FIREWALL_RULE_CREATE', 'DB_FIREWALL_RULE_UPDATE', 'DB_FIREWALL_RULE_DELETE', 'DB_MASKING_POLICY_CREATE', 'DB_MASKING_POLICY_UPDATE', 'DB_MASKING_POLICY_DELETE', 'DB_QUERY_PLAN_REQUESTED', 'DB_QUERY_AI_OPTIMIZED', 'DB_INTROSPECTION_REQUESTED'],
  },
];

const SIDEBAR_PAGE_SIZE = 30;

/* ------------------------------------------------------------------ */
/*  Component                                                         */
/* ------------------------------------------------------------------ */

export default function AuditSidePanel() {
  const auditLogAction = useUiPreferencesStore((s) => s.auditLogAction);
  const auditLogSearch = useUiPreferencesStore((s) => s.auditLogSearch);
  const auditLogSortBy = useUiPreferencesStore((s) => s.auditLogSortBy);
  const auditLogSortOrder = useUiPreferencesStore((s) => s.auditLogSortOrder);
  const setUiPref = useUiPreferencesStore((s) => s.set);

  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [searchInput, setSearchInput] = useState(auditLogSearch);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => {
      setUiPref('auditLogSearch', searchInput);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput, setUiPref]);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const params: AuditLogParams = {
        page: 1,
        limit: SIDEBAR_PAGE_SIZE,
        sortBy: auditLogSortBy as 'createdAt' | 'action',
        sortOrder: auditLogSortOrder as 'asc' | 'desc',
      };
      if (auditLogAction) params.action = auditLogAction as AuditAction;
      if (auditLogSearch) params.search = auditLogSearch;
      const result = await getAuditLogs(params);
      setLogs(result.data);
    } catch {
      setError('Failed to load audit logs');
    } finally {
      setLoading(false);
    }
  }, [auditLogAction, auditLogSearch, auditLogSortBy, auditLogSortOrder]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const hasActiveFilters = Boolean(auditLogAction || auditLogSearch);

  return (
    <SidebarGroup>
      <SidebarGroupLabel>
        <History className="size-4" />
        Activity
      </SidebarGroupLabel>
      <SidebarGroupAction
        title="Refresh"
        onClick={() => fetchLogs()}
        className={cn(loading && 'animate-spin')}
      >
        <RefreshCw className="size-4" />
      </SidebarGroupAction>
      <SidebarGroupContent>
        {/* Search */}
        <div className="px-2 pb-1">
          <div className="relative">
            <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <SidebarInput
              placeholder="Search audit..."
              className="pl-7 h-7 text-xs"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
            />
          </div>
        </div>

        {/* Action filter */}
        <div className="px-2 pb-2">
          <Select
            value={auditLogAction || '__all__'}
            onValueChange={(v) =>
              setUiPref('auditLogAction', v === '__all__' ? '' : v)
            }
          >
            <SelectTrigger className="h-7 text-xs w-full">
              <SelectValue placeholder="All actions" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">All actions</SelectItem>
              {ACTION_GROUPS.map((group) => (
                <div key={group.label}>
                  <div className="px-2 py-1.5 text-[0.65rem] font-semibold text-muted-foreground uppercase tracking-wider">
                    {group.label}
                  </div>
                  {group.actions
                    .filter((a) => ALL_ACTIONS.includes(a))
                    .map((action) => (
                      <SelectItem key={action} value={action} className="text-xs">
                        {ACTION_LABELS[action]}
                      </SelectItem>
                    ))}
                </div>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Content */}
        <div className="flex flex-col overflow-y-auto max-h-[calc(100vh-14rem)]">
          {error && (
            <div className="mx-2 rounded-md border border-destructive/50 bg-destructive/10 px-2 py-1.5 text-xs text-destructive">
              {error}
            </div>
          )}

          {loading ? (
            <div className="flex justify-center py-6">
              <Loader2 className="size-5 animate-spin text-muted-foreground" />
            </div>
          ) : logs.length === 0 ? (
            <div className="px-2 py-6 text-center text-xs text-muted-foreground">
              {hasActiveFilters
                ? 'No logs match your filters'
                : 'No activity recorded yet'}
            </div>
          ) : (
            <ul className="flex flex-col gap-px">
              {logs.map((log) => {
                const color = getActionColor(log.action);
                const Icon = getActionIcon(log.action);
                const isExpanded = expandedId === log.id;
                const isFlagged =
                  log.flags?.includes('IMPOSSIBLE_TRAVEL') ||
                  log.action === 'TOKEN_HIJACK_ATTEMPT' ||
                  log.action === 'ANOMALOUS_LATERAL_MOVEMENT';
                return (
                  <li key={log.id}>
                    <button
                      type="button"
                      className={cn(
                        'flex w-full items-start gap-2 px-2 py-1.5 text-left text-xs transition-colors hover:bg-accent/50 rounded-sm',
                        isExpanded && 'bg-accent/50',
                      )}
                      onClick={() =>
                        setExpandedId(isExpanded ? null : log.id)
                      }
                    >
                      {/* Color dot + icon */}
                      <span
                        className={cn(
                          'mt-0.5 flex size-5 shrink-0 items-center justify-center rounded',
                          DOT_COLOR[color] ? `${DOT_COLOR[color]}/15` : 'bg-muted',
                        )}
                      >
                        <Icon
                          className={cn(
                            'size-3',
                            DOT_COLOR[color]
                              ? DOT_COLOR[color].replace('bg-', 'text-')
                              : 'text-muted-foreground',
                          )}
                        />
                      </span>

                      {/* Text content */}
                      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                        <span className="flex items-center gap-1">
                          <span className="truncate font-medium leading-tight">
                            {ACTION_LABELS[log.action] || log.action}
                          </span>
                          {isFlagged && (
                            <AlertTriangle className="size-3 shrink-0 text-yellow-500" />
                          )}
                        </span>
                        <span className="flex items-center gap-1 text-[0.65rem] text-muted-foreground leading-tight">
                          <span className="truncate">
                            {log.targetType
                              ? `${log.targetType}${log.targetId ? ` ${log.targetId.slice(0, 8)}` : ''}`
                              : '\u2014'}
                          </span>
                          <span className="shrink-0">&middot;</span>
                          <span className="shrink-0">
                            {formatRelativeTime(log.createdAt)}
                          </span>
                        </span>
                      </span>
                    </button>

                    {/* Expanded detail */}
                    {isExpanded && (
                      <div className="ml-7 mr-2 mb-1 rounded-md border bg-card p-2 text-[0.65rem] space-y-1">
                        <div className="flex items-center gap-1.5">
                          <Badge
                            variant="outline"
                            className={cn(
                              'border text-[0.6rem] px-1.5 py-0',
                              BADGE_COLOR[color] || '',
                            )}
                          >
                            {ACTION_LABELS[log.action] || log.action}
                          </Badge>
                        </div>
                        <div className="text-muted-foreground">
                          {new Date(log.createdAt).toLocaleString()}
                        </div>
                        {log.ipAddress && (
                          <div className="text-muted-foreground">
                            IP: {log.ipAddress}
                            {log.geoCity && ` (${log.geoCity}${log.geoCountry ? `, ${log.geoCountry}` : ''})`}
                          </div>
                        )}
                        {log.targetId && (
                          <div className="break-all text-muted-foreground">
                            Target: {log.targetType} {log.targetId}
                          </div>
                        )}
                        {log.details &&
                          typeof log.details === 'object' &&
                          Object.keys(log.details).length > 0 && (
                            <div className="space-y-0.5 pt-0.5 border-t border-border/50">
                              {Object.entries(log.details).map(
                                ([key, value]) => (
                                  <div
                                    key={key}
                                    className="flex gap-1 text-muted-foreground"
                                  >
                                    <span className="font-medium shrink-0">
                                      {key}:
                                    </span>
                                    <span className="break-all">
                                      {Array.isArray(value)
                                        ? value.join(', ')
                                        : String(value)}
                                    </span>
                                  </div>
                                ),
                              )}
                            </div>
                          )}
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </SidebarGroupContent>
    </SidebarGroup>
  );
}
