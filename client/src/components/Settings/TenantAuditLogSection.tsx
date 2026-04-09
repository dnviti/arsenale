import { lazy, Suspense, useCallback, useEffect, useState } from 'react';
import { Card, CardContent } from '@/components/ui/card';
import {
  getTenantAuditLogs,
  getTenantAuditGateways,
  getTenantAuditCountries,
  type AuditAction,
  type AuditGateway,
  type TenantAuditLogEntry,
  type TenantAuditLogParams,
} from '../../api/audit.api';
import { useFeatureFlagsStore } from '../../store/featureFlagsStore';
import { useTenantStore } from '../../store/tenantStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import { SettingsPanel } from './settings-ui';
import {
  TenantAuditFiltersCard,
  TenantAuditPagination,
  TenantAuditResults,
  TenantAuditSummary,
  TenantAuditToolbar,
  exportTenantAuditCsv,
} from './tenantAuditLogUi';
import { countActiveFilters } from './tenantAuditLogUtils';

const AuditGeoMap = lazy(() => import('../Audit/AuditGeoMap'));

interface TenantAuditLogSectionProps {
  onViewUserProfile?: (userId: string) => void;
  onGeoIpClick?: (ip: string) => void;
}

export default function TenantAuditLogSection({
  onViewUserProfile,
  onGeoIpClick,
}: TenantAuditLogSectionProps) {
  const tenantAuditLogAction = useUiPreferencesStore((state) => state.tenantAuditLogAction);
  const tenantAuditLogSearch = useUiPreferencesStore((state) => state.tenantAuditLogSearch);
  const tenantAuditLogTargetType = useUiPreferencesStore((state) => state.tenantAuditLogTargetType);
  const tenantAuditLogGatewayId = useUiPreferencesStore((state) => state.tenantAuditLogGatewayId);
  const tenantAuditLogUserId = useUiPreferencesStore((state) => state.tenantAuditLogUserId);
  const tenantAuditLogSortBy = useUiPreferencesStore((state) => state.tenantAuditLogSortBy);
  const tenantAuditLogSortOrder = useUiPreferencesStore((state) => state.tenantAuditLogSortOrder);
  const tenantAuditLogViewMode = useUiPreferencesStore((state) => state.tenantAuditLogViewMode);
  const setUiPref = useUiPreferencesStore((state) => state.set);
  const ipGeolocationEnabled = useFeatureFlagsStore((state) => state.ipGeolocationEnabled);
  const users = useTenantStore((state) => state.users);
  const fetchUsers = useTenantStore((state) => state.fetchUsers);

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
  const [searchInput, setSearchInput] = useState(tenantAuditLogSearch);
  const [gateways, setGateways] = useState<AuditGateway[]>([]);
  const [countries, setCountries] = useState<string[]>([]);
  const [geoCountry, setGeoCountry] = useState('');
  const [flaggedOnly, setFlaggedOnly] = useState(false);

  useEffect(() => {
    if (users.length === 0) {
      fetchUsers();
    }
  }, [fetchUsers, users.length]);

  useEffect(() => {
    const timeout = setTimeout(() => {
      setUiPref('tenantAuditLogSearch', searchInput);
      setPage(0);
    }, 300);
    return () => clearTimeout(timeout);
  }, [searchInput, setUiPref]);

  useEffect(() => {
    getTenantAuditGateways().then(setGateways).catch(() => {});
    getTenantAuditCountries().then(setCountries).catch(() => {});
  }, []);

  useEffect(() => {
    if (!ipGeolocationEnabled && tenantAuditLogViewMode === 'map') {
      setUiPref('tenantAuditLogViewMode', 'table');
    }
  }, [ipGeolocationEnabled, setUiPref, tenantAuditLogViewMode]);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    setError('');

    try {
      const params: TenantAuditLogParams = {
        page: page + 1,
        limit: rowsPerPage,
        sortBy: tenantAuditLogSortBy as 'createdAt' | 'action',
        sortOrder: tenantAuditLogSortOrder as 'asc' | 'desc',
      };

      if (tenantAuditLogAction) params.action = tenantAuditLogAction as AuditAction;
      if (tenantAuditLogSearch) params.search = tenantAuditLogSearch;
      if (tenantAuditLogTargetType) params.targetType = tenantAuditLogTargetType;
      if (tenantAuditLogGatewayId) params.gatewayId = tenantAuditLogGatewayId;
      if (tenantAuditLogUserId) params.userId = tenantAuditLogUserId;
      if (ipAddress) params.ipAddress = ipAddress;
      if (geoCountry) params.geoCountry = geoCountry;
      if (startDate) params.startDate = startDate;
      if (endDate) params.endDate = endDate;
      if (flaggedOnly) params.flaggedOnly = true;

      const response = await getTenantAuditLogs(params);
      setLogs(response.data);
      setTotal(response.total);
    } catch {
      setError('Failed to load the organization audit log.');
    } finally {
      setLoading(false);
    }
  }, [
    endDate,
    flaggedOnly,
    geoCountry,
    ipAddress,
    page,
    rowsPerPage,
    startDate,
    tenantAuditLogAction,
    tenantAuditLogGatewayId,
    tenantAuditLogSearch,
    tenantAuditLogSortBy,
    tenantAuditLogSortOrder,
    tenantAuditLogTargetType,
    tenantAuditLogUserId,
  ]);

  useEffect(() => {
    void fetchLogs();
  }, [fetchLogs]);

  const effectiveViewMode = ipGeolocationEnabled && tenantAuditLogViewMode === 'map' ? 'map' : 'table';
  const flaggedEntriesOnPage = logs.filter((log) => log.flags?.length).length;
  const usersOnPage = new Set(logs.map((log) => log.userId).filter(Boolean)).size;
  const filtersApplied = countActiveFilters([
    tenantAuditLogAction,
    tenantAuditLogSearch,
    tenantAuditLogTargetType,
    tenantAuditLogGatewayId,
    tenantAuditLogUserId,
    ipAddress,
    geoCountry,
    startDate,
    endDate,
    flaggedOnly,
  ]);
  const totalPages = Math.max(1, Math.ceil(total / rowsPerPage));

  const clearFilters = () => {
    setSearchInput('');
    setUiPref('tenantAuditLogSearch', '');
    setUiPref('tenantAuditLogAction', '');
    setUiPref('tenantAuditLogTargetType', '');
    setUiPref('tenantAuditLogGatewayId', '');
    setUiPref('tenantAuditLogUserId', '');
    setUiPref('tenantAuditLogSortBy', 'createdAt');
    setUiPref('tenantAuditLogSortOrder', 'desc');
    setStartDate('');
    setEndDate('');
    setIpAddress('');
    setGeoCountry('');
    setFlaggedOnly(false);
    setPage(0);
  };

  return (
    <SettingsPanel
      title="Organization Audit Log"
      description="Review tenant-wide activity with grouped filters, export, and map-based geo inspection."
      heading={(
        <TenantAuditToolbar
          effectiveViewMode={effectiveViewMode}
          hasLogs={logs.length > 0}
          ipGeolocationEnabled={ipGeolocationEnabled}
          onExport={() => exportTenantAuditCsv(logs)}
          onViewModeChange={(mode) => setUiPref('tenantAuditLogViewMode', mode)}
        />
      )}
      contentClassName="space-y-6"
    >
      <TenantAuditSummary
        filtersApplied={filtersApplied}
        flaggedEntriesOnPage={flaggedEntriesOnPage}
        total={total}
        usersOnPage={usersOnPage}
      />

      <TenantAuditFiltersCard
        action={tenantAuditLogAction}
        countries={countries}
        endDate={endDate}
        flaggedOnly={flaggedOnly}
        gatewayId={tenantAuditLogGatewayId}
        gateways={gateways}
        geoCountry={geoCountry}
        ipAddress={ipAddress}
        onActionChange={(value) => {
          setUiPref('tenantAuditLogAction', value);
          setPage(0);
        }}
        onClearFilters={clearFilters}
        onCountryChange={(value) => {
          setGeoCountry(value);
          setPage(0);
        }}
        onEndDateChange={(value) => {
          setEndDate(value);
          setPage(0);
        }}
        onFlaggedToggle={() => {
          setFlaggedOnly((current) => !current);
          setPage(0);
        }}
        onGatewayChange={(value) => {
          setUiPref('tenantAuditLogGatewayId', value);
          setPage(0);
        }}
        onIpAddressChange={(value) => {
          setIpAddress(value);
          setPage(0);
        }}
        onSearchChange={setSearchInput}
        onSortByChange={(value) => {
          setUiPref('tenantAuditLogSortBy', value);
          setPage(0);
        }}
        onSortOrderChange={(value) => {
          setUiPref('tenantAuditLogSortOrder', value);
          setPage(0);
        }}
        onStartDateChange={(value) => {
          setStartDate(value);
          setPage(0);
        }}
        onTargetTypeChange={(value) => {
          setUiPref('tenantAuditLogTargetType', value);
          setPage(0);
        }}
        onUserChange={(value) => {
          setUiPref('tenantAuditLogUserId', value);
          setPage(0);
        }}
        searchInput={searchInput}
        sortBy={tenantAuditLogSortBy}
        sortOrder={tenantAuditLogSortOrder}
        startDate={startDate}
        targetType={tenantAuditLogTargetType}
        userId={tenantAuditLogUserId}
        users={users}
      />

      {effectiveViewMode === 'map' ? (
        <Suspense
          fallback={(
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              Loading geo map.
            </div>
          )}
        >
          <Card>
            <CardContent className="p-0">
              <AuditGeoMap
                onSelectCountry={(country) => {
                  setGeoCountry(country);
                  setUiPref('tenantAuditLogViewMode', 'table');
                  setPage(0);
                }}
              />
            </CardContent>
          </Card>
        </Suspense>
      ) : (
        <>
          <TenantAuditResults
            emptyMessage={filtersApplied > 0 ? 'No activity matches the current filters.' : 'No activity recorded yet.'}
            error={error}
            expandedRowId={expandedRowId}
            loading={loading}
            logs={logs}
            onGeoIpClick={onGeoIpClick}
            onToggleRow={(logId) => setExpandedRowId((current) => current === logId ? null : logId)}
            onViewUserProfile={onViewUserProfile}
          />
          {logs.length > 0 && !loading && !error ? (
            <TenantAuditPagination
              page={page}
              rowsPerPage={rowsPerPage}
              total={total}
              totalPages={totalPages}
              onNextPage={() => setPage((current) => current + 1)}
              onPreviousPage={() => setPage((current) => current - 1)}
              onRowsPerPageChange={(value) => {
                setRowsPerPage(value);
                setPage(0);
              }}
            />
          ) : null}
        </>
      )}
    </SettingsPanel>
  );
}
