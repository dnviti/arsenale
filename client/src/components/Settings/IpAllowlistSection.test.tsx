import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import IpAllowlistSection from './IpAllowlistSection';
import { useAuthStore } from '../../store/authStore';
import { useNotificationStore } from '../../store/notificationStore';

const { getIpAllowlist, updateIpAllowlist } = vi.hoisted(() => ({
  getIpAllowlist: vi.fn(),
  updateIpAllowlist: vi.fn(),
}));

vi.mock('../../api/tenant.api', async () => {
  const actual = await vi.importActual<typeof import('../../api/tenant.api')>('../../api/tenant.api');
  return {
    ...actual,
    getIpAllowlist,
    updateIpAllowlist,
  };
});

describe('IpAllowlistSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useNotificationStore.setState({ notification: null });
    useAuthStore.setState({ user: { tenantId: 'tenant-1' } as never });

    getIpAllowlist.mockResolvedValue({
      enabled: false,
      mode: 'flag',
      entries: [],
    });
    updateIpAllowlist.mockResolvedValue({
      enabled: true,
      mode: 'block',
      entries: ['203.0.113.0/24'],
    });
  });

  it('updates allowlist mode, entries, and test results', async () => {
    render(<IpAllowlistSection />);

    fireEvent.click(await screen.findByRole('switch', { name: 'Enable IP allowlist' }));
    fireEvent.click(screen.getByRole('radio', { name: 'Block unauthorized logins' }));

    fireEvent.change(screen.getByLabelText('Allowlist Entry'), {
      target: { value: '203.0.113.0/24' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));

    fireEvent.change(screen.getByLabelText('Test IP Address'), {
      target: { value: '203.0.113.5' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Check' }));

    expect(await screen.findByText(/would be/i)).toHaveTextContent('allowed');

    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(updateIpAllowlist).toHaveBeenCalledWith('tenant-1', {
        enabled: true,
        mode: 'block',
        entries: ['203.0.113.0/24'],
      });
    });
  });
});
