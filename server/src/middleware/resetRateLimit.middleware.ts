import { createRateLimiter } from './rateLimitFactory';

const FIFTEEN_MINUTES_MS = 15 * 60 * 1000;
const TEN_MINUTES_MS = 10 * 60 * 1000;

export const forgotPasswordLimiter = createRateLimiter({
  windowMs: FIFTEEN_MINUTES_MS,
  max: 3,
  message: 'Too many password reset requests. Please try again later.',
});

export const resetPasswordLimiter = createRateLimiter({
  windowMs: FIFTEEN_MINUTES_MS,
  max: 5,
  message: 'Too many attempts. Please try again later.',
});

export const resetSmsLimiter = createRateLimiter({
  windowMs: TEN_MINUTES_MS,
  max: 3,
  message: 'Too many SMS requests. Please try again later.',
});
