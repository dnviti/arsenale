import crypto from 'crypto';
import { config } from '../config';

/**
 * HMAC-sign a JSON payload for OAuth/SAML state parameters.
 * Prevents user-controlled bypass by ensuring state integrity.
 */
export function signState(payload: Record<string, unknown>): string {
  const json = JSON.stringify(payload);
  const data = Buffer.from(json).toString('base64url');
  const hmac = crypto.createHmac('sha256', config.jwtSecret).update(data).digest('base64url');
  return `${data}.${hmac}`;
}

/**
 * Verify and decode an HMAC-signed state token.
 * Returns the parsed payload or null if the signature is invalid.
 */
export function verifyState<T = Record<string, unknown>>(token: string | string[]): T | null {
  // Reject arrays (e.g. from query param tampering like ?state=a&state=b)
  if (Array.isArray(token)) return null;
  if (typeof token !== 'string') return null;
  const dotIndex = token.indexOf('.');
  if (dotIndex === -1) return null;

  const data = token.substring(0, dotIndex);
  const signature = token.substring(dotIndex + 1);

  const expected = crypto.createHmac('sha256', config.jwtSecret).update(data).digest('base64url');

  if (signature.length !== expected.length) return null;
  if (!crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expected))) return null;

  try {
    return JSON.parse(Buffer.from(data, 'base64url').toString()) as T;
  } catch {
    return null;
  }
}
