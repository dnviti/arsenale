import prisma, { DbQueryType, RateLimitAction, Prisma } from '../lib/prisma';
import { config } from '../config';
import { logger } from '../utils/logger';
import { AppError } from '../middleware/error.middleware';

const log = logger.child('db-rate-limit');

// ---- Types ----

export interface RateLimitPolicyInput {
  tenantId: string;
  name: string;
  queryType?: DbQueryType | null;
  windowMs?: number;
  maxQueries?: number;
  burstMax?: number;
  exemptRoles?: string[];
  scope?: string;
  action?: RateLimitAction;
  enabled?: boolean;
  priority?: number;
}

export interface RateLimitPolicy {
  id: string;
  tenantId: string;
  name: string;
  queryType: DbQueryType | null;
  windowMs: number;
  maxQueries: number;
  burstMax: number;
  exemptRoles: string[];
  scope: string | null;
  action: RateLimitAction;
  enabled: boolean;
  priority: number;
  createdAt: Date;
  updatedAt: Date;
}

export interface RateLimitEvaluation {
  allowed: boolean;
  policy: RateLimitPolicy | null;
  remaining: number;
  retryAfterMs: number;
}

// ---- Token bucket implementation ----

interface TokenBucket {
  tokens: number;
  lastRefillTime: number;
  windowMs: number;
  maxTokens: number;
  refillRate: number; // tokens per ms
}

/**
 * In-memory token bucket map.
 * Key format: `userId:tenantId:queryType:policyId`
 */
const buckets = new Map<string, TokenBucket>();

/**
 * Cleanup interval handle — used to periodically sweep expired buckets.
 */
let cleanupTimer: ReturnType<typeof setInterval> | null = null;

function getBucketKey(userId: string, tenantId: string, queryType: string, policyId: string): string {
  return `${userId}:${tenantId}:${queryType}:${policyId}`;
}

function getOrCreateBucket(key: string, policy: RateLimitPolicy): TokenBucket {
  const existing = buckets.get(key);
  if (existing) {
    // Refill tokens based on elapsed time
    const now = Date.now();
    const elapsed = now - existing.lastRefillTime;
    const refillAmount = elapsed * existing.refillRate;
    existing.tokens = Math.min(existing.maxTokens, existing.tokens + refillAmount);
    existing.lastRefillTime = now;

    // Ensure bucket parameters reflect the latest policy so changes take effect immediately
    existing.windowMs = policy.windowMs;
    existing.maxTokens = policy.burstMax;
    existing.refillRate = policy.maxQueries / policy.windowMs;
    if (existing.tokens > existing.maxTokens) {
      existing.tokens = existing.maxTokens;
    }
    return existing;
  }

  const bucket: TokenBucket = {
    tokens: policy.burstMax,
    lastRefillTime: Date.now(),
    windowMs: policy.windowMs,
    maxTokens: policy.burstMax,
    refillRate: policy.maxQueries / policy.windowMs,
  };
  buckets.set(key, bucket);
  return bucket;
}

function clearBucketsForPolicy(policyId: string): void {
  for (const key of buckets.keys()) {
    if (key.endsWith(`:${policyId}`)) {
      buckets.delete(key);
    }
  }
}

function consumeToken(bucket: TokenBucket): boolean {
  if (bucket.tokens >= 1) {
    bucket.tokens -= 1;
    return true;
  }
  return false;
}

function calculateRetryAfterMs(bucket: TokenBucket): number {
  if (bucket.tokens >= 1) return 0;
  // Time until 1 token is available
  const tokensNeeded = 1 - bucket.tokens;
  return Math.ceil(tokensNeeded / bucket.refillRate);
}

// ---- Periodic cleanup ----

function cleanupExpiredBuckets(): void {
  const now = Date.now();
  for (const [key, bucket] of buckets) {
    // Remove buckets that haven't been used for 2x the window duration
    const idleTime = now - bucket.lastRefillTime;
    if (idleTime > bucket.windowMs * 2) {
      buckets.delete(key);
    }
  }
}

/** Start the periodic cleanup timer (idempotent). */
export function startCleanup(): void {
  if (cleanupTimer) return;
  cleanupTimer = setInterval(cleanupExpiredBuckets, config.dbRateLimitCleanupIntervalMs);
  // Ensure the timer doesn't prevent Node from exiting
  if (cleanupTimer && typeof cleanupTimer === 'object' && 'unref' in cleanupTimer) {
    cleanupTimer.unref();
  }
}

/** Stop the periodic cleanup timer. */
export function stopCleanup(): void {
  if (cleanupTimer) {
    clearInterval(cleanupTimer);
    cleanupTimer = null;
  }
}

// Auto-start cleanup on module load
startCleanup();

// ---- Evaluation ----

/**
 * Evaluate rate limits for a query.
 * Returns the evaluation result indicating whether the query is allowed
 * and which policy matched (if any).
 *
 * Policies are evaluated in priority order (higher priority first).
 * Only the first matching policy is applied.
 */
