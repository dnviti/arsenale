import { useAuthStore } from '../store/authStore';
import { refreshAccessToken } from './client';

export interface SSEEventMessage {
  event: string;
  data: unknown;
}

export interface ConnectSSEOptions {
  url: string;
  accessToken: string;
  onEvent: (message: SSEEventMessage) => void;
  onOpen?: () => void;
  onError?: (error: Error) => void;
  retryMs?: number;
  shouldRetry?: (status?: number, error?: Error) => boolean;
}

class SSEConnectionError extends Error {
  status?: number;

  constructor(message: string, status?: number) {
    super(message);
    this.name = 'SSEConnectionError';
    this.status = status;
  }
}

const DEFAULT_RETRY_MS = 3000;

export function connectSSE(options: ConnectSSEOptions): () => void {
  let closed = false;
  let controller: AbortController | null = null;
  let retryTimer: ReturnType<typeof setTimeout> | null = null;

  const shouldRetry = (status?: number, error?: Error) => {
    if (options.shouldRetry) return options.shouldRetry(status, error);
    return status == null || status === 408 || status === 429 || (status >= 500 && status !== 501);
  };

  const scheduleReconnect = () => {
    if (closed || retryTimer) return;
    retryTimer = setTimeout(() => {
      retryTimer = null;
      void openStream(true);
    }, options.retryMs ?? DEFAULT_RETRY_MS);
  };

  const openStream = async (allowRefresh: boolean) => {
    controller = new AbortController();
    try {
      const currentToken = useAuthStore.getState().accessToken ?? options.accessToken;
      const response = await fetch(options.url, {
        method: 'GET',
        headers: {
          Accept: 'text/event-stream',
          Authorization: `Bearer ${currentToken}`,
        },
        cache: 'no-store',
        credentials: 'include',
        signal: controller.signal,
      });

      if (response.status === 401 && allowRefresh && useAuthStore.getState().isAuthenticated) {
        await refreshAccessToken();
        if (!closed) {
          void openStream(false);
        }
        return;
      }

      if (!response.ok) {
        throw new SSEConnectionError(`Stream request failed with ${response.status}`, response.status);
      }

      if (!response.body) {
        throw new SSEConnectionError('Stream response body is unavailable');
      }

      options.onOpen?.();
      await readStream(response.body, options.onEvent, controller.signal);

      if (!closed) {
        scheduleReconnect();
      }
    } catch (error) {
      if (closed || controller?.signal.aborted) return;
      const normalized = error instanceof Error ? error : new Error('Stream connection failed');
      options.onError?.(normalized);
      const status = normalized instanceof SSEConnectionError ? normalized.status : undefined;
      if (shouldRetry(status, normalized)) {
        scheduleReconnect();
      }
    }
  };

  void openStream(true);

  return () => {
    closed = true;
    controller?.abort();
    if (retryTimer) {
      clearTimeout(retryTimer);
      retryTimer = null;
    }
  };
}

async function readStream(
  body: ReadableStream<Uint8Array>,
  onEvent: (message: SSEEventMessage) => void,
  signal: AbortSignal,
) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  while (!signal.aborted) {
    const { value, done } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, '\n');

    let separatorIndex = buffer.indexOf('\n\n');
    while (separatorIndex >= 0) {
      const chunk = buffer.slice(0, separatorIndex);
      buffer = buffer.slice(separatorIndex + 2);
      const parsed = parseEventChunk(chunk);
      if (parsed) onEvent(parsed);
      separatorIndex = buffer.indexOf('\n\n');
    }
  }
}

function parseEventChunk(chunk: string): SSEEventMessage | null {
  let event = 'message';
  const dataLines: string[] = [];

  for (const rawLine of chunk.split('\n')) {
    if (!rawLine || rawLine.startsWith(':')) continue;

    const separatorIndex = rawLine.indexOf(':');
    const field = separatorIndex >= 0 ? rawLine.slice(0, separatorIndex) : rawLine;
    let value = separatorIndex >= 0 ? rawLine.slice(separatorIndex + 1) : '';
    if (value.startsWith(' ')) {
      value = value.slice(1);
    }

    switch (field) {
      case 'event':
        if (value) event = value;
        break;
      case 'data':
        dataLines.push(value);
        break;
      default:
        break;
    }
  }

  if (dataLines.length === 0) return null;

  const rawData = dataLines.join('\n');
  try {
    return { event, data: JSON.parse(rawData) };
  } catch {
    return { event, data: rawData };
  }
}
