import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { SettingsPanel, SettingsSwitchRow } from './settings-ui';
import { useDesktopNotifications } from '../../hooks/useDesktopNotifications';

export default function NotificationsSection() {
  const {
    supported,
    permission,
    enabled,
    setEnabled,
  } = useDesktopNotifications();

  return (
    <SettingsPanel
      title="Desktop Notifications"
      description="Receive native notifications while Arsenale is in the background."
    >
      {!supported ? (
        <Alert variant="warning">
          <AlertTitle>Unsupported Browser</AlertTitle>
          <AlertDescription>
            Your browser does not support desktop notifications.
          </AlertDescription>
        </Alert>
      ) : (
        <div className="space-y-3">
          <SettingsSwitchRow
            title="Enable desktop notifications"
            description="Show native notifications when new alerts arrive and the app is not focused."
            checked={enabled}
            disabled={permission === 'denied'}
            onCheckedChange={(checked) => {
              void setEnabled(checked);
            }}
          />

          {permission === 'denied' && (
            <Alert variant="info">
              <AlertTitle>Permission Blocked</AlertTitle>
              <AlertDescription>
                Notification permission was denied by the browser. Re-enable it in the site
                settings for this page.
              </AlertDescription>
            </Alert>
          )}

          {permission === 'default' && enabled && (
            <Alert variant="info">
              <AlertDescription>
                The browser will ask for permission when the next notification is triggered.
              </AlertDescription>
            </Alert>
          )}

          {permission === 'granted' && enabled && (
            <p className="text-sm text-muted-foreground">
              Desktop notifications are active and will appear while the app is not in focus.
            </p>
          )}
        </div>
      )}
    </SettingsPanel>
  );
}
