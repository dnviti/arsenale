import { Response } from 'express';
import { AuthRequest, assertTenantAuthenticated } from '../types';
import * as syncService from '../services/sync.service';
import * as auditService from '../services/audit.service';
import { getClientIp } from '../utils/ip';
import type { TriggerSyncInput } from '../schemas/sync.schemas';

export async function create(req: AuthRequest, res: Response) {
  assertTenantAuthenticated(req);
  const data = req.body as syncService.CreateSyncProfileInput;
  const result = await syncService.createSyncProfile(req.user.userId, req.user.tenantId, data);

  auditService.log({
    userId: req.user.userId,
    action: 'SYNC_PROFILE_CREATE',
    targetType: 'SyncProfile',
    targetId: result.id,
    details: { name: data.name, provider: data.provider },
    ipAddress: getClientIp(req),
  });

  res.status(201).json(result);
}

export async function list(req: AuthRequest, res: Response) {
  assertTenantAuthenticated(req);
  const result = await syncService.listSyncProfiles(req.user.tenantId);
  res.json(result);
}

export async function get(req: AuthRequest, res: Response) {
  assertTenantAuthenticated(req);
  const profileId = req.params.id as string;
  const result = await syncService.getSyncProfile(profileId, req.user.tenantId);
  res.json(result);
}

export async function update(req: AuthRequest, res: Response) {
  assertTenantAuthenticated(req);
  const profileId = req.params.id as string;
  const data = req.body as syncService.UpdateSyncProfileInput;
  const result = await syncService.updateSyncProfile(
    req.user.userId,
    profileId,
    req.user.tenantId,
    data,
  );

  auditService.log({
    userId: req.user.userId,
    action: 'SYNC_PROFILE_UPDATE',
    targetType: 'SyncProfile',
    targetId: profileId,
    details: { name: data.name },
    ipAddress: getClientIp(req),
  });

  res.json(result);
}

export async function remove(req: AuthRequest, res: Response) {
  assertTenantAuthenticated(req);
  const profileId = req.params.id as string;
  await syncService.deleteSyncProfile(profileId, req.user.tenantId);

  auditService.log({
    userId: req.user.userId,
    action: 'SYNC_PROFILE_DELETE',
    targetType: 'SyncProfile',
    targetId: profileId,
    ipAddress: getClientIp(req),
  });

  res.status(204).end();
}

export async function testConnection(req: AuthRequest, res: Response) {
  assertTenantAuthenticated(req);
  const profileId = req.params.id as string;
  const result = await syncService.testConnection(profileId, req.user.tenantId);
  res.json(result);
}

export async function triggerSync(req: AuthRequest, res: Response) {
  assertTenantAuthenticated(req);
  const profileId = req.params.id as string;
  const { dryRun } = req.body as TriggerSyncInput;

  auditService.log({
    userId: req.user.userId,
    action: 'SYNC_START',
    targetType: 'SyncProfile',
    targetId: profileId,
    details: { dryRun },
    ipAddress: getClientIp(req),
  });

  const result = await syncService.triggerSync(
    req.user.userId,
    profileId,
    req.user.tenantId,
    dryRun,
  );

  const action = result.result
    ? (result.result.failed > 0 ? 'SYNC_ERROR' : 'SYNC_COMPLETE')
    : 'SYNC_COMPLETE';

  auditService.log({
    userId: req.user.userId,
    action,
    targetType: 'SyncProfile',
    targetId: profileId,
    details: result.result ? JSON.parse(JSON.stringify(result.result)) : { dryRun: true },
    ipAddress: getClientIp(req),
  });

  res.json(result);
}

export async function getLogs(req: AuthRequest, res: Response) {
  assertTenantAuthenticated(req);
  const profileId = req.params.id as string;
  const page = parseInt(req.query.page as string) || 1;
  const limit = Math.min(parseInt(req.query.limit as string) || 20, 100);
  const result = await syncService.getSyncLogs(profileId, req.user.tenantId, page, limit);
  res.json(result);
}
