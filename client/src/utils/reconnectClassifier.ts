/**
 * Error classification utilities for auto-reconnect logic.
 * Distinguishes transient (retryable) disconnections from permanent errors.
 */

// Guacamole status codes that indicate permanent failure (no retry)
const GUAC_PERMANENT_CODES = new Set([
  0x0203, // SESSION_CLOSED
  0x0301, // UNAUTHORIZED
  0x0308, // SESSION_TIMEOUT
]);

const PERMANENT_ERROR_PATTERNS = [
  'terminated by administrator',
  'session expired',
  'session timeout',
  'unauthorized',
  'authentication',
  'permission denied',
  'access denied',
];

/**
 * Returns true if a Guacamole error is permanent and should NOT trigger reconnection.
 */
export function isGuacPermanentError(errorMessage: string, statusCode?: number): boolean {
  if (statusCode !== undefined && GUAC_PERMANENT_CODES.has(statusCode)) {
    return true;
  }
  const lower = errorMessage.toLowerCase();
  return PERMANENT_ERROR_PATTERNS.some((p) => lower.includes(p));
}

/**
 * Returns true if an SSH Socket.IO event represents a permanent error (no retry).
 */
export function isSshPermanentError(eventName: string, data?: { message?: string }): boolean {
  // These events are always permanent
  if (eventName === 'session:timeout' || eventName === 'session:terminated') {
    return true;
  }
  if (eventName === 'session:error') {
    // SSH session errors (auth failure, invalid connection, etc.) are permanent
    return true;
  }
  if (eventName === 'connect_error' && data?.message) {
    const lower = data.message.toLowerCase();
    if (lower.includes('invalid token') || lower.includes('authentication')) {
      return true;
    }
  }
  return false;
}

/**
 * Returns true if a Socket.IO disconnect reason indicates a transient failure (retryable).
 */
export function isTransientDisconnect(reason: string): boolean {
  return (
    reason === 'transport close' ||
    reason === 'transport error' ||
    reason === 'ping timeout'
  );
}
