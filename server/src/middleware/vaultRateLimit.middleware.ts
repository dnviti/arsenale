import { config } from '../config';
import { createRateLimiter } from './rateLimitFactory';

export const vaultUnlockRateLimiter = createRateLimiter({
  windowMs: config.vaultRateLimitWindowMs,
  max: config.vaultRateLimitMaxAttempts,
  message: 'Too many vault unlock attempts. Please try again later.',
  keyPrefix: 'vault',
});

export const vaultMfaRateLimiter = createRateLimiter({
  windowMs: config.vaultRateLimitWindowMs,
  max: config.vaultMfaRateLimitMaxAttempts,
  message: 'Too many vault unlock attempts. Please try again later.',
  keyPrefix: 'vault-mfa',
});
