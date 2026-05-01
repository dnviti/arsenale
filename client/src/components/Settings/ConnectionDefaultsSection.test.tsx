import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ConnectionDefaultsSection from './ConnectionDefaultsSection';
import { useNotificationStore } from '../../store/notificationStore';
import { useRdpSettingsStore } from '../../store/rdpSettingsStore';
import { useTerminalSettingsStore } from '../../store/terminalSettingsStore';

const { getProfile, updateSshDefaults, updateRdpDefaults } = vi.hoisted(() => ({
  getProfile: vi.fn(),
  updateSshDefaults: vi.fn(),
  updateRdpDefaults: vi.fn(),
}));

vi.mock('../../api/user.api', () => ({
  getProfile,
  updateSshDefaults,
  updateRdpDefaults,
}));

describe('ConnectionDefaultsSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    useNotificationStore.setState({ notification: null });
    useTerminalSettingsStore.setState({ userDefaults: null, loaded: false, loading: false });
    useRdpSettingsStore.setState({ userDefaults: null, loaded: false, loading: false });

    getProfile.mockResolvedValue({
      sshDefaults: { fontSize: 16 },
      rdpDefaults: { dpi: 144 },
    });
    updateSshDefaults.mockResolvedValue({ sshDefaults: { fontSize: 16 } });
    updateRdpDefaults.mockResolvedValue({ rdpDefaults: { dpi: 144 } });
  });

  it('loads SSH and RDP defaults and saves each tab', async () => {
    render(<ConnectionDefaultsSection />);

    fireEvent.click(await screen.findByRole('button', { name: 'Save SSH Defaults' }, { timeout: 10000 }));
    await waitFor(() => {
      expect(updateSshDefaults).toHaveBeenCalledWith({ fontSize: 16 });
    });

    const rdpTab = screen.getByRole('tab', { name: 'RDP' });
    fireEvent.mouseDown(rdpTab, { button: 0, ctrlKey: false });
    fireEvent.click(rdpTab);
    await waitFor(() => {
      expect(rdpTab).toHaveAttribute('aria-selected', 'true');
    });
    fireEvent.click(await screen.findByRole('button', { name: 'Save RDP Defaults' }, { timeout: 10000 }));
    await waitFor(() => {
      expect(updateRdpDefaults).toHaveBeenCalledWith({ dpi: 144 });
    });
  }, 10000);
});
