import type { Request, Response, NextFunction } from 'express';
import { config } from '../config';
import { createRateLimiter } from './rateLimitFactory';

let _flowLimiter = createRateLimiter({
  windowMs: config.oauthFlowRateLimitWindowMs,
  max: config.oauthFlowRateLimitMaxAttempts,
  message: 'Too many OAuth requests. Please try again later.',
});

let _linkLimiter = createRateLimiter({
  windowMs: config.oauthLinkRateLimitWindowMs,
  max: config.oauthLinkRateLimitMaxAttempts,
  message: 'Too many account linking attempts. Please try again later.',
});

let _acctLimiter = createRateLimiter({
  windowMs: config.oauthAccountRateLimitWindowMs,
  max: config.oauthAccountRateLimitMaxAttempts,
  message: 'Too many OAuth account requests. Please try again later.',
  keyPrefix: 'oauth-account',
});

export function oauthFlowRateLimiter(req: Request, res: Response, next: NextFunction) {
  _flowLimiter(req, res, next);
}

export function oauthLinkRateLimiter(req: Request, res: Response, next: NextFunction) {
  _linkLimiter(req, res, next);
}

export function oauthAccountRateLimiter(req: Request, res: Response, next: NextFunction) {
  _acctLimiter(req, res, next);
}

/** Rebuild all OAuth rate limiters with current config values. */
export function rebuildOauthRateLimiters(): void {
  _flowLimiter = createRateLimiter({
    windowMs: config.oauthFlowRateLimitWindowMs,
    max: config.oauthFlowRateLimitMaxAttempts,
    message: 'Too many OAuth requests. Please try again later.',
  });
  _linkLimiter = createRateLimiter({
    windowMs: config.oauthLinkRateLimitWindowMs,
    max: config.oauthLinkRateLimitMaxAttempts,
    message: 'Too many account linking attempts. Please try again later.',
  });
  _acctLimiter = createRateLimiter({
    windowMs: config.oauthAccountRateLimitWindowMs,
    max: config.oauthAccountRateLimitMaxAttempts,
    message: 'Too many OAuth account requests. Please try again later.',
    keyPrefix: 'oauth-account',
  });
}
