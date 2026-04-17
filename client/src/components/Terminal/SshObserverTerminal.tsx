import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Loader2, Maximize, Minimize, Power } from 'lucide-react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { useTerminalSettingsStore } from '@/store/terminalSettingsStore';
import { useThemeStore } from '@/store/themeStore';
import { useAutoReconnect } from '@/hooks/useAutoReconnect';
import { useKeyboardCapture } from '@/hooks/useKeyboardCapture';
import type { ObserveSshSessionResponse } from '@/api/sessions.api';
import type { SshTerminalConfig } from '@/constants/terminalThemes';
import { mergeTerminalConfig, resolveThemeForMode, THEME_PRESETS, toXtermOptions } from '@/constants/terminalThemes';
import DockedToolbar, { type ToolbarAction } from '@/components/shared/DockedToolbar';
import ReconnectOverlay from '@/components/shared/ReconnectOverlay';
import '@xterm/xterm/css/xterm.css';

interface SshObserverTerminalProps {
  session: ObserveSshSessionResponse;
}

interface TerminalBrokerMessage {
  type: 'ready' | 'data' | 'pong' | 'closed' | 'error';
  data?: string;
  code?: string;
  message?: string;
}

function isPermanentTerminalCode(code?: string): boolean {
  return code === 'INVALID_TOKEN'
    || code === 'SESSION_TIMEOUT'
    || code === 'SESSION_TERMINATED'
    || code === 'SESSION_CLOSED';
}

function resolveBrowserWebSocketUrl(session: ObserveSshSessionResponse): string {
  if (session.webSocketUrl) {
    return session.webSocketUrl;
  }

  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${wsProtocol}//${window.location.host}${session.webSocketPath}?token=${encodeURIComponent(session.token)}`;
}

