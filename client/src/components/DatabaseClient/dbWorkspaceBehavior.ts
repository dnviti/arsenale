import type { DbQueryResult, DbSessionConfig } from '../../api/database.api';

export type WorkspaceQueryType = 'SELECT' | 'INSERT' | 'UPDATE' | 'DELETE' | 'DDL' | 'EXEC' | 'OTHER';

export interface QuerySubTab {
  id: string;
  label: string;
  sql: string;
  result: DbQueryResult | null;
  executing: boolean;
}

export interface PersistedQuerySubTabs {
  tabs: Array<{ id: string; label: string; sql: string }>;
  activeId: string;
}

let subTabCounter = 0;

export function createQuerySubTab(): QuerySubTab {
  subTabCounter += 1;
  return {
    id: `qtab-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
    label: `Query ${subTabCounter}`,
    sql: '',
    result: null,
    executing: false,
  };
}

export function restoreQuerySubTabs(persisted?: PersistedQuerySubTabs): QuerySubTab[] {
  if (!persisted?.tabs?.length) {
    return [createQuerySubTab()];
  }

  const restored = persisted.tabs.map((tab) => ({
    ...tab,
    result: null as DbQueryResult | null,
    executing: false,
  }));
  const maxNum = restored.reduce((max, tab) => {
    const match = tab.label.match(/^Query (\d+)$/);
    return match ? Math.max(max, parseInt(match[1], 10)) : max;
  }, 0);
  if (maxNum > subTabCounter) subTabCounter = maxNum;
  return restored;
}

export function activeQueryTabIdForTabs(tabs: QuerySubTab[], persisted?: PersistedQuerySubTabs): string {
  if (persisted?.activeId && tabs.some((tab) => tab.id === persisted.activeId)) {
    return persisted.activeId;
  }
  return tabs[0]?.id ?? createQuerySubTab().id;
}

export function persistableQuerySubTabs(tabs: QuerySubTab[], activeId: string): PersistedQuerySubTabs {
  return {
    tabs: tabs.map(({ id, label, sql }) => ({ id, label, sql })),
    activeId,
  };
}

export function hasSessionConfigValues(config: DbSessionConfig): boolean {
  return Object.values(config).some((value) => value !== undefined && value !== '');
}

export function stripLeadingComments(sql: string): string {
  let remaining = sql.trim();
  for (;;) {
    if (remaining.startsWith('--')) {
      const newline = remaining.indexOf('\n');
      remaining = (newline === -1 ? '' : remaining.slice(newline + 1)).trimStart();
    } else if (remaining.startsWith('/*')) {
      const end = remaining.indexOf('*/');
      remaining = (end === -1 ? '' : remaining.slice(end + 2)).trimStart();
    } else {
      break;
    }
  }
  return remaining;
}

export function classifyQueryType(sql: string): WorkspaceQueryType {
  const text = stripLeadingComments(sql);
  if (/^SELECT\b/i.test(text)) return 'SELECT';
  if (/^INSERT\b/i.test(text)) return 'INSERT';
  if (/^UPDATE\b/i.test(text)) return 'UPDATE';
  if (/^DELETE\b/i.test(text)) return 'DELETE';
  if (/^(CREATE|ALTER|DROP|TRUNCATE)\b/i.test(text)) return 'DDL';
  if (/^WITH\b/i.test(text)) {
    if (/\)\s*INSERT\b/i.test(text)) return 'INSERT';
    if (/\)\s*UPDATE\b/i.test(text)) return 'UPDATE';
    if (/\)\s*DELETE\b/i.test(text)) return 'DELETE';
    return 'SELECT';
  }
  if (/^(EXPLAIN|DESCRIBE|DESC|SHOW)\b/i.test(text)) return 'SELECT';
  if (/^(GRANT|REVOKE|SET)\b/i.test(text)) return 'DDL';
  if (/^MERGE\b/i.test(text)) return 'UPDATE';
  if (/^(CALL|EXEC|EXECUTE)\b/i.test(text)) return 'EXEC';
  return 'OTHER';
}

export function defaultSessionConfigForProtocol(protocol: string, databaseName?: string): DbSessionConfig {
  const normalized = protocol.toLowerCase();
  const defaults: DbSessionConfig = {};

  switch (normalized) {
    case 'postgresql':
      defaults.timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
      if (databaseName) {
        defaults.activeDatabase = databaseName;
        defaults.searchPath = 'public';
      }
      return defaults;
    case 'mysql':
      defaults.timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
      if (databaseName) {
        defaults.activeDatabase = databaseName;
      }
      return defaults;
    case 'mssql':
      if (databaseName) {
        defaults.activeDatabase = databaseName;
      }
      return defaults;
    case 'oracle':
      defaults.timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
      return defaults;
    default:
      return defaults;
  }
}

export function resultToCsv(result: DbQueryResult): string {
  const header = result.columns.join(',');
  const rows = result.rows.map((row) =>
    result.columns
      .map((col) => {
        const val = row[col];
        if (val === null || val === undefined) return '';
        const str = String(val);
        if (str.includes(',') || str.includes('"') || str.includes('\n')) {
          return `"${str.replace(/"/g, '""')}"`;
        }
        return str;
      })
      .join(','),
  );
  return [header, ...rows].join('\n');
}
