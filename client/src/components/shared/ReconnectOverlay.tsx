import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Loader2, AlertTriangle, WifiOff } from 'lucide-react';

interface ReconnectOverlayProps {
  state: 'reconnecting' | 'unstable' | 'failed';
  attempt: number;
  maxRetries: number;
  onRetry?: () => void;
  onClose?: () => void;
  protocol: 'RDP' | 'VNC' | 'SSH' | 'DATABASE';
}

export default function ReconnectOverlay({ state, attempt, maxRetries, onRetry, onClose, protocol }: ReconnectOverlayProps) {
  if (state === 'unstable') {
    return (
      <div className="absolute top-2 left-1/2 -translate-x-1/2 z-10">
        <Badge className="bg-yellow-500/15 text-yellow-400 border-yellow-500/30 gap-1">
          <WifiOff className="h-3.5 w-3.5" />
          Connection unstable
        </Badge>
      </div>
    );
  }

  return (
    <div className="absolute inset-0 flex flex-col items-center justify-center z-10 bg-black/70 gap-3">
      {state === 'reconnecting' && (
        <>
          <Loader2 className="h-8 w-8 animate-spin" />
          <span>
            Reconnecting to {protocol} session... (attempt {attempt + 1}/{maxRetries})
          </span>
        </>
      )}
      {state === 'failed' && (
        <>
          <AlertTriangle className="h-12 w-12 text-red-500" />
          <h3 className="text-lg font-semibold">Reconnection failed</h3>
          <p className="text-sm text-muted-foreground">
            Could not restore the {protocol} session after {maxRetries} attempts.
          </p>
          <div className="flex gap-2 mt-2">
            {onRetry && (
              <Button size="sm" onClick={onRetry}>
                Retry
              </Button>
            )}
            {onClose && (
              <Button variant="outline" size="sm" onClick={onClose}>
                Close Tab
              </Button>
            )}
          </div>
        </>
      )}
    </div>
  );
}
