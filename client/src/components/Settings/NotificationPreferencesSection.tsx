import { useCallback, useEffect, useMemo, useState } from 'react';
import { BellRing, Mail, MoonStar } from 'lucide-react';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Input } from '@/components/ui/input';
import { Switch } from '@/components/ui/switch';
import {
  SettingsFieldGroup,
  SettingsPanel,
  SettingsSwitchRow,
} from './settings-ui';
import {
  getNotificationSchedule,
  getPreferences,
  updateNotificationSchedule,
  updatePreference,
  type NotificationPreference,
  type NotificationSchedule,
  type NotificationType,
} from '../../api/notifications.api';
import { useFeatureFlagsStore } from '../../store/featureFlagsStore';
import { extractApiError } from '../../utils/apiError';

interface NotificationCategory {
  label: string;
  types: NotificationType[];
}

const CATEGORIES: NotificationCategory[] = [
  {
    label: 'Sharing',
    types: ['CONNECTION_SHARED', 'SHARE_PERMISSION_UPDATED', 'SHARE_REVOKED'],
  },
  {
    label: 'Secrets',
    types: ['SECRET_SHARED', 'SECRET_SHARE_REVOKED', 'SECRET_EXPIRING', 'SECRET_EXPIRED'],
  },
  {
    label: 'Security',
    types: ['IMPOSSIBLE_TRAVEL_DETECTED', 'LATERAL_MOVEMENT_ALERT'],
  },
  {
    label: 'Organization',
    types: ['TENANT_INVITATION'],
  },
  {
    label: 'Sessions',
    types: ['RECORDING_READY'],
  },
];

const TYPE_LABELS: Record<NotificationType, string> = {
  CONNECTION_SHARED: 'Connection Shared With You',
  SHARE_PERMISSION_UPDATED: 'Share Permission Updated',
  SHARE_REVOKED: 'Share Revoked',
  SECRET_SHARED: 'Secret Shared With You',
  SECRET_SHARE_REVOKED: 'Secret Share Revoked',
  SECRET_EXPIRING: 'Secret Expiring Soon',
  SECRET_EXPIRED: 'Secret Expired',
  IMPOSSIBLE_TRAVEL_DETECTED: 'Impossible Travel Detected',
  LATERAL_MOVEMENT_ALERT: 'Lateral Movement Anomaly',
  TENANT_INVITATION: 'Organization Invitation',
  RECORDING_READY: 'Session Recording Ready',
};

function getTimezoneOptions(): string[] {
  try {
    return (Intl as unknown as { supportedValuesOf(key: string): string[] })
      .supportedValuesOf('timeZone');
  } catch {
    return [
      'UTC',
      'America/New_York',
      'America/Chicago',
      'America/Denver',
      'America/Los_Angeles',
      'America/Sao_Paulo',
      'Europe/London',
      'Europe/Paris',
      'Europe/Berlin',
      'Europe/Rome',
      'Europe/Moscow',
      'Asia/Dubai',
      'Asia/Kolkata',
      'Asia/Shanghai',
      'Asia/Tokyo',
      'Australia/Sydney',
      'Pacific/Auckland',
    ];
  }
}

const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

function PreferenceRow({
  label,
  inApp,
  email,
  disabled,
  onInAppChange,
  onEmailChange,
}: {
  label: string;
  inApp: boolean;
  email: boolean;
  disabled: boolean;
  onInAppChange: (checked: boolean) => void;
  onEmailChange: (checked: boolean) => void;
}) {
  return (
    <div className="grid gap-3 rounded-xl border border-border/70 bg-background/60 px-4 py-3 md:grid-cols-[minmax(0,1fr)_140px_140px] md:items-center">
      <div className="text-sm font-medium text-foreground">{label}</div>
      <label className="flex items-center justify-between gap-3 rounded-lg bg-background px-3 py-2 md:justify-center md:bg-transparent md:px-0 md:py-0">
        <span className="text-xs uppercase tracking-[0.2em] text-muted-foreground md:hidden">
          In-App
        </span>
        <Switch
          checked={inApp}
          disabled={disabled}
          onCheckedChange={onInAppChange}
          aria-label={`${label} in-app notifications`}
        />
      </label>
      <label className="flex items-center justify-between gap-3 rounded-lg bg-background px-3 py-2 md:justify-center md:bg-transparent md:px-0 md:py-0">
        <span className="text-xs uppercase tracking-[0.2em] text-muted-foreground md:hidden">
          Email
        </span>
        <Switch
          checked={email}
          disabled={disabled}
          onCheckedChange={onEmailChange}
          aria-label={`${label} email notifications`}
        />
      </label>
    </div>
  );
}

