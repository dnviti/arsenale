import { config } from '../config';
import { createRateLimiter } from './rateLimitFactory';

export const loginRateLimiter = createRateLimiter({
  windowMs: config.loginRateLimitWindowMs,
  max: config.loginRateLimitMaxAttempts,
  message: 'Too many login attempts. Please try again later.',
});
