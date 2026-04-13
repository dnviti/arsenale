import { lazy, Suspense, useState } from 'react';
import {
  ChevronDown,
  ChevronUp,
  Download,
  ListFilter,
  Loader2,
  Map,
  Search,
  SlidersHorizontal,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type {
  AuditGateway,
  AuditLogEntry,
  AuditAction,
  TenantAuditLogEntry,
  TenantGeoSummaryParams,
} from '../../api/audit.api';
import type { Recording } from '../../api/recordings.api';
import type { TenantUser } from '../../api/tenant.api';
import { countActiveFilters, exportTenantAuditCsv } from '../Settings/tenantAuditLogUtils';
import RecordingPlayerDialog from '../Recording/RecordingPlayerDialog';
import AuditLogTenantFilters from './AuditLogTenantFilters';
import AuditLogTenantRow from './AuditLogTenantRow';

const AuditGeoMap = lazy(() => import('../Audit/AuditGeoMap'));

interface AuditLogTenantViewProps {
  action: string;
  countries: string[];
  effectiveViewMode: 'map' | 'table';
  endDate: string;
  error: string;
  expandedRowId: string | null;
  flaggedOnly: boolean;
  gatewayId: string;
  gateways: AuditGateway[];
  geoCountry: string;
  ipAddress: string;
  ipGeolocationEnabled: boolean;
  loading: boolean;
  loadingRecordingId: string | null;
  onActionChange: (value: string) => void;
  onClearFilters: () => void;
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
  onSortByChange: (value: string) => void;
  onSortOrderChange: (value: string) => void;
  onStartDateChange: (value: string) => void;
  onTargetTypeChange: (value: string) => void;
  onToggleRow: (logId: string) => void;
  onUserChange: (value: string) => void;
  onViewModeChange: (mode: 'map' | 'table') => void;
  onViewRecording: (log: AuditLogEntry) => void;
  onViewUserProfile?: (userId: string) => void;
  page: number;
  recordingPlayerOpen: boolean;
  rowsPerPage: number;
  search: string;
  searchInput: string;
  selectedRecording: Recording | null;
  sortBy: string;
  sortOrder: string;
  startDate: string;
  targetType: string;
  tenantLogs: TenantAuditLogEntry[];
  total: number;
  userId: string;
  users: TenantUser[];
  onCloseRecordingPlayer: () => void;
}

export default function AuditLogTenantView({
  action,
  countries,
  effectiveViewMode,
  endDate,
  error,
  expandedRowId,
  flaggedOnly,
  gatewayId,
  gateways,
  geoCountry,
  ipAddress,
  ipGeolocationEnabled,
  loading,
  loadingRecordingId,
  onActionChange,
  onClearFilters,
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
  onSortByChange,
  onSortOrderChange,
  onStartDateChange,
  onTargetTypeChange,
  onToggleRow,
  onUserChange,
  onViewModeChange,
  onViewRecording,
  onViewUserProfile,
  page,
  recordingPlayerOpen,
  rowsPerPage,
  search,
  searchInput,
  selectedRecording,
  sortBy,
  sortOrder,
  startDate,
  targetType,
  tenantLogs,
  total,
  userId,
  users,
}: AuditLogTenantViewProps) {
  const advancedFiltersCount = countActiveFilters([
    action,
    targetType,
    gatewayId,
    userId,
    ipAddress,
    geoCountry,
    startDate,
    endDate,
    flaggedOnly,
  ]);
  const [advancedOpen, setAdvancedOpen] = useState(advancedFiltersCount > 0);
  const hasActiveFilters = Boolean(searchInput || advancedFiltersCount > 0);
  const totalPages = Math.max(1, Math.ceil(total / rowsPerPage));
  const mapFilters: TenantGeoSummaryParams = {
    action: action ? action as AuditAction : undefined,
    search: search || undefined,
    targetType: targetType || undefined,
    gatewayId: gatewayId || undefined,
    userId: userId || undefined,
    ipAddress: ipAddress || undefined,
    geoCountry: geoCountry || undefined,
    startDate: startDate || undefined,
    endDate: endDate || undefined,
    flaggedOnly: flaggedOnly || undefined,
  };

  return (
    <>
      <div className="space-y-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="space-y-1">
            <h3 className="font-heading text-lg font-medium tracking-tight text-foreground">
              Organization Activity
            </h3>
            <p className="text-sm leading-6 text-muted-foreground">
              Review tenant activity in the audit table, expand into advanced filters, and switch to the map to plot the same filtered geolocated events.
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant={effectiveViewMode === 'table' ? 'default' : 'outline'}
              size="sm"
              onClick={() => onViewModeChange('table')}
            >
              <ListFilter className="size-4" />
              Activity
            </Button>
            {ipGeolocationEnabled ? (
              <Button
                type="button"
                variant={effectiveViewMode === 'map' ? 'default' : 'outline'}
                size="sm"
                onClick={() => onViewModeChange('map')}
              >
                <Map className="size-4" />
                Map
              </Button>
            ) : null}
            <Button type="button" variant="outline" size="sm" disabled={!tenantLogs.length} onClick={() => exportTenantAuditCsv(tenantLogs)}>
              <Download className="size-4" />
              Export CSV
            </Button>
          </div>
        </div>

        <Collapsible open={advancedOpen} onOpenChange={setAdvancedOpen}>
          <div className="rounded-lg border bg-card p-3">
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
              <div className="relative flex-1">
                <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  aria-label="Search activity"
                  className="pl-9"
                  placeholder="Search activity, targets, IPs, and details..."
                  value={searchInput}
                  onChange={(event) => onSearchChange(event.target.value)}
                />
              </div>
              <CollapsibleTrigger asChild>
                <Button type="button" variant="outline" size="sm" className="gap-2">
                  <SlidersHorizontal className="size-4" />
                  Advanced Search
                  {advancedFiltersCount ? ` (${advancedFiltersCount})` : ''}
                  {advancedOpen ? <ChevronUp className="size-4" /> : <ChevronDown className="size-4" />}
                </Button>
              </CollapsibleTrigger>
            </div>

            <CollapsibleContent>
              <AuditLogTenantFilters
                action={action}
                countries={countries}
                endDate={endDate}
                flaggedOnly={flaggedOnly}
                gatewayId={gatewayId}
                gateways={gateways}
                geoCountry={geoCountry}
                ipAddress={ipAddress}
                onActionChange={onActionChange}
                onClearFilters={onClearFilters}
                onCountryChange={onCountryChange}
                onEndDateChange={onEndDateChange}
                onFlaggedToggle={onFlaggedToggle}
                onGatewayChange={onGatewayChange}
                onIpAddressChange={onIpAddressChange}
                onSortByChange={onSortByChange}
                onSortOrderChange={onSortOrderChange}
                onStartDateChange={onStartDateChange}
                onTargetTypeChange={onTargetTypeChange}
                onUserChange={onUserChange}
                sortBy={sortBy}
                sortOrder={sortOrder}
                startDate={startDate}
                targetType={targetType}
                userId={userId}
                users={users}
              />
            </CollapsibleContent>
          </div>
        </Collapsible>

        {effectiveViewMode === 'map' ? (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              The map plots the same filtered geolocated activity shown in the table and aggregates nearby points as you zoom out.
            </p>
            <div className="overflow-hidden rounded-lg border bg-card">
              <Suspense
                fallback={(
                  <div className="flex h-[28rem] items-center justify-center gap-2 text-sm text-muted-foreground">
                    <Loader2 className="size-4 animate-spin" />
                    Loading activity map...
                  </div>
                )}
              >
                <AuditGeoMap
                  countLabel="audit events"
                  emptyMessage="No geolocated audit entries matched the current filters."
                  filters={mapFilters}
                  onSelectCountry={(country) => {
                    onCountryChange(country);
                  }}
                />
              </Suspense>
            </div>
          </div>
        ) : (
          <>
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
              ) : tenantLogs.length === 0 ? (
                <div className="py-12 text-center">
                  <p className="text-sm text-muted-foreground">
                    {hasActiveFilters ? 'No audit entries match your filters' : 'No activity recorded yet'}
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
                            {sortBy === 'createdAt' ? (
                              sortOrder === 'asc' ? <ChevronUp className="size-3" /> : <ChevronDown className="size-3" />
                            ) : null}
                          </button>
                        </th>
                        <th className="px-3 py-2 text-left font-medium">User</th>
                        <th className="px-3 py-2 text-left font-medium">
                          <button className="inline-flex items-center gap-1 hover:text-foreground" onClick={() => onHandleSort('action')}>
                            Action
                            {sortBy === 'action' ? (
                              sortOrder === 'asc' ? <ChevronUp className="size-3" /> : <ChevronDown className="size-3" />
                            ) : null}
                          </button>
                        </th>
                        <th className="px-3 py-2 text-left font-medium">Target</th>
                        <th className="px-3 py-2 text-left font-medium">IP Address</th>
                        <th className="px-3 py-2 text-left font-medium">Details</th>
                      </tr>
                    </thead>
                    <tbody>
                      {tenantLogs.map((log) => (
                        <AuditLogTenantRow
                          key={log.id}
                          expanded={expandedRowId === log.id}
                          loadingRecordingId={loadingRecordingId}
                          log={log}
                          onGeoIpClick={onGeoIpClick}
                          onToggle={() => onToggleRow(log.id)}
                          onViewRecording={() => onViewRecording(log)}
                          onViewUserProfile={onViewUserProfile}
                        />
                      ))}
                    </tbody>
                  </table>

                  <div className="flex items-center justify-between border-t px-4 py-2 text-sm text-muted-foreground">
            <div className="flex items-center gap-2">
              <span>Rows per page:</span>
              <Select value={String(rowsPerPage)} onValueChange={(value) => onRowsPerPageChange(Number.parseInt(value, 10))}>
                <SelectTrigger className="h-8 w-[90px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {[25, 50, 100].map((value) => (
                    <SelectItem key={value} value={String(value)}>
                      {value}
                    </SelectItem>
                  ))}
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
          </>
        )}
      </div>

      <RecordingPlayerDialog
        open={recordingPlayerOpen}
        onClose={onCloseRecordingPlayer}
        recording={selectedRecording}
      />
    </>
  );
}
