import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthStore } from '../../store/authStore';
import { useGatewayStore } from '../../store/gatewayStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';
import GatewaySection from './GatewaySection';

vi.mock('../gateway/GatewayDialog', () => ({
  default: () => <div data-testid="gateway-dialog" />,
}));

vi.mock('../gateway/GatewayTemplateSection', () => ({
  default: () => <div>Gateway templates</div>,
}));

vi.mock('../orchestration/SessionDashboard', () => ({
  default: () => <div>Session dashboard</div>,
}));

vi.mock('../orchestration/ScalingControls', () => ({
  default: () => <div>Scaling controls</div>,
}));

vi.mock('../orchestration/GatewayInstanceList', () => ({
  default: () => <div>Gateway instances</div>,
}));

describe('GatewaySection', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    useUiPreferencesStore.setState({
      gatewayActiveSubTab: 'gateways',
    });
  });

  it('shows an organization setup call to action when the user has no tenant', () => {
    useAuthStore.setState({
      user: {
        id: 'user-1',
        email: 'user@example.com',
        username: 'User',
        avatarData: null,
        tenantId: undefined,
        tenantRole: undefined,
      },
      permissionsLoaded: true,
      permissions: {
        ...useAuthStore.getState().permissions,
        canManageGateways: true,
        canManageSessions: true,
      },
    });

    render(<GatewaySection />);

    expect(screen.getByText('Gateway access')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Set Up Organization' })).toBeInTheDocument();
  });

  it('renders the shadcn gateway inventory and key management panels', () => {
    useAuthStore.setState({
      user: {
        id: 'user-1',
        email: 'admin@example.com',
        username: 'Admin',
        avatarData: null,
        tenantId: 'tenant-1',
        tenantRole: 'OWNER',
      },
      permissionsLoaded: true,
      permissions: {
        ...useAuthStore.getState().permissions,
        canManageGateways: true,
        canManageSessions: true,
      },
    });

    useGatewayStore.setState({
      gateways: [
        {
          id: 'gateway-1',
          name: 'Tunnel SSH',
          type: 'MANAGED_SSH',
          host: 'ssh-gateway',
          port: 2222,
          deploymentMode: 'MANAGED_GROUP',
          description: 'Primary managed SSH route.',
          isDefault: true,
          hasSshKey: true,
          apiPort: 9022,
          inactivityTimeoutSeconds: 3600,
          tenantId: 'tenant-1',
          createdById: 'user-1',
          createdAt: '2026-04-08T00:00:00.000Z',
          updatedAt: '2026-04-08T00:00:00.000Z',
          monitoringEnabled: true,
          monitorIntervalMs: 5000,
          lastHealthStatus: 'REACHABLE',
          lastCheckedAt: '2026-04-08T00:00:00.000Z',
          lastLatencyMs: 24,
          lastError: null,
          isManaged: true,
          publishPorts: false,
          lbStrategy: 'ROUND_ROBIN',
          desiredReplicas: 1,
          autoScale: false,
          minReplicas: 1,
          maxReplicas: 3,
          sessionsPerInstance: 10,
          scaleDownCooldownSeconds: 300,
          lastScaleAction: null,
          templateId: null,
          totalInstances: 1,
          healthyInstances: 1,
          runningInstances: 1,
          tunnelEnabled: true,
          tunnelConnected: true,
          tunnelConnectedAt: '2026-04-08T00:00:00.000Z',
          tunnelClientCertExp: null,
          operationalStatus: 'HEALTHY',
          operationalReason: 'Tunnel is connected and reporting a healthy heartbeat.',
        },
      ],
      loading: false,
      sshKeyPair: {
        id: 'key-1',
        publicKey: 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGatewayKey',
        fingerprint: 'SHA256:test',
        algorithm: 'ed25519',
        createdAt: '2026-04-08T00:00:00.000Z',
        updatedAt: '2026-04-08T00:00:00.000Z',
      },
      sshKeyLoading: false,
      tunnelStatuses: {
        'gateway-1': {
          gatewayId: 'gateway-1',
          connected: true,
          connectedAt: '2026-04-08T00:00:00.000Z',
          rttMs: 19,
          activeStreams: 2,
          agentVersion: '1.0.0',
          checkedAt: '2026-04-08T00:00:00.000Z',
        },
      },
      fetchGateways: vi.fn().mockResolvedValue(undefined),
      createGateway: vi.fn().mockResolvedValue(undefined),
      updateGateway: vi.fn().mockResolvedValue(undefined),
      deleteGateway: vi.fn().mockResolvedValue(undefined),
      applyHealthUpdate: vi.fn(),
      applyInstancesUpdate: vi.fn(),
      applyScalingUpdate: vi.fn(),
      applyGatewayUpdate: vi.fn(),
      applyGatewayStreamSnapshot: vi.fn(),
      applyActiveSessionStreamSnapshot: vi.fn(),
      fetchSshKeyPair: vi.fn().mockResolvedValue(undefined),
      generateSshKeyPair: vi.fn().mockResolvedValue(undefined),
      rotateSshKeyPair: vi.fn().mockResolvedValue({}),
      pushKeyToGateway: vi.fn().mockResolvedValue({ ok: true }),
      activeSessions: [],
      sessionCount: 0,
      sessionCountByGateway: [],
      scalingStatus: {},
      instances: {},
      sessionsLoading: false,
      fetchActiveSessions: vi.fn().mockResolvedValue(undefined),
      fetchSessionCount: vi.fn().mockResolvedValue(undefined),
      fetchSessionCountByGateway: vi.fn().mockResolvedValue(undefined),
      terminateSession: vi.fn().mockResolvedValue(undefined),
      fetchScalingStatus: vi.fn().mockResolvedValue(undefined),
      fetchInstances: vi.fn().mockResolvedValue(undefined),
      watchScalingStatus: vi.fn(),
      unwatchScalingStatus: vi.fn(),
      watchInstances: vi.fn(),
      unwatchInstances: vi.fn(),
      deployGateway: vi.fn().mockResolvedValue(undefined),
      undeployGateway: vi.fn().mockResolvedValue(undefined),
      scaleGateway: vi.fn().mockResolvedValue(undefined),
      updateScalingConfig: vi.fn().mockResolvedValue(undefined),
      restartInstance: vi.fn().mockResolvedValue(undefined),
      templates: [],
      templatesLoading: false,
      fetchTemplates: vi.fn().mockResolvedValue(undefined),
      createTemplate: vi.fn().mockResolvedValue(undefined),
      updateTemplate: vi.fn().mockResolvedValue(undefined),
      deleteTemplate: vi.fn().mockResolvedValue(undefined),
      deployFromTemplate: vi.fn().mockResolvedValue(undefined),
      generateTunnelToken: vi.fn().mockResolvedValue(undefined),
      revokeTunnelToken: vi.fn().mockResolvedValue(undefined),
      applyTunnelStatusUpdate: vi.fn(),
      tunnelOverview: null,
      tunnelOverviewLoading: false,
      fetchTunnelOverview: vi.fn().mockResolvedValue(undefined),
      watchedScalingGatewayIds: {},
      watchedInstanceGatewayIds: {},
      reset: vi.fn(),
    });

    render(<GatewaySection />);

    expect(screen.getByText('SSH Key Pair')).toBeInTheDocument();
    expect(screen.getByText('Gateway Inventory')).toBeInTheDocument();
    expect(screen.getByText('Tunnel SSH')).toBeInTheDocument();
    expect(screen.getByText('Tunnel healthy')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Push Key' })).toBeInTheDocument();
  });
});
