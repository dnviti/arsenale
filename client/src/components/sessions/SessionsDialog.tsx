import { useState } from 'react';
import { Activity, X } from 'lucide-react';
import {
  Dialog,
  DialogContent,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import {
  ControlledSessionsConsole,
} from './SessionsConsole';
import { resolveSessionsRouteState, type SessionsRouteState } from './sessionConsoleRoute';

interface SessionsDialogProps {
  open: boolean;
  onClose: () => void;
  initialState?: Partial<SessionsRouteState>;
}

export default function SessionsDialog({ open, onClose, initialState }: SessionsDialogProps) {
  if (!open) {
    return null;
  }

  const resetKey = JSON.stringify(resolveSessionsRouteState(initialState));

  return (
    <SessionsDialogContent
      key={resetKey}
      onClose={onClose}
      initialState={initialState}
    />
  );
}

function SessionsDialogContent({ onClose, initialState }: Omit<SessionsDialogProps, 'open'>) {
  const [routeState, setRouteState] = useState(() => resolveSessionsRouteState(initialState));

  return (
    <Dialog open onOpenChange={(nextOpen) => { if (!nextOpen) onClose(); }}>
      <DialogContent
        showCloseButton={false}
        className="flex h-[100dvh] w-screen max-w-none flex-col gap-0 rounded-none border-0 p-0 sm:h-[94vh] sm:w-[96vw] sm:max-w-[1700px] sm:overflow-hidden sm:rounded-2xl sm:border"
      >
        <div className="flex h-10 shrink-0 items-center gap-2 border-b px-3 sm:px-4">
          <Activity className="size-4 text-muted-foreground" />
          <span className="text-sm font-medium">Sessions</span>
          <div className="ml-auto">
            <Button type="button" variant="ghost" size="icon-xs" onClick={onClose} aria-label="Close sessions console">
              <X className="size-3.5" />
            </Button>
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-hidden">
          <ControlledSessionsConsole
            routeState={routeState}
            onRouteStateChange={setRouteState}
            layout="dialog"
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
