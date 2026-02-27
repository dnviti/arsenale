import prisma, { AuditAction, Prisma } from '../lib/prisma';
import { logger } from '../utils/logger';

export { AuditAction };

export interface AuditLogInput {
  userId: string;
  action: AuditAction;
  targetType?: string;
  targetId?: string;
  details?: Record<string, unknown>;
  ipAddress?: string | string[];
}

/**
 * Fire-and-forget audit logger. Never throws — errors are logged internally.
 */
export function log(input: AuditLogInput): void {
  prisma.auditLog
    .create({
      data: {
        userId: input.userId,
        action: input.action,
        targetType: input.targetType ?? null,
        targetId: input.targetId ?? null,
        details: (input.details as Prisma.InputJsonValue) ?? undefined,
        ipAddress: (Array.isArray(input.ipAddress) ? input.ipAddress[0] : input.ipAddress) ?? null,
      },
    })
    .catch((err) => {
      logger.error('Failed to write audit log:', err);
    });
}

export interface AuditLogQuery {
  userId: string;
  page?: number;
  limit?: number;
  action?: AuditAction;
  startDate?: Date;
  endDate?: Date;
}

export interface AuditLogEntry {
  id: string;
  action: AuditAction;
  targetType: string | null;
  targetId: string | null;
  details: unknown;
  ipAddress: string | null;
  createdAt: Date;
}

export interface PaginatedAuditLogs {
  data: AuditLogEntry[];
  total: number;
  page: number;
  limit: number;
  totalPages: number;
}

export async function getAuditLogs(query: AuditLogQuery): Promise<PaginatedAuditLogs> {
  const page = query.page ?? 1;
  const limit = Math.min(query.limit ?? 50, 100);
  const skip = (page - 1) * limit;

  const where: Record<string, unknown> = { userId: query.userId };
  if (query.action) where.action = query.action;
  if (query.startDate || query.endDate) {
    where.createdAt = {
      ...(query.startDate && { gte: query.startDate }),
      ...(query.endDate && { lte: query.endDate }),
    };
  }

  const [data, total] = await Promise.all([
    prisma.auditLog.findMany({
      where,
      orderBy: { createdAt: 'desc' },
      skip,
      take: limit,
      select: {
        id: true,
        action: true,
        targetType: true,
        targetId: true,
        details: true,
        ipAddress: true,
        createdAt: true,
      },
    }),
    prisma.auditLog.count({ where }),
  ]);

  return {
    data,
    total,
    page,
    limit,
    totalPages: Math.ceil(total / limit),
  };
}
