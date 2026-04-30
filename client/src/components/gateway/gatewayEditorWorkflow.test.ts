import { describe, expect, it } from 'vitest';
import type { GatewayData } from '../../api/gateway.api';
import {
  buildGatewayCreateInput,
  buildGatewayUpdate,
  defaultGatewayEditorForm,
  gatewayToEditorForm,
  nextGatewayTypeForm,
  nextPublishPortsForm,
  validateGatewayEditorForm,
} from './gatewayEditorWorkflow';

describe('gatewayEditorWorkflow', () => {
  it('locks SSH bastions to single-instance mode and applies the default port', () => {
    const form = {
      ...defaultGatewayEditorForm(),
      deploymentMode: 'MANAGED_GROUP' as const,
      port: '4822',
      apiPort: '9022',
    };

    expect(nextGatewayTypeForm(form, 'SSH_BASTION')).toMatchObject({
      type: 'SSH_BASTION',
      deploymentMode: 'SINGLE_INSTANCE',
      port: '22',
      apiPort: '',
    });
  });

  it('does not require host for managed groups', () => {
    const form = {
      ...defaultGatewayEditorForm(),
      name: 'Managed GUACD',
      deploymentMode: 'MANAGED_GROUP' as const,
      port: '4822',
    };

    expect(validateGatewayEditorForm(form)).toBeNull();
    expect(buildGatewayCreateInput(form)).toMatchObject({
      name: 'Managed GUACD',
      host: '',
      port: 4822,
      deploymentMode: 'MANAGED_GROUP',
      lbStrategy: 'ROUND_ROBIN',
    });
  });

  it('requires host for single-instance gateways', () => {
    const form = {
      ...defaultGatewayEditorForm(),
      name: 'Standalone',
      deploymentMode: 'SINGLE_INSTANCE' as const,
      port: '4822',
    };

    expect(validateGatewayEditorForm(form)).toBe('Host is required');
  });

  it('uses gateway data as the edit form source of truth', () => {
    const gateway = makeGateway({
      type: 'DB_PROXY',
      deploymentMode: 'MANAGED_GROUP',
      port: 5432,
      autoScale: true,
      minReplicas: 2,
      publishPorts: true,
      lbStrategy: 'LEAST_CONNECTIONS',
    });

    expect(gatewayToEditorForm(gateway)).toMatchObject({
      type: 'DB_PROXY',
      deploymentMode: 'MANAGED_GROUP',
      port: '5432',
      autoScaleEnabled: true,
      minReplicasVal: '2',
      publishPorts: true,
      lbStrategy: 'LEAST_CONNECTIONS',
    });
  });

  it('builds sparse gateway updates', () => {
    const gateway = makeGateway({
      name: 'Old',
      host: 'old.example.com',
      port: 2222,
      type: 'MANAGED_SSH',
      apiPort: 9022,
    });
    const form = {
      ...gatewayToEditorForm(gateway),
      name: 'New',
      host: 'new.example.com',
      apiPort: '9122',
      monitorIntervalMs: '5000',
    };

    expect(buildGatewayUpdate(gateway, form)).toEqual({
      name: 'New',
      host: 'new.example.com',
      apiPort: 9122,
    });
  });

  it('resets service ports when publishing managed instances', () => {
    const form = {
      ...defaultGatewayEditorForm(),
      type: 'DB_PROXY' as const,
      deploymentMode: 'MANAGED_GROUP' as const,
      port: '15432',
    };

    expect(nextPublishPortsForm(form, true)).toMatchObject({
      publishPorts: true,
      port: '5432',
    });
  });
});

function makeGateway(overrides: Partial<GatewayData> = {}): GatewayData {
  return {
    id: 'gw-1',
    name: 'Gateway',
    type: 'GUACD',
    host: 'gateway.local',
    port: 4822,
    deploymentMode: 'SINGLE_INSTANCE',
    description: null,
    isDefault: false,
    hasSshKey: false,
    apiPort: null,
    inactivityTimeoutSeconds: 3600,
    tenantId: 'tenant-1',
    createdById: 'user-1',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    monitoringEnabled: true,
    monitorIntervalMs: 5000,
    lastHealthStatus: 'UNKNOWN',
    lastCheckedAt: null,
    lastLatencyMs: null,
    lastError: null,
    isManaged: false,
    publishPorts: false,
    lbStrategy: 'ROUND_ROBIN',
    desiredReplicas: 1,
    autoScale: false,
    minReplicas: 0,
    maxReplicas: 5,
    sessionsPerInstance: 10,
    scaleDownCooldownSeconds: 300,
    lastScaleAction: null,
    templateId: null,
    totalInstances: 0,
    healthyInstances: 0,
    runningInstances: 0,
    tunnelEnabled: false,
    tunnelConnected: false,
    tunnelConnectedAt: null,
    tunnelClientCertExp: null,
    operationalStatus: 'UNKNOWN',
    operationalReason: '',
    ...overrides,
  };
}
