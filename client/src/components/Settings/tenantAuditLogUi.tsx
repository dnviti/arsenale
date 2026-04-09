import type { ReactNode } from 'react';
import { Download, ListFilter, Loader2, Map, Search, UserRound, X } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import type { AuditGateway, TenantAuditLogEntry } from '../../api/audit.api';
import type { TenantUser } from '../../api/tenant.api';
import { ACTION_LABELS, ALL_ACTIONS, TARGET_TYPES } from '../Audit/auditConstants';
import { SettingsButtonRow, SettingsSummaryGrid, SettingsSummaryItem } from './settings-ui';
import { ALL_VALUE, exportTenantAuditCsv } from './tenantAuditLogUtils';
import TenantAuditEntryCard from './tenantAuditLogEntryCard';

function FilterField({
  children,
  label,
}: {
  children: ReactNode;
  label: string;
}) {
  return <div className="space-y-2"><Label>{label}</Label>{children}</div>;
}

export function TenantAuditToolbar({
  effectiveViewMode,
  hasLogs,
  ipGeolocationEnabled,
  onExport,
  onViewModeChange,
}: {
  effectiveViewMode: 'map' | 'table';
  hasLogs: boolean;
  ipGeolocationEnabled: boolean;
  onExport: () => void;
  onViewModeChange: (mode: 'map' | 'table') => void;
}) {
  return (
    <SettingsButtonRow>
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
      <Button type="button" variant="outline" size="sm" disabled={!hasLogs} onClick={onExport}>
        <Download className="size-4" />
        Export CSV
      </Button>
    </SettingsButtonRow>
  );
}

