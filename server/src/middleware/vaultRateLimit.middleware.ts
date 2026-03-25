import type { Request, Response, NextFunction } from 'express';
import { config } from '../config';
import { createRateLimiter } from './rateLimitFactory';

let _unlockLimiter = createRateLimiter({
  windowMs: config.vaultRateLimitWindowMs,
  max: config.vaultRateLimitMaxAttempts,
  message: 'Too many vault unlock attempts. Please try again later.',
  keyPrefix: 'vault',
});

let _mfaLimiter = createRateLimiter({
  windowMs: config.vaultRateLimitWindowMs,
  max: config.vaultMfaRateLimitMaxAttempts,
  message: 'Too many vault unlock attempts. Please try again later.',
  keyPrefix: 'vault-mfa',
});

export function vaultUnlockRateLimiter(req: Request, res: Response, next: NextFunction) {
  _unlockLimiter(req, res, next);
}

export function vaultMfaRateLimiter(req: Request, res: Response, next: NextFunction) {
  _mfaLimiter(req, res, next);
}

/** Rebuild both vault rate limiters with current config values. */
export function rebuildVaultRateLimiters(): void {
  _unlockLimiter = createRateLimiter({
    windowMs: config.vaultRateLimitWindowMs,
    max: config.vaultRateLimitMaxAttempts,
    message: 'Too many vault unlock attempts. Please try again later.',
    keyPrefix: 'vault',
  });
  _mfaLimiter = createRateLimiter({
    windowMs: config.vaultRateLimitWindowMs,
    max: config.vaultMfaRateLimitMaxAttempts,
    message: 'Too many vault unlock attempts. Please try again later.',
    keyPrefix: 'vault-mfa',
  });
}
