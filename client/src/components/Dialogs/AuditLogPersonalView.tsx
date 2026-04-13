import {
  AlertTriangle,
  ChevronDown,
  ChevronUp,
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
import type { AuditGateway, AuditLogEntry } from '../../api/audit.api';
import type { Recording } from '../../api/recordings.api';
import {
  ACTION_LABELS,
  ALL_ACTIONS,
  TARGET_TYPES,
} from '../Audit/auditConstants';
import RecordingPlayerDialog from '../Recording/RecordingPlayerDialog';
import AuditLogPersonalRow from './AuditLogPersonalRow';

interface AuditLogPersonalViewProps {
  action: string;
  auditLogSortBy: string;
  auditLogSortOrder: string;
  countries: string[];
  endDate: string;
  error: string;
  expandedRowId: string | null;
  flaggedOnly: boolean;
  gatewayId: string;
  gateways: AuditGateway[];
  geoCountry: string;
  ipAddress: string;
  loading: boolean;
  loadingRecordingId: string | null;
  logs: AuditLogEntry[];
  onActionChange: (value: string) => void;
  onCountryChange: (value: string) => void;
  onEndDateChange: (value: string) => void;
  onFlaggedToggle: () => void;
  onGatewayChange: (value: string) => void;
  onGeoIpClick?: (ip: string) => void;
  onHandleSort: (field: 'createdAt' | 'action') => void;
  onIpAddressChange: (value: string) => void;
  onNextPage: () => void;
  onPreviousPage: () => void;
  onRowsPerPageChange: (value: number) => void;
  onSearchChange: (value: string) => void;
  onStartDateChange: (value: string) => void;
  onTargetTypeChange: (value: string) => void;
  onToggleRow: (logId: string) => void;
  onViewRecording: (log: AuditLogEntry) => void;
  page: number;
  recordingPlayerOpen: boolean;
  rowsPerPage: number;
  searchInput: string;
  selectedRecording: Recording | null;
  startDate: string;
  targetType: string;
  total: number;
  onCloseRecordingPlayer: () => void;
}

export default function AuditLogPersonalView({
  action,
  auditLogSortBy,
  auditLogSortOrder,
  countries,
  endDate,
  error,
  expandedRowId,
  flaggedOnly,
  gatewayId,
  gateways,
  geoCountry,
  ipAddress,
  loading,
  loadingRecordingId,
  logs,
  onActionChange,
  onCloseRecordingPlayer,
  onCountryChange,
  onEndDateChange,
  onFlaggedToggle,
  onGatewayChange,
  onGeoIpClick,
  onHandleSort,
  onIpAddressChange,
  onNextPage,
  onPreviousPage,
  onRowsPerPageChange,
  onSearchChange,
  onStartDateChange,
  onTargetTypeChange,
  onToggleRow,
  onViewRecording,
  page,
  recordingPlayerOpen,
  rowsPerPage,
  searchInput,
  selectedRecording,
  startDate,
  targetType,
  total,
}: AuditLogPersonalViewProps) {
  const hasActiveFilters = Boolean(
    action || searchInput || targetType || gatewayId || ipAddress || geoCountry || startDate || endDate || flaggedOnly,
  );
  const totalPages = Math.max(1, Math.ceil(total / rowsPerPage));

  return (
    <>
      <div className="space-y-4">
        <div className="rounded-lg border bg-card p-3">
          <div className="relative mb-3">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              className="pl-9"
              placeholder="Search across target, IP address, and details..."
              value={searchInput}
              onChange={(event) => onSearchChange(event.target.value)}
            />
          </div>
          <div className="flex flex-wrap items-center gap-3">
            <div className="min-w-[200px] space-y-1">
              <Label className="text-xs">Action</Label>
              <Select
                value={action || '__all__'}
                onValueChange={(value) => onActionChange(value === '__all__' ? '' : value)}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">All Actions</SelectItem>
                  {ALL_ACTIONS.map((item) => (
                    <SelectItem key={item} value={item}>{ACTION_LABELS[item]}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="min-w-[160px] space-y-1">
              <Label className="text-xs">Target Type</Label>
              <Select
                value={targetType || '__all__'}
                onValueChange={(value) => onTargetTypeChange(value === '__all__' ? '' : value)}
              >
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="__all__">All Types</SelectItem>
                  {TARGET_TYPES.map((item) => (
                    <SelectItem key={item} value={item}>{item}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            {gateways.length > 0 ? (
              <div className="min-w-[160px] space-y-1">
                <Label className="text-xs">Gateway</Label>
                <Select
                  value={gatewayId || '__all__'}
                  onValueChange={(value) => onGatewayChange(value === '__all__' ? '' : value)}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__all__">All Gateways</SelectItem>
                    {gateways.map((gateway) => (
                      <SelectItem key={gateway.id} value={gateway.id}>{gateway.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : null}
            {countries.length > 0 ? (
              <div className="min-w-[160px] space-y-1">
                <Label className="text-xs">Country</Label>
                <Select
                  value={geoCountry || '__all__'}
                  onValueChange={(value) => onCountryChange(value === '__all__' ? '' : value)}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__all__">All Countries</SelectItem>
                    {countries.map((country) => (
                      <SelectItem key={country} value={country}>{country}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            ) : null}
            <div className="w-[160px] space-y-1">
              <Label className="text-xs">IP Address</Label>
              <Input value={ipAddress} onChange={(event) => onIpAddressChange(event.target.value)} />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">From</Label>
              <Input type="date" value={startDate} onChange={(event) => onStartDateChange(event.target.value)} />
            </div>
            <div className="space-y-1">
              <Label className="text-xs">To</Label>
              <Input type="date" value={endDate} onChange={(event) => onEndDateChange(event.target.value)} />
            </div>
            <Badge
              variant={flaggedOnly ? 'default' : 'outline'}
              className={cn(
                'mt-5 cursor-pointer gap-1',
                flaggedOnly ? 'bg-yellow-600/15 text-yellow-500 border-yellow-600/30' : '',
              )}
              onClick={onFlaggedToggle}
              title="Show only flagged entries"
            >
              <AlertTriangle className="size-3" />
              Flagged
            </Badge>
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
                {hasActiveFilters ? 'No logs match your filters' : 'No activity recorded yet'}
              </p>
            </div>
          ) : (
            <>
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50">
                    <th className="w-8 px-2 py-2" />
                    <th className="px-3 py-2 text-left font-medium">
                      <button className="inline-flex items-center gap-1 hover:text-foreground" onClick={() => onHandleSort('createdAt')}>
                        Date/Time
                        {auditLogSortBy === 'createdAt' ? (
                          auditLogSortOrder === 'asc' ? <ChevronUp className="size-3" /> : <ChevronDown className="size-3" />
                        ) : null}
                      </button>
                    </th>
                    <th className="px-3 py-2 text-left font-medium">
                      <button className="inline-flex items-center gap-1 hover:text-foreground" onClick={() => onHandleSort('action')}>
                        Action
                        {auditLogSortBy === 'action' ? (
                          auditLogSortOrder === 'asc' ? <ChevronUp className="size-3" /> : <ChevronDown className="size-3" />
                        ) : null}
                      </button>
                    </th>
                    <th className="px-3 py-2 text-left font-medium">Target</th>
                    <th className="px-3 py-2 text-left font-medium">IP Address</th>
                    <th className="px-3 py-2 text-left font-medium">Details</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((log) => {
                    const isExpanded = expandedRowId === log.id;
                    return (
                      <AuditLogPersonalRow
                        key={log.id}
                        expanded={isExpanded}
                        loadingRecordingId={loadingRecordingId}
                        log={log}
                        onGeoIpClick={onGeoIpClick}
                        onToggle={() => onToggleRow(log.id)}
                        onViewRecording={() => onViewRecording(log)}
                      />
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

      <RecordingPlayerDialog
        open={recordingPlayerOpen}
        onClose={onCloseRecordingPlayer}
        recording={selectedRecording}
      />
    </>
  );
}
