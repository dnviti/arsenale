import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import TunnelConfigSection from './TunnelConfigSection';
import { useAuthStore } from '../../store/authStore';
import { useGatewayStore } from '../../store/gatewayStore';
import { useNotificationStore } from '../../store/notificationStore';
import { useTenantStore } from '../../store/tenantStore';

describe('TunnelConfigSection', () => {
  const updateTenant = vi.fn();
  const fetchTenant = vi.fn();
  const fetchTunnelOverview = vi.fn();

  beforeEach(() => {
    vi.resetAllMocks();
    updateTenant.mockResolvedValue(undefined);
    fetchTenant.mockResolvedValue(undefined);
    fetchTunnelOverview.mockResolvedValue(undefined);

    useNotificationStore.setState({ notification: null });
    useAuthStore.setState({ user: { tenantId: 'tenant-1' } as never });
    useTenantStore.setState({
      tenant: {
        id: 'tenant-1',
        tunnelDefaultEnabled: true,
        tunnelRequireForRemote: false,
        tunnelAutoTokenRotation: true,
        tunnelTokenRotationDays: 90,
        tunnelTokenMaxLifetimeDays: null,
        tunnelAgentAllowedCidrs: ['10.0.0.0/8'],
      } as never,
      updateTenant,
      fetchTenant,
    });
    useGatewayStore.setState({
      tunnelOverview: { total: 3, connected: 2, disconnected: 1, avgRttMs: 45 },
      tunnelOverviewLoading: false,
      fetchTunnelOverview,
    });
  });

  it('updates tunnel settings and preserves shared CIDR validation', async () => {
    render(<TunnelConfigSection />);

    fireEvent.click(screen.getByRole('switch', { name: 'Require tunnels for remote gateways' }));
    fireEvent.change(screen.getByLabelText('Allowed Agent Network'), {
      target: { value: '203.0.113.0/24' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(updateTenant).toHaveBeenCalledWith({
        tunnelDefaultEnabled: true,
        tunnelRequireForRemote: true,
        tunnelAutoTokenRotation: true,
        tunnelTokenRotationDays: 90,
        tunnelTokenMaxLifetimeDays: null,
        tunnelAgentAllowedCidrs: ['10.0.0.0/8', '203.0.113.0/24'],
      });
    });

    expect(fetchTenant).toHaveBeenCalled();
  });
});
