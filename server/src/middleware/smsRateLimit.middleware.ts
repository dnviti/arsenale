import rateLimit from 'express-rate-limit';

export const smsRateLimiter = rateLimit({
  windowMs: 10 * 60 * 1000, // 10 minutes
  max: 3,
  message: { error: 'Too many SMS requests. Please try again later.' },
  standardHeaders: true,
  legacyHeaders: false,
  keyGenerator: (req) => {
    const authReq = req as { user?: { userId: string } };
    return `sms:${authReq.user?.userId ?? req.ip}`;
  },
});

export const smsLoginRateLimiter = rateLimit({
  windowMs: 10 * 60 * 1000,
  max: 3,
  message: { error: 'Too many SMS requests. Please try again later.' },
  standardHeaders: true,
  legacyHeaders: false,
});
