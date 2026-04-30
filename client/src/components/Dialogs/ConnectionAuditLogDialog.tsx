import { useState, useEffect, useCallback, Fragment } from 'react';
import {
  Dialog, DialogContent, DialogTitle, DialogDescription,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import {
  Select, SelectTrigger, SelectValue, SelectContent, SelectItem,
} from '@/components/ui/select';
import {
  X, Search, ChevronDown, ChevronUp, Download, AlertTriangle,
} from 'lucide-react';
import { Loader2 } from 'lucide-react';
import { cn } from '@/lib/utils';
import {
  getConnectionAuditLogs, getConnectionAuditUsers, getAuditGateways, getAuditCountries,
  TenantAuditLogEntry, AuditAction, ConnectionAuditLogParams, AuditGateway, ConnectionAuditUser,
} from '../../api/audit.api';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import { useAuthStore } from '../../store/authStore';
import { ACTION_LABELS, getActionColor, formatDetails, ALL_ACTIONS } from '../Audit/auditConstants';
import IpGeoCell from '../Audit/IpGeoCell';
import { hasAnyRole } from '../../utils/roles';

const ACTION_COLOR_MAP: Record<string, string> = {
  default: '',
  primary: 'bg-primary/15 text-primary border-primary/30',
  secondary: 'bg-muted text-muted-foreground',
  error: 'bg-destructive/15 text-destructive border-destructive/30',
  warning: 'bg-yellow-600/15 text-yellow-500 border-yellow-600/30',
  success: 'bg-emerald-600/15 text-emerald-400 border-emerald-600/30',
  info: 'bg-blue-600/15 text-blue-400 border-blue-600/30',
};

function exportCsv(logs: TenantAuditLogEntry[], connectionName: string) {
  const header = 'Date,User,Email,Action,IP Address,Country,City,Details';
  const rows = logs.map((log) => {
    const date = new Date(log.createdAt).toISOString();
    const user = (log.userName ?? '').replace(/"/g, '""');
    const email = (log.userEmail ?? '').replace(/"/g, '""');
    const action = ACTION_LABELS[log.action] || log.action;
    const ip = log.ipAddress ?? '';
    const country = log.geoCountry ?? '';
    const city = log.geoCity ?? '';
    const details = formatDetails(log.details as Record<string, unknown> | null).replace(/"/g, '""');
    return `"${date}","${user}","${email}","${action}","${ip}","${country}","${city}","${details}"`;
  });
  const csv = [header, ...rows].join('\n');
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `connection-audit-log-${connectionName.replace(/\s+/g, '-')}-${new Date().toISOString().slice(0, 10)}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

interface ConnectionAuditLogDialogProps {
  open: boolean;
  onClose: () => void;
  connectionId: string;
  connectionName: string;
  onGeoIpClick?: (ip: string) => void;
}

export default function ConnectionAuditLogDialog({ open, onClose, connectionId, connectionName, onGeoIpClick }: ConnectionAuditLogDialogProps) {
  const connAuditLogAction = useUiPreferencesStore((s) => s.connAuditLogAction);
  const connAuditLogSearch = useUiPreferencesStore((s) => s.connAuditLogSearch);
  const connAuditLogGatewayId = useUiPreferencesStore((s) => s.connAuditLogGatewayId);
  const connAuditLogUserId = useUiPreferencesStore((s) => s.connAuditLogUserId);
  const connAuditLogSortBy = useUiPreferencesStore((s) => s.connAuditLogSortBy);
  const connAuditLogSortOrder = useUiPreferencesStore((s) => s.connAuditLogSortOrder);
  const setUiPref = useUiPreferencesStore((s) => s.set);

  const tenantRole = useAuthStore((s) => s.user?.tenantRole);
  const isAdmin = hasAnyRole(tenantRole, 'ADMIN', 'OWNER', 'AUDITOR');

  const [logs, setLogs] = useState<TenantAuditLogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [ipAddress, setIpAddress] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [expandedRowId, setExpandedRowId] = useState<string | null>(null);
  const [searchInput, setSearchInput] = useState(connAuditLogSearch);
  const [gateways, setGateways] = useState<AuditGateway[]>([]);
  const [auditUsers, setAuditUsers] = useState<ConnectionAuditUser[]>([]);
  const [countries, setCountries] = useState<string[]>([]);
  const [geoCountry, setGeoCountry] = useState('');
  const [flaggedOnly, setFlaggedOnly] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => {
      setUiPref('connAuditLogSearch', searchInput);
      setPage(0);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput, setUiPref]);

  const fetchLogs = useCallback(async () => {
    if (!connectionId) return;
    setLoading(true);
    setError('');
    try {
      const params: ConnectionAuditLogParams = {
        page: page + 1,
        limit: rowsPerPage,
        sortBy: connAuditLogSortBy as 'createdAt' | 'action',
        sortOrder: connAuditLogSortOrder as 'asc' | 'desc',
      };
      if (connAuditLogAction) params.action = connAuditLogAction as AuditAction;
      if (connAuditLogSearch) params.search = connAuditLogSearch;
      if (connAuditLogGatewayId) params.gatewayId = connAuditLogGatewayId;
      if (connAuditLogUserId) params.userId = connAuditLogUserId;
      if (ipAddress) params.ipAddress = ipAddress;
      if (geoCountry) params.geoCountry = geoCountry;
      if (startDate) params.startDate = startDate;
      if (endDate) params.endDate = endDate;
      if (flaggedOnly) params.flaggedOnly = true;

      const result = await getConnectionAuditLogs(connectionId, params);
      setLogs(result.data);
      setTotal(result.total);
    } catch {
      setError('Failed to load audit logs');
    } finally {
      setLoading(false);
    }
  }, [connectionId, page, rowsPerPage, connAuditLogAction, connAuditLogSearch, connAuditLogGatewayId, connAuditLogUserId, ipAddress, geoCountry, startDate, endDate, connAuditLogSortBy, connAuditLogSortOrder, flaggedOnly]);

  useEffect(() => {
    if (open && connectionId) {
      fetchLogs();
      getAuditGateways().then(setGateways).catch(() => {});
      getAuditCountries().then(setCountries).catch(() => {});
      if (isAdmin) {
        getConnectionAuditUsers(connectionId).then(setAuditUsers).catch(() => {});
      }
    }
  }, [open, connectionId, fetchLogs, isAdmin]);

  const handleSort = (field: 'createdAt' | 'action') => {
    if (connAuditLogSortBy === field) {
      setUiPref('connAuditLogSortOrder', connAuditLogSortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setUiPref('connAuditLogSortBy', field);
      setUiPref('connAuditLogSortOrder', field === 'createdAt' ? 'desc' : 'asc');
    }
    setPage(0);
  };

  const colSpan = isAdmin ? 7 : 6;
  const hasActiveFilters = connAuditLogAction || connAuditLogSearch || connAuditLogGatewayId || connAuditLogUserId || ipAddress || geoCountry || startDate || endDate || flaggedOnly;
  const totalPages = Math.ceil(total / rowsPerPage);

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <DialogContent
        className="h-[100dvh] w-screen max-w-none gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
        showCloseButton={false}
      >
        <DialogTitle className="sr-only">Activity Log — {connectionName}</DialogTitle>
        <DialogDescription className="sr-only">Connection audit log</DialogDescription>

        {/* Header */}
        <div className="flex items-center gap-3 border-b px-4 py-2.5 bg-card">
          <Button variant="ghost" size="icon" onClick={onClose} className="size-8">
            <X className="size-4" />
          </Button>
          <h2 className="flex-1 text-lg font-semibold">Activity Log &mdash; {connectionName}</h2>
          {isAdmin && (
            <Button
              variant="ghost"
              size="sm"
              className="gap-1.5"
              onClick={() => exportCsv(logs, connectionName)}
              disabled={logs.length === 0}
            >
              <Download className="size-4" />
              Export CSV
            </Button>
          )}
        </div>

        {/* Body */}
        <div className="flex-1 overflow-auto p-4">
          {/* Filters */}
          <div className="rounded-lg border bg-card p-3 mb-4">
            <div className="relative mb-3">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
              <Input
                className="pl-9"
                placeholder="Search across IP address and details..."
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
              />
            </div>
            <div className="flex flex-wrap items-center gap-3">
              {isAdmin && auditUsers.length > 0 && (
                <div className="min-w-[220px] space-y-1">
                  <Label className="text-xs">User</Label>
                  <Select value={connAuditLogUserId || '__all__'} onValueChange={(v) => { setUiPref('connAuditLogUserId', v === '__all__' ? '' : v); setPage(0); }}>
                    <SelectTrigger>
                      <SelectValue placeholder="All Users" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__all__">All Users</SelectItem>
                      {auditUsers.map((u) => (
                        <SelectItem key={u.id} value={u.id}>{u.username ?? u.email}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
              <div className="min-w-[200px] space-y-1">
                <Label className="text-xs">Action</Label>
                <Select value={connAuditLogAction || '__all__'} onValueChange={(v) => { setUiPref('connAuditLogAction', v === '__all__' ? '' : v); setPage(0); }}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__all__">All Actions</SelectItem>
                    {ALL_ACTIONS.map((action) => (
                      <SelectItem key={action} value={action}>{ACTION_LABELS[action]}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              {gateways.length > 0 && (
                <div className="min-w-[160px] space-y-1">
                  <Label className="text-xs">Gateway</Label>
                  <Select value={connAuditLogGatewayId || '__all__'} onValueChange={(v) => { setUiPref('connAuditLogGatewayId', v === '__all__' ? '' : v); setPage(0); }}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__all__">All Gateways</SelectItem>
                      {gateways.map((gw) => (
                        <SelectItem key={gw.id} value={gw.id}>{gw.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
              <div className="w-[160px] space-y-1">
                <Label className="text-xs">IP Address</Label>
                <Input value={ipAddress} onChange={(e) => { setIpAddress(e.target.value); setPage(0); }} />
              </div>
              {countries.length > 0 && (
                <div className="min-w-[160px] space-y-1">
                  <Label className="text-xs">Country</Label>
                  <Select value={geoCountry || '__all__'} onValueChange={(v) => { setGeoCountry(v === '__all__' ? '' : v); setPage(0); }}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__all__">All Countries</SelectItem>
                      {countries.map((c) => (
                        <SelectItem key={c} value={c}>{c}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
              <div className="space-y-1">
                <Label className="text-xs">From</Label>
                <Input type="date" value={startDate} onChange={(e) => { setStartDate(e.target.value); setPage(0); }} />
              </div>
              <div className="space-y-1">
                <Label className="text-xs">To</Label>
                <Input type="date" value={endDate} onChange={(e) => { setEndDate(e.target.value); setPage(0); }} />
              </div>
              <Badge
                variant={flaggedOnly ? 'default' : 'outline'}
                className={cn(
                  'cursor-pointer gap-1 mt-5',
                  flaggedOnly ? 'bg-yellow-600/15 text-yellow-500 border-yellow-600/30' : '',
                )}
                onClick={() => { setFlaggedOnly(!flaggedOnly); setPage(0); }}
                title="Show only flagged entries (e.g. impossible travel)"
              >
                <AlertTriangle className="size-3" />
                Flagged
              </Badge>
            </div>
          </div>

          {error && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive mb-4">
              {error}
            </div>
          )}

          {/* Table */}
          <div className="rounded-lg border bg-card">
            {loading ? (
              <div className="flex justify-center py-12">
                <Loader2 className="size-8 animate-spin text-muted-foreground" />
              </div>
            ) : logs.length === 0 ? (
              <div className="text-center py-12">
                <p className="text-sm text-muted-foreground">
                  {hasActiveFilters ? 'No logs match your filters' : 'No activity recorded yet'}
                </p>
              </div>
            ) : (
              <>
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/50">
                      <th className="w-8 px-2 py-2" />
                      <th className="text-left px-3 py-2 font-medium">
                        <button className="inline-flex items-center gap-1 hover:text-foreground" onClick={() => handleSort('createdAt')}>
                          Date/Time
                          {connAuditLogSortBy === 'createdAt' && (
                            connAuditLogSortOrder === 'asc' ? <ChevronUp className="size-3" /> : <ChevronDown className="size-3" />
                          )}
                        </button>
                      </th>
                      {isAdmin && <th className="text-left px-3 py-2 font-medium">User</th>}
                      <th className="text-left px-3 py-2 font-medium">
                        <button className="inline-flex items-center gap-1 hover:text-foreground" onClick={() => handleSort('action')}>
                          Action
                          {connAuditLogSortBy === 'action' && (
                            connAuditLogSortOrder === 'asc' ? <ChevronUp className="size-3" /> : <ChevronDown className="size-3" />
                          )}
                        </button>
                      </th>
                      <th className="text-left px-3 py-2 font-medium">IP Address</th>
                      <th className="text-left px-3 py-2 font-medium">Details</th>
                    </tr>
                  </thead>
                  <tbody>
                    {logs.map((log) => {
                      const isExpanded = expandedRowId === log.id;
                      return (
                        <Fragment key={log.id}>
                          <tr
                            className="border-b hover:bg-accent/50 cursor-pointer"
                            onClick={() => setExpandedRowId(isExpanded ? null : log.id)}
                          >
                            <td className="px-2 py-2">
                              <Button variant="ghost" size="icon" className="size-6">
                                {isExpanded ? <ChevronUp className="size-3.5" /> : <ChevronDown className="size-3.5" />}
                              </Button>
                            </td>
                            <td className="px-3 py-2 whitespace-nowrap">
                              {new Date(log.createdAt).toLocaleString()}
                            </td>
                            {isAdmin && (
                              <td className="px-3 py-2 whitespace-nowrap">
                                {log.userName ?? log.userEmail ?? '\u2014'}
                              </td>
                            )}
                            <td className="px-3 py-2">
                              <div className="inline-flex items-center gap-1.5">
                                <Badge variant="outline" className={cn('border', ACTION_COLOR_MAP[getActionColor(log.action) as string] || '')}>
                                  {ACTION_LABELS[log.action] || log.action}
                                </Badge>
                                {log.flags?.includes('IMPOSSIBLE_TRAVEL') && (
                                  <span title="Impossible travel detected"><AlertTriangle className="size-4 text-yellow-500" /></span>
                                )}
                              </div>
                            </td>
                            <td className="px-3 py-2">
                              <IpGeoCell ipAddress={log.ipAddress} geoCountry={log.geoCountry} geoCity={log.geoCity} onGeoIpClick={onGeoIpClick} />
                            </td>
                            <td className="px-3 py-2 max-w-[300px] overflow-hidden text-ellipsis whitespace-nowrap">
                              {formatDetails(log.details as Record<string, unknown> | null)}
                            </td>
                          </tr>
                          {isExpanded && (
                            <tr>
                              <td colSpan={colSpan} className="px-6 py-4 border-b">
                                {log.details && typeof log.details === 'object' && Object.keys(log.details as object).length > 0 ? (
                                  <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 max-w-[600px]">
                                    {Object.entries(log.details as Record<string, unknown>).map(([key, value]) => (
                                      <Fragment key={key}>
                                        <span className="text-sm font-semibold text-muted-foreground">{key}</span>
                                        <span className="text-sm break-all">
                                          {Array.isArray(value) ? value.join(', ') : String(value)}
                                        </span>
                                      </Fragment>
                                    ))}
                                  </div>
                                ) : (
                                  <p className="text-sm text-muted-foreground">No additional details</p>
                                )}
                                {isAdmin && log.userEmail && (
                                  <p className="text-xs text-muted-foreground mt-2">Email: {log.userEmail}</p>
                                )}
                              </td>
                            </tr>
                          )}
                        </Fragment>
                      );
                    })}
                  </tbody>
                </table>
                {/* Pagination */}
                <div className="flex items-center justify-between px-4 py-2 border-t text-sm text-muted-foreground">
                  <div className="flex items-center gap-2">
                    <span>Rows per page:</span>
                    <Select value={String(rowsPerPage)} onValueChange={(v) => { setRowsPerPage(parseInt(v, 10)); setPage(0); }}>
                      <SelectTrigger className="h-8 w-[70px]">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="25">25</SelectItem>
                        <SelectItem value="50">50</SelectItem>
                        <SelectItem value="100">100</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="flex items-center gap-2">
                    <span>{page * rowsPerPage + 1}-{Math.min((page + 1) * rowsPerPage, total)} of {total}</span>
                    <Button variant="ghost" size="sm" disabled={page === 0} onClick={() => setPage((p) => p - 1)}>Previous</Button>
                    <Button variant="ghost" size="sm" disabled={page + 1 >= totalPages} onClick={() => setPage((p) => p + 1)}>Next</Button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