export default function NotificationPreferencesSection() {
  const recordingsEnabled = useFeatureFlagsStore((state) => state.recordingsEnabled);
  const [prefs, setPrefs] = useState<Map<NotificationType, NotificationPreference>>(new Map());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState<Set<NotificationType>>(new Set());
  const [schedule, setSchedule] = useState<NotificationSchedule>({
    dndEnabled: false,
    quietHoursStart: null,
    quietHoursEnd: null,
    quietHoursTimezone: null,
  });
  const [scheduleSaving, setScheduleSaving] = useState(false);

  const timezoneOptions = useMemo(() => getTimezoneOptions(), []);
  const categories = useMemo(
    () =>
      CATEGORIES.filter(
        (category) =>
          recordingsEnabled || !category.types.includes('RECORDING_READY'),
      ),
    [recordingsEnabled],
  );

  useEffect(() => {
    Promise.all([getPreferences(), getNotificationSchedule()])
      .then(([list, notificationSchedule]) => {
        setPrefs(new Map(list.map((preference) => [preference.type, preference])));
        setSchedule(notificationSchedule);
      })
      .catch((err) => {
        setError(extractApiError(err, 'Failed to load preferences'));
      })
      .finally(() => setLoading(false));
  }, []);

  const handleToggle = useCallback(
    async (type: NotificationType, channel: 'inApp' | 'email', value: boolean) => {
      setPrefs((previous) => {
        const next = new Map(previous);
        const current = next.get(type);
        if (current) {
          next.set(type, { ...current, [channel]: value });
        }
        return next;
      });
      setSaving((previous) => new Set([...previous, type]));

      try {
        const updated = await updatePreference(type, { [channel]: value });
        setPrefs((previous) => {
          const next = new Map(previous);
          next.set(type, updated);
          return next;
        });
      } catch (err) {
        setPrefs((previous) => {
          const next = new Map(previous);
          const current = next.get(type);
          if (current) {
            next.set(type, { ...current, [channel]: !value });
          }
          return next;
        });
        setError(extractApiError(err, 'Failed to update preference'));
      } finally {
        setSaving((previous) => {
          const next = new Set(previous);
          next.delete(type);
          return next;
        });
      }
    },
    [],
  );

  const handleScheduleChange = useCallback(
    async (update: Partial<NotificationSchedule>) => {
      const previous = { ...schedule };
      setSchedule((current) => ({ ...current, ...update }));
      setScheduleSaving(true);
      try {
        const updated = await updateNotificationSchedule(update);
        setSchedule(updated);
      } catch (err) {
        setSchedule(previous);
        setError(extractApiError(err, 'Failed to update notification schedule'));
      } finally {
        setScheduleSaving(false);
      }
    },
    [schedule],
  );

  if (loading) {
    return (
      <SettingsPanel
        title="Notification Preferences"
        description="Choose which events reach you in the app or by email."
      >
        <p className="text-sm text-muted-foreground">Loading notification preferences...</p>
      </SettingsPanel>
    );
  }

  return (
    <div className="space-y-4">
      <SettingsPanel
        title="Notification Preferences"
        description="Choose which events reach you in the app or by email."
      >
        <SettingsFieldGroup>
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {categories.map((category) => (
            <section key={category.label} className="space-y-3">
              <div className="flex items-center justify-between gap-3">
                <h4 className="text-sm font-semibold uppercase tracking-[0.2em] text-muted-foreground">
                  {category.label}
                </h4>
                <div className="hidden items-center gap-6 text-xs uppercase tracking-[0.2em] text-muted-foreground md:flex">
                  <span className="inline-flex items-center gap-2">
                    <BellRing className="size-3.5" />
                    In-App
                  </span>
                  <span className="inline-flex items-center gap-2">
                    <Mail className="size-3.5" />
                    Email
                  </span>
                </div>
              </div>

              <div className="space-y-2">
                {category.types.map((type) => {
                  const preference = prefs.get(type);
                  const isSaving = saving.has(type);
                  return (
                    <PreferenceRow
                      key={type}
                      label={TYPE_LABELS[type]}
                      inApp={preference?.inApp ?? true}
                      email={preference?.email ?? false}
                      disabled={isSaving}
                      onInAppChange={(checked) => {
                        void handleToggle(type, 'inApp', checked);
                      }}
                      onEmailChange={(checked) => {
                        void handleToggle(type, 'email', checked);
                      }}
                    />
                  );
                })}
              </div>
            </section>
          ))}
        </SettingsFieldGroup>
      </SettingsPanel>

      <SettingsPanel
        title="Quiet Hours"
        description="Suppress non-critical real-time notifications during specific hours. Security-critical alerts still break through."
      >
        <div className="space-y-4">
          <SettingsSwitchRow
            title="Do Not Disturb"
            description="Immediately suppress all non-critical real-time notifications."
            checked={schedule.dndEnabled}
            disabled={scheduleSaving}
            onCheckedChange={(checked) => {
              void handleScheduleChange({ dndEnabled: checked });
            }}
          />

          <div className="rounded-xl border border-border/70 bg-background/60 p-4">
            <div className="mb-3 flex items-center gap-2">
              <MoonStar className="size-4 text-muted-foreground" />
              <h4 className="text-sm font-medium text-foreground">Scheduled Quiet Hours</h4>
            </div>

            <div className="grid gap-4 md:grid-cols-[150px_auto_150px] md:items-center">
              <Input
                type="time"
                value={schedule.quietHoursStart ?? ''}
                disabled={scheduleSaving}
                onChange={(event) => {
                  void handleScheduleChange({
                    quietHoursStart: event.target.value || null,
                  });
                }}
                aria-label="Quiet hours start time"
              />
              <div className="text-center text-sm text-muted-foreground">to</div>
              <Input
                type="time"
                value={schedule.quietHoursEnd ?? ''}
                disabled={scheduleSaving}
                onChange={(event) => {
                  void handleScheduleChange({
                    quietHoursEnd: event.target.value || null,
                  });
                }}
                aria-label="Quiet hours end time"
              />
            </div>

            <div className="mt-4 space-y-2">
              <label className="text-sm font-medium text-foreground" htmlFor="quiet-hours-timezone">
                Timezone
              </label>
              <Input
                id="quiet-hours-timezone"
                list="notification-timezones"
                value={schedule.quietHoursTimezone ?? browserTimezone}
                disabled={scheduleSaving}
                onChange={(event) => {
                  void handleScheduleChange({
                    quietHoursTimezone: event.target.value || browserTimezone,
                  });
                }}
              />
              <datalist id="notification-timezones">
                {timezoneOptions.map((timezone) => (
                  <option key={timezone} value={timezone} />
                ))}
              </datalist>
            </div>
          </div>
        </div>
      </SettingsPanel>
    </div>
  );
}
