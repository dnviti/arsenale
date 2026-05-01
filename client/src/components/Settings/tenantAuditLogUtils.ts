import type { AuditAction, TenantAuditLogEntry } from '../../api/audit.api';
import { ACTION_LABELS, formatDetails, getActionColor } from '../Audit/auditConstants';

export const ALL_VALUE = '__all__';

export function exportTenantAuditCsv(logs: TenantAuditLogEntry[]) {
  const header = 'Date,User,Email,Action,Target Type,Target ID,IP Address,Country,City,Details';
  const rows = logs.map((log) => {
    const date = new Date(log.createdAt).toISOString();
    const user = (log.userName ?? '').replace(/"/g, '""');
    const email = (log.userEmail ?? '').replace(/"/g, '""');
    const action = ACTION_LABELS[log.action] || log.action;
    const targetType = log.targetType ?? '';
    const targetId = log.targetId ?? '';
    const ip = log.ipAddress ?? '';
    const country = log.geoCountry ?? '';
    const city = log.geoCity ?? '';
    const details = formatDetails(log.details as Record<string, unknown> | null).replace(/"/g, '""');
    return `"${date}","${user}","${email}","${action}","${targetType}","${targetId}","${ip}","${country}","${city}","${details}"`;
  });

  const blob = new Blob([[header, ...rows].join('\n')], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `tenant-audit-log-${new Date().toISOString().slice(0, 10)}.csv`;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function actionBadgeClasses(action: AuditAction) {
  switch (getActionColor(action)) {
    case 'error':
      return 'border-destructive/30 bg-destructive/10 text-destructive';
    case 'warning':
      return 'border-chart-5/30 bg-chart-5/10 text-foreground';
    case 'success':
    case 'info':
      return 'border-primary/25 bg-primary/10 text-primary';
    case 'secondary':
      return 'border-secondary/25 bg-secondary/15 text-secondary-foreground';
    default:
      return 'border-border bg-background text-foreground';
  }
}

export function tenantAuditTargetLabel(log: TenantAuditLogEntry) {
  if (!log.targetType) return 'No target';
  return log.targetId ? `${log.targetType} · ${log.targetId.slice(0, 8)}...` : log.targetType;
}

export function countActiveFilters(filters: Array<string | boolean>) {
  return filters.filter(Boolean).length;
}
