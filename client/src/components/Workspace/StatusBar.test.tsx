import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useAuthStore } from '@/store/authStore';
import { useFeatureFlagsStore } from '@/store/featureFlagsStore';
import { useGatewayStore } from '@/store/gatewayStore';
import { useSecretStore } from '@/store/secretStore';
import { useTabsStore } from '@/store/tabsStore';
import { useUiPreferencesStore } from '@/store/uiPreferencesStore';
import { useVaultStore } from '@/store/vaultStore';
import { TooltipProvider } from '@/components/ui/tooltip';
import StatusBar from './StatusBar';

describe('StatusBar', () => {
  beforeEach(() => {
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
        canConnect: true,
        canCreateConnections: true,
        canManageConnections: true,
        canViewCredentials: true,
        canShareConnections: true,
        canViewAuditLog: true,
        canManageSessions: true,
        canManageGateways: true,
        canManageUsers: true,
        canManageSecrets: true,
        canManageTenantSettings: true,
      },
    });
    useFeatureFlagsStore.setState({ loaded: true, keychainEnabled: false });
    useGatewayStore.setState({
      gateways: [
        {
          id: 'gateway-1',
          name: 'Tunnel SSH',
          type: 'MANAGED_SSH',
          host: 'ssh-gateway',
          port: 2222,
          deploymentMode: 'MANAGED_GROUP',
          description: null,
          isDefault: false,
          hasSshKey: true,
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
          healthyInstances: 1,
          runningInstances: 1,
          tunnelEnabled: true,
          tunnelConnected: true,
          tunnelConnectedAt: '2026-04-09T00:00:00Z',
          tunnelClientCertExp: null,
          operationalStatus: 'HEALTHY',
          operationalReason: 'Tunnel is connected and reporting a healthy heartbeat.',
        },
      ],
    });
    useSecretStore.setState({ expiringCount: 0, pwnedCount: 0 });
    useTabsStore.setState({ tabs: [] });
    useUiPreferencesStore.setState({ uiZoomLevel: 100 });
    useVaultStore.setState({ unlocked: false, initialized: false });
  });

  it('opens the infrastructure settings concern when the gateway indicator is clicked', () => {
    const onOpenSettings = vi.fn();

    render(
      <TooltipProvider>
        <StatusBar onOpenSettings={onOpenSettings} />
      </TooltipProvider>,
    );

    fireEvent.click(screen.getByRole('button', { name: /1\/1/i }));

    expect(onOpenSettings).toHaveBeenCalledWith('infrastructure');
  });

  it('keeps the gateway indicator visible while the initial gateway check is loading', () => {
    useGatewayStore.setState({
      gateways: [],
      loading: true,
    });

    render(
      <TooltipProvider>
        <StatusBar />
      </TooltipProvider>,
    );

    expect(screen.getByRole('button', { name: /checking/i })).toBeInTheDocument();
  });
});
