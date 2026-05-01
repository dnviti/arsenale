import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SettingValue } from '../../api/systemSettings.api';
import SettingField from './SettingField';

const { updateSystemSetting } = vi.hoisted(() => ({
  updateSystemSetting: vi.fn(),
}));

vi.mock('../../api/systemSettings.api', () => ({
  updateSystemSetting,
}));

function buildSetting(overrides: Partial<SettingValue>): SettingValue {
  return {
    key: 'TEST_SETTING',
    value: '',
    source: 'default',
    envLocked: false,
    canEdit: true,
    type: 'string',
    default: '',
    group: 'general',
    label: 'Test Setting',
    description: 'Test description',
    restartRequired: false,
    sensitive: false,
    ...overrides,
  };
}

describe('SettingField', () => {
  beforeEach(() => {
    vi.resetAllMocks();
    updateSystemSetting.mockResolvedValue({ key: 'TEST_SETTING', value: 'value', source: 'db' });
  });

  it('auto-saves boolean values when toggled', async () => {
    const onUpdated = vi.fn();

    render(
      <SettingField
        setting={buildSetting({
          key: 'BOOL_SETTING',
          label: 'Boolean Setting',
          type: 'boolean',
          value: true,
        })}
        onUpdated={onUpdated}
      />,
    );

    fireEvent.click(screen.getByRole('switch', { name: 'Boolean Setting' }));

    await waitFor(() => {
      expect(updateSystemSetting).toHaveBeenCalledWith('BOOL_SETTING', false);
    });
    expect(onUpdated).toHaveBeenCalledWith('BOOL_SETTING', false);
  });

  it('saves string array selections as a comma-separated value', async () => {
    const onUpdated = vi.fn();

    render(
      <SettingField
        setting={buildSetting({
          key: 'GATEWAY_REQUIRED_TYPES',
          label: 'Required Gateway Types',
          type: 'string[]',
          value: 'MANAGED_SSH',
          options: ['MANAGED_SSH', 'GUACD', 'DB_PROXY'],
        })}
        onUpdated={onUpdated}
      />,
    );

    fireEvent.click(screen.getByText('GUACD'));
    fireEvent.click(screen.getByRole('button', { name: 'Save Selection' }));

    await waitFor(() => {
      expect(updateSystemSetting).toHaveBeenCalledWith(
        'GATEWAY_REQUIRED_TYPES',
        'MANAGED_SSH,GUACD',
      );
    });
    expect(onUpdated).toHaveBeenCalledWith('GATEWAY_REQUIRED_TYPES', 'MANAGED_SSH,GUACD');
  });

  it('redacts sensitive values after save', async () => {
    const onUpdated = vi.fn();

    render(
      <SettingField
        setting={buildSetting({
          key: 'CLIENT_SECRET',
          label: 'Client Secret',
          sensitive: true,
          value: '[REDACTED]',
        })}
        onUpdated={onUpdated}
      />,
    );

    fireEvent.change(screen.getByLabelText('Client Secret'), {
      target: { value: 'super-secret-value' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(updateSystemSetting).toHaveBeenCalledWith('CLIENT_SECRET', 'super-secret-value');
    });
    expect(onUpdated).toHaveBeenCalledWith('CLIENT_SECRET', '[REDACTED]');
  });
});
