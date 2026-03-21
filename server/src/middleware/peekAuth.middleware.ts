import type { Response, NextFunction } from 'express';
import type { AuthPayload, AuthRequest } from '../types';
import { verifyJwt } from '../utils/jwt';

/**
 * Lightweight auth peek — attempts to extract user identity from the
 * Authorization header WITHOUT rejecting unauthenticated requests.
 *
 * Used before the global rate limiter so authenticated requests can be
 * keyed by userId (higher limit) instead of IP (lower limit).
 *
 * This does NOT replace the per-route `authenticate` middleware which
 * enforces authentication and returns 401 on failure.
 */
export function peekAuth(req: AuthRequest, _res: Response, next: NextFunction): void {
  const authHeader = req.headers.authorization;
  if (authHeader?.startsWith('Bearer ')) {
    try {
      req.user = verifyJwt<AuthPayload>(authHeader.slice(7));
    } catch {
      // Token invalid/expired — treat as anonymous for rate limiting
    }
  }
  next();
}
