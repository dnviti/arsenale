import { config } from '../config';
import { createRateLimiter } from './rateLimitFactory';

export const sessionRateLimiter = createRateLimiter({
  windowMs: config.sessionRateLimitWindowMs,
  max: config.sessionRateLimitMaxAttempts,
  message: 'Too many session requests. Please try again later.',
  keyPrefix: 'session',
});
