import prisma, { AuditAction, Prisma } from '../lib/prisma';
import { logger } from '../utils/logger';

export { AuditAction };

export interface AuditLogInput {
  userId?: string | null;
  action: AuditAction;
  targetType?: string;
  targetId?: string;
  details?: Record<string, unknown>;
  ipAddress?: string | string[];
  gatewayId?: string | null;
}

/**
 * Fire-and-forget audit logger. Never throws — errors are logged internally.
 */
export function log(input: AuditLogInput): void {
  prisma.auditLog
    .create({
      data: {
        userId: input.userId ?? null,
        action: input.action,
        targetType: input.targetType ?? null,
        targetId: input.targetId ?? null,
        details: (input.details as Prisma.InputJsonValue) ?? undefined,
        ipAddress: (Array.isArray(input.ipAddress) ? input.ipAddress[0] : input.ipAddress) ?? null,
        gatewayId: input.gatewayId ?? null,
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
  search?: string;
  targetType?: string;
  ipAddress?: string;
  gatewayId?: string;
  sortBy?: 'createdAt' | 'action';
  sortOrder?: 'asc' | 'desc';
}

export interface AuditLogEntry {
  id: string;
  action: AuditAction;
  targetType: string | null;
  targetId: string | null;
  details: unknown;
  ipAddress: string | null;
  gatewayId: string | null;
  createdAt: Date;
}

export interface AuditGateway {
  id: string;
  name: string;
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

  const where: Prisma.AuditLogWhereInput = { userId: query.userId };
  if (query.action) where.action = query.action;
  if (query.startDate || query.endDate) {
    where.createdAt = {
      ...(query.startDate && { gte: query.startDate }),
      ...(query.endDate && { lte: query.endDate }),
    };
  }
  if (query.targetType) {
    where.targetType = query.targetType;
  }
  if (query.ipAddress) {
    where.ipAddress = { contains: query.ipAddress, mode: 'insensitive' };
  }
  if (query.gatewayId) {
    where.gatewayId = query.gatewayId;
  }
  if (query.search) {
    const term = query.search;
    where.AND = [
      {
        OR: [
          { targetType: { contains: term, mode: 'insensitive' } },
          { targetId: { contains: term, mode: 'insensitive' } },
          { ipAddress: { contains: term, mode: 'insensitive' } },
          { details: { string_contains: term } },
        ],
      },
    ];
  }

  const sortBy = query.sortBy ?? 'createdAt';
  const sortOrder = query.sortOrder ?? 'desc';
  const orderBy: Prisma.AuditLogOrderByWithRelationInput = { [sortBy]: sortOrder };

  const [data, total] = await Promise.all([
    prisma.auditLog.findMany({
      where,
      orderBy,
      skip,
      take: limit,
      select: {
        id: true,
        action: true,
        targetType: true,
        targetId: true,
        details: true,
        ipAddress: true,
        gatewayId: true,
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

export async function getAuditGateways(userId: string): Promise<AuditGateway[]> {
  const rows = await prisma.auditLog.findMany({
    where: { userId, gatewayId: { not: null } },
    select: { gatewayId: true },
    distinct: ['gatewayId'],
  });

  const gatewayIds = rows.map((r) => r.gatewayId!);
  if (gatewayIds.length === 0) return [];

  const gateways = await prisma.gateway.findMany({
    where: { id: { in: gatewayIds } },
    select: { id: true, name: true },
  });

  const nameMap = new Map(gateways.map((g) => [g.id, g.name]));
  return gatewayIds.map((id) => ({
    id,
    name: nameMap.get(id) ?? `Deleted (${id.slice(0, 8)}…)`,
  }));
}
