import {
  AlertTriangle,
  ChevronDown,
  ChevronUp,
  Loader2,
  Play,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import type { AuditAction, TenantAuditLogEntry } from '../../api/audit.api';
import {
  ACTION_LABELS,
  formatDetails,
  getActionColor,
} from '../Audit/auditConstants';
import IpGeoCell from '../Audit/IpGeoCell';

const ACTION_COLOR_MAP: Record<string, string> = {
  default: '',
  primary: 'bg-primary/15 text-primary border-primary/30',
  secondary: 'bg-muted text-muted-foreground',
  error: 'bg-destructive/15 text-destructive border-destructive/30',
  warning: 'bg-yellow-600/15 text-yellow-500 border-yellow-600/30',
  success: 'bg-emerald-600/15 text-emerald-400 border-emerald-600/30',
  info: 'bg-blue-600/15 text-blue-400 border-blue-600/30',
};

interface AuditLogTenantRowProps {
  expanded: boolean;
  loadingRecordingId: string | null;
  log: TenantAuditLogEntry;
  onGeoIpClick?: (ip: string) => void;
  onToggle: () => void;
  onViewRecording: () => void;
  onViewUserProfile?: (userId: string) => void;
}

export default function AuditLogTenantRow({
  expanded,
  loadingRecordingId,
  log,
  onGeoIpClick,
  onToggle,
  onViewRecording,
  onViewUserProfile,
}: AuditLogTenantRowProps) {
  const canViewRecording = ['SESSION_START', 'SESSION_END', 'SESSION_TERMINATED_POLICY_VIOLATION'].includes(log.action)
    && Boolean((log.details as Record<string, unknown>)?.sessionId || (log.details as Record<string, unknown>)?.recordingId);

  const userId = log.userId ?? null;
  const userLabel = log.userName || log.userEmail || (log.userId ? log.userId.slice(0, 8) : '\u2014');
  const details = formatDetails(log.details);

  return (
    <>
      <tr className="cursor-pointer border-b hover:bg-accent/50" onClick={onToggle}>
        <td className="px-2 py-2">
          <Button variant="ghost" size="icon" className="size-6">
            {expanded ? <ChevronUp className="size-3.5" /> : <ChevronDown className="size-3.5" />}
          </Button>
        </td>
        <td className="whitespace-nowrap px-3 py-2">{new Date(log.createdAt).toLocaleString()}</td>
        <td className="px-3 py-2">
          <div className="flex min-w-[12rem] flex-col">
            {userId && onViewUserProfile ? (
              <button
                type="button"
                className="w-fit text-left text-primary hover:underline"
                onClick={(event) => {
                  event.stopPropagation();
                  onViewUserProfile(userId);
                }}
              >
                {userLabel}
              </button>
            ) : (
              <span>{userLabel}</span>
            )}
            {log.userEmail && log.userEmail !== userLabel ? (
              <span className="text-xs text-muted-foreground">{log.userEmail}</span>
            ) : null}
          </div>
        </td>
        <td className="px-3 py-2">
          <div className="inline-flex items-center gap-1.5">
            <Badge variant="outline" className={cn('border', ACTION_COLOR_MAP[getActionColor(log.action)] || '')}>
              {ACTION_LABELS[log.action as AuditAction] || log.action}
            </Badge>
            {log.flags?.length ? (
              <span title={log.flags.join(', ')}>
                <AlertTriangle className="size-4 text-yellow-500" />
              </span>
            ) : null}
            {canViewRecording ? (
              <Button
                variant="ghost"
                size="icon"
                className="size-6"
                onClick={(event) => {
                  event.stopPropagation();
                  onViewRecording();
                }}
                disabled={loadingRecordingId === log.id}
                title="View Recording"
              >
                {loadingRecordingId === log.id ? (
                  <Loader2 className="size-3.5 animate-spin" />
                ) : (
                  <Play className="size-3.5" />
                )}
              </Button>
            ) : null}
          </div>
        </td>
        <td className="px-3 py-2">
          {log.targetType ? `${log.targetType}${log.targetId ? ` ${log.targetId.slice(0, 8)}...` : ''}` : '\u2014'}
        </td>
        <td className="px-3 py-2">
          <IpGeoCell
            ipAddress={log.ipAddress}
            geoCountry={log.geoCountry}
            geoCity={log.geoCity}
            onGeoIpClick={onGeoIpClick}
          />
        </td>
        <td className="max-w-[320px] overflow-hidden px-3 py-2 text-ellipsis whitespace-nowrap">
          {details || '\u2014'}
        </td>
      </tr>
      {expanded ? (
        <tr>
          <td colSpan={7} className="border-b px-6 py-4">
            <div className="space-y-4">
              {log.flags?.length ? (
                <div className="flex flex-wrap gap-2">
                  {log.flags.map((flag) => (
                    <Badge key={flag} variant="outline" className="border-yellow-600/30 bg-yellow-600/10 text-yellow-600">
                      {flag.replaceAll('_', ' ')}
                    </Badge>
                  ))}
                </div>
              ) : null}
              {log.details && typeof log.details === 'object' && Object.keys(log.details).length > 0 ? (
                <div className="grid max-w-[700px] grid-cols-[auto_1fr] gap-x-4 gap-y-1">
                  {Object.entries(log.details).map(([key, value]) => (
                    <div key={key} className="contents">
                      <span className="text-sm font-semibold text-muted-foreground">{key}</span>
                      <span className="break-all text-sm">
                        {Array.isArray(value) ? value.join(', ') : String(value)}
                      </span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No additional details</p>
              )}
              <div className="grid gap-1 text-xs text-muted-foreground">
                {log.userId ? <p>User ID: {log.userId}</p> : null}
                {log.targetId ? <p>Full Target ID: {log.targetId}</p> : null}
                {log.gatewayId ? <p>Gateway ID: {log.gatewayId}</p> : null}
              </div>
            </div>
          </td>
        </tr>
      ) : null}
    </>
  );
}
