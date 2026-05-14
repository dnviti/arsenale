import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import GatewayDialog from './GatewayDialog';
import type { GatewayData, TunnelTokenResponse } from '../../api/gateway.api';
import { useGatewayStore } from '../../store/gatewayStore';
import { useUiPreferencesStore } from '../../store/uiPreferencesStore';

const baseGateway: GatewayData = {
  id: 'gateway-1',
  name: 'Remote GUACD',
  type: 'GUACD',
  host: 'remote-guacd.local',
  port: 4822,
  deploymentMode: 'SINGLE_INSTANCE',
  description: null,
  isDefault: false,
  hasSshKey: false,
  apiPort: null,
  inactivityTimeoutSeconds: 3600,
  tenantId: 'tenant-1',
  createdById: 'user-1',
  createdAt: '2026-05-14T00:00:00.000Z',
  updatedAt: '2026-05-14T00:00:00.000Z',
  monitoringEnabled: true,
  monitorIntervalMs: 5000,
  lastHealthStatus: 'UNKNOWN',
  lastCheckedAt: null,
  lastLatencyMs: null,
  lastError: null,
  isManaged: false,
  publishPorts: false,
  lbStrategy: 'ROUND_ROBIN',
  desiredReplicas: 0,
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
  operationalReason: 'No tunnel configured.',
};

const tunnelBundle: TunnelTokenResponse = {
  token: 'tok-secret',
  tunnelEnabled: true,
  tunnelConnected: false,
  gatewayId: 'gateway-1',
  gatewayType: 'GUACD',
  tunnelLocalHost: '127.0.0.1',
  tunnelLocalPort: 4822,
  tunnelClientCert: '-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----',
  tunnelClientKey: '-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----',
  tunnelClientCertExp: '2026-08-14T00:00:00.000Z',
};

describe('GatewayDialog', () => {
  beforeEach(() => {
    useGatewayStore.setState({
      gateways: [],
      createGateway: vi.fn().mockResolvedValue(baseGateway),
      generateTunnelToken: vi.fn().mockResolvedValue(tunnelBundle),
    });
    useUiPreferencesStore.setState({
      tunnelSectionOpen: true,
      tunnelEventLogOpen: false,
      tunnelMetricsOpen: false,
    });
  });

  it('uses viewport-relative sizing for the gateway editor popup', () => {
    render(<GatewayDialog open onClose={vi.fn()} gateway={null} />);

    const dialog = screen.getByRole('dialog', { name: /new gateway/i });

    expect(dialog).toHaveClass('w-[calc(100vw-1rem)]');
    expect(dialog).toHaveClass('sm:w-[90vw]');
    expect(dialog).toHaveClass('sm:max-w-[90vw]');
    expect(dialog).toHaveClass('overflow-hidden');
  });

  it('keeps header and footer outside the scrollable gateway form body', () => {
    render(<GatewayDialog open onClose={vi.fn()} gateway={null} />);

    const dialog = screen.getByRole('dialog', { name: /new gateway/i });
    const title = screen.getByText('New Gateway');
    const saveButton = screen.getByRole('button', { name: 'Create' });
    const scrollBody = dialog.querySelector('.min-h-0');

    expect(scrollBody).toHaveClass('flex-1');
    expect(scrollBody).toHaveClass('overflow-y-auto');
    expect(scrollBody).toHaveClass('overflow-x-hidden');
    expect(scrollBody).not.toContainElement(title);
    expect(scrollBody).not.toContainElement(saveButton);
    expect(dialog).toContainElement(title);
    expect(dialog).toContainElement(saveButton);
  });

  it('lets operators enable zero-trust tunneling while creating a gateway', async () => {
    const user = userEvent.setup();

    render(<GatewayDialog open onClose={vi.fn()} gateway={null} />);

    await user.click(screen.getByRole('button', { name: 'Enable Zero-Trust Tunnel' }));

    expect(screen.getByText(/Tunnel will be enabled when the gateway is created/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create and Enable Tunnel' })).toBeInTheDocument();
  });

  it('creates a gateway, enables the tunnel, and shows remote install commands', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const createGateway = vi.fn().mockResolvedValue(baseGateway);
    const generateTunnelToken = vi.fn().mockResolvedValue(tunnelBundle);
    useGatewayStore.setState({ createGateway, generateTunnelToken });

    render(<GatewayDialog open onClose={onClose} gateway={null} />);

    await user.type(screen.getByLabelText('Name'), 'Remote GUACD');
    await user.type(screen.getByLabelText('Host'), 'remote-guacd.local');
    await user.type(screen.getByLabelText('Port'), '4822');
    await user.click(screen.getByRole('button', { name: 'Enable Zero-Trust Tunnel' }));
    await user.click(screen.getByRole('button', { name: 'Create and Enable Tunnel' }));

    await waitFor(() => expect(createGateway).toHaveBeenCalled());
    expect(generateTunnelToken).toHaveBeenCalledWith('gateway-1');
    expect(onClose).not.toHaveBeenCalled();
    expect(await screen.findByText(/Copy these values now/i)).toBeInTheDocument();
    expect(screen.getByDisplayValue(/TUNNEL_TOKEN="tok-secret"/i)).toBeInTheDocument();
    expect(screen.getByDisplayValue(/docker compose --env-file tunnel.env up -d/i)).toBeInTheDocument();
    expect(screen.getByDisplayValue(/-----BEGIN PRIVATE KEY-----/i)).toBeInTheDocument();
  });

  it('keeps the created gateway open when tunnel activation fails after create', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const createGateway = vi.fn().mockResolvedValue(baseGateway);
    const generateTunnelToken = vi.fn().mockRejectedValue(new Error('token unavailable'));
    useGatewayStore.setState({ createGateway, generateTunnelToken });

    render(<GatewayDialog open onClose={onClose} gateway={null} />);

    await user.type(screen.getByLabelText('Name'), 'Remote GUACD');
    await user.type(screen.getByLabelText('Host'), 'remote-guacd.local');
    await user.type(screen.getByLabelText('Port'), '4822');
    await user.click(screen.getByRole('button', { name: 'Enable Zero-Trust Tunnel' }));
    await user.click(screen.getByRole('button', { name: 'Create and Enable Tunnel' }));

    await waitFor(() => expect(createGateway).toHaveBeenCalledTimes(1));
    expect(generateTunnelToken).toHaveBeenCalledWith('gateway-1');
    expect(onClose).not.toHaveBeenCalled();
    expect(await screen.findByRole('dialog', { name: /gateway created/i })).toBeInTheDocument();
    expect(screen.getByText(/Gateway was created, but tunnel activation failed/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Enable Zero-Trust Tunnel' })).toBeInTheDocument();
  });
});