export default function SshObserverTerminal({ session }: SshObserverTerminalProps) {
  const termRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const webSocketRef = useRef<WebSocket | null>(null);
  const heartbeatIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const triggerReconnectRef = useRef<() => void>(() => {});
  const resetReconnectRef = useRef<() => void>(() => {});
  const cancelledRef = useRef(false);
  const permanentErrorRef = useRef(false);
  const wasConnectedRef = useRef(false);

  const [status, setStatus] = useState<'connecting' | 'connected' | 'error'>('connecting');
  const [error, setError] = useState('');

  const userDefaults = useTerminalSettingsStore((s) => s.userDefaults);
  const webUiMode = useThemeStore((s) => s.mode);
  const safeUserDefaults = useMemo<Partial<SshTerminalConfig>>(() => userDefaults ?? {}, [userDefaults]);

  const observerConfig = useMemo<Partial<SshTerminalConfig>>(
    () => ({
      fontSize: safeUserDefaults.fontSize,
      fontFamily: safeUserDefaults.fontFamily,
      lineHeight: safeUserDefaults.lineHeight,
    }),
    [safeUserDefaults.fontFamily, safeUserDefaults.fontSize, safeUserDefaults.lineHeight],
  );

  const xtermOptions = useMemo(() => {
    const options = toXtermOptions(mergeTerminalConfig(safeUserDefaults, observerConfig), webUiMode);
    return { ...options, disableStdin: true };
  }, [observerConfig, safeUserDefaults, webUiMode]);

  useEffect(() => {
    const terminal = terminalRef.current;
    if (!terminal) {
      return;
    }

    const merged = mergeTerminalConfig(safeUserDefaults, observerConfig);
    const effectiveTheme = resolveThemeForMode(merged, webUiMode);
    const colors = effectiveTheme === 'custom'
      ? merged.customColors
      : THEME_PRESETS[effectiveTheme] ?? THEME_PRESETS['default-dark'];

    terminal.options.theme = {
      background: colors.background,
      foreground: colors.foreground,
      cursor: colors.cursor,
      selectionBackground: colors.selectionBackground,
      black: colors.black,
      red: colors.red,
      green: colors.green,
      yellow: colors.yellow,
      blue: colors.blue,
      magenta: colors.magenta,
      cyan: colors.cyan,
      white: colors.white,
      brightBlack: colors.brightBlack,
      brightRed: colors.brightRed,
      brightGreen: colors.brightGreen,
      brightYellow: colors.brightYellow,
      brightBlue: colors.brightBlue,
      brightMagenta: colors.brightMagenta,
      brightCyan: colors.brightCyan,
      brightWhite: colors.brightWhite,
    };
  }, [observerConfig, safeUserDefaults, webUiMode]);

  const cleanupTransport = useCallback(() => {
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current);
      heartbeatIntervalRef.current = null;
    }

    if (webSocketRef.current) {
      const ws = webSocketRef.current;
      webSocketRef.current = null;
      ws.close();
    }
  }, []);

  const connectSession = useCallback(async () => {
    cleanupTransport();
    setStatus('connecting');
    setError('');

    const terminal = terminalRef.current;
    if (!terminal) {
      return;
    }

    await new Promise<void>((resolve, reject) => {
      const ws = new WebSocket(resolveBrowserWebSocketUrl(session));
      webSocketRef.current = ws;

      let resolved = false;
      let failedBeforeReady = false;

      const fail = (message: string, permanent = false) => {
        if (permanent) {
          permanentErrorRef.current = true;
        }
        setStatus('error');
        setError(message);
        if (!resolved) {
          failedBeforeReady = true;
          reject(new Error(message));
          return;
        }
        terminal.write(`\r\n\x1b[31m${message}\x1b[0m\r\n`);
      };

      ws.onmessage = (event) => {
        let message: TerminalBrokerMessage;
        try {
          message = JSON.parse(String(event.data)) as TerminalBrokerMessage;
        } catch {
          fail('Invalid terminal broker payload');
          return;
        }

        switch (message.type) {
          case 'ready':
            resolved = true;
            wasConnectedRef.current = true;
            setStatus('connected');
            resetReconnectRef.current();
            heartbeatIntervalRef.current = setInterval(() => {
              if (webSocketRef.current?.readyState === WebSocket.OPEN) {
                webSocketRef.current.send(JSON.stringify({ type: 'ping' }));
              }
            }, 30_000);
            resolve();
            return;
          case 'data':
            terminal.write(message.data ?? '');
            return;
          case 'pong':
            return;
          case 'closed':
            terminal.write('\r\n\x1b[33mObserved session closed.\x1b[0m\r\n');
            return;
          case 'error': {
            const errorMessage = message.message || 'Observer session failed';
            if (isPermanentTerminalCode(message.code)) {
              fail(errorMessage, true);
              return;
            }
            fail(errorMessage);
            return;
          }
        }
      };

      ws.onerror = () => {
        fail('Failed to connect to observed terminal session.');
      };

      ws.onclose = () => {
        if (heartbeatIntervalRef.current) {
          clearInterval(heartbeatIntervalRef.current);
          heartbeatIntervalRef.current = null;
        }

        if (cancelledRef.current || permanentErrorRef.current) {
          return;
        }

        if (failedBeforeReady) {
          return;
        }

        if (resolved && wasConnectedRef.current) {
          terminal.write('\r\n\x1b[33m[Reconnecting observer... ]\x1b[0m\r\n');
          triggerReconnectRef.current();
          return;
        }

        fail('Observer connection lost');
      };
    });
  }, [cleanupTransport, session]);

  const { reconnectState, attempt, maxRetries, triggerReconnect, cancelReconnect, resetReconnect } = useAutoReconnect(
    connectSession,
  );

  useEffect(() => {
    triggerReconnectRef.current = triggerReconnect;
    resetReconnectRef.current = resetReconnect;
  }, [triggerReconnect, resetReconnect]);

  const { isFullscreen, toggleFullscreen } = useKeyboardCapture({
    focusRef: termRef,
    fullscreenRef: containerRef,
    isActive: true,
    onFullscreenChange: () => {
      setTimeout(() => fitAddonRef.current?.fit(), 100);
    },
    suppressBrowserKeys: false,
    onRequestFocus: () => terminalRef.current?.focus(),
  });

  const toolbarActions = useMemo<ToolbarAction[]>(() => ([
    {
      id: 'fullscreen',
      icon: isFullscreen ? <Minimize className="h-4 w-4" /> : <Maximize className="h-4 w-4" />,
      tooltip: isFullscreen ? 'Exit Fullscreen' : 'Fullscreen',
      onClick: toggleFullscreen,
      active: isFullscreen,
    },
    {
      id: 'disconnect',
      icon: <Power className="h-4 w-4" />,
      tooltip: 'Disconnect',
      onClick: () => window.close(),
      color: 'error.main',
    },
  ]), [isFullscreen, toggleFullscreen]);

  useEffect(() => {
    if (!termRef.current) {
      return;
    }

    cancelledRef.current = false;
    permanentErrorRef.current = false;
    wasConnectedRef.current = false;

    const terminal = new Terminal(xtermOptions);
    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(termRef.current);
    fitAddon.fit();
    terminal.write('\x1b[2mRead-only observer connected. Input is disabled.\x1b[0m\r\n\r\n');

    terminalRef.current = terminal;
    fitAddonRef.current = fitAddon;

    const resizeObserver = new ResizeObserver(() => fitAddon.fit());
    resizeObserver.observe(termRef.current);

    const connectTimer = window.setTimeout(() => {
      void connectSession().catch((err: unknown) => {
        if (cancelledRef.current || permanentErrorRef.current) {
          return;
        }
        setStatus('error');
        setError(err instanceof Error ? err.message : 'Failed to connect observer');
      });
    }, 0);

    return () => {
      cancelledRef.current = true;
      window.clearTimeout(connectTimer);
      cancelReconnect();
      resizeObserver.disconnect();
      cleanupTransport();
      terminal.dispose();
    };
  }, [cancelReconnect, cleanupTransport, connectSession, xtermOptions]);

  return (
    <div ref={containerRef} data-viewer-type="ssh-observer" className="flex flex-1 flex-row overflow-hidden">
      {status === 'connected' && reconnectState === 'idle' ? <DockedToolbar actions={toolbarActions} /> : null}
      <div className="relative flex min-w-0 flex-1 flex-col">
        {status === 'connecting' && reconnectState === 'idle' ? (
          <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/70">
            <Loader2 className="mr-2 h-5 w-5 animate-spin" />
            <span>Connecting to observed terminal...</span>
          </div>
        ) : null}
        {status === 'error' && reconnectState === 'idle' ? (
          <div className="m-2 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
            {error}
          </div>
        ) : null}
        {reconnectState === 'reconnecting' ? (
          <ReconnectOverlay state="reconnecting" attempt={attempt} maxRetries={maxRetries} protocol="SSH" />
        ) : null}
        {reconnectState === 'failed' ? (
          <ReconnectOverlay
            state="failed"
            attempt={attempt}
            maxRetries={maxRetries}
            protocol="SSH"
            onRetry={() => {
              permanentErrorRef.current = false;
              wasConnectedRef.current = true;
              triggerReconnect();
            }}
          />
        ) : null}
        <div
          ref={termRef}
          tabIndex={-1}
          className="flex-1 overflow-hidden [&_.xterm]:h-full [&_.xterm]:p-1"
        />
      </div>
    </div>
  );
}
