import { fireEvent, waitFor } from '@testing-library/dom';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import NotificationPreferencesSection from './NotificationPreferencesSection';
import { useFeatureFlagsStore } from '../../store/featureFlagsStore';

const {
  getPreferences,
  updatePreference,
  getNotificationSchedule,
  updateNotificationSchedule,
} = vi.hoisted(() => ({
  getPreferences: vi.fn(),
  updatePreference: vi.fn(),
  getNotificationSchedule: vi.fn(),
  updateNotificationSchedule: vi.fn(),
}));

vi.mock('../../api/notifications.api', () => ({
  getPreferences,
  updatePreference,
  getNotificationSchedule,
  updateNotificationSchedule,
}));

describe('NotificationPreferencesSection', () => {
  beforeEach(() => {
    vi.resetAllMocks();

    useFeatureFlagsStore.setState({
      recordingsEnabled: false,
    });

    getPreferences.mockResolvedValue([
      { type: 'CONNECTION_SHARED', inApp: true, email: false },
      { type: 'SECRET_EXPIRING', inApp: true, email: true },
    ]);
    getNotificationSchedule.mockResolvedValue({
      dndEnabled: false,
      quietHoursStart: '22:00',
      quietHoursEnd: '07:00',
      quietHoursTimezone: 'UTC',
    });
    updatePreference.mockImplementation(async (type, update) => ({
      type,
      inApp: update.inApp ?? true,
      email: update.email ?? false,
    }));
    updateNotificationSchedule.mockResolvedValue({
      dndEnabled: true,
      quietHoursStart: '22:00',
      quietHoursEnd: '07:00',
      quietHoursTimezone: 'UTC',
    });
  });

  it('loads categories and omits recording preferences when recordings are disabled', async () => {
    render(<NotificationPreferencesSection />);

    expect(
      await screen.findByText('Connection Shared With You'),
    ).toBeInTheDocument();
    expect(screen.getByText('Quiet Hours')).toBeInTheDocument();
    expect(screen.queryByText('Session Recording Ready')).not.toBeInTheDocument();
  });

  it('updates a notification channel toggle', async () => {
    render(<NotificationPreferencesSection />);

    const emailSwitch = await screen.findByRole('switch', {
      name: 'Connection Shared With You email notifications',
    });
    fireEvent.click(emailSwitch);

    await waitFor(() => {
      expect(updatePreference).toHaveBeenCalledWith('CONNECTION_SHARED', {
        email: true,
      });
    });
  });
});
