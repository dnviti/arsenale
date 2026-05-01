import { describe, expect, it } from 'vitest';
import { summarizeGatewayStatuses } from './gatewayStatus';
import type { GatewayData } from '../api/gateway.api';

function gateway(status: GatewayData['operationalStatus']): GatewayData {
  return {
    id: `gateway-${status}`,
    name: `Gateway ${status}`,
    type: 'MANAGED_SSH',
    host: 'gateway.internal',
    port: 2222,
    deploymentMode: 'MANAGED_GROUP',
    description: null,
    isDefault: false,
    hasSshKey: false,
    apiPort: 9022,
    inactivityTimeoutSeconds: 3600,
    tenantId: 'tenant-1',
    createdById: 'user-1',
    createdAt: '2026-04-09T00:00:00Z',
    updatedAt: '2026-04-09T00:00:00Z',
    monitoringEnabled: true,
    monitorIntervalMs: 5000,
    lastHealthStatus: 'UNKNOWN',
    lastCheckedAt: null,
    lastLatencyMs: null,
    lastError: null,
    isManaged: true,
    publishPorts: false,
    lbStrategy: 'ROUND_ROBIN',
    desiredReplicas: 1,
    autoScale: false,
    minReplicas: 1,
    maxReplicas: 1,
    sessionsPerInstance: 10,
    scaleDownCooldownSeconds: 300,
    lastScaleAction: null,
    templateId: null,
    totalInstances: 1,
    healthyInstances: status === 'HEALTHY' ? 1 : 0,
    runningInstances: 1,
    tunnelEnabled: false,
    tunnelConnected: false,
    tunnelConnectedAt: null,
    tunnelClientCertExp: null,
    operationalStatus: status,
    operationalReason: `${status} reason`,
  };
}

describe('summarizeGatewayStatuses', () => {
  it('counts gateways by operational status', () => {
    const summary = summarizeGatewayStatuses([
      gateway('HEALTHY'),
      gateway('DEGRADED'),
      gateway('UNHEALTHY'),
      gateway('UNKNOWN'),
    ]);

    expect(summary).toEqual({
      total: 4,
      healthy: 1,
      degraded: 1,
      unhealthy: 1,
      unknown: 1,
    });
  });
});