export function TenantAuditFiltersCard({
  action,
  countries,
  endDate,
  flaggedOnly,
  gatewayId,
  gateways,
  geoCountry,
  ipAddress,
  onActionChange,
  onClearFilters,
  onCountryChange,
  onEndDateChange,
  onFlaggedToggle,
  onGatewayChange,
  onIpAddressChange,
  onSearchChange,
  onSortByChange,
  onSortOrderChange,
  onStartDateChange,
  onTargetTypeChange,
  onUserChange,
  searchInput,
  sortBy,
  sortOrder,
  startDate,
  targetType,
  userId,
  users,
}: {
  action: string;
  countries: string[];
  endDate: string;
  flaggedOnly: boolean;
  gatewayId: string;
  gateways: AuditGateway[];
  geoCountry: string;
  ipAddress: string;
  onActionChange: (value: string) => void;
  onClearFilters: () => void;
  onCountryChange: (value: string) => void;
  onEndDateChange: (value: string) => void;
  onFlaggedToggle: () => void;
  onGatewayChange: (value: string) => void;
  onIpAddressChange: (value: string) => void;
  onSearchChange: (value: string) => void;
  onSortByChange: (value: string) => void;
  onSortOrderChange: (value: string) => void;
  onStartDateChange: (value: string) => void;
  onTargetTypeChange: (value: string) => void;
  onUserChange: (value: string) => void;
  searchInput: string;
  sortBy: string;
  sortOrder: string;
  startDate: string;
  targetType: string;
  userId: string;
  users: TenantUser[];
}) {
  return (
    <Card>
      <CardContent className="space-y-4 pt-6">
        <div className="space-y-2">
          <Label htmlFor="tenant-audit-search">Search activity</Label>
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              id="tenant-audit-search"
              value={searchInput}
              placeholder="Search targets, IPs, and details"
              className="pl-9"
              onChange={(event) => onSearchChange(event.target.value)}
            />
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <FilterField label="User">
            <Select value={userId || ALL_VALUE} onValueChange={(value) => onUserChange(value === ALL_VALUE ? '' : value)}>
              <SelectTrigger>
                <SelectValue placeholder="All users" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>All users</SelectItem>
                {users.map((user) => (
                  <SelectItem key={user.id} value={user.id}>{user.username ?? user.email}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FilterField>

          <FilterField label="Action">
            <Select value={action || ALL_VALUE} onValueChange={(value) => onActionChange(value === ALL_VALUE ? '' : value)}>
              <SelectTrigger>
                <SelectValue placeholder="All actions" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>All actions</SelectItem>
                {ALL_ACTIONS.map((entry) => (
                  <SelectItem key={entry} value={entry}>{ACTION_LABELS[entry]}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FilterField>

          <FilterField label="Target type">
            <Select value={targetType || ALL_VALUE} onValueChange={(value) => onTargetTypeChange(value === ALL_VALUE ? '' : value)}>
              <SelectTrigger>
                <SelectValue placeholder="All target types" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>All target types</SelectItem>
                {TARGET_TYPES.map((type) => (
                  <SelectItem key={type} value={type}>{type}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FilterField>

          <FilterField label="Gateway">
            <Select value={gatewayId || ALL_VALUE} onValueChange={(value) => onGatewayChange(value === ALL_VALUE ? '' : value)}>
              <SelectTrigger>
                <SelectValue placeholder="All gateways" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>All gateways</SelectItem>
                {gateways.map((gateway) => (
                  <SelectItem key={gateway.id} value={gateway.id}>{gateway.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FilterField>

          <FilterField label="IP address">
            <Input value={ipAddress} onChange={(event) => onIpAddressChange(event.target.value)} />
          </FilterField>

          <FilterField label="Country">
            <Select value={geoCountry || ALL_VALUE} onValueChange={(value) => onCountryChange(value === ALL_VALUE ? '' : value)}>
              <SelectTrigger>
                <SelectValue placeholder="All countries" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_VALUE}>All countries</SelectItem>
                {countries.map((country) => (
                  <SelectItem key={country} value={country}>{country}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FilterField>

          <FilterField label="From">
            <Input type="date" value={startDate} onChange={(event) => onStartDateChange(event.target.value)} />
          </FilterField>

          <FilterField label="To">
            <Input type="date" value={endDate} onChange={(event) => onEndDateChange(event.target.value)} />
          </FilterField>

          <FilterField label="Sort by">
            <Select value={sortBy} onValueChange={onSortByChange}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="createdAt">Date</SelectItem>
                <SelectItem value="action">Action</SelectItem>
              </SelectContent>
            </Select>
          </FilterField>

          <FilterField label="Order">
            <Select value={sortOrder} onValueChange={onSortOrderChange}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="desc">Newest first</SelectItem>
                <SelectItem value="asc">Oldest first</SelectItem>
              </SelectContent>
            </Select>
          </FilterField>
        </div>

        <SettingsButtonRow>
          <Button type="button" variant={flaggedOnly ? 'default' : 'outline'} size="sm" onClick={onFlaggedToggle}>
            <UserRound className="size-4" />
            Flagged Only
          </Button>
          <Button type="button" variant="ghost" size="sm" onClick={onClearFilters}>
            <X className="size-4" />
            Clear Filters
          </Button>
        </SettingsButtonRow>
      </CardContent>
    </Card>
  );
}

export function TenantAuditResults({
  emptyMessage,
  error,
  expandedRowId,
  loading,
  logs,
  onGeoIpClick,
  onToggleRow,
  onViewUserProfile,
}: {
  emptyMessage: string;
  error: string;
  expandedRowId: string | null;
  loading: boolean;
  logs: TenantAuditLogEntry[];
  onGeoIpClick?: (ip: string) => void;
  onToggleRow: (logId: string) => void;
  onViewUserProfile?: (userId: string) => void;
}) {
  if (error) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Audit log unavailable</AlertTitle>
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    );
  }

  if (loading) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="size-4 animate-spin" />
        Loading audit entries.
      </div>
    );
  }

  if (logs.length === 0) {
    return (
      <Card className="border-dashed">
        <CardContent className="py-10 text-center text-sm text-muted-foreground">
          {emptyMessage}
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-3">
      {logs.map((log) => (
        <TenantAuditEntryCard
          key={log.id}
          expanded={expandedRowId === log.id}
          log={log}
          onGeoIpClick={onGeoIpClick}
          onToggle={() => onToggleRow(log.id)}
          onViewUserProfile={onViewUserProfile}
        />
      ))}
    </div>
  );
}

export function TenantAuditPagination({
  page,
  rowsPerPage,
  total,
  totalPages,
  onNextPage,
  onPreviousPage,
  onRowsPerPageChange,
}: {
  onNextPage: () => void;
  onPreviousPage: () => void;
  onRowsPerPageChange: (value: number) => void;
  page: number;
  rowsPerPage: number;
  total: number;
  totalPages: number;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border/70 bg-background/60 px-4 py-3">
      <div className="text-sm text-muted-foreground">
        Page {page + 1} of {totalPages} · {total} total events
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Select value={String(rowsPerPage)} onValueChange={(value) => onRowsPerPageChange(Number(value))}>
          <SelectTrigger className="w-[120px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {[25, 50, 100].map((value) => (
              <SelectItem key={value} value={String(value)}>{value} / page</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button type="button" variant="outline" size="sm" disabled={page === 0} onClick={onPreviousPage}>
          Previous
        </Button>
        <Button type="button" variant="outline" size="sm" disabled={page + 1 >= totalPages} onClick={onNextPage}>
          Next
        </Button>
      </div>
    </div>
  );
}

export function TenantAuditSummary({
  filtersApplied,
  flaggedEntriesOnPage,
  total,
  usersOnPage,
}: {
  filtersApplied: number;
  flaggedEntriesOnPage: number;
  total: number;
  usersOnPage: number;
}) {
  return <SettingsSummaryGrid>
    <SettingsSummaryItem label="Results" value={String(total)} />
    <SettingsSummaryItem label="Filters applied" value={String(filtersApplied)} />
    <SettingsSummaryItem label="Flagged on page" value={String(flaggedEntriesOnPage)} />
    <SettingsSummaryItem label="Users on page" value={String(usersOnPage)} />
  </SettingsSummaryGrid>;
}

export { exportTenantAuditCsv };
