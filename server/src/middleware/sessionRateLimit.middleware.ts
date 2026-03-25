import type { Request, Response, NextFunction } from 'express';
import { config } from '../config';
import { createRateLimiter } from './rateLimitFactory';

let _limiter = createRateLimiter({
  windowMs: config.sessionRateLimitWindowMs,
  max: config.sessionRateLimitMaxAttempts,
  message: 'Too many session requests. Please try again later.',
  keyPrefix: 'session',
});

export function sessionRateLimiter(req: Request, res: Response, next: NextFunction) {
  _limiter(req, res, next);
}

/** Rebuild session rate limiter with current config values. */
export function rebuildSessionRateLimiter(): void {
  _limiter = createRateLimiter({
    windowMs: config.sessionRateLimitWindowMs,
    max: config.sessionRateLimitMaxAttempts,
    message: 'Too many session requests. Please try again later.',
    keyPrefix: 'session',
  });
}
