import type { SessionConsoleSession, SessionConsoleStatus } from '@/api/sessions.api';

export const SESSION_STATUS_FILTER_OPTIONS: Array<{ value: SessionConsoleStatus; label: string }> = [
  { value: 'ACTIVE', label: 'Active' },
  { value: 'IDLE', label: 'Idle' },
  { value: 'PAUSED', label: 'Paused' },
  { value: 'CLOSED', label: 'Closed' },
];

export const SESSION_PAGE_SIZE = 25;

export function formatSessionTimestamp(value: string | null | undefined) {
  if (!value) {
    return '—';
  }

  return new Date(value).toLocaleString();
}

export function formatRecordingSize(bytes: number | null | undefined) {
  if (bytes == null) {
    return '—';
  }
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function formatRecordingDuration(seconds: number | null | undefined) {
  if (seconds == null) {
    return '—';
  }
  const minutes = Math.floor(seconds / 60);
  const remainder = seconds % 60;
  return `${minutes}:${String(remainder).padStart(2, '0')}`;
}

export function recordingExtension(format: string | null | undefined) {
  if (format === 'asciicast') {
    return 'cast';
  }
  return format || 'dat';
}

export function matchesTextFilter(session: SessionConsoleSession, query: string) {
  const normalized = query.trim().toLowerCase();
  if (!normalized) {
    return true;
  }

  return [
    session.username,
    session.email,
    session.connectionName,
    session.connectionHost,
    session.gatewayName,
    session.instanceName,
  ]
    .filter(Boolean)
    .some((value) => value!.toLowerCase().includes(normalized));
}

export function matchesStatusFilter(session: SessionConsoleSession, statuses: readonly SessionConsoleStatus[]) {
  return statuses.includes(session.status);
}

export function getSessionStatusBadgeClass(status: SessionConsoleStatus) {
  switch (status) {
    case 'ACTIVE':
      return 'border-primary/30 bg-primary/10 text-primary';
    case 'IDLE':
      return 'border-amber-500/30 bg-amber-500/10 text-amber-300';
    case 'PAUSED':
      return 'border-sky-500/30 bg-sky-500/10 text-sky-300';
    case 'CLOSED':
      return 'border-border bg-muted/50 text-muted-foreground';
    default:
      return 'border-border bg-muted/50 text-muted-foreground';
  }
}

export function getConsoleServerStatuses(
  statuses: readonly SessionConsoleStatus[],
  scope: 'own' | 'tenant' | null,
): SessionConsoleStatus[] | undefined {
  const filtered = scope === 'own'
    ? statuses.filter((status) => status !== 'CLOSED')
    : [...statuses];

  return filtered.length > 0 ? filtered : undefined;
}

export function formatStatusFilterLabel(statuses: readonly SessionConsoleStatus[]) {
  if (statuses.length === 0) {
    return 'Status';
  }
  if (statuses.length === SESSION_STATUS_FILTER_OPTIONS.length) {
    return 'All statuses';
  }
  return statuses
    .map((status) => SESSION_STATUS_FILTER_OPTIONS.find((option) => option.value === status)?.label ?? status)
    .join(', ');
}
