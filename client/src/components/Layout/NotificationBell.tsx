import { useEffect, useState } from 'react';
import { Bell, CheckCheck, X } from 'lucide-react';
import { connectSSE } from '@/api/sse';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useAuthStore } from '@/store/authStore';
import { useNotificationListStore } from '@/store/notificationListStore';
import { useDesktopNotifications } from '@/hooks/useDesktopNotifications';
import type { NotificationEntry } from '@/api/notifications.api';
import type { NotificationStreamSnapshot } from '@/api/live.api';
import {
  getNotificationIcon,
  getOnNavigate,
  type NavigationActions,
} from '@/utils/notificationActions';
import { CounterBadge, HeaderIconButton } from './layoutUi';

function timeAgo(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diffSec = Math.floor((now - then) / 1000);
  if (diffSec < 60) return 'just now';
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

interface NotificationBellProps {
  navigationActions: NavigationActions;
}

export default function NotificationBell({ navigationActions }: NotificationBellProps) {
  const accessToken = useAuthStore((state) => state.accessToken);
  const notifications = useNotificationListStore((state) => state.notifications);
  const unreadCount = useNotificationListStore((state) => state.unreadCount);
  const applySnapshot = useNotificationListStore((state) => state.applySnapshot);
  const markAsRead = useNotificationListStore((state) => state.markAsRead);
  const markAllAsRead = useNotificationListStore((state) => state.markAllAsRead);
  const removeNotification = useNotificationListStore((state) => state.removeNotification);
  const [open, setOpen] = useState(false);
  const { sendDesktopNotification, setOnClick } = useDesktopNotifications();
  const [latestNotificationId, setLatestNotificationId] = useState<string | null>(null);

  useEffect(() => {
    setOnClick(() => {
      setOpen(true);
    });
  }, [setOnClick]);

  useEffect(() => {
    if (!accessToken) {
      return undefined;
    }

    return connectSSE({
      url: '/api/notifications/stream',
      accessToken,
      onEvent: ({ event, data }) => {
        if (event !== 'snapshot') {
          return;
        }

        const snapshot = data as NotificationStreamSnapshot;
        const latest = snapshot.data[0];

        applySnapshot(snapshot);

        if (latest) {
          if (latestNotificationId && latestNotificationId !== latest.id && !latest.read) {
            sendDesktopNotification('New notification', {
              body: latest.message,
              tag: latest.id,
            });
          }
          setLatestNotificationId(latest.id);
        }
      },
    });
  }, [accessToken, applySnapshot, latestNotificationId, sendDesktopNotification]);

  const handleClick = (notification: NotificationEntry) => {
    if (!notification.read) {
      void markAsRead(notification.id);
    }

    const navigate = getOnNavigate(notification.type);
    if (navigate) {
      navigate(notification, navigationActions);
      setOpen(false);
    }
  };

  return (
    <DropdownMenu open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <HeaderIconButton aria-label="Notifications" title="Notifications">
          <Bell className="size-4" />
          <CounterBadge count={unreadCount} />
        </HeaderIconButton>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-[22rem] p-0">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <div>
            <p className="text-sm font-medium text-foreground">Notifications</p>
            <p className="text-xs text-muted-foreground">
              {unreadCount > 0 ? `${unreadCount} unread` : 'All caught up'}
            </p>
          </div>
          {unreadCount > 0 ? (
            <button
              type="button"
              className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-primary transition-colors hover:bg-accent"
              onClick={() => void markAllAsRead()}
            >
              <CheckCheck className="size-3.5" />
              Mark all read
            </button>
          ) : null}
        </div>

        {notifications.length === 0 ? (
          <div className="px-4 py-8 text-center text-sm text-muted-foreground">
            No notifications
          </div>
        ) : (
          <ScrollArea className="max-h-96">
            <div className="divide-y">
              {notifications.map((notification) => (
                <div
                  key={notification.id}
                  className={notification.read ? 'bg-transparent' : 'bg-primary/5'}
                >
                  <div className="flex items-start gap-2 px-3 py-3">
                    <button
                      type="button"
                      className="flex min-w-0 flex-1 items-start gap-3 text-left"
                      onClick={() => handleClick(notification)}
                    >
                      <span className="mt-0.5 shrink-0">{getNotificationIcon(notification.type)}</span>
                      <span className="min-w-0 space-y-1">
                        <span className={`block text-sm ${notification.read ? 'text-foreground' : 'font-medium text-foreground'}`}>
                          {notification.message}
                        </span>
                        <span className="block text-xs text-muted-foreground">
                          {timeAgo(notification.createdAt)}
                        </span>
                      </span>
                    </button>

                    <button
                      type="button"
                      aria-label="Dismiss notification"
                      className="mt-0.5 inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                      onClick={(event) => {
                        event.stopPropagation();
                        void removeNotification(notification.id);
                      }}
                    >
                      <X className="size-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
