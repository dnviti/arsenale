import { useState, useEffect, useCallback, Fragment } from "react";
import {
  Dialog,
  DialogContent,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import {
  X,
  Search,
  ChevronDown,
  ChevronUp,
  Pause,
  Play,
  AlertTriangle,
  Database,
  List,
  Eye,
  Loader2,
} from "lucide-react";
import { cn } from "@/lib/utils";
import {
  getAuditLogs,
  getAuditGateways,
  getAuditCountries,
  AuditLogEntry,
  AuditAction,
  AuditLogParams,
  AuditGateway,
} from "../../api/audit.api";
import {
  getDbAuditLogs,
  getDbAuditConnections,
  getDbAuditUsers,
  DbAuditLogEntry,
  DbAuditLogParams,
  DbAuditConnection,
  DbAuditUser,
  DbQueryType,
} from "../../api/dbAudit.api";
import type {
  AuditStreamSnapshot,
  DbAuditStreamSnapshot,
} from "../../api/live.api";
import { connectSSE } from "../../api/sse";
import { useUiPreferencesStore } from "../../store/uiPreferencesStore";
import { useAuthStore } from "../../store/authStore";
import { useFeatureFlagsStore } from "../../store/featureFlagsStore";
import {
  ACTION_LABELS,
  getActionColor,
  formatDetails,
  ALL_ACTIONS,
  TARGET_TYPES,
} from "../Audit/auditConstants";
import IpGeoCell from "../Audit/IpGeoCell";
import QueryVisualizer from "../DatabaseClient/QueryVisualizer";
import RecordingPlayerDialog from "../Recording/RecordingPlayerDialog";
import { getRecording } from "../../api/recordings.api";
import type { Recording } from "../../api/recordings.api";
import { getSessionRecording } from "../../api/audit.api";

interface AuditLogDialogProps {
  open: boolean;
  onClose: () => void;
  onGeoIpClick?: (ip: string) => void;
}

const QUERY_TYPE_LABELS: Record<DbQueryType, string> = {
  SELECT: "SELECT",
  INSERT: "INSERT",
  UPDATE: "UPDATE",
  DELETE: "DELETE",
  DDL: "DDL",
  OTHER: "Other",
};

const QUERY_TYPE_COLORS: Record<DbQueryType, string> = {
  SELECT: "bg-blue-600/15 text-blue-400 border-blue-600/30",
  INSERT: "bg-emerald-600/15 text-emerald-400 border-emerald-600/30",
  UPDATE: "bg-primary/15 text-primary border-primary/30",
  DELETE: "bg-destructive/15 text-destructive border-destructive/30",
  DDL: "bg-yellow-600/15 text-yellow-500 border-yellow-600/30",
  OTHER: "",
};

const ACTION_COLOR_MAP: Record<string, string> = {
  default: "",
  primary: "bg-primary/15 text-primary border-primary/30",
  secondary: "bg-muted text-muted-foreground",
  error: "bg-destructive/15 text-destructive border-destructive/30",
  warning: "bg-yellow-600/15 text-yellow-500 border-yellow-600/30",
  success: "bg-emerald-600/15 text-emerald-400 border-emerald-600/30",
  info: "bg-blue-600/15 text-blue-400 border-blue-600/30",
};

const ALL_QUERY_TYPES: DbQueryType[] = [
  "SELECT",
  "INSERT",
  "UPDATE",
  "DELETE",
  "DDL",
  "OTHER",
];

export default function AuditLogDialog({
  open,
  onClose,
  onGeoIpClick,
}: AuditLogDialogProps) {
  const user = useAuthStore((s) => s.user);
  const accessToken = useAuthStore((s) => s.accessToken);
  const databaseProxyEnabled = useFeatureFlagsStore(
    (s) => s.databaseProxyEnabled,
  );
  const hasTenant = Boolean(user?.tenantId);
  const auditLogAction = useUiPreferencesStore((s) => s.auditLogAction);
  const auditLogSearch = useUiPreferencesStore((s) => s.auditLogSearch);
  const auditLogTargetType = useUiPreferencesStore((s) => s.auditLogTargetType);
  const auditLogGatewayId = useUiPreferencesStore((s) => s.auditLogGatewayId);
  const auditLogSortBy = useUiPreferencesStore((s) => s.auditLogSortBy);
  const auditLogSortOrder = useUiPreferencesStore((s) => s.auditLogSortOrder);
  const autoRefreshPaused = useUiPreferencesStore(
    (s) => s.auditLogAutoRefreshPaused,
  );
  const auditLogTab = useUiPreferencesStore((s) => s.auditLogDialogTab);
  const setUiPref = useUiPreferencesStore((s) => s.set);

  // ---- General Audit Log state ----
  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [startDate, setStartDate] = useState("");
  const [endDate, setEndDate] = useState("");
  const [ipAddress, setIpAddress] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [expandedRowId, setExpandedRowId] = useState<string | null>(null);
  const [searchInput, setSearchInput] = useState(auditLogSearch);
  const [gateways, setGateways] = useState<AuditGateway[]>([]);
  const [countries, setCountries] = useState<string[]>([]);
  const [geoCountry, setGeoCountry] = useState("");
  const [flaggedOnly, setFlaggedOnly] = useState(false);

  // ---- SQL Audit Log state ----
  const [dbLogs, setDbLogs] = useState<DbAuditLogEntry[]>([]);
  const [dbTotal, setDbTotal] = useState(0);
  const [dbPage, setDbPage] = useState(0);
  const [dbRowsPerPage, setDbRowsPerPage] = useState(25);
  const [dbLoading, setDbLoading] = useState(false);
  const [dbError, setDbError] = useState("");
  const [dbSearch, setDbSearch] = useState("");
  const [dbQueryType, setDbQueryType] = useState("");
  const [dbConnectionId, setDbConnectionId] = useState("");
  const [dbUserId, setDbUserId] = useState("");
  const [dbBlocked, setDbBlocked] = useState("");
  const [dbStartDate, setDbStartDate] = useState("");
  const [dbEndDate, setDbEndDate] = useState("");
  const [dbExpandedRowId, setDbExpandedRowId] = useState<string | null>(null);
  const [dbConnections, setDbConnections] = useState<DbAuditConnection[]>([]);
  const [dbUsers, setDbUsers] = useState<DbAuditUser[]>([]);

  // ---- Query Visualizer state ----
  const [visualizerEntry, setVisualizerEntry] =
    useState<DbAuditLogEntry | null>(null);

  // ---- Recording Player state ----
  const [selectedRecording, setSelectedRecording] = useState<Recording | null>(
    null,
  );
  const [recordingPlayerOpen, setRecordingPlayerOpen] = useState(false);
  const [loadingRecordingId, setLoadingRecordingId] = useState<string | null>(
    null,
  );

  const sqlAuditVisible = hasTenant && databaseProxyEnabled;
  const activeTab =
    sqlAuditVisible && auditLogTab === "sql" ? "sql" : "general";

  useEffect(() => {
    if (open && auditLogTab === "sql" && !sqlAuditVisible) {
      setUiPref("auditLogDialogTab", "general");
    }
  }, [open, auditLogTab, setUiPref, sqlAuditVisible]);

  // Debounce search input -> store
  useEffect(() => {
    const timer = setTimeout(() => {
      setUiPref("auditLogSearch", searchInput);
      setPage(0);
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput, setUiPref]);

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const params: AuditLogParams = {
        page: page + 1,
        limit: rowsPerPage,
        sortBy: auditLogSortBy as "createdAt" | "action",
        sortOrder: auditLogSortOrder as "asc" | "desc",
      };
      if (auditLogAction) params.action = auditLogAction as AuditAction;
      if (auditLogSearch) params.search = auditLogSearch;
      if (auditLogTargetType) params.targetType = auditLogTargetType;
      if (auditLogGatewayId) params.gatewayId = auditLogGatewayId;
      if (ipAddress) params.ipAddress = ipAddress;
      if (geoCountry) params.geoCountry = geoCountry;
      if (startDate) params.startDate = startDate;
      if (endDate) params.endDate = endDate;
      if (flaggedOnly) params.flaggedOnly = true;

      const result = await getAuditLogs(params);
      setLogs(result.data);
      setTotal(result.total);
    } catch {
      setError("Failed to load audit logs");
    } finally {
      setLoading(false);
    }
  }, [
    page,
    rowsPerPage,
    auditLogAction,
    auditLogSearch,
    auditLogTargetType,
    auditLogGatewayId,
    ipAddress,
    geoCountry,
    startDate,
    endDate,
    auditLogSortBy,
    auditLogSortOrder,
    flaggedOnly,
  ]);

  const fetchDbLogs = useCallback(async () => {
    setDbLoading(true);
    setDbError("");
    try {
      const params: DbAuditLogParams = {
        page: dbPage + 1,
        limit: dbRowsPerPage,
        sortBy: "createdAt",
        sortOrder: "desc",
      };
      if (dbSearch) params.search = dbSearch;
      if (dbQueryType) params.queryType = dbQueryType as DbQueryType;
      if (dbConnectionId) params.connectionId = dbConnectionId;
      if (dbUserId) params.userId = dbUserId;
      if (dbBlocked === "true") params.blocked = true;
      if (dbBlocked === "false") params.blocked = false;
      if (dbStartDate) params.startDate = dbStartDate;
      if (dbEndDate) params.endDate = dbEndDate;

      const result = await getDbAuditLogs(params);
      setDbLogs(result.data);
      setDbTotal(result.total);
    } catch {
      setDbError("Failed to load SQL audit logs");
    } finally {
      setDbLoading(false);
    }
  }, [
    dbPage,
    dbRowsPerPage,
    dbSearch,
    dbQueryType,
    dbConnectionId,
    dbUserId,
    dbBlocked,
    dbStartDate,
    dbEndDate,
  ]);

  useEffect(() => {
    if (open && activeTab === "general") {
      fetchLogs();
      getAuditGateways()
        .then(setGateways)
        .catch(() => {});
      getAuditCountries()
        .then(setCountries)
        .catch(() => {});
    }
  }, [open, fetchLogs, activeTab]);

  useEffect(() => {
    if (open && activeTab === "sql" && hasTenant) {
      fetchDbLogs();
      getDbAuditConnections()
        .then(setDbConnections)
        .catch(() => {});
      getDbAuditUsers()
        .then(setDbUsers)
        .catch(() => {});
    }
  }, [open, fetchDbLogs, activeTab, hasTenant]);

  useEffect(() => {
    if (
      !open ||
      !accessToken ||
      autoRefreshPaused ||
      activeTab !== "general" ||
      page !== 0
    )
      return undefined;

    const params = new URLSearchParams({
      page: "1",
      limit: String(rowsPerPage),
      sortBy: auditLogSortBy as string,
      sortOrder: auditLogSortOrder as string,
    });
    if (auditLogAction) params.set("action", auditLogAction);
    if (auditLogSearch) params.set("search", auditLogSearch);
    if (auditLogTargetType) params.set("targetType", auditLogTargetType);
    if (auditLogGatewayId) params.set("gatewayId", auditLogGatewayId);
    if (ipAddress) params.set("ipAddress", ipAddress);
    if (geoCountry) params.set("geoCountry", geoCountry);
    if (startDate) params.set("startDate", startDate);
    if (endDate) params.set("endDate", endDate);
    if (flaggedOnly) params.set("flaggedOnly", "true");

    return connectSSE({
      url: `/api/audit/stream?${params.toString()}`,
      accessToken,
      onEvent: ({ event, data }) => {
        if (event !== "snapshot") return;
        const snapshot = data as AuditStreamSnapshot;
        setLogs(snapshot.data);
        setTotal(snapshot.total);
        setLoading(false);
        setError("");
      },
    });
  }, [
    open,
    accessToken,
    autoRefreshPaused,
    activeTab,
    page,
    rowsPerPage,
    auditLogAction,
    auditLogSearch,
    auditLogTargetType,
    auditLogGatewayId,
    auditLogSortBy,
    auditLogSortOrder,
    ipAddress,
    geoCountry,
    startDate,
    endDate,
    flaggedOnly,
  ]);

  useEffect(() => {
    if (
      !open ||
      !accessToken ||
      !hasTenant ||
      autoRefreshPaused ||
      activeTab !== "sql" ||
      dbPage !== 0
    )
      return undefined;

    const params = new URLSearchParams({
      page: "1",
      limit: String(dbRowsPerPage),
      sortBy: "createdAt",
      sortOrder: "desc",
    });
    if (dbSearch) params.set("search", dbSearch);
    if (dbQueryType) params.set("queryType", dbQueryType);
    if (dbConnectionId) params.set("connectionId", dbConnectionId);
    if (dbUserId) params.set("userId", dbUserId);
    if (dbBlocked === "true" || dbBlocked === "false")
      params.set("blocked", dbBlocked);
    if (dbStartDate) params.set("startDate", dbStartDate);
    if (dbEndDate) params.set("endDate", dbEndDate);

    return connectSSE({
      url: `/api/db-audit/logs/stream?${params.toString()}`,
      accessToken,
      onEvent: ({ event, data }) => {
        if (event !== "snapshot") return;
        const snapshot = data as DbAuditStreamSnapshot;
        setDbLogs(snapshot.data);
        setDbTotal(snapshot.total);
        setDbLoading(false);
        setDbError("");
      },
    });
  }, [
    open,
    accessToken,
    hasTenant,
    autoRefreshPaused,
    activeTab,
    dbPage,
    dbRowsPerPage,
    dbSearch,
    dbQueryType,
    dbConnectionId,
    dbUserId,
    dbBlocked,
    dbStartDate,
    dbEndDate,
  ]);

  const handleSort = (field: "createdAt" | "action") => {
    if (auditLogSortBy === field) {
      setUiPref(
        "auditLogSortOrder",
        auditLogSortOrder === "asc" ? "desc" : "asc",
      );
    } else {
      setUiPref("auditLogSortBy", field);
      setUiPref("auditLogSortOrder", field === "createdAt" ? "desc" : "asc");
    }
    setPage(0);
  };

  const hasActiveFilters =
    auditLogAction ||
    auditLogSearch ||
    auditLogTargetType ||
    auditLogGatewayId ||
    ipAddress ||
    geoCountry ||
    startDate ||
    endDate ||
    flaggedOnly;
  const hasDbActiveFilters =
    dbSearch ||
    dbQueryType ||
    dbConnectionId ||
    dbUserId ||
    dbBlocked ||
    dbStartDate ||
    dbEndDate;

  const totalPages = Math.ceil(total / rowsPerPage);
  const dbTotalPages = Math.ceil(dbTotal / dbRowsPerPage);

  const handleViewRecording = async (log: AuditLogEntry) => {
    const sessionId = (log.details as Record<string, unknown>)?.sessionId as
      | string
      | undefined;
    const recordingId = (log.details as Record<string, unknown>)
      ?.recordingId as string | undefined;
    if (!sessionId && !recordingId) return;

    setLoadingRecordingId(log.id);
    try {
      let recording: Recording;
      if (recordingId) {
        recording = await getRecording(recordingId);
      } else if (sessionId) {
        recording = await getSessionRecording(sessionId);
      } else {
        return;
      }
      setSelectedRecording(recording);
      setRecordingPlayerOpen(true);
    } catch {
      // Recording not found or not available
    } finally {
      setLoadingRecordingId(null);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(next) => { if (!next) onClose(); }}>
      <DialogContent
        className="h-[100dvh] w-screen max-w-none gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1500px] sm:overflow-hidden sm:rounded-2xl sm:border"
        showCloseButton={false}
      >
        <DialogTitle className="sr-only">Activity Log</DialogTitle>
        <DialogDescription className="sr-only">System audit log</DialogDescription>

        {/* Header */}
        <div className="border-b bg-card">
          <div className="flex items-center gap-3 px-4 py-2.5">
            <Button variant="ghost" size="icon" onClick={onClose} className="size-8">
              <X className="size-4" />
            </Button>
            <h2 className="flex-1 text-lg font-semibold">Activity Log</h2>
            <Button
              variant="ghost"
              size="icon"
              className="size-8"
              onClick={() =>
                setUiPref("auditLogAutoRefreshPaused", !autoRefreshPaused)
              }
              title={autoRefreshPaused ? "Resume live updates" : "Pause live updates"}
            >
              {autoRefreshPaused ? <Play className="size-4" /> : <Pause className="size-4" />}
            </Button>
            <Badge
              variant={autoRefreshPaused ? "outline" : "default"}
              className={cn(
                "font-semibold",
                autoRefreshPaused
                  ? ""
                  : "bg-emerald-600/15 text-emerald-400 border-emerald-600/30",
              )}
            >
              {autoRefreshPaused ? (
                "Paused"
              ) : (
                <span className="inline-flex items-center gap-1.5">
                  <span className="size-1.5 rounded-full bg-current animate-pulse" />
                  Live
                </span>
              )}
            </Badge>
          </div>
          {hasTenant && (
            <Tabs
              value={activeTab}
              onValueChange={(v) => setUiPref("auditLogDialogTab", v)}
              className="px-4"
            >
              <TabsList className="h-9">
                <TabsTrigger value="general" className="gap-1.5 text-xs">
                  <List className="size-3.5" />
                  General
                </TabsTrigger>
                {sqlAuditVisible && (
                  <TabsTrigger value="sql" className="gap-1.5 text-xs">
                    <Database className="size-3.5" />
                    SQL Audit
                  </TabsTrigger>
                )}
              </TabsList>
            </Tabs>
          )}
        </div>

        {/* Body */}
        <div className="flex-1 overflow-auto p-4">
          {activeTab === "general" && (
            <>
              {/* General Filters */}
              <div className="rounded-lg border bg-card p-3 mb-4">
                <div className="relative mb-3">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
                  <Input
                    className="pl-9"
                    placeholder="Search across target, IP address, and details..."
                    value={searchInput}
                    onChange={(e) => setSearchInput(e.target.value)}
                  />
                </div>
                <div className="flex flex-wrap items-center gap-3">
                  <div className="min-w-[200px] space-y-1">
                    <Label className="text-xs">Action</Label>
                    <Select value={auditLogAction || "__all__"} onValueChange={(v) => { setUiPref("auditLogAction", v === "__all__" ? "" : v); setPage(0); }}>
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="__all__">All Actions</SelectItem>
                        {ALL_ACTIONS.map((action) => (
                          <SelectItem key={action} value={action}>{ACTION_LABELS[action]}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="min-w-[160px] space-y-1">
                    <Label className="text-xs">Target Type</Label>
                    <Select value={auditLogTargetType || "__all__"} onValueChange={(v) => { setUiPref("auditLogTargetType", v === "__all__" ? "" : v); setPage(0); }}>
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="__all__">All Types</SelectItem>
                        {TARGET_TYPES.map((t) => (
                          <SelectItem key={t} value={t}>{t}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  {gateways.length > 0 && (
                    <div className="min-w-[160px] space-y-1">
                      <Label className="text-xs">Gateway</Label>
                      <Select value={auditLogGatewayId || "__all__"} onValueChange={(v) => { setUiPref("auditLogGatewayId", v === "__all__" ? "" : v); setPage(0); }}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="__all__">All Gateways</SelectItem>
                          {gateways.map((gw) => (
                            <SelectItem key={gw.id} value={gw.id}>{gw.name}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}
                  {countries.length > 0 && (
                    <div className="min-w-[160px] space-y-1">
                      <Label className="text-xs">Country</Label>
                      <Select value={geoCountry || "__all__"} onValueChange={(v) => { setGeoCountry(v === "__all__" ? "" : v); setPage(0); }}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="__all__">All Countries</SelectItem>
                          {countries.map((c) => (
                            <SelectItem key={c} value={c}>{c}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}
                  <div className="w-[160px] space-y-1">
                    <Label className="text-xs">IP Address</Label>
                    <Input value={ipAddress} onChange={(e) => { setIpAddress(e.target.value); setPage(0); }} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">From</Label>
                    <Input type="date" value={startDate} onChange={(e) => { setStartDate(e.target.value); setPage(0); }} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">To</Label>
                    <Input type="date" value={endDate} onChange={(e) => { setEndDate(e.target.value); setPage(0); }} />
                  </div>
                  <Badge
                    variant={flaggedOnly ? "default" : "outline"}
                    className={cn(
                      "cursor-pointer gap-1 mt-5",
                      flaggedOnly ? "bg-yellow-600/15 text-yellow-500 border-yellow-600/30" : "",
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

              {/* General Table */}
              <div className="rounded-lg border bg-card">
                {loading ? (
                  <div className="flex justify-center py-12">
                    <Loader2 className="size-8 animate-spin text-muted-foreground" />
                  </div>
                ) : logs.length === 0 ? (
                  <div className="text-center py-12">
                    <p className="text-sm text-muted-foreground">
                      {hasActiveFilters ? "No logs match your filters" : "No activity recorded yet"}
                    </p>
                  </div>
                ) : (
                  <>
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b bg-muted/50">
                          <th className="w-8 px-2 py-2" />
                          <th className="text-left px-3 py-2 font-medium">
                            <button className="inline-flex items-center gap-1 hover:text-foreground" onClick={() => handleSort("createdAt")}>
                              Date/Time
                              {auditLogSortBy === "createdAt" && (auditLogSortOrder === "asc" ? <ChevronUp className="size-3" /> : <ChevronDown className="size-3" />)}
                            </button>
                          </th>
                          <th className="text-left px-3 py-2 font-medium">
                            <button className="inline-flex items-center gap-1 hover:text-foreground" onClick={() => handleSort("action")}>
                              Action
                              {auditLogSortBy === "action" && (auditLogSortOrder === "asc" ? <ChevronUp className="size-3" /> : <ChevronDown className="size-3" />)}
                            </button>
                          </th>
                          <th className="text-left px-3 py-2 font-medium">Target</th>
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
                                <td className="px-3 py-2">
                                  <div className="inline-flex items-center gap-1.5">
                                    <Badge variant="outline" className={cn("border", ACTION_COLOR_MAP[getActionColor(log.action) as string] || "")}>
                                      {ACTION_LABELS[log.action] || log.action}
                                    </Badge>
                                    {log.flags?.includes("IMPOSSIBLE_TRAVEL") && (
                                      <span title="Impossible travel detected"><AlertTriangle className="size-4 text-yellow-500" /></span>
                                    )}
                                    {["SESSION_START", "SESSION_END", "SESSION_TERMINATED_POLICY_VIOLATION"].includes(log.action) &&
                                      Boolean((log.details as Record<string, unknown>)?.sessionId) && (
                                        <Button
                                          variant="ghost"
                                          size="icon"
                                          className="size-6"
                                          onClick={(e) => { e.stopPropagation(); handleViewRecording(log); }}
                                          disabled={loadingRecordingId === log.id}
                                          title="View Recording"
                                        >
                                          {loadingRecordingId === log.id ? (
                                            <Loader2 className="size-3.5 animate-spin" />
                                          ) : (
                                            <Play className="size-3.5" />
                                          )}
                                        </Button>
                                    )}
                                  </div>
                                </td>
                                <td className="px-3 py-2">
                                  {log.targetType
                                    ? `${log.targetType}${log.targetId ? ` ${log.targetId.slice(0, 8)}...` : ""}`
                                    : "\u2014"}
                                </td>
                                <td className="px-3 py-2">
                                  <IpGeoCell ipAddress={log.ipAddress} geoCountry={log.geoCountry} geoCity={log.geoCity} onGeoIpClick={onGeoIpClick} />
                                </td>
                                <td className="px-3 py-2 max-w-[300px] overflow-hidden text-ellipsis whitespace-nowrap">
                                  {formatDetails(log.details)}
                                </td>
                              </tr>
                              {isExpanded && (
                                <tr>
                                  <td colSpan={6} className="px-6 py-4 border-b">
                                    {log.details && typeof log.details === "object" && Object.keys(log.details).length > 0 ? (
                                      <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 max-w-[600px]">
                                        {Object.entries(log.details).map(([key, value]) => (
                                          <Fragment key={key}>
                                            <span className="text-sm font-semibold text-muted-foreground">{key}</span>
                                            <span className="text-sm break-all">
                                              {Array.isArray(value) ? value.join(", ") : String(value)}
                                            </span>
                                          </Fragment>
                                        ))}
                                      </div>
                                    ) : (
                                      <p className="text-sm text-muted-foreground">No additional details</p>
                                    )}
                                    {log.targetId && (
                                      <p className="text-xs text-muted-foreground mt-2">Full Target ID: {log.targetId}</p>
                                    )}
                                  </td>
                                </tr>
                              )}
                            </Fragment>
                          );
                        })}
                      </tbody>
                    </table>
                    <div className="flex items-center justify-between px-4 py-2 border-t text-sm text-muted-foreground">
                      <div className="flex items-center gap-2">
                        <span>Rows per page:</span>
                        <Select value={String(rowsPerPage)} onValueChange={(v) => { setRowsPerPage(parseInt(v, 10)); setPage(0); }}>
                          <SelectTrigger className="h-8 w-[70px]"><SelectValue /></SelectTrigger>
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
            </>
          )}

          {activeTab === "sql" && hasTenant && (
            <>
              {/* SQL Filters */}
              <div className="rounded-lg border bg-card p-3 mb-4">
                <div className="relative mb-3">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground" />
                  <Input
                    className="pl-9"
                    placeholder="Search SQL queries, tables, or block reasons..."
                    value={dbSearch}
                    onChange={(e) => { setDbSearch(e.target.value); setDbPage(0); }}
                  />
                </div>
                <div className="flex flex-wrap items-center gap-3">
                  <div className="min-w-[140px] space-y-1">
                    <Label className="text-xs">Query Type</Label>
                    <Select value={dbQueryType || "__all__"} onValueChange={(v) => { setDbQueryType(v === "__all__" ? "" : v); setDbPage(0); }}>
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="__all__">All Types</SelectItem>
                        {ALL_QUERY_TYPES.map((qt) => (
                          <SelectItem key={qt} value={qt}>{QUERY_TYPE_LABELS[qt]}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  {dbConnections.length > 0 && (
                    <div className="min-w-[160px] space-y-1">
                      <Label className="text-xs">Connection</Label>
                      <Select value={dbConnectionId || "__all__"} onValueChange={(v) => { setDbConnectionId(v === "__all__" ? "" : v); setDbPage(0); }}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="__all__">All Connections</SelectItem>
                          {dbConnections.map((c) => (
                            <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}
                  {dbUsers.length > 0 && (
                    <div className="min-w-[160px] space-y-1">
                      <Label className="text-xs">User</Label>
                      <Select value={dbUserId || "__all__"} onValueChange={(v) => { setDbUserId(v === "__all__" ? "" : v); setDbPage(0); }}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="__all__">All Users</SelectItem>
                          {dbUsers.map((u) => (
                            <SelectItem key={u.id} value={u.id}>{u.username || u.email}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  )}
                  <div className="min-w-[130px] space-y-1">
                    <Label className="text-xs">Status</Label>
                    <Select value={dbBlocked || "__all__"} onValueChange={(v) => { setDbBlocked(v === "__all__" ? "" : v); setDbPage(0); }}>
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
                    <Input type="date" value={dbStartDate} onChange={(e) => { setDbStartDate(e.target.value); setDbPage(0); }} />
                  </div>
                  <div className="space-y-1">
                    <Label className="text-xs">To</Label>
                    <Input type="date" value={dbEndDate} onChange={(e) => { setDbEndDate(e.target.value); setDbPage(0); }} />
                  </div>
                </div>
              </div>

              {dbError && (
                <div className="rounded-md border border-destructive/50 bg-destructive/10 px-4 py-3 text-sm text-destructive mb-4">
                  {dbError}
                </div>
              )}

              {/* SQL Table */}
              <div className="rounded-lg border bg-card">
                {dbLoading ? (
                  <div className="flex justify-center py-12">
                    <Loader2 className="size-8 animate-spin text-muted-foreground" />
                  </div>
                ) : dbLogs.length === 0 ? (
                  <div className="text-center py-12">
                    <p className="text-sm text-muted-foreground">
                      {hasDbActiveFilters ? "No SQL audit logs match your filters" : "No SQL queries recorded yet"}
                    </p>
                  </div>
                ) : (
                  <>
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b bg-muted/50">
                          <th className="w-8 px-2 py-2" />
                          <th className="text-left px-3 py-2 font-medium">Date/Time</th>
                          <th className="text-left px-3 py-2 font-medium">User</th>
                          <th className="text-left px-3 py-2 font-medium">Connection</th>
                          <th className="text-left px-3 py-2 font-medium">Type</th>
                          <th className="text-left px-3 py-2 font-medium">Tables</th>
                          <th className="text-left px-3 py-2 font-medium">Status</th>
                          <th className="text-left px-3 py-2 font-medium">Time (ms)</th>
                        </tr>
                      </thead>
                      <tbody>
                        {dbLogs.map((entry) => {
                          const isExpanded = dbExpandedRowId === entry.id;
                          return (
                            <Fragment key={entry.id}>
                              <tr
                                className="border-b hover:bg-accent/50 cursor-pointer"
                                onClick={() => setDbExpandedRowId(isExpanded ? null : entry.id)}
                              >
                                <td className="px-2 py-2">
                                  <Button variant="ghost" size="icon" className="size-6">
                                    {isExpanded ? <ChevronUp className="size-3.5" /> : <ChevronDown className="size-3.5" />}
                                  </Button>
                                </td>
                                <td className="px-3 py-2 whitespace-nowrap">
                                  {new Date(entry.createdAt).toLocaleString()}
                                </td>
                                <td className="px-3 py-2">
                                  {entry.userName || entry.userEmail || entry.userId.slice(0, 8)}
                                </td>
                                <td className="px-3 py-2">
                                  {entry.connectionName || entry.connectionId.slice(0, 8)}
                                </td>
                                <td className="px-3 py-2">
                                  <Badge variant="outline" className={cn("border", QUERY_TYPE_COLORS[entry.queryType] || "")}>
                                    {QUERY_TYPE_LABELS[entry.queryType] || entry.queryType}
                                  </Badge>
                                </td>
                                <td className="px-3 py-2 max-w-[200px] overflow-hidden text-ellipsis whitespace-nowrap">
                                  {entry.tablesAccessed.length > 0 ? entry.tablesAccessed.join(", ") : "\u2014"}
                                </td>
                                <td className="px-3 py-2">
                                  {entry.blocked ? (
                                    <Badge variant="outline" className="border bg-destructive/15 text-destructive border-destructive/30">Blocked</Badge>
                                  ) : entry.blockReason ? (
                                    <Badge variant="outline" className="border bg-yellow-600/15 text-yellow-500 border-yellow-600/30">Alert</Badge>
                                  ) : (
                                    <Badge variant="outline" className="border bg-emerald-600/15 text-emerald-400 border-emerald-600/30">OK</Badge>
                                  )}
                                </td>
                                <td className="px-3 py-2">
                                  {entry.executionTimeMs !== null ? `${entry.executionTimeMs}` : "\u2014"}
                                </td>
                              </tr>
                              {isExpanded && (
                                <tr>
                                  <td colSpan={8} className="px-6 py-4 border-b max-w-[800px]">
                                    <p className="text-sm font-semibold text-muted-foreground mb-1">Query</p>
                                    <div className="p-3 rounded bg-accent/50 font-mono text-[0.85rem] whitespace-pre-wrap break-all mb-3">
                                      {entry.queryText}
                                    </div>
                                    <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1">
                                      <span className="text-sm font-semibold text-muted-foreground">Rows Affected</span>
                                      <span className="text-sm">{entry.rowsAffected ?? "\u2014"}</span>
                                      {entry.blockReason && (
                                        <>
                                          <span className="text-sm font-semibold text-muted-foreground">
                                            {entry.blocked ? "Block Reason" : "Firewall Alert"}
                                          </span>
                                          <span className={cn("text-sm", entry.blocked ? "text-destructive" : "text-yellow-500")}>
                                            {entry.blockReason}
                                          </span>
                                        </>
                                      )}
                                    </div>
                                    <div className="mt-3">
                                      <Button
                                        variant="ghost"
                                        size="icon"
                                        className="size-7 text-primary"
                                        onClick={(e) => { e.stopPropagation(); setVisualizerEntry(entry); }}
                                        title="Open query visualizer"
                                      >
                                        <Eye className="size-4" />
                                      </Button>
                                    </div>
                                  </td>
                                </tr>
                              )}
                            </Fragment>
                          );
                        })}
                      </tbody>
                    </table>
                    <div className="flex items-center justify-between px-4 py-2 border-t text-sm text-muted-foreground">
                      <div className="flex items-center gap-2">
                        <span>Rows per page:</span>
                        <Select value={String(dbRowsPerPage)} onValueChange={(v) => { setDbRowsPerPage(parseInt(v, 10)); setDbPage(0); }}>
                          <SelectTrigger className="h-8 w-[70px]"><SelectValue /></SelectTrigger>
                          <SelectContent>
                            <SelectItem value="25">25</SelectItem>
                            <SelectItem value="50">50</SelectItem>
                            <SelectItem value="100">100</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      <div className="flex items-center gap-2">
                        <span>{dbPage * dbRowsPerPage + 1}-{Math.min((dbPage + 1) * dbRowsPerPage, dbTotal)} of {dbTotal}</span>
                        <Button variant="ghost" size="sm" disabled={dbPage === 0} onClick={() => setDbPage((p) => p - 1)}>Previous</Button>
                        <Button variant="ghost" size="sm" disabled={dbPage + 1 >= dbTotalPages} onClick={() => setDbPage((p) => p + 1)}>Next</Button>
                      </div>
                    </div>
                  </>
                )}
              </div>
            </>
          )}
        </div>

        {/* Query Visualizer drawer */}
        <QueryVisualizer
          open={Boolean(visualizerEntry)}
          onClose={() => setVisualizerEntry(null)}
          queryText={visualizerEntry?.queryText ?? ""}
          queryType={visualizerEntry?.queryType ?? ""}
          executionTimeMs={visualizerEntry?.executionTimeMs ?? null}
          rowsAffected={visualizerEntry?.rowsAffected ?? null}
          tablesAccessed={visualizerEntry?.tablesAccessed ?? []}
          blocked={visualizerEntry?.blocked ?? false}
          blockReason={visualizerEntry?.blockReason}
          storedExecutionPlan={visualizerEntry?.executionPlan ?? null}
        />

        <RecordingPlayerDialog
          open={recordingPlayerOpen}
          onClose={() => {
            setRecordingPlayerOpen(false);
            setSelectedRecording(null);
          }}
          recording={selectedRecording}
        />
      </DialogContent>
    </Dialog>
  );
}
