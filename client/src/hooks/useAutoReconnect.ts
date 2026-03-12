import { useCallback, useEffect, useRef, useState } from 'react';

interface UseAutoReconnectOptions {
  maxRetries?: number;
  baseDelayMs?: number;
  maxDelayMs?: number;
  totalTimeoutMs?: number;
}

interface UseAutoReconnectReturn {
  reconnectState: 'idle' | 'reconnecting' | 'failed';
  attempt: number;
  maxRetries: number;
  triggerReconnect: () => void;
  cancelReconnect: () => void;
  resetReconnect: () => void;
}

const DEFAULTS = {
  maxRetries: 5,
  baseDelayMs: 1000,
  maxDelayMs: 15000,
  totalTimeoutMs: 60000,
};

export function useAutoReconnect(
  connectFn: () => Promise<void>,
  options?: UseAutoReconnectOptions,
): UseAutoReconnectReturn {
  const {
    maxRetries = DEFAULTS.maxRetries,
    baseDelayMs = DEFAULTS.baseDelayMs,
    maxDelayMs = DEFAULTS.maxDelayMs,
    totalTimeoutMs = DEFAULTS.totalTimeoutMs,
  } = options ?? {};

  const [state, setState] = useState<'idle' | 'reconnecting' | 'failed'>('idle');
  const [attempt, setAttempt] = useState(0);

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const startTimeRef = useRef<number>(0);
  const attemptRef = useRef(0);
  const connectFnRef = useRef(connectFn);
  const cancelledRef = useRef(false);
  const scheduleRef = useRef<(currentAttempt: number) => void>(null);

  // Keep connectFn ref up to date
  useEffect(() => {
    connectFnRef.current = connectFn;
  }, [connectFn]);

  const clearTimer = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  // Store schedule function in a ref to allow recursive calls without useCallback circular dep
  useEffect(() => {
    scheduleRef.current = (currentAttempt: number) => {
      const elapsed = Date.now() - startTimeRef.current;
      if (currentAttempt >= maxRetries || elapsed >= totalTimeoutMs) {
        setState('failed');
        return;
      }

      // Exponential backoff with jitter
      const delay = Math.min(
        baseDelayMs * Math.pow(2, currentAttempt) + Math.random() * 500,
        maxDelayMs,
      );

      timerRef.current = setTimeout(async () => {
        if (cancelledRef.current) return;

        attemptRef.current = currentAttempt + 1;
        setAttempt(currentAttempt + 1);

        try {
          await connectFnRef.current();
          // Success — caller should call resetReconnect() when connection is confirmed
        } catch {
          if (!cancelledRef.current) {
            scheduleRef.current?.(currentAttempt + 1);
          }
        }
      }, delay);
    };
  }, [maxRetries, baseDelayMs, maxDelayMs, totalTimeoutMs]);

  const triggerReconnect = useCallback(() => {
    if (state === 'reconnecting') return; // Already reconnecting
    cancelledRef.current = false;
    startTimeRef.current = Date.now();
    attemptRef.current = 0;
    setAttempt(0);
    setState('reconnecting');
    scheduleRef.current?.(0);
  }, [state]);

  const cancelReconnect = useCallback(() => {
    cancelledRef.current = true;
    clearTimer();
    setState('idle');
    setAttempt(0);
    attemptRef.current = 0;
  }, [clearTimer]);

  const resetReconnect = useCallback(() => {
    cancelledRef.current = false;
    clearTimer();
    setState('idle');
    setAttempt(0);
    attemptRef.current = 0;
  }, [clearTimer]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      cancelledRef.current = true;
      clearTimer();
    };
  }, [clearTimer]);

  return {
    reconnectState: state,
    attempt,
    maxRetries,
    triggerReconnect,
    cancelReconnect,
    resetReconnect,
  };
}
