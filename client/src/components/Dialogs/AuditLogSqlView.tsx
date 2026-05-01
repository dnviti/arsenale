import { useCallback, useEffect, useState } from 'react';
import { connectSSE } from '../../api/sse';
import {
  getDbAuditConnections,
  getDbAuditLogs,
  getDbAuditUsers,
  type DbAuditConnection,
  type DbAuditLogEntry,
  type DbAuditLogParams,
  type DbAuditUser,
  type DbQueryType,
} from '../../api/dbAudit.api';
import type { DbAuditStreamSnapshot } from '../../api/live.api';
import { useAuthStore } from '../../store/authStore';
import { useFeatureFlagsStore } from '../../store/featureFlagsStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import AuditLogSqlActivityView from './AuditLogSqlActivityView';

interface AuditLogSqlViewProps {
  open: boolean;
}

export default function AuditLogSqlView({ open }: AuditLogSqlViewProps) {
  const accessToken = useAuthStore((state) => state.accessToken);
  const user = useAuthStore((state) => state.user);
  const autoRefreshPaused = useUiPreferencesStore((state) => state.auditLogAutoRefreshPaused);
  const databaseProxyEnabled = useFeatureFlagsStore((state) => state.databaseProxyEnabled);

  const [dbLogs, setDbLogs] = useState<DbAuditLogEntry[]>([]);
  const [dbTotal, setDbTotal] = useState(0);
  const [dbPage, setDbPage] = useState(0);
  const [dbRowsPerPage, setDbRowsPerPage] = useState(25);
  const [dbLoading, setDbLoading] = useState(false);
  const [dbError, setDbError] = useState('');
  const [dbSearch, setDbSearch] = useState('');
  const [dbQueryType, setDbQueryType] = useState('');
  const [dbConnectionId, setDbConnectionId] = useState('');
  const [dbUserId, setDbUserId] = useState('');
  const [dbBlocked, setDbBlocked] = useState('');
  const [dbStartDate, setDbStartDate] = useState('');
  const [dbEndDate, setDbEndDate] = useState('');
  const [dbExpandedRowId, setDbExpandedRowId] = useState<string | null>(null);
  const [dbConnections, setDbConnections] = useState<DbAuditConnection[]>([]);
  const [dbUsers, setDbUsers] = useState<DbAuditUser[]>([]);

  const hasTenant = Boolean(user?.tenantId);
  const showSqlAudit = hasTenant && databaseProxyEnabled;

  const fetchDbLogs = useCallback(async () => {
    setDbLoading(true);
    setDbError('');
    try {
      const params: DbAuditLogParams = {
        page: dbPage + 1,
        limit: dbRowsPerPage,
        sortBy: 'createdAt',
        sortOrder: 'desc',
      };
      if (dbSearch) params.search = dbSearch;
      if (dbQueryType) params.queryType = dbQueryType as DbQueryType;
      if (dbConnectionId) params.connectionId = dbConnectionId;
      if (dbUserId) params.userId = dbUserId;
      if (dbBlocked === 'true') params.blocked = true;
      if (dbBlocked === 'false') params.blocked = false;
      if (dbStartDate) params.startDate = dbStartDate;
      if (dbEndDate) params.endDate = dbEndDate;

      const result = await getDbAuditLogs(params);
      setDbLogs(result.data);
      setDbTotal(result.total);
    } catch {
      setDbError('Failed to load SQL audit logs');
    } finally {
      setDbLoading(false);
    }
  }, [
    dbBlocked,
    dbConnectionId,
    dbEndDate,
    dbPage,
    dbQueryType,
    dbRowsPerPage,
    dbSearch,
    dbStartDate,
    dbUserId,
  ]);

  useEffect(() => {
    if (!open || !showSqlAudit) {
      return;
    }
    void fetchDbLogs();
    void getDbAuditConnections().then(setDbConnections).catch(() => setDbConnections([]));
    void getDbAuditUsers().then(setDbUsers).catch(() => setDbUsers([]));
  }, [fetchDbLogs, open, showSqlAudit]);

  useEffect(() => {
    if (!open || !showSqlAudit || !accessToken || autoRefreshPaused || dbPage !== 0) {
      return undefined;
    }

    const params = new URLSearchParams({
      page: '1',
      limit: String(dbRowsPerPage),
      sortBy: 'createdAt',
      sortOrder: 'desc',
    });
    if (dbSearch) params.set('search', dbSearch);
    if (dbQueryType) params.set('queryType', dbQueryType);
    if (dbConnectionId) params.set('connectionId', dbConnectionId);
    if (dbUserId) params.set('userId', dbUserId);
    if (dbBlocked === 'true' || dbBlocked === 'false') params.set('blocked', dbBlocked);
    if (dbStartDate) params.set('startDate', dbStartDate);
    if (dbEndDate) params.set('endDate', dbEndDate);

    return connectSSE({
      url: `/api/db-audit/logs/stream?${params.toString()}`,
      accessToken,
      onEvent: ({ event, data }) => {
        if (event !== 'snapshot') {
          return;
        }
        const snapshot = data as DbAuditStreamSnapshot;
        setDbLogs(snapshot.data);
        setDbTotal(snapshot.total);
        setDbLoading(false);
        setDbError('');
      },
    });
  }, [
    accessToken,
    autoRefreshPaused,
    dbBlocked,
    dbConnectionId,
    dbEndDate,
    dbPage,
    dbQueryType,
    dbRowsPerPage,
    dbSearch,
    dbStartDate,
    dbUserId,
    open,
    showSqlAudit,
  ]);

  const hasDbActiveFilters = Boolean(
    dbSearch || dbQueryType || dbConnectionId || dbUserId || dbBlocked || dbStartDate || dbEndDate,
  );
  const dbTotalPages = Math.ceil(dbTotal / dbRowsPerPage);

  if (!showSqlAudit) {
    return null;
  }

  return (
    <AuditLogSqlActivityView
      connections={dbConnections}
      endDate={dbEndDate}
      error={dbError}
      expandedRowId={dbExpandedRowId}
      hasActiveFilters={hasDbActiveFilters}
      loading={dbLoading}
      logs={dbLogs}
      onBlockedChange={(value) => {
        setDbBlocked(value);
        setDbPage(0);
      }}
      onConnectionChange={(value) => {
        setDbConnectionId(value);
        setDbPage(0);
      }}
      onEndDateChange={(value) => {
        setDbEndDate(value);
        setDbPage(0);
      }}
      onNextPage={() => setDbPage((current) => current + 1)}
      onPreviousPage={() => setDbPage((current) => current - 1)}
      onQueryTypeChange={(value) => {
        setDbQueryType(value as DbQueryType | '');
        setDbPage(0);
      }}
      onRowsPerPageChange={(value) => {
        setDbRowsPerPage(value);
        setDbPage(0);
      }}
      onSearchChange={(value) => {
        setDbSearch(value);
        setDbPage(0);
      }}
      onStartDateChange={(value) => {
        setDbStartDate(value);
        setDbPage(0);
      }}
      onToggleRow={(rowId) => {
        setDbExpandedRowId((current) => current === rowId ? null : rowId);
      }}
      onUserChange={(value) => {
        setDbUserId(value);
        setDbPage(0);
      }}
      page={dbPage}
      rowsPerPage={dbRowsPerPage}
      search={dbSearch}
      selectedBlocked={dbBlocked}
      selectedConnectionId={dbConnectionId}
      selectedQueryType={dbQueryType}
      selectedUserId={dbUserId}
      startDate={dbStartDate}
      total={dbTotal}
      totalPages={dbTotalPages}
      users={dbUsers}
    />
  );
}
