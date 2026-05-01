import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import RdGatewayConfigSection from './RdGatewayConfigSection';
import { useAuthStore } from '../../store/authStore';
import { useNotificationStore } from '../../store/notificationStore';

const {
  getRdGatewayConfig,
  getRdGatewayStatus,
  updateRdGatewayConfig,
} = vi.hoisted(() => ({
  getRdGatewayConfig: vi.fn(),
  getRdGatewayStatus: vi.fn(),
  updateRdGatewayConfig: vi.fn(),
}));

vi.mock('../../api/rdGateway.api', () => ({
  getRdGatewayConfig,
  getRdGatewayStatus,
  updateRdGatewayConfig,
}));

describe('RdGatewayConfigSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useNotificationStore.setState({ notification: null });
    useAuthStore.setState({
      isAuthenticated: true,
      user: {
        id: 'user-1',
        email: 'admin@example.com',
        username: 'admin',
        avatarData: null,
        tenantId: 'tenant-1',
        tenantRole: 'ADMIN',
      },
    });

    getRdGatewayConfig.mockResolvedValue({
      enabled: true,
      externalHostname: 'rdgw.example.com',
      port: 443,
      idleTimeoutSeconds: 3600,
    });
    getRdGatewayStatus.mockResolvedValue({
      activeTunnels: 2,
      activeChannels: 5,
    });
    updateRdGatewayConfig.mockResolvedValue({
      enabled: true,
      externalHostname: 'gateway.example.com',
      port: 443,
      idleTimeoutSeconds: 3600,
    });
  });

  it('loads the configuration and saves edited values', async () => {
    render(<RdGatewayConfigSection />);

    const hostnameInput = await screen.findByLabelText('External Hostname');
    fireEvent.change(hostnameInput, { target: { value: 'gateway.example.com' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(updateRdGatewayConfig).toHaveBeenCalledWith({
        enabled: true,
        externalHostname: 'gateway.example.com',
        port: 443,
        idleTimeoutSeconds: 3600,
      });
    });

    expect(useNotificationStore.getState().notification).toMatchObject({
      message: 'RD Gateway configuration saved',
      severity: 'success',
    });
  });
});
