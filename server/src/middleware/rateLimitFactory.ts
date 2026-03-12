import rateLimit, { type Options, ipKeyGenerator } from 'express-rate-limit';

interface RateLimitOpts {
  windowMs: number;
  max: number;
  message: string;
  /** If provided, keys rate limit by authenticated userId with this prefix (falls back to IP). */
  keyPrefix?: string;
  /** Extra options passed through to express-rate-limit. */
  extra?: Partial<Options>;
}

/** Create a rate limiter with shared defaults (standardHeaders, legacyHeaders). */
export function createRateLimiter({ windowMs, max, message, keyPrefix, extra }: RateLimitOpts) {
  return rateLimit({
    windowMs,
    max,
    message: { error: message },
    standardHeaders: true,
    legacyHeaders: false,
    ...(keyPrefix && {
      keyGenerator: (req) => {
        const authReq = req as { user?: { userId: string } };
        if (authReq.user?.userId) return `${keyPrefix}:${authReq.user.userId}`;
        return `${keyPrefix}:${ipKeyGenerator(req.ip ?? '127.0.0.1')}`;
      },
    }),
    ...extra,
  });
}
