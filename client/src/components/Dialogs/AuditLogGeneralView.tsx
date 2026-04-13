import { useCallback, useEffect, useState } from 'react';
import {
  getAuditCountries,
  getAuditGateways,
  getAuditLogs,
  getSessionRecording,
  getTenantAuditCountries,
  getTenantAuditGateways,
  getTenantAuditLogs,
  type AuditAction,
  type AuditGateway,
  type AuditLogEntry,
  type AuditLogParams,
  type TenantAuditLogEntry,
  type TenantAuditLogParams,
} from '../../api/audit.api';
import { getRecording, type Recording } from '../../api/recordings.api';
import { useAuthStore } from '../../store/authStore';
import { useFeatureFlagsStore } from '../../store/featureFlagsStore';
import { useTenantStore } from '../../store/tenantStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import { hasAnyRole } from '../../utils/roles';
import AuditLogPersonalView from './AuditLogPersonalView';
import AuditLogTenantView from './AuditLogTenantView';

const AUTO_REFRESH_INTERVAL_MS = 10000;

interface AuditLogGeneralViewProps {
  open: boolean;
  onGeoIpClick?: (ip: string) => void;
  onViewUserProfile?: (userId: string) => void;
}

export default function AuditLogGeneralView({ open, onGeoIpClick, onViewUserProfile }: AuditLogGeneralViewProps) {
  const user = useAuthStore((state) => state.user);
  const permissionsLoaded = useAuthStore((state) => state.permissionsLoaded);
  const canViewAuditLog = useAuthStore((state) => state.permissions.canViewAuditLog);
  const autoRefreshPaused = useUiPreferencesStore((state) => state.auditLogAutoRefreshPaused);
  const ipGeolocationEnabled = useFeatureFlagsStore((state) => state.ipGeolocationEnabled);
  const users = useTenantStore((state) => state.users);
  const fetchUsers = useTenantStore((state) => state.fetchUsers);

  const auditLogAction = useUiPreferencesStore((state) => state.auditLogAction);
  const auditLogSearch = useUiPreferencesStore((state) => state.auditLogSearch);
  const auditLogTargetType = useUiPreferencesStore((state) => state.auditLogTargetType);
  const auditLogGatewayId = useUiPreferencesStore((state) => state.auditLogGatewayId);
  const auditLogSortBy = useUiPreferencesStore((state) => state.auditLogSortBy);
  const auditLogSortOrder = useUiPreferencesStore((state) => state.auditLogSortOrder);
  const tenantAuditLogAction = useUiPreferencesStore((state) => state.tenantAuditLogAction);
  const tenantAuditLogSearch = useUiPreferencesStore((state) => state.tenantAuditLogSearch);
  const tenantAuditLogTargetType = useUiPreferencesStore((state) => state.tenantAuditLogTargetType);
  const tenantAuditLogGatewayId = useUiPreferencesStore((state) => state.tenantAuditLogGatewayId);
  const tenantAuditLogUserId = useUiPreferencesStore((state) => state.tenantAuditLogUserId);
  const tenantAuditLogSortBy = useUiPreferencesStore((state) => state.tenantAuditLogSortBy);
  const tenantAuditLogSortOrder = useUiPreferencesStore((state) => state.tenantAuditLogSortOrder);
  const tenantAuditLogViewMode = useUiPreferencesStore((state) => state.tenantAuditLogViewMode);
  const setUiPref = useUiPreferencesStore((state) => state.set);

  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [tenantLogs, setTenantLogs] = useState<TenantAuditLogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [ipAddress, setIpAddress] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [expandedRowId, setExpandedRowId] = useState<string | null>(null);
  const [searchInput, setSearchInput] = useState('');
  const [gateways, setGateways] = useState<AuditGateway[]>([]);
  const [countries, setCountries] = useState<string[]>([]);
  const [geoCountry, setGeoCountry] = useState('');
  const [flaggedOnly, setFlaggedOnly] = useState(false);
  const [selectedRecording, setSelectedRecording] = useState<Recording | null>(null);
  const [recordingPlayerOpen, setRecordingPlayerOpen] = useState(false);
  const [loadingRecordingId, setLoadingRecordingId] = useState<string | null>(null);

  const hasTenant = Boolean(user?.tenantId);
  const hasTenantAuditAccess = hasTenant && (permissionsLoaded ? canViewAuditLog : hasAnyRole(user?.tenantRole, 'ADMIN', 'OWNER', 'AUDITOR'));

  const action = hasTenantAuditAccess ? tenantAuditLogAction : auditLogAction;
  const search = hasTenantAuditAccess ? tenantAuditLogSearch : auditLogSearch;
  const targetType = hasTenantAuditAccess ? tenantAuditLogTargetType : auditLogTargetType;
  const gatewayId = hasTenantAuditAccess ? tenantAuditLogGatewayId : auditLogGatewayId;
  const sortBy = hasTenantAuditAccess ? tenantAuditLogSortBy : auditLogSortBy;
  const sortOrder = hasTenantAuditAccess ? tenantAuditLogSortOrder : auditLogSortOrder;
  const effectiveViewMode = ipGeolocationEnabled && tenantAuditLogViewMode === 'map' ? 'map' : 'table';

  useEffect(() => {
    setSearchInput(search);
  }, [hasTenantAuditAccess, search]);

  useEffect(() => {
    if (!hasTenantAuditAccess || users.length > 0) {
      return;
    }
    void fetchUsers();
  }, [fetchUsers, hasTenantAuditAccess, users.length]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setUiPref(hasTenantAuditAccess ? 'tenantAuditLogSearch' : 'auditLogSearch', searchInput);
      setPage(0);
    }, 300);
    return () => window.clearTimeout(timer);
  }, [hasTenantAuditAccess, searchInput, setUiPref]);

  const fetchMetadata = useCallback(async () => {
    try {
      const [nextGateways, nextCountries] = await Promise.all([
        hasTenantAuditAccess ? getTenantAuditGateways() : getAuditGateways(),
        hasTenantAuditAccess ? getTenantAuditCountries() : getAuditCountries(),
      ]);
      setGateways(nextGateways);
      setCountries(nextCountries);
    } catch {
      setGateways([]);
      setCountries([]);
    }
  }, [hasTenantAuditAccess]);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    setError('');

    try {
      if (hasTenantAuditAccess) {
        const params: TenantAuditLogParams = {
          page: page + 1,
          limit: rowsPerPage,
          sortBy: sortBy as 'createdAt' | 'action',
          sortOrder: sortOrder as 'asc' | 'desc',
        };
        if (action) params.action = action as AuditAction;
        if (search) params.search = search;
        if (targetType) params.targetType = targetType;
        if (gatewayId) params.gatewayId = gatewayId;
        if (tenantAuditLogUserId) params.userId = tenantAuditLogUserId;
        if (ipAddress) params.ipAddress = ipAddress;
        if (geoCountry) params.geoCountry = geoCountry;
        if (startDate) params.startDate = startDate;
        if (endDate) params.endDate = endDate;
        if (flaggedOnly) params.flaggedOnly = true;

        const result = await getTenantAuditLogs(params);
        setTenantLogs(result.data);
        setTotal(result.total);
        return;
      }

      const params: AuditLogParams = {
        page: page + 1,
        limit: rowsPerPage,
        sortBy: sortBy as 'createdAt' | 'action',
        sortOrder: sortOrder as 'asc' | 'desc',
      };
      if (action) params.action = action as AuditAction;
      if (search) params.search = search;
      if (targetType) params.targetType = targetType;
      if (gatewayId) params.gatewayId = gatewayId;
      if (ipAddress) params.ipAddress = ipAddress;
      if (geoCountry) params.geoCountry = geoCountry;
      if (startDate) params.startDate = startDate;
      if (endDate) params.endDate = endDate;
      if (flaggedOnly) params.flaggedOnly = true;

      const result = await getAuditLogs(params);
      setLogs(result.data);
      setTotal(result.total);
    } catch {
      setError(hasTenantAuditAccess ? 'Failed to load the organization audit log.' : 'Failed to load audit logs');
    } finally {
      setLoading(false);
    }
  }, [
    action,
    endDate,
    flaggedOnly,
    gatewayId,
    geoCountry,
    hasTenantAuditAccess,
    ipAddress,
    page,
    rowsPerPage,
    search,
    sortBy,
    sortOrder,
    startDate,
    targetType,
    tenantAuditLogUserId,
  ]);

  useEffect(() => {
    if (!open) {
      return;
    }
    void fetchLogs();
    void fetchMetadata();
  }, [fetchLogs, fetchMetadata, open]);

  useEffect(() => {
    if (!open || autoRefreshPaused || page !== 0) {
      return undefined;
    }
    const intervalId = window.setInterval(() => {
      void fetchLogs();
    }, AUTO_REFRESH_INTERVAL_MS);
    return () => window.clearInterval(intervalId);
  }, [autoRefreshPaused, fetchLogs, open, page]);

  const handleSort = (field: 'createdAt' | 'action') => {
    if (auditLogSortBy === field) {
      setUiPref('auditLogSortOrder', auditLogSortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setUiPref('auditLogSortBy', field);
      setUiPref('auditLogSortOrder', field === 'createdAt' ? 'desc' : 'asc');
    }
    setPage(0);
  };

  const handleTenantSort = (field: 'createdAt' | 'action') => {
    if (tenantAuditLogSortBy === field) {
      setUiPref('tenantAuditLogSortOrder', tenantAuditLogSortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setUiPref('tenantAuditLogSortBy', field);
      setUiPref('tenantAuditLogSortOrder', field === 'createdAt' ? 'desc' : 'asc');
    }
    setPage(0);
  };

  const handleViewRecording = async (log: AuditLogEntry) => {
    const sessionId = (log.details as Record<string, unknown>)?.sessionId as string | undefined;
    const recordingId = (log.details as Record<string, unknown>)?.recordingId as string | undefined;
    if (!sessionId && !recordingId) {
      return;
    }

    setLoadingRecordingId(log.id);
    try {
      const recording = recordingId
        ? await getRecording(recordingId)
        : sessionId
          ? await getSessionRecording(sessionId)
          : null;
      if (!recording) {
        return;
      }
      setSelectedRecording(recording);
      setRecordingPlayerOpen(true);
    } catch {
      // Best-effort enhancement from the audit surface.
    } finally {
      setLoadingRecordingId(null);
    }
  };

  const clearFilters = () => {
    setSearchInput('');
    if (hasTenantAuditAccess) {
      setUiPref('tenantAuditLogSearch', '');
      setUiPref('tenantAuditLogAction', '');
      setUiPref('tenantAuditLogTargetType', '');
      setUiPref('tenantAuditLogGatewayId', '');
      setUiPref('tenantAuditLogUserId', '');
      setUiPref('tenantAuditLogSortBy', 'createdAt');
      setUiPref('tenantAuditLogSortOrder', 'desc');
    } else {
      setUiPref('auditLogSearch', '');
      setUiPref('auditLogAction', '');
      setUiPref('auditLogTargetType', '');
      setUiPref('auditLogGatewayId', '');
      setUiPref('auditLogSortBy', 'createdAt');
      setUiPref('auditLogSortOrder', 'desc');
    }
    setStartDate('');
    setEndDate('');
    setIpAddress('');
    setGeoCountry('');
    setFlaggedOnly(false);
    setPage(0);
  };

  if (hasTenantAuditAccess) {
    return (
      <AuditLogTenantView
        action={action}
        countries={countries}
        effectiveViewMode={effectiveViewMode}
        endDate={endDate}
        error={error}
        expandedRowId={expandedRowId}
        flaggedOnly={flaggedOnly}
        gatewayId={gatewayId}
        gateways={gateways}
        geoCountry={geoCountry}
        ipAddress={ipAddress}
        ipGeolocationEnabled={ipGeolocationEnabled}
        loading={loading}
        loadingRecordingId={loadingRecordingId}
        onActionChange={(value) => { setUiPref('tenantAuditLogAction', value); setPage(0); }}
        onClearFilters={clearFilters}
        onCloseRecordingPlayer={() => { setRecordingPlayerOpen(false); setSelectedRecording(null); }}
        onCountryChange={(value) => { setGeoCountry(value); setPage(0); }}
        onEndDateChange={(value) => { setEndDate(value); setPage(0); }}
        onFlaggedToggle={() => { setFlaggedOnly((current) => !current); setPage(0); }}
        onGatewayChange={(value) => { setUiPref('tenantAuditLogGatewayId', value); setPage(0); }}
        onGeoIpClick={onGeoIpClick}
        onHandleSort={handleTenantSort}
        onIpAddressChange={(value) => { setIpAddress(value); setPage(0); }}
        onNextPage={() => setPage((current) => current + 1)}
        onPreviousPage={() => setPage((current) => current - 1)}
        onRowsPerPageChange={(value) => { setRowsPerPage(value); setPage(0); }}
        search={search}
        onSearchChange={setSearchInput}
        onSortByChange={(value) => { setUiPref('tenantAuditLogSortBy', value); setPage(0); }}
        onSortOrderChange={(value) => { setUiPref('tenantAuditLogSortOrder', value); setPage(0); }}
        onStartDateChange={(value) => { setStartDate(value); setPage(0); }}
        onTargetTypeChange={(value) => { setUiPref('tenantAuditLogTargetType', value); setPage(0); }}
        onToggleRow={(logId) => { setExpandedRowId((current) => current === logId ? null : logId); }}
        onUserChange={(value) => { setUiPref('tenantAuditLogUserId', value); setPage(0); }}
        onViewModeChange={(mode) => setUiPref('tenantAuditLogViewMode', mode)}
        onViewRecording={handleViewRecording}
        onViewUserProfile={onViewUserProfile}
        page={page}
        recordingPlayerOpen={recordingPlayerOpen}
        rowsPerPage={rowsPerPage}
        searchInput={searchInput}
        selectedRecording={selectedRecording}
        sortBy={sortBy}
        sortOrder={sortOrder}
        startDate={startDate}
        targetType={targetType}
        tenantLogs={tenantLogs}
        total={total}
        userId={tenantAuditLogUserId}
        users={users}
      />
    );
  }

  return (
    <AuditLogPersonalView
      action={action}
      auditLogSortBy={auditLogSortBy}
      auditLogSortOrder={auditLogSortOrder}
      countries={countries}
      endDate={endDate}
      error={error}
      expandedRowId={expandedRowId}
      flaggedOnly={flaggedOnly}
      gatewayId={gatewayId}
      gateways={gateways}
      geoCountry={geoCountry}
      ipAddress={ipAddress}
      loading={loading}
      loadingRecordingId={loadingRecordingId}
      logs={logs}
      onActionChange={(value) => { setUiPref('auditLogAction', value); setPage(0); }}
      onCloseRecordingPlayer={() => { setRecordingPlayerOpen(false); setSelectedRecording(null); }}
      onCountryChange={(value) => { setGeoCountry(value); setPage(0); }}
      onEndDateChange={(value) => { setEndDate(value); setPage(0); }}
      onFlaggedToggle={() => { setFlaggedOnly((current) => !current); setPage(0); }}
      onGatewayChange={(value) => { setUiPref('auditLogGatewayId', value); setPage(0); }}
      onGeoIpClick={onGeoIpClick}
      onHandleSort={handleSort}
      onIpAddressChange={(value) => { setIpAddress(value); setPage(0); }}
      onNextPage={() => setPage((current) => current + 1)}
      onPreviousPage={() => setPage((current) => current - 1)}
      onRowsPerPageChange={(value) => { setRowsPerPage(value); setPage(0); }}
      onSearchChange={setSearchInput}
      onStartDateChange={(value) => { setStartDate(value); setPage(0); }}
      onTargetTypeChange={(value) => { setUiPref('auditLogTargetType', value); setPage(0); }}
      onToggleRow={(logId) => { setExpandedRowId((current) => current === logId ? null : logId); }}
      onViewRecording={handleViewRecording}
      page={page}
      recordingPlayerOpen={recordingPlayerOpen}
      rowsPerPage={rowsPerPage}
      searchInput={searchInput}
      selectedRecording={selectedRecording}
      startDate={startDate}
      targetType={targetType}
      total={total}
    />
  );
}
