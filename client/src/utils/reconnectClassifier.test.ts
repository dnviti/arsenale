import { isGuacPermanentError, isSshPermanentError, isTransientDisconnect } from './reconnectClassifier';

describe('isGuacPermanentError', () => {
  it('returns true for STATUS_CODE 0x0203 (SESSION_CLOSED)', () => {
    expect(isGuacPermanentError('', 0x0203)).toBe(true);
  });

  it('returns true for status code 0x0301 (UNAUTHORIZED)', () => {
    expect(isGuacPermanentError('', 0x0301)).toBe(true);
  });

  it('returns true for status code 0x0308 (SESSION_TIMEOUT)', () => {
    expect(isGuacPermanentError('', 0x0308)).toBe(true);
  });

  it('falls back to message check for status code 0x0200 (not in set)', () => {
    expect(isGuacPermanentError('connection lost', 0x0200)).toBe(false);
    expect(isGuacPermanentError('unauthorized access', 0x0200)).toBe(true);
  });

  it('returns true for "Terminated by administrator" without status code (case insensitive)', () => {
    expect(isGuacPermanentError('Terminated by administrator')).toBe(true);
  });

  it('returns true for "Session expired" without status code', () => {
    expect(isGuacPermanentError('Session expired')).toBe(true);
  });

  it('returns true for "unauthorized access" without status code', () => {
    expect(isGuacPermanentError('unauthorized access')).toBe(true);
  });

  it('returns false for "connection lost" without status code (not a permanent pattern)', () => {
    expect(isGuacPermanentError('connection lost')).toBe(false);
  });

  it('returns false for empty message without status code', () => {
    expect(isGuacPermanentError('')).toBe(false);
  });

  it('returns true when permanent code is present even with non-matching message (code takes precedence)', () => {
    expect(isGuacPermanentError('connection lost', 0x0203)).toBe(true);
  });
});

describe('isSshPermanentError', () => {
  it('returns true for session:timeout', () => {
    expect(isSshPermanentError('session:timeout')).toBe(true);
  });

  it('returns true for session:terminated', () => {
    expect(isSshPermanentError('session:terminated')).toBe(true);
  });

  it('returns true for session:error (always permanent)', () => {
    expect(isSshPermanentError('session:error')).toBe(true);
  });

  it('returns true for connect_error with "invalid token"', () => {
    expect(isSshPermanentError('connect_error', { message: 'invalid token' })).toBe(true);
  });

  it('returns true for connect_error with "Authentication failed"', () => {
    expect(isSshPermanentError('connect_error', { message: 'Authentication failed' })).toBe(true);
  });

  it('returns false for connect_error with "ECONNREFUSED" (transient)', () => {
    expect(isSshPermanentError('connect_error', { message: 'ECONNREFUSED' })).toBe(false);
  });

  it('returns false for connect_error with no data', () => {
    expect(isSshPermanentError('connect_error')).toBe(false);
  });

  it('returns false for disconnect', () => {
    expect(isSshPermanentError('disconnect')).toBe(false);
  });

  it('returns false for data', () => {
    expect(isSshPermanentError('data')).toBe(false);
  });
});

describe('isTransientDisconnect', () => {
  it('returns true for "transport close"', () => {
    expect(isTransientDisconnect('transport close')).toBe(true);
  });

  it('returns true for "transport error"', () => {
    expect(isTransientDisconnect('transport error')).toBe(true);
  });

  it('returns true for "ping timeout"', () => {
    expect(isTransientDisconnect('ping timeout')).toBe(true);
  });

  it('returns false for "io server disconnect"', () => {
    expect(isTransientDisconnect('io server disconnect')).toBe(false);
  });

  it('returns false for "io client disconnect"', () => {
    expect(isTransientDisconnect('io client disconnect')).toBe(false);
  });

  it('returns false for empty string', () => {
    expect(isTransientDisconnect('')).toBe(false);
  });
});
