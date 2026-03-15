import prisma, { Prisma } from '../lib/prisma';
import type { AccessPolicyTargetType } from '../lib/prisma';
import { AppError } from '../middleware/error.middleware';

export interface AccessPolicyData {
  id: string;
  targetType: AccessPolicyTargetType;
  targetId: string;
  allowedTimeWindows: string | null;
  requireTrustedDevice: boolean;
  requireMfaStepUp: boolean;
  createdAt: Date;
  updatedAt: Date;
}

/**
 * List all access policies for targets belonging to the given tenant.
 * Includes TENANT-level policies (targetId === tenantId) as well as
 * TEAM and FOLDER policies whose targets belong to this tenant.
 */
export async function listPolicies(tenantId: string): Promise<AccessPolicyData[]> {
  // Gather target IDs belonging to this tenant
  const teamRows = await prisma.team.findMany({ where: { tenantId }, select: { id: true } });
  const teamIds = teamRows.map((r: { id: string }) => r.id);

  // Folders belong to teams (team folders) or users. For tenant scope,
  // fetch folders whose team belongs to this tenant.
  let folderIds: string[] = [];
  if (teamIds.length > 0) {
    const folderRows = await prisma.folder.findMany({ where: { teamId: { in: teamIds } }, select: { id: true } });
    folderIds = folderRows.map((r: { id: string }) => r.id);
  }

  const orConditions: Prisma.AccessPolicyWhereInput[] = [
    { targetType: 'TENANT', targetId: tenantId },
  ];
  if (teamIds.length > 0) {
    orConditions.push({ targetType: 'TEAM', targetId: { in: teamIds } });
  }
  if (folderIds.length > 0) {
    orConditions.push({ targetType: 'FOLDER', targetId: { in: folderIds } });
  }

  return prisma.accessPolicy.findMany({
    where: { OR: orConditions },
    orderBy: { createdAt: 'desc' },
  });
}

/**
 * Create a new access policy with target validation.
 * Prevents duplicate policies for the same (targetType, targetId) pair.
 */
export async function createPolicy(
  tenantId: string,
  data: {
    targetType: AccessPolicyTargetType;
    targetId: string;
    allowedTimeWindows?: string | null;
    requireTrustedDevice?: boolean;
    requireMfaStepUp?: boolean;
  },
): Promise<AccessPolicyData> {
  await validateTarget(tenantId, data.targetType, data.targetId);

  // Prevent duplicate policies for the same target
  const existing = await prisma.accessPolicy.findFirst({
    where: { targetType: data.targetType, targetId: data.targetId },
  });
  if (existing) {
    throw new AppError('A policy already exists for this target. Edit the existing policy instead.', 409);
  }

  return prisma.accessPolicy.create({
    data: {
      targetType: data.targetType,
      targetId: data.targetId,
      allowedTimeWindows: data.allowedTimeWindows ?? null,
      requireTrustedDevice: data.requireTrustedDevice ?? false,
      requireMfaStepUp: data.requireMfaStepUp ?? false,
    },
  });
}

/**
 * Update an existing access policy.
 */
export async function updatePolicy(
  tenantId: string,
  policyId: string,
  data: {
    allowedTimeWindows?: string | null;
    requireTrustedDevice?: boolean;
    requireMfaStepUp?: boolean;
  },
): Promise<AccessPolicyData> {
  const existing = await prisma.accessPolicy.findUnique({ where: { id: policyId } });
  if (!existing) throw new AppError('Policy not found', 404);

  // Verify the policy belongs to this tenant
  await validateTarget(tenantId, existing.targetType as AccessPolicyTargetType, existing.targetId);

  return prisma.accessPolicy.update({
    where: { id: policyId },
    data: {
      ...(data.allowedTimeWindows !== undefined && { allowedTimeWindows: data.allowedTimeWindows }),
      ...(data.requireTrustedDevice !== undefined && { requireTrustedDevice: data.requireTrustedDevice }),
      ...(data.requireMfaStepUp !== undefined && { requireMfaStepUp: data.requireMfaStepUp }),
    },
  });
}

/**
 * Delete an access policy.
 */
export async function deletePolicy(tenantId: string, policyId: string): Promise<void> {
  const existing = await prisma.accessPolicy.findUnique({ where: { id: policyId } });
  if (!existing) throw new AppError('Policy not found', 404);

  // Verify the policy belongs to this tenant
  await validateTarget(tenantId, existing.targetType as AccessPolicyTargetType, existing.targetId);

  await prisma.accessPolicy.delete({ where: { id: policyId } });
}

/**
 * Validate that the target exists and belongs to the given tenant.
 */
async function validateTarget(
  tenantId: string,
  targetType: AccessPolicyTargetType,
  targetId: string,
): Promise<void> {
  switch (targetType) {
    case 'TENANT': {
      if (targetId !== tenantId) {
        throw new AppError('Target tenant does not match your tenant', 403);
      }
      break;
    }
    case 'TEAM': {
      const team = await prisma.team.findUnique({ where: { id: targetId }, select: { tenantId: true } });
      if (!team || team.tenantId !== tenantId) {
        throw new AppError('Team not found or does not belong to your tenant', 404);
      }
      break;
    }
    case 'FOLDER': {
      // Folders don't have tenantId directly; check via the team relation
      const folder = await prisma.folder.findUnique({
        where: { id: targetId },
        select: { teamId: true, team: { select: { tenantId: true } } },
      });
      if (!folder || !folder.team || folder.team.tenantId !== tenantId) {
        throw new AppError('Folder not found or does not belong to your tenant', 404);
      }
      break;
    }
    default:
      throw new AppError('Invalid target type', 400);
  }
}
