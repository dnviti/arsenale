import type { Request, Response, NextFunction } from 'express';
import { config } from '../config';
import { createRateLimiter } from './rateLimitFactory';

let _limiter = createRateLimiter({
  windowMs: config.loginRateLimitWindowMs,
  max: config.loginRateLimitMaxAttempts,
  message: 'Too many login attempts. Please try again later.',
});

export function loginRateLimiter(req: Request, res: Response, next: NextFunction) {
  _limiter(req, res, next);
}

/** Rebuild login rate limiter with current config values. */
export function rebuildLoginRateLimiter(): void {
  _limiter = createRateLimiter({
    windowMs: config.loginRateLimitWindowMs,
    max: config.loginRateLimitMaxAttempts,
    message: 'Too many login attempts. Please try again later.',
  });
}
