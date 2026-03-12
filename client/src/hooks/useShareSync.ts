import { useEffect, useRef } from 'react';
import { useNotificationListStore } from '../store/notificationListStore';
import { getOnReceive } from '../utils/notificationActions';

const DEBOUNCE_MS = 500;

/**
 * Subscribes to the notification list store and triggers data refreshes
 * when share-related notifications arrive. Uses the notification action
 * registry to determine which refresh to call per notification type.
 *
 * Mount once in MainLayout alongside useGatewayMonitor().
 */
export function useShareSync() {
  const connectionsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const secretsTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    const unsubscribe = useNotificationListStore.subscribe(
      (state, prevState) => {
        if (state.notifications.length <= prevState.notifications.length) return;
        const latest = state.notifications[0];
        if (!latest) return;

        const handler = getOnReceive(latest.type);
        if (!handler) return;

        // Use separate debounce timers based on handler identity so that
        // a connection share event doesn't delay a simultaneous secret refresh
        const isConnectionHandler = latest.type.startsWith('CONNECTION_') || latest.type === 'SHARE_REVOKED' || latest.type === 'SHARE_PERMISSION_UPDATED';
        const timerRef = isConnectionHandler ? connectionsTimerRef : secretsTimerRef;

        if (timerRef.current) clearTimeout(timerRef.current);
        timerRef.current = setTimeout(() => {
          handler(latest);
          timerRef.current = null;
        }, DEBOUNCE_MS);
      },
    );

    return () => {
      unsubscribe();
      // eslint-disable-next-line react-hooks/exhaustive-deps -- timer refs are stable and not React-rendered nodes
      if (connectionsTimerRef.current) clearTimeout(connectionsTimerRef.current);
      // eslint-disable-next-line react-hooks/exhaustive-deps
      if (secretsTimerRef.current) clearTimeout(secretsTimerRef.current);
    };
  }, []);
}
