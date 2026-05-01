import type { GatewayData, GatewayOperationalStatus } from '../api/gateway.api';

export interface GatewayStatusSummary {
  total: number;
  healthy: number;
  degraded: number;
  unhealthy: number;
  unknown: number;
}

export function summarizeGatewayStatuses(gateways: GatewayData[]): GatewayStatusSummary {
  return gateways.reduce<GatewayStatusSummary>((summary, gateway) => {
    summary.total += 1;
    switch (gateway.operationalStatus) {
      case 'HEALTHY':
        summary.healthy += 1;
        break;
      case 'DEGRADED':
        summary.degraded += 1;
        break;
      case 'UNHEALTHY':
        summary.unhealthy += 1;
        break;
      default:
        summary.unknown += 1;
        break;
    }
    return summary;
  }, {
    total: 0,
    healthy: 0,
    degraded: 0,
    unhealthy: 0,
    unknown: 0,
  });
}

export function gatewayStatusLabel(status: GatewayOperationalStatus): string {
  switch (status) {
    case 'HEALTHY':
      return 'Healthy';
    case 'DEGRADED':
      return 'Degraded';
    case 'UNHEALTHY':
      return 'Unhealthy';
    default:
      return 'Unknown';
  }
}

export function gatewayStatusTone(status: GatewayOperationalStatus): 'success' | 'warning' | 'destructive' | 'neutral' {
  switch (status) {
    case 'HEALTHY':
      return 'success';
    case 'DEGRADED':
      return 'warning';
    case 'UNHEALTHY':
      return 'destructive';
    default:
      return 'neutral';
  }
}

export function gatewayStatusBadgeClass(status: GatewayOperationalStatus): string {
  switch (status) {
    case 'HEALTHY':
      return 'bg-green-500/15 text-green-400 border-green-500/30';
    case 'DEGRADED':
      return 'bg-amber-500/15 text-amber-400 border-amber-500/30';
    case 'UNHEALTHY':
      return 'bg-red-500/15 text-red-400 border-red-500/30';
    default:
      return 'bg-zinc-500/15 text-zinc-400 border-zinc-500/30';
  }
}
