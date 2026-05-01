import type { ReactNode } from 'react';
import {
  Bell,
  BellOff,
  KeyRound,
  KeySquare,
  Pencil,
  Plane,
  Share2,
  TriangleAlert,
  UserRoundPlus,
  Video,
  XCircle,
} from 'lucide-react';
import type { NotificationEntry, NotificationType } from '../api/notifications.api';
import { useConnectionsStore } from '../store/connectionsStore';
import { useSecretStore } from '../store/secretStore';

export interface NavigationActions {
  openKeychain: () => void;
  openRecordings: () => void;
  openSettings: (tab?: string) => void;
  openAuditLog: () => void;
  selectConnection: (connectionId: string) => void;
}

interface NotificationActionDef {
  icon: ReactNode;
  onNavigate?: (notification: NotificationEntry, actions: NavigationActions) => void;
  onReceive?: (notification: NotificationEntry) => void;
}

function refreshConnections() {
  useConnectionsStore.getState().fetchConnections();
}

function refreshSecrets() {
  useSecretStore.getState().fetchSecrets();
}

function iconBadge(icon: ReactNode, tone: 'destructive' | 'primary' | 'success' | 'warning' | 'muted') {
  const toneClasses = {
    primary: 'bg-primary/10 text-primary',
    success: 'bg-primary/10 text-primary',
    warning: 'bg-chart-5/15 text-chart-5',
    destructive: 'bg-destructive/10 text-destructive',
    muted: 'bg-muted text-muted-foreground',
  } as const;

  return (
    <span className={`inline-flex size-7 items-center justify-center rounded-full ${toneClasses[tone]}`}>
      {icon}
    </span>
  );
}

const NOTIFICATION_ACTIONS: Record<NotificationType, NotificationActionDef> = {
  CONNECTION_SHARED: {
    icon: iconBadge(<Share2 className="size-4" />, 'primary'),
    onReceive: refreshConnections,
    onNavigate: (notification, actions) => {
      if (notification.relatedId) {
        actions.selectConnection(notification.relatedId);
      }
    },
  },
  SHARE_PERMISSION_UPDATED: {
    icon: iconBadge(<Pencil className="size-4" />, 'warning'),
    onReceive: refreshConnections,
    onNavigate: (notification, actions) => {
      if (notification.relatedId) {
        actions.selectConnection(notification.relatedId);
      }
    },
  },
  SHARE_REVOKED: {
    icon: iconBadge(<XCircle className="size-4" />, 'destructive'),
    onReceive: refreshConnections,
  },
  SECRET_SHARED: {
    icon: iconBadge(<KeyRound className="size-4" />, 'primary'),
    onReceive: refreshSecrets,
    onNavigate: (_notification, actions) => actions.openKeychain(),
  },
  SECRET_SHARE_REVOKED: {
    icon: iconBadge(<BellOff className="size-4" />, 'destructive'),
    onReceive: refreshSecrets,
  },
  SECRET_EXPIRING: {
    icon: iconBadge(<TriangleAlert className="size-4" />, 'warning'),
    onNavigate: (_notification, actions) => actions.openKeychain(),
  },
  SECRET_EXPIRED: {
    icon: iconBadge(<TriangleAlert className="size-4" />, 'destructive'),
    onNavigate: (_notification, actions) => actions.openKeychain(),
  },
  TENANT_INVITATION: {
    icon: iconBadge(<UserRoundPlus className="size-4" />, 'primary'),
    onNavigate: (_notification, actions) => actions.openSettings('organization'),
  },
  RECORDING_READY: {
    icon: iconBadge(<Video className="size-4" />, 'success'),
    onNavigate: (_notification, actions) => actions.openRecordings(),
  },
  IMPOSSIBLE_TRAVEL_DETECTED: {
    icon: iconBadge(<Plane className="size-4" />, 'destructive'),
    onNavigate: (_notification, actions) => actions.openAuditLog(),
  },
  LATERAL_MOVEMENT_ALERT: {
    icon: iconBadge(<KeySquare className="size-4" />, 'destructive'),
    onNavigate: (_notification, actions) => actions.openAuditLog(),
  },
};

export function getNotificationIcon(type: NotificationType): ReactNode {
  return NOTIFICATION_ACTIONS[type]?.icon ?? iconBadge(<Bell className="size-4" />, 'muted');
}

export function getOnReceive(type: NotificationType) {
  return NOTIFICATION_ACTIONS[type]?.onReceive;
}

export function getOnNavigate(type: NotificationType) {
  return NOTIFICATION_ACTIONS[type]?.onNavigate;
}
