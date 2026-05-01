import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Loader2, Maximize, Minimize, Power } from 'lucide-react';
import * as Guacamole from '@glokon/guacamole-common-js';
import { useAutoReconnect } from '@/hooks/useAutoReconnect';
import { useKeyboardCapture } from '@/hooks/useKeyboardCapture';
import type { ObserveDesktopSessionResponse } from '@/api/sessions.api';
import DockedToolbar, { type ToolbarAction } from '@/components/shared/DockedToolbar';
import ReconnectOverlay from '@/components/shared/ReconnectOverlay';
import { isGuacPermanentError } from '@/utils/reconnectClassifier';

interface DesktopObserverViewerProps {
  protocol: 'RDP' | 'VNC';
  session: ObserveDesktopSessionResponse;
}

function resolveDesktopWebSocketUrl(session: ObserveDesktopSessionResponse): string {
  if (session.webSocketUrl) {
    return session.webSocketUrl;
  }

  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${wsProtocol}//${window.location.host}${session.webSocketPath}`;
}

export default function DesktopObserverViewer({ protocol, session }: DesktopObserverViewerProps) {
  const displayRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const clientRef = useRef<Guacamole.Client | null>(null);
  const resizeObserverRef = useRef<ResizeObserver | null>(null);
  const triggerReconnectRef = useRef<() => void>(() => {});
  const resetReconnectRef = useRef<() => void>(() => {});
  const lastGuacErrorRef = useRef('');
  const wasConnectedRef = useRef(false);
  const permanentErrorRef = useRef(false);
  const connectedAtRef = useRef(0);

  const [status, setStatus] = useState<'connecting' | 'connected' | 'unstable' | 'error'>('connecting');
  const [error, setError] = useState('');

  const STABLE_THRESHOLD_MS = 5_000;

  const connectSession = useCallback(async () => {
    if (!displayRef.current) {
      return;
    }

    if (resizeObserverRef.current) {
      resizeObserverRef.current.disconnect();
      resizeObserverRef.current = null;
    }
    if (clientRef.current) {
      clientRef.current.onstatechange = null;
      clientRef.current.onerror = null;
      clientRef.current.disconnect();
      clientRef.current = null;
    }
    displayRef.current.innerHTML = '';
    setStatus('connecting');
    setError('');

    const tunnel = new Guacamole.WebSocketTunnel(resolveDesktopWebSocketUrl(session));
    const client = new Guacamole.Client(tunnel);
    clientRef.current = client;

    const display = client.getDisplay().getElement();
    displayRef.current.appendChild(display);

    let connected = false;

    const handleResize = () => {
      if (!connected || !displayRef.current) {
        return;
      }

      const width = displayRef.current.clientWidth;
      const height = displayRef.current.clientHeight;
      if (width <= 0 || height <= 0) {
        return;
      }

      const guacDisplay = client.getDisplay();
      const scale = Math.min(width / guacDisplay.getWidth(), height / guacDisplay.getHeight());
      if (isFinite(scale) && scale > 0) {
        guacDisplay.scale(scale);
      }
    };

    (client.getDisplay() as unknown as { onresize: (() => void) | null }).onresize = handleResize;

    client.onstatechange = (state: number) => {
      switch (state) {
        case 3:
          connected = true;
          wasConnectedRef.current = true;
          connectedAtRef.current = Date.now();
          lastGuacErrorRef.current = '';
          setStatus('connected');
          resetReconnectRef.current();
          setTimeout(() => {
            handleResize();
            if (displayRef.current && !resizeObserverRef.current) {
              resizeObserverRef.current = new ResizeObserver(handleResize);
              resizeObserverRef.current.observe(displayRef.current);
            }
          }, 100);
          return;
        case 4:
          if (connected) {
            setStatus('unstable');
          }
          return;
        case 5: {
          connected = false;
          if (permanentErrorRef.current) {
            return;
          }
          const uptime = connectedAtRef.current ? Date.now() - connectedAtRef.current : 0;
          connectedAtRef.current = 0;
          if (wasConnectedRef.current && uptime >= STABLE_THRESHOLD_MS && !isGuacPermanentError(lastGuacErrorRef.current)) {
            triggerReconnectRef.current();
            return;
          }
          setStatus('error');
          setError(lastGuacErrorRef.current || `Disconnected from observed ${protocol} session`);
        }
      }
    };

    client.onerror = (err: { message?: string }) => {
      const message = err.message || `${protocol} observer connection error`;
      lastGuacErrorRef.current = message;
      if (isGuacPermanentError(message)) {
        permanentErrorRef.current = true;
        setStatus('error');
        setError(message);
      }
    };

    client.connect(`token=${encodeURIComponent(session.token)}`);
  }, [protocol, session]);

  const { reconnectState, attempt, maxRetries, triggerReconnect, cancelReconnect, resetReconnect } = useAutoReconnect(
    connectSession,
  );

  useEffect(() => {
    triggerReconnectRef.current = triggerReconnect;
    resetReconnectRef.current = resetReconnect;
  }, [triggerReconnect, resetReconnect]);

  const { isFullscreen, toggleFullscreen } = useKeyboardCapture({
    focusRef: displayRef,
    fullscreenRef: containerRef,
    isActive: true,
    suppressBrowserKeys: false,
  });

  const toolbarActions = useMemo<ToolbarAction[]>(() => ([
    {
      id: 'fullscreen',
      icon: isFullscreen ? <Minimize className="size-4" /> : <Maximize className="size-4" />,
      tooltip: isFullscreen ? 'Exit Fullscreen' : 'Fullscreen',
      onClick: toggleFullscreen,
      active: isFullscreen,
    },
    {
      id: 'disconnect',
      icon: <Power className="size-4" />,
      tooltip: 'Disconnect',
      onClick: () => window.close(),
      color: 'error.main',
    },
  ]), [isFullscreen, toggleFullscreen]);

  useEffect(() => {
    if (!displayRef.current) {
      return;
    }

    permanentErrorRef.current = false;
    wasConnectedRef.current = false;
    lastGuacErrorRef.current = '';
    connectedAtRef.current = 0;

    const connectTimer = window.setTimeout(() => {
      void connectSession().catch((err: unknown) => {
        setStatus('error');
        setError(err instanceof Error ? err.message : `Failed to observe ${protocol} session`);
      });
    }, 0);

    return () => {
      const displayElement = displayRef.current;
      window.clearTimeout(connectTimer);
      cancelReconnect();
      if (resizeObserverRef.current) {
        resizeObserverRef.current.disconnect();
        resizeObserverRef.current = null;
      }
      if (clientRef.current) {
        clientRef.current.onstatechange = null;
        clientRef.current.onerror = null;
        clientRef.current.disconnect();
        clientRef.current = null;
      }
      if (displayElement) {
        displayElement.innerHTML = '';
      }
    };
  }, [cancelReconnect, connectSession, protocol]);

  return (
    <div ref={containerRef} className="flex flex-1 flex-row overflow-hidden">
      {status === 'connected' && reconnectState === 'idle' ? <DockedToolbar actions={toolbarActions} /> : null}
      <div className="relative flex min-w-0 flex-1 flex-col">
        {status === 'connecting' && reconnectState === 'idle' ? (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/70">
            <Loader2 className="mr-2 h-5 w-5 animate-spin" />
            <span>Connecting to observed {protocol} session...</span>
          </div>
        ) : null}
        {status === 'error' && reconnectState === 'idle' ? (
          <div className="m-2 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
            {error}
          </div>
        ) : null}
        {status === 'unstable' && reconnectState === 'idle' ? (
          <ReconnectOverlay state="unstable" attempt={0} maxRetries={maxRetries} protocol={protocol} />
        ) : null}
        {reconnectState === 'reconnecting' ? (
          <ReconnectOverlay state="reconnecting" attempt={attempt} maxRetries={maxRetries} protocol={protocol} />
        ) : null}
        {reconnectState === 'failed' ? (
          <ReconnectOverlay
            state="failed"
            attempt={attempt}
            maxRetries={maxRetries}
            protocol={protocol}
            onRetry={() => {
              permanentErrorRef.current = false;
              wasConnectedRef.current = true;
              triggerReconnect();
            }}
          />
        ) : null}
        <div
          ref={displayRef}
          tabIndex={-1}
          className="flex-1 overflow-hidden outline-none cursor-default [&>div]:!h-full [&>div]:!w-full"
        />
      </div>
    </div>
  );
}
