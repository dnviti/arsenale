import prisma, { CheckoutStatus, Prisma } from '../lib/prisma';
import { AppError } from '../middleware/error.middleware';
import { createNotificationAsync } from './notification.service';
import { emitNotification } from '../socket/notification.handler';
import * as auditService from './audit.service';
import { logger } from '../utils/logger';

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface CheckoutRequestInput {
  secretId?: string;
  connectionId?: string;
  durationMinutes: number;
  reason?: string;
}

export interface CheckoutRequestEntry {
  id: string;
  secretId: string | null;
  connectionId: string | null;
  requesterId: string;
  approverId: string | null;
  status: CheckoutStatus;
  durationMinutes: number;
  reason: string | null;
  expiresAt: Date | null;
  createdAt: Date;
  updatedAt: Date;
  requester: { email: string; username: string | null };
  approver?: { email: string; username: string | null } | null;
  secretName?: string | null;
  connectionName?: string | null;
}

export interface PaginatedCheckoutRequests {
  data: CheckoutRequestEntry[];
  total: number;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const checkoutSelect = {
  id: true,
  secretId: true,
  connectionId: true,
  requesterId: true,
  approverId: true,
  status: true,
  durationMinutes: true,
  reason: true,
  expiresAt: true,
  createdAt: true,
  updatedAt: true,
  requester: { select: { email: true, username: true } },
  approver: { select: { email: true, username: true } },
} as const;

function displayName(u: { username: string | null; email: string }): string {
  return u.username || u.email;
}

/**
 * Find OWNER/ADMIN users who can approve checkout requests for a given
 * secret or connection. Returns the owner of the secret/connection,
 * plus any tenant OWNER/ADMIN members.
 */
async function findApprovers(secretId?: string | null, connectionId?: string | null): Promise<string[]> {
  const approverIds = new Set<string>();

  if (secretId) {
    const secret = await prisma.vaultSecret.findUnique({
      where: { id: secretId },
      select: { userId: true, tenantId: true },
    });
    if (secret) {
      approverIds.add(secret.userId);
      if (secret.tenantId) {
        const admins = await prisma.tenantMember.findMany({
          where: { tenantId: secret.tenantId, role: { in: ['OWNER', 'ADMIN'] } },
          select: { userId: true },
        });
        for (const a of admins) approverIds.add(a.userId);
      }
    }
  }

  if (connectionId) {
    const conn = await prisma.connection.findUnique({
      where: { id: connectionId },
      select: { userId: true, teamId: true },
    });
    if (conn) {
      approverIds.add(conn.userId);
      // If the connection belongs to a team, add team admins
      if (conn.teamId) {
        const teamAdmins = await prisma.teamMember.findMany({
          where: { teamId: conn.teamId, role: 'TEAM_ADMIN' },
          select: { userId: true },
        });
        for (const a of teamAdmins) approverIds.add(a.userId);
      }
    }
  }

  return Array.from(approverIds);
}

// ---------------------------------------------------------------------------
// Service functions
// ---------------------------------------------------------------------------

/**
 * Request temporary checkout of a secret or connection.
 */
export async function requestCheckout(
  requesterId: string,
  input: CheckoutRequestInput,
  ipAddress?: string | string[],
): Promise<CheckoutRequestEntry> {
  if (!input.secretId && !input.connectionId) {
    throw new AppError('Either secretId or connectionId is required', 400);
  }
  if (input.secretId && input.connectionId) {
    throw new AppError('Provide either secretId or connectionId, not both', 400);
  }
  if (input.durationMinutes < 1 || input.durationMinutes > 1440) {
    throw new AppError('Duration must be between 1 and 1440 minutes (24h)', 400);
  }

  // Verify the target resource exists
  let targetName = '';
  if (input.secretId) {
    const secret = await prisma.vaultSecret.findUnique({
      where: { id: input.secretId },
      select: { id: true, name: true, userId: true },
    });
    if (!secret) throw new AppError('Secret not found', 404);
    if (secret.userId === requesterId) {
      throw new AppError('Cannot check out your own secret', 400);
    }
    targetName = secret.name;
  }
  if (input.connectionId) {
    const conn = await prisma.connection.findUnique({
      where: { id: input.connectionId },
      select: { id: true, name: true, userId: true },
    });
    if (!conn) throw new AppError('Connection not found', 404);
    if (conn.userId === requesterId) {
      throw new AppError('Cannot check out your own connection', 400);
    }
    targetName = conn.name;
  }

  // Check for existing pending request
  const existing = await prisma.secretCheckoutRequest.findFirst({
    where: {
      requesterId,
      status: 'PENDING',
      ...(input.secretId ? { secretId: input.secretId } : {}),
      ...(input.connectionId ? { connectionId: input.connectionId } : {}),
    },
  });
  if (existing) {
    throw new AppError('A pending checkout request already exists for this resource', 409);
  }

  const request = await prisma.secretCheckoutRequest.create({
    data: {
      secretId: input.secretId ?? null,
      connectionId: input.connectionId ?? null,
      requesterId,
      durationMinutes: input.durationMinutes,
      reason: input.reason ?? null,
    },
    select: checkoutSelect,
  });

  // Audit log
  auditService.log({
    userId: requesterId,
    action: 'SECRET_CHECKOUT_REQUESTED',
    targetType: input.secretId ? 'VaultSecret' : 'Connection',
    targetId: input.secretId ?? input.connectionId ?? undefined,
    details: {
      checkoutId: request.id,
      durationMinutes: input.durationMinutes,
      reason: input.reason,
    },
    ipAddress,
  });

  // Notify approvers
  const approverIds = await findApprovers(input.secretId, input.connectionId);
  const requesterName = displayName(request.requester);
  const resourceType = input.secretId ? 'secret' : 'connection';

  for (const approverId of approverIds) {
    if (approverId === requesterId) continue;
    const msg = `${requesterName} requests temporary access to ${resourceType} "${targetName}" for ${input.durationMinutes} minutes`;
    createNotificationAsync({
      userId: approverId,
      type: 'SECRET_CHECKOUT_REQUESTED',
      message: msg,
      relatedId: request.id,
    });
    emitNotification(approverId, {
      id: '',
      type: 'SECRET_CHECKOUT_REQUESTED',
      message: msg,
      read: false,
      relatedId: request.id,
      createdAt: new Date(),
    });
  }

  return {
    ...request,
    secretName: input.secretId ? targetName : null,
    connectionName: input.connectionId ? targetName : null,
  };
}

/**
 * Approve a pending checkout request. Creates a time-limited share.
 */
export async function approveCheckout(
  approverId: string,
  requestId: string,
  ipAddress?: string | string[],
): Promise<CheckoutRequestEntry> {
  const request = await prisma.secretCheckoutRequest.findUnique({
    where: { id: requestId },
    select: { ...checkoutSelect, secretId: true, connectionId: true, requesterId: true, durationMinutes: true, status: true },
  });
  if (!request) throw new AppError('Checkout request not found', 404);
  if (request.status !== 'PENDING') {
    throw new AppError(`Request is already ${request.status.toLowerCase()}`, 400);
  }

  // Verify the approver has authority
  const approverIds = await findApprovers(request.secretId, request.connectionId);
  if (!approverIds.includes(approverId)) {
    throw new AppError('You are not authorized to approve this request', 403);
  }

  const expiresAt = new Date(Date.now() + request.durationMinutes * 60 * 1000);

  const updated = await prisma.secretCheckoutRequest.update({
    where: { id: requestId },
    data: {
      status: 'APPROVED',
      approverId,
      expiresAt,
    },
    select: checkoutSelect,
  });

  // Audit log
  auditService.log({
    userId: approverId,
    action: 'SECRET_CHECKOUT_APPROVED',
    targetType: request.secretId ? 'VaultSecret' : 'Connection',
    targetId: request.secretId ?? request.connectionId ?? undefined,
    details: {
      checkoutId: requestId,
      requesterId: request.requesterId,
      durationMinutes: request.durationMinutes,
      expiresAt: expiresAt.toISOString(),
    },
    ipAddress,
  });

  // Notify requester
  const approverUser = await prisma.user.findUnique({
    where: { id: approverId },
    select: { username: true, email: true },
  });
  const approverName = approverUser ? displayName(approverUser) : 'An administrator';
  let targetName = '';

  if (request.secretId) {
    const secret = await prisma.vaultSecret.findUnique({ where: { id: request.secretId }, select: { name: true } });
    targetName = secret?.name ?? 'a secret';
  } else if (request.connectionId) {
    const conn = await prisma.connection.findUnique({ where: { id: request.connectionId }, select: { name: true } });
    targetName = conn?.name ?? 'a connection';
  }

  const resourceType = request.secretId ? 'secret' : 'connection';
  const msg = `${approverName} approved your checkout of ${resourceType} "${targetName}" for ${request.durationMinutes} minutes`;
  createNotificationAsync({
    userId: request.requesterId,
    type: 'SECRET_CHECKOUT_APPROVED',
    message: msg,
    relatedId: requestId,
  });
  emitNotification(request.requesterId, {
    id: '',
    type: 'SECRET_CHECKOUT_APPROVED',
    message: msg,
    read: false,
    relatedId: requestId,
    createdAt: new Date(),
  });

  return updated;
}

/**
 * Reject a pending checkout request.
 */
export async function rejectCheckout(
  approverId: string,
  requestId: string,
  ipAddress?: string | string[],
): Promise<CheckoutRequestEntry> {
  const request = await prisma.secretCheckoutRequest.findUnique({
    where: { id: requestId },
    select: { ...checkoutSelect, secretId: true, connectionId: true, requesterId: true, status: true },
  });
  if (!request) throw new AppError('Checkout request not found', 404);
  if (request.status !== 'PENDING') {
    throw new AppError(`Request is already ${request.status.toLowerCase()}`, 400);
  }

  // Verify the approver has authority
  const approverIds = await findApprovers(request.secretId, request.connectionId);
  if (!approverIds.includes(approverId)) {
    throw new AppError('You are not authorized to reject this request', 403);
  }

  const updated = await prisma.secretCheckoutRequest.update({
    where: { id: requestId },
    data: {
      status: 'REJECTED',
      approverId,
    },
    select: checkoutSelect,
  });

  // Audit log
  auditService.log({
    userId: approverId,
    action: 'SECRET_CHECKOUT_DENIED',
    targetType: request.secretId ? 'VaultSecret' : 'Connection',
    targetId: request.secretId ?? request.connectionId ?? undefined,
    details: {
      checkoutId: requestId,
      requesterId: request.requesterId,
    },
    ipAddress,
  });

  // Notify requester
  const approverUser = await prisma.user.findUnique({
    where: { id: approverId },
    select: { username: true, email: true },
  });
  const approverName = approverUser ? displayName(approverUser) : 'An administrator';
  let targetName = '';

  if (request.secretId) {
    const secret = await prisma.vaultSecret.findUnique({ where: { id: request.secretId }, select: { name: true } });
    targetName = secret?.name ?? 'a secret';
  } else if (request.connectionId) {
    const conn = await prisma.connection.findUnique({ where: { id: request.connectionId }, select: { name: true } });
    targetName = conn?.name ?? 'a connection';
  }

  const resourceType = request.secretId ? 'secret' : 'connection';
  const msg = `${approverName} denied your checkout of ${resourceType} "${targetName}"`;
  createNotificationAsync({
    userId: request.requesterId,
    type: 'SECRET_CHECKOUT_DENIED',
    message: msg,
    relatedId: requestId,
  });
  emitNotification(request.requesterId, {
    id: '',
    type: 'SECRET_CHECKOUT_DENIED',
    message: msg,
    read: false,
    relatedId: requestId,
    createdAt: new Date(),
  });

  return updated;
}

/**
 * Manually check in (return) a checked-out credential before expiry.
 */
export async function checkinCheckout(
  userId: string,
  requestId: string,
  ipAddress?: string | string[],
): Promise<CheckoutRequestEntry> {
  const request = await prisma.secretCheckoutRequest.findUnique({
    where: { id: requestId },
    select: checkoutSelect,
  });
  if (!request) throw new AppError('Checkout request not found', 404);
  if (request.status !== 'APPROVED') {
    throw new AppError('Only approved checkouts can be checked in', 400);
  }
  // Only the requester or an approver can check in
  if (request.requesterId !== userId) {
    const approverIds = await findApprovers(request.secretId, request.connectionId);
    if (!approverIds.includes(userId)) {
      throw new AppError('You are not authorized to check in this request', 403);
    }
  }

  const updated = await prisma.secretCheckoutRequest.update({
    where: { id: requestId },
    data: { status: 'CHECKED_IN' },
    select: checkoutSelect,
  });

  auditService.log({
    userId,
    action: 'SECRET_CHECKOUT_CHECKED_IN',
    targetType: request.secretId ? 'VaultSecret' : 'Connection',
    targetId: request.secretId ?? request.connectionId ?? undefined,
    details: { checkoutId: requestId },
    ipAddress,
  });

  return updated;
}

/**
 * List checkout requests for the current user (as requester or approver).
 */
export async function listCheckoutRequests(
  userId: string,
  role: 'requester' | 'approver' | 'all',
  status?: CheckoutStatus,
  limit = 50,
  offset = 0,
): Promise<PaginatedCheckoutRequests> {
  const safeLimit = Math.min(limit, 100);
  const where: Prisma.SecretCheckoutRequestWhereInput = {};

  if (role === 'requester') {
    where.requesterId = userId;
  } else if (role === 'approver') {
    // Show requests where this user could be an approver
    // (owns the secret/connection, or is admin)
    where.OR = [
      { secretId: { not: null }, requester: { id: { not: userId } } },
      { connectionId: { not: null }, requester: { id: { not: userId } } },
    ];
    // Filtered further below
  } else {
    where.OR = [
      { requesterId: userId },
      { approverId: userId },
    ];
  }

  if (status) {
    where.status = status;
  }

  const [data, total] = await Promise.all([
    prisma.secretCheckoutRequest.findMany({
      where,
      orderBy: { createdAt: 'desc' },
      skip: offset,
      take: safeLimit,
      select: checkoutSelect,
    }),
    prisma.secretCheckoutRequest.count({ where }),
  ]);

  // Enrich with resource names
  const enriched: CheckoutRequestEntry[] = [];
  for (const item of data) {
    let secretName: string | null = null;
    let connectionName: string | null = null;
    if (item.secretId) {
      const s = await prisma.vaultSecret.findUnique({ where: { id: item.secretId }, select: { name: true } });
      secretName = s?.name ?? null;
    }
    if (item.connectionId) {
      const c = await prisma.connection.findUnique({ where: { id: item.connectionId }, select: { name: true } });
      connectionName = c?.name ?? null;
    }
    enriched.push({ ...item, secretName, connectionName });
  }

  return { data: enriched, total };
}

/**
 * Get a single checkout request by ID.
 */
export async function getCheckoutRequest(
  requestId: string,
): Promise<CheckoutRequestEntry | null> {
  const request = await prisma.secretCheckoutRequest.findUnique({
    where: { id: requestId },
    select: checkoutSelect,
  });
  if (!request) return null;

  let secretName: string | null = null;
  let connectionName: string | null = null;
  if (request.secretId) {
    const s = await prisma.vaultSecret.findUnique({ where: { id: request.secretId }, select: { name: true } });
    secretName = s?.name ?? null;
  }
  if (request.connectionId) {
    const c = await prisma.connection.findUnique({ where: { id: request.connectionId }, select: { name: true } });
    connectionName = c?.name ?? null;
  }

  return { ...request, secretName, connectionName };
}

/**
 * Process expired checkout requests (called by scheduler).
 * Marks APPROVED requests whose expiresAt has passed as EXPIRED.
 */
export async function processExpiredCheckouts(): Promise<number> {
  const now = new Date();

  const expired = await prisma.secretCheckoutRequest.findMany({
    where: {
      status: 'APPROVED',
      expiresAt: { not: null, lte: now },
    },
    select: {
      id: true,
      secretId: true,
      connectionId: true,
      requesterId: true,
      requester: { select: { email: true, username: true } },
    },
  });

  if (expired.length === 0) return 0;

  for (const item of expired) {
    try {
      await prisma.secretCheckoutRequest.update({
        where: { id: item.id },
        data: { status: 'EXPIRED' },
      });

      auditService.log({
        action: 'SECRET_CHECKOUT_EXPIRED',
        targetType: item.secretId ? 'VaultSecret' : 'Connection',
        targetId: item.secretId ?? item.connectionId ?? undefined,
        details: { checkoutId: item.id, requesterId: item.requesterId },
      });

      // Resolve target name for notification
      let targetName = '';
      const resourceType = item.secretId ? 'secret' : 'connection';
      if (item.secretId) {
        const s = await prisma.vaultSecret.findUnique({ where: { id: item.secretId }, select: { name: true } });
        targetName = s?.name ?? 'a secret';
      } else if (item.connectionId) {
        const c = await prisma.connection.findUnique({ where: { id: item.connectionId }, select: { name: true } });
        targetName = c?.name ?? 'a connection';
      }

      const msg = `Your temporary access to ${resourceType} "${targetName}" has expired (auto check-in)`;
      createNotificationAsync({
        userId: item.requesterId,
        type: 'SECRET_CHECKOUT_EXPIRED',
        message: msg,
        relatedId: item.id,
      });
      emitNotification(item.requesterId, {
        id: '',
        type: 'SECRET_CHECKOUT_EXPIRED',
        message: msg,
        read: false,
        relatedId: item.id,
        createdAt: new Date(),
      });
    } catch (err) {
      logger.error(`[checkout] Failed to expire checkout ${item.id}:`, (err as Error).message);
    }
  }

  logger.info(`[checkout] Expired ${expired.length} checkout(s)`);
  return expired.length;
}
