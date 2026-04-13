import { Fragment, useState } from 'react';
import {
  ChevronDown,
  ChevronUp,
  Eye,
  Loader2,
  Search,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn } from '@/lib/utils';
import type {
  DbAuditConnection,
  DbAuditLogEntry,
  DbAuditUser,
  DbQueryType,
} from '../../api/dbAudit.api';
import QueryVisualizer from '../DatabaseClient/QueryVisualizer';

const QUERY_TYPE_LABELS: Record<DbQueryType, string> = {
  SELECT: 'SELECT',
  INSERT: 'INSERT',
  UPDATE: 'UPDATE',
  DELETE: 'DELETE',
  DDL: 'DDL',
  OTHER: 'Other',
};

const QUERY_TYPE_COLORS: Record<DbQueryType, string> = {
  SELECT: 'bg-blue-600/15 text-blue-400 border-blue-600/30',
  INSERT: 'bg-emerald-600/15 text-emerald-400 border-emerald-600/30',
  UPDATE: 'bg-primary/15 text-primary border-primary/30',
  DELETE: 'bg-destructive/15 text-destructive border-destructive/30',
  DDL: 'bg-yellow-600/15 text-yellow-500 border-yellow-600/30',
  OTHER: '',
};

const ALL_QUERY_TYPES: DbQueryType[] = ['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'DDL', 'OTHER'];

interface AuditLogSqlActivityViewProps {
  connections: DbAuditConnection[];
  endDate: string;
  error: string;
  expandedRowId: string | null;
  hasActiveFilters: boolean;
  loading: boolean;
  logs: DbAuditLogEntry[];
  onBlockedChange: (value: string) => void;
  onConnectionChange: (value: string) => void;
  onEndDateChange: (value: string) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onQueryTypeChange: (value: string) => void;
  onRowsPerPageChange: (value: number) => void;
  onSearchChange: (value: string) => void;
  onStartDateChange: (value: string) => void;
  onToggleRow: (rowId: string) => void;
  onUserChange: (value: string) => void;
  page: number;
  rowsPerPage: number;
  search: string;
  selectedBlocked: string;
  selectedConnectionId: string;
  selectedQueryType: string;
  selectedUserId: string;
  startDate: string;
  total: number;
  totalPages: number;
  users: DbAuditUser[];
}

export default function AuditLogSqlActivityView({
  connections,
  endDate,
  error,
  expandedRowId,
  hasActiveFilters,
  loading,
  logs,
  onBlockedChange,
  onConnectionChange,
  onEndDateChange,
  onNextPage,
  onPreviousPage,
  onQueryTypeChange,
  onRowsPerPageChange,
  onSearchChange,
  onStartDateChange,
  onToggleRow,
  onUserChange,
  page,
  rowsPerPage,
  search,
  selectedBlocked,
  selectedConnectionId,
  selectedQueryType,
  selectedUserId,
  startDate,
  total,
  totalPages,
  users,
}: AuditLogSqlActivityViewProps) {
  const [visualizerEntry, setVisualizerEntry] = useState<DbAuditLogEntry | null>(null);

  return (
    <>
      <div className="space-y-4">
        <div className="rounded-lg border bg-card p-3">
          <div className="relative mb-3">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-9"
              placeholder="Search SQL queries, tables, or block reasons..."
              value={search}
              onChange={(event) => onSearchChange(event.target.value)}
            />
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <div className="min-w-[140px] space-y-1">
              <Label className="text-xs">Query Type</Label>
              <Select
                value={selectedQueryType || '__all__'}
                onValueChange={(value) => onQueryTypeChange(value === '__all__' ? '' : value)}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">All Types</SelectItem>
                  {ALL_QUERY_TYPES.map((queryType) => (
                    <SelectItem key={queryType} value={queryType}>{QUERY_TYPE_LABELS[queryType]}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {connections.length > 0 ? (
              <div className="min-w-[160px] space-y-1">
                <Label className="text-xs">Connection</Label>
                <Select
                  value={selectedConnectionId || '__all__'}
                  onValueChange={(value) => onConnectionChange(value === '__all__' ? '' : value)}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__all__">All Connections</SelectItem>
                    {connections.map((connection) => (
                      <SelectItem key={connection.id} value={connection.id}>{connection.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : null}
            {users.length > 0 ? (
              <div className="min-w-[160px] space-y-1">
                <Label className="text-xs">User</Label>
                <Select
                  value={selectedUserId || '__all__'}
                  onValueChange={(value) => onUserChange(value === '__all__' ? '' : value)}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__all__">All Users</SelectItem>
                    {users.map((entry) => (
                      <SelectItem key={entry.id} value={entry.id}>{entry.username || entry.email}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : null}
            <div className="min-w-[130px] space-y-1">
              <Label className="text-xs">Status</Label>
              <Select
                value={selectedBlocked || '__all__'}
                onValueChange={(value) => onBlockedChange(value === '__all__' ? '' : value)}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">All</SelectItem>
                  <SelectItem value="true">Blocked</SelectItem>
                  <SelectItem value="false">Allowed</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label className="text-xs">From</Label>
              <Input type="date" value={startDate} onChange={(event) => onStartDateChange(event.target.value)} />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">To</Label>
              <Input type="date" value={endDate} onChange={(event) => onEndDateChange(event.target.value)} />
            </div>
          </div>
        </div>

        {error ? (
          <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive">
            {error}
          </div>
        ) : null}

        <div className="rounded-lg border bg-card">
          {loading ? (
            <div className="flex justify-center py-12">
              <Loader2 className="size-8 animate-spin text-muted-foreground" />
            </div>
          ) : logs.length === 0 ? (
            <div className="py-12 text-center">
              <p className="text-sm text-muted-foreground">
                {hasActiveFilters ? 'No SQL audit logs match your filters' : 'No SQL queries recorded yet'}
              </p>
            </div>
          ) : (
            <>
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="w-8 px-2 py-2" />
                    <th className="px-3 py-2 text-left font-medium">Date/Time</th>
                    <th className="px-3 py-2 text-left font-medium">User</th>
                    <th className="px-3 py-2 text-left font-medium">Connection</th>
                    <th className="px-3 py-2 text-left font-medium">Type</th>
                    <th className="px-3 py-2 text-left font-medium">Tables</th>
                    <th className="px-3 py-2 text-left font-medium">Status</th>
                    <th className="px-3 py-2 text-left font-medium">Time (ms)</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((entry) => {
                    const isExpanded = expandedRowId === entry.id;
                    return (
                      <Fragment key={entry.id}>
                        <tr
                          className="cursor-pointer border-b hover:bg-accent/50"
                          onClick={() => onToggleRow(entry.id)}
                        >
                          <td className="px-2 py-2">
                            <Button variant="ghost" size="icon" className="size-6">
                              {isExpanded ? <ChevronUp className="size-3.5" /> : <ChevronDown className="size-3.5" />}
                            </Button>
                          </td>
                          <td className="whitespace-nowrap px-3 py-2">{new Date(entry.createdAt).toLocaleString()}</td>
                          <td className="px-3 py-2">{entry.userName || entry.userEmail || entry.userId.slice(0, 8)}</td>
                          <td className="px-3 py-2">{entry.connectionName || entry.connectionId.slice(0, 8)}</td>
                          <td className="px-3 py-2">
                            <Badge variant="outline" className={cn('border', QUERY_TYPE_COLORS[entry.queryType] || '')}>
                              {QUERY_TYPE_LABELS[entry.queryType] || entry.queryType}
                            </Badge>
                          </td>
                          <td className="max-w-[200px] overflow-hidden px-3 py-2 text-ellipsis whitespace-nowrap">
                            {entry.tablesAccessed.length > 0 ? entry.tablesAccessed.join(', ') : '\u2014'}
                          </td>
                          <td className="px-3 py-2">
                            {entry.blocked ? (
                              <Badge variant="outline" className="border border-destructive/30 bg-destructive/15 text-destructive">Blocked</Badge>
                            ) : entry.blockReason ? (
                              <Badge variant="outline" className="border border-yellow-600/30 bg-yellow-600/15 text-yellow-500">Alert</Badge>
                            ) : (
                              <Badge variant="outline" className="border border-emerald-600/30 bg-emerald-600/15 text-emerald-400">OK</Badge>
                            )}
                          </td>
                          <td className="px-3 py-2">
                            {entry.executionTimeMs !== null ? `${entry.executionTimeMs}` : '\u2014'}
                          </td>
                        </tr>
                        {isExpanded ? (
                          <tr>
                            <td colSpan={8} className="max-w-[800px] border-b px-6 py-4">
                              <p className="mb-1 text-sm font-semibold text-muted-foreground">Query</p>
                              <div className="mb-3 rounded bg-accent/50 p-3 font-mono text-[0.85rem] whitespace-pre-wrap break-all">
                                {entry.queryText}
                              </div>
                              <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
                                <span className="text-sm font-semibold text-muted-foreground">Rows Affected</span>
                                <span className="text-sm">{entry.rowsAffected ?? '\u2014'}</span>
                                {entry.blockReason ? (
                                  <>
                                    <span className="text-sm font-semibold text-muted-foreground">
                                      {entry.blocked ? 'Block Reason' : 'Firewall Alert'}
                                    </span>
                                    <span className={cn('text-sm', entry.blocked ? 'text-destructive' : 'text-yellow-500')}>
                                      {entry.blockReason}
                                    </span>
                                  </>
                                ) : null}
                              </div>
                              <div className="mt-3">
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="size-7 text-primary"
                                  onClick={(event) => {
                                    event.stopPropagation();
                                    setVisualizerEntry(entry);
                                  }}
                                  title="Open query visualizer"
                                >
                                  <Eye className="size-4" />
                                </Button>
                              </div>
                            </td>
                          </tr>
                        ) : null}
                      </Fragment>
                    );
                  })}
                </tbody>
              </table>
              <div className="flex items-center justify-between border-t px-4 py-2 text-sm text-muted-foreground">
                <div className="flex items-center gap-2">
                  <span>Rows per page:</span>
                  <Select
                    value={String(rowsPerPage)}
                    onValueChange={(value) => onRowsPerPageChange(Number.parseInt(value, 10))}
                  >
                    <SelectTrigger className="h-8 w-[70px]"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="25">25</SelectItem>
                      <SelectItem value="50">50</SelectItem>
                      <SelectItem value="100">100</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex items-center gap-2">
                  <span>
                    {page * rowsPerPage + 1}-{Math.min((page + 1) * rowsPerPage, total)} of {total}
                  </span>
                  <Button variant="ghost" size="sm" disabled={page === 0} onClick={onPreviousPage}>
                    Previous
                  </Button>
                  <Button variant="ghost" size="sm" disabled={page + 1 >= totalPages} onClick={onNextPage}>
                    Next
                  </Button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      <QueryVisualizer
        open={Boolean(visualizerEntry)}
        onClose={() => setVisualizerEntry(null)}
        queryText={visualizerEntry?.queryText ?? ''}
        queryType={visualizerEntry?.queryType ?? ''}
        executionTimeMs={visualizerEntry?.executionTimeMs ?? null}
        rowsAffected={visualizerEntry?.rowsAffected ?? null}
        tablesAccessed={visualizerEntry?.tablesAccessed ?? []}
        blocked={visualizerEntry?.blocked ?? false}
        blockReason={visualizerEntry?.blockReason}
        storedExecutionPlan={visualizerEntry?.executionPlan ?? null}
      />
    </>
  );
}
