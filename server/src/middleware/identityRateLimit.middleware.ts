import { createRateLimiter } from './rateLimitFactory';

export const identityVerificationLimiter = createRateLimiter({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 3,
  message: 'Too many verification requests. Please try again later.',
  extra: {
    // Keyed by authenticated userId (this runs after authenticate middleware).
    // The IP fallback is a safety net — suppress the IPv6 validation warning.
    validate: { keyGeneratorIpFallback: false },
    keyGenerator: (req) => {
      const authReq = req as { user?: { userId: string } };
      return authReq.user?.userId ?? req.ip ?? '127.0.0.1';
    },
  },
});
