import { CheckCircle2, XCircle, Hourglass, SlidersHorizontal } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { cn } from '@/lib/utils';

export type DbConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error';

interface DbConnectionStatusProps {
  state: DbConnectionState;
  protocol: string;
  databaseName?: string;
  error?: string;
  hasSessionConfig?: boolean;
}

export default function DbConnectionStatus({
  state,
  protocol,
  databaseName,
  error,
  hasSessionConfig,
}: DbConnectionStatusProps) {
  const statusConfig: Record<
    DbConnectionState,
    { label: string; badgeClass: string; icon: React.ReactElement }
  > = {
    connecting: {
      label: 'Connecting',
      badgeClass: 'border-yellow-500/50 text-yellow-400',
      icon: <Hourglass className="size-3.5" />,
    },
    connected: {
      label: 'Connected',
      badgeClass: 'border-green-500/50 text-green-400',
      icon: <CheckCircle2 className="size-3.5" />,
    },
    disconnected: {
      label: 'Disconnected',
      badgeClass: 'border-border text-muted-foreground',
      icon: <XCircle className="size-3.5" />,
    },
    error: {
      label: 'Error',
      badgeClass: 'border-red-500/50 text-red-400',
      icon: <XCircle className="size-3.5" />,
    },
  };

  const { label, badgeClass, icon } = statusConfig[state];

  return (
    <div className="flex items-center gap-2">
      <Badge variant="outline" className={cn('h-6 gap-1', badgeClass)}>
        {icon}
        {label}
      </Badge>
      <span className="text-xs text-muted-foreground">
        {protocol.toUpperCase()}
      </span>
      {databaseName && (
        <span className="text-xs text-muted-foreground">
          / {databaseName}
        </span>
      )}
      {hasSessionConfig && (
        <SlidersHorizontal className="size-3 text-primary ml-0.5" />
      )}
      {state === 'error' && error && (
        <span className="text-xs text-red-400 ml-2">
          {error}
        </span>
      )}
    </div>
  );
}
