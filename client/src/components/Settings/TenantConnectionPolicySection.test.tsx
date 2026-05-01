import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import TenantConnectionPolicySection from './TenantConnectionPolicySection';
import { useNotificationStore } from '../../store/notificationStore';
import { useTenantStore } from '../../store/tenantStore';

describe('TenantConnectionPolicySection', () => {
  const updateTenant = vi.fn();

  beforeEach(() => {
    vi.resetAllMocks();
    updateTenant.mockResolvedValue(undefined);

    useNotificationStore.setState({ notification: null });
    useTenantStore.setState({
      tenant: {
        id: 'tenant-1',
        enforcedConnectionSettings: {
          ssh: { fontSize: 15 },
          rdp: { dpi: 120 },
        },
      } as never,
      updateTenant,
    });
  });

  it('saves and clears enforced connection policies', async () => {
    render(<TenantConnectionPolicySection />);

    fireEvent.click(screen.getByRole('button', { name: 'Save Policy' }));
    await waitFor(() => {
      expect(updateTenant).toHaveBeenCalledWith({
        enforcedConnectionSettings: {
          ssh: { fontSize: 15 },
          rdp: { dpi: 120 },
        },
      });
    }, { timeout: 5000 });

    const clearButton = screen.getByRole('button', { name: 'Clear All' });
    await waitFor(() => {
      expect(clearButton).not.toBeDisabled();
    }, { timeout: 5000 });

    fireEvent.click(clearButton);
    await waitFor(() => {
      expect(updateTenant).toHaveBeenLastCalledWith({ enforcedConnectionSettings: null });
    }, { timeout: 5000 });
  }, 15000);
});
