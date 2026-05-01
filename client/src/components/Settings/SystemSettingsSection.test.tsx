import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import SystemSettingsSection from './SystemSettingsSection';

const {
  getSystemSettings,
  getAdminDbStatus,
  updateSystemSetting,
} = vi.hoisted(() => ({
  getSystemSettings: vi.fn(),
  getAdminDbStatus: vi.fn(),
  updateSystemSetting: vi.fn(),
}));

vi.mock('../../api/systemSettings.api', () => ({
  getSystemSettings,
  getAdminDbStatus,
  updateSystemSetting,
}));

describe('SystemSettingsSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    getSystemSettings.mockResolvedValue({
      groups: [{ key: 'general', label: 'General', order: 1 }],
      settings: [
        {
          key: 'ALLOW_EXTERNAL_SHARING',
          value: false,
          source: 'default',
          envLocked: false,
          canEdit: true,
          type: 'boolean',
          default: false,
          group: 'general',
          label: 'Allow External Sharing',
          description: 'Allow sharing outside the tenant.',
          restartRequired: false,
          sensitive: false,
        },
      ],
    });
    getAdminDbStatus.mockResolvedValue({
      host: 'postgres',
      port: 5432,
      database: 'arsenale',
      connected: true,
      version: '16.2',
    });
    updateSystemSetting.mockResolvedValue({
      key: 'ALLOW_EXTERNAL_SHARING',
      value: true,
      source: 'db',
    });
  });

  it('loads grouped settings and refreshes database status', async () => {
    render(<SystemSettingsSection />);

    expect(await screen.findByText('General')).toBeInTheDocument();
    expect(screen.getByText('1 settings')).toBeInTheDocument();
    expect(screen.getByText('Connected')).toBeInTheDocument();
    fireEvent.click(await screen.findByRole('button', { name: 'Refresh Status' }));

    await waitFor(() => {
      expect(getAdminDbStatus).toHaveBeenCalledTimes(2);
    });
  });
});
