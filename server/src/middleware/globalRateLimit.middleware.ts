import rateLimit, { ipKeyGenerator } from 'express-rate-limit';
import type { Request } from 'express';
import type { AuthRequest } from '../types';
import { config } from '../config';
import { isIpAllowed } from '../utils/ipAllowlist';

const env = (key: string, fallback: number) =>
  Number(process.env[key]) || fallback;

/**
 * Global API rate limiter applied to all /api routes.
 *
 * Tiers:
 * 1. Whitelisted IPs (loopback, RFC 1918 by default): skip rate limiting entirely
 * 2. Authenticated requests: keyed by userId (200 req / 60 s default)
 * 3. Unauthenticated requests: keyed by IP   (60 req / 60 s default)
 *
 * Requires `peekAuth` middleware to run first so `req.user` is populated
 * for authenticated requests.
 *
 * Per-route limiters (login, vault, sessions, etc.) still apply on top
 * of this and are typically stricter.
 */
export const globalRateLimit = rateLimit({
  windowMs: env('GLOBAL_RATE_LIMIT_WINDOW_MS', 60_000),
  max: (req: Request) => {
    const authReq = req as AuthRequest;
    return authReq.user?.userId
      ? env('GLOBAL_RATE_LIMIT_MAX_AUTHENTICATED', 200)
      : env('GLOBAL_RATE_LIMIT_MAX_ANONYMOUS', 60);
  },
  keyGenerator: (req: Request) => {
    const authReq = req as AuthRequest;
    if (authReq.user?.userId) return `global:${authReq.user.userId}`;
    return `global:${ipKeyGenerator(req.ip ?? '127.0.0.1')}`;
  },
  message: { error: 'Too many requests. Please try again later.' },
  standardHeaders: true,
  legacyHeaders: false,
  skipSuccessfulRequests: false,
  skip: (req: Request) => {
    // Never rate-limit health probes
    if (req.path === '/health' || req.path === '/ready') return true;
    // Skip whitelisted IPs (loopback + private ranges by default)
    const clientIp = req.ip ?? '127.0.0.1';
    if (config.rateLimitWhitelistCidrs.length > 0 &&
        isIpAllowed(clientIp, config.rateLimitWhitelistCidrs)) {
      return true;
    }
    return false;
  },
});
