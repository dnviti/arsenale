import type {
  ActiveSessionData,
  ContainerLogsData,
  GatewayData,
  ManagedInstanceData,
  ScalingStatusData,
  TunnelOverviewData,
} from './gateway.api';
import type { NotificationsResponse } from './notifications.api';
import type { VaultStatusResponse } from './vault.api';
import type { AuditLogResponse } from './audit.api';
import type { DbAuditLogResponse } from './dbAudit.api';

export interface GatewayStreamSnapshot {
  gateways: GatewayData[];
  tunnelOverview: TunnelOverviewData;
  scalingStatus: Record<string, ScalingStatusData>;
  instances: Record<string, ManagedInstanceData[]>;
}

export interface ActiveSessionStreamSnapshot {
  activeSessions: ActiveSessionData[];
  sessionCount: number;
}

export type NotificationStreamSnapshot = NotificationsResponse;
export type VaultStatusStreamSnapshot = VaultStatusResponse;
export type AuditStreamSnapshot = AuditLogResponse;
export type DbAuditStreamSnapshot = DbAuditLogResponse;
export type ContainerLogStreamSnapshot = ContainerLogsData;