export async function evaluateRateLimit(
  userId: string,
  tenantId: string,
  queryType: DbQueryType,
  tenantRole?: string,
  database?: string,
  table?: string,
): Promise<RateLimitEvaluation> {
  try {
    const policies = await prisma.dbRateLimitPolicy.findMany({
      where: { tenantId, enabled: true },
      orderBy: { priority: 'desc' },
    });

    for (const policy of policies) {
      // Check if policy applies to this query type
      if (policy.queryType !== null && policy.queryType !== queryType) {
        continue;
      }

      // Check scope matching (mirrors sqlFirewall.matchesRule): match against database or table
      if (policy.scope) {
        const scopeLower = policy.scope.toLowerCase();
        const dbMatch = database && database.toLowerCase() === scopeLower;
        const tableMatch = table && table.toLowerCase() === scopeLower;
        if (!dbMatch && !tableMatch) continue;
      }

      // Check role exemptions
      if (tenantRole && policy.exemptRoles.length > 0) {
        if (policy.exemptRoles.includes(tenantRole)) {
          continue;
        }
      }

      // Found a matching policy — evaluate token bucket
      const bucketKey = getBucketKey(userId, tenantId, policy.queryType ?? 'ALL', policy.id);
      const bucket = getOrCreateBucket(bucketKey, policy);
      const consumed = consumeToken(bucket);

      if (!consumed) {
        const retryAfterMs = calculateRetryAfterMs(bucket);
        return {
          allowed: policy.action === 'LOG_ONLY',
          policy,
          remaining: Math.max(0, Math.floor(bucket.tokens)),
          retryAfterMs,
        };
      }

      return {
        allowed: true,
        policy,
        remaining: Math.max(0, Math.floor(bucket.tokens)),
        retryAfterMs: 0,
      };
    }
  } catch (err) {
    log.error('Rate limit evaluation error — allowing query as fallback:', err instanceof Error ? err.message : 'Unknown error');
  }

  return { allowed: true, policy: null, remaining: -1, retryAfterMs: 0 };
}

// ---- CRUD operations ----

export async function listPolicies(tenantId: string): Promise<RateLimitPolicy[]> {
  return prisma.dbRateLimitPolicy.findMany({
    where: { tenantId },
    orderBy: [{ priority: 'desc' }, { createdAt: 'desc' }],
  });
}

export async function getPolicy(tenantId: string, policyId: string): Promise<RateLimitPolicy | null> {
  return prisma.dbRateLimitPolicy.findFirst({
    where: { id: policyId, tenantId },
  });
}

function validatePolicyValues(windowMs?: number, maxQueries?: number, burstMax?: number): void {
  if (windowMs !== undefined && windowMs < 1) {
    throw new AppError('windowMs must be at least 1', 400);
  }
  if (maxQueries !== undefined && maxQueries < 1) {
    throw new AppError('maxQueries must be at least 1', 400);
  }
  if (burstMax !== undefined && burstMax < 1) {
    throw new AppError('burstMax must be at least 1', 400);
  }
}

export async function createPolicy(input: RateLimitPolicyInput): Promise<RateLimitPolicy> {
  validatePolicyValues(input.windowMs, input.maxQueries, input.burstMax);

  // Application-level uniqueness check (@@unique doesn't work with NULLs in Postgres)
  const duplicate = await prisma.dbRateLimitPolicy.findFirst({
    where: {
      tenantId: input.tenantId,
      queryType: input.queryType ?? null,
      scope: input.scope ?? null,
    },
  });
  if (duplicate) {
    throw new AppError('A rate limit policy already exists for this tenant/queryType/scope combination', 409);
  }

  return prisma.dbRateLimitPolicy.create({
    data: {
      tenantId: input.tenantId,
      name: input.name,
      queryType: input.queryType ?? null,
      windowMs: input.windowMs ?? config.dbRateLimitDefaultWindowMs,
      maxQueries: input.maxQueries ?? config.dbRateLimitDefaultMaxQueries,
      burstMax: input.burstMax ?? 10,
      exemptRoles: input.exemptRoles ?? [],
      scope: input.scope ?? null,
      action: input.action ?? 'REJECT',
      enabled: input.enabled ?? true,
      priority: input.priority ?? 0,
    },
  });
}

export async function updatePolicy(
  tenantId: string,
  policyId: string,
  updates: Partial<Omit<RateLimitPolicyInput, 'tenantId'>>,
): Promise<RateLimitPolicy> {
  const existing = await prisma.dbRateLimitPolicy.findFirst({ where: { id: policyId, tenantId } });
  if (!existing) throw new AppError('Rate limit policy not found', 404);

  validatePolicyValues(updates.windowMs, updates.maxQueries, updates.burstMax);

  const data: Prisma.DbRateLimitPolicyUpdateInput = {};
  if (updates.name !== undefined) data.name = updates.name;
  if (updates.queryType !== undefined) data.queryType = updates.queryType ?? null;
  if (updates.windowMs !== undefined) data.windowMs = updates.windowMs;
  if (updates.maxQueries !== undefined) data.maxQueries = updates.maxQueries;
  if (updates.burstMax !== undefined) data.burstMax = updates.burstMax;
  if (updates.exemptRoles !== undefined) data.exemptRoles = updates.exemptRoles ?? [];
  if (updates.scope !== undefined) data.scope = updates.scope ?? null;
  if (updates.action !== undefined) data.action = updates.action;
  if (updates.enabled !== undefined) data.enabled = updates.enabled;
  if (updates.priority !== undefined) data.priority = updates.priority;

  return prisma.dbRateLimitPolicy.update({
    where: { id: policyId },
    data,
  });
}

export async function deletePolicy(tenantId: string, policyId: string): Promise<void> {
  const existing = await prisma.dbRateLimitPolicy.findFirst({ where: { id: policyId, tenantId } });
  if (!existing) throw new AppError('Rate limit policy not found', 404);

  await prisma.dbRateLimitPolicy.delete({ where: { id: policyId } });

  // Clear in-memory buckets for this policy so stale state doesn't linger
  clearBucketsForPolicy(policyId);
}
