import { createRateLimiter } from './rateLimitFactory';

const SMS_WINDOW_MS = 10 * 60 * 1000; // 10 minutes

export const smsRateLimiter = createRateLimiter({
  windowMs: SMS_WINDOW_MS,
  max: 3,
  message: 'Too many SMS requests. Please try again later.',
  keyPrefix: 'sms',
});

export const smsLoginRateLimiter = createRateLimiter({
  windowMs: SMS_WINDOW_MS,
  max: 3,
  message: 'Too many SMS requests. Please try again later.',
});
