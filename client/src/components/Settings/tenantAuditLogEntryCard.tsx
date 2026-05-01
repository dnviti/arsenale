import { Play } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import type { TenantAuditLogEntry } from '../../api/audit.api';
import { ACTION_LABELS, formatDetails } from '../Audit/auditConstants';
import IpGeoCell from '../Audit/IpGeoCell';
import { SettingsFieldCard } from './settings-ui';
import { actionBadgeClasses, tenantAuditTargetLabel } from './tenantAuditLogUtils';

export default function TenantAuditEntryCard({
  expanded,
  log,
  onGeoIpClick,
  onToggle,
  onViewRecording,
  onViewUserProfile,
  recordingLoading = false,
}: {
  expanded: boolean;
  log: TenantAuditLogEntry;
  onGeoIpClick?: (ip: string) => void;
  onToggle: () => void;
  onViewRecording?: () => void;
  onViewUserProfile?: (userId: string) => void;
  recordingLoading?: boolean;
}) {
  const detailsPreview = formatDetails(log.details as Record<string, unknown> | null);
  const hasFlag = log.flags?.includes('IMPOSSIBLE_TRAVEL');
  const userLabel = log.userName ?? log.userEmail ?? '\u2014';
  const canViewRecording = ['SESSION_START', 'SESSION_END', 'SESSION_TERMINATED_POLICY_VIOLATION'].includes(log.action)
    && Boolean((log.details as Record<string, unknown>)?.sessionId || (log.details as Record<string, unknown>)?.recordingId);

  return (
    <SettingsFieldCard
      label={userLabel}
      description={`${new Date(log.createdAt).toLocaleString()} · ${tenantAuditTargetLabel(log)}`}
      aside={(
        <div className="flex flex-wrap items-center justify-end gap-2">
          <Badge className={cn('border', actionBadgeClasses(log.action))}>{ACTION_LABELS[log.action] || log.action}</Badge>
          {hasFlag ? <Badge className="border border-chart-5/30 bg-chart-5/10 text-foreground">Flagged</Badge> : null}
          {canViewRecording && onViewRecording ? (
            <Button type="button" variant="ghost" size="sm" onClick={onViewRecording} disabled={recordingLoading}>
              <Play className="size-4" />
              {recordingLoading ? 'Loading...' : 'Recording'}
            </Button>
          ) : null}
          <Button type="button" variant="ghost" size="sm" onClick={onToggle}>
            {expanded ? 'Hide Details' : 'Show Details'}
          </Button>
        </div>
      )}
      contentClassName="space-y-4"
    >
      <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <div className="space-y-2">
          <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">User</div>
          {log.userId && onViewUserProfile ? (
            <button
              type="button"
              className="text-left text-sm font-medium text-foreground underline-offset-4 hover:underline"
              onClick={() => {
                if (log.userId) {
                  onViewUserProfile(log.userId);
                }
              }}
            >
              {userLabel}
            </button>
          ) : (
            <div className="text-sm font-medium text-foreground">{userLabel}</div>
          )}
          {log.userEmail ? <div className="text-sm text-muted-foreground">{log.userEmail}</div> : null}
        </div>
        <div className="space-y-2">
          <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">IP & Geography</div>
          <IpGeoCell
            ipAddress={log.ipAddress}
            geoCountry={log.geoCountry}
            geoCity={log.geoCity}
            onGeoIpClick={onGeoIpClick}
          />
        </div>
      </div>

      <div className="space-y-2">
        <div className="text-xs uppercase tracking-[0.18em] text-muted-foreground">Details</div>
        <p className="text-sm leading-6 text-muted-foreground">
          {detailsPreview || 'No additional details captured for this event.'}
        </p>
      </div>

      {expanded ? (
        <div className="grid gap-2 rounded-xl border border-border/70 bg-background/60 p-4 md:grid-cols-[180px_minmax(0,1fr)]">
          <div className="text-sm font-medium text-foreground">Action metadata</div>
          <div className="space-y-2 text-sm text-muted-foreground">
            <div>Target: {tenantAuditTargetLabel(log)}</div>
            {log.targetId ? <div>Full target ID: {log.targetId}</div> : null}
            {log.flags?.length ? <div>Flags: {log.flags.join(', ')}</div> : null}
          </div>
          <div className="text-sm font-medium text-foreground">Structured details</div>
          <div className="grid gap-2 text-sm text-muted-foreground">
            {log.details && typeof log.details === 'object' && Object.keys(log.details).length > 0 ? (
              Object.entries(log.details as Record<string, unknown>).map(([key, value]) => (
                <div key={key}>
                  <span className="font-medium text-foreground">{key}:</span>{' '}
                  {Array.isArray(value) ? value.join(', ') : String(value)}
                </div>
              ))
            ) : (
              <div>No additional structured details.</div>
            )}
          </div>
        </div>
      ) : null}
    </SettingsFieldCard>
  );
}
