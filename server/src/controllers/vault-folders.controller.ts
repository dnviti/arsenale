import { Response } from 'express';
import { AuthRequest, assertAuthenticated } from '../types';
import * as vaultFolderService from '../services/vault-folder.service';
import * as auditService from '../services/audit.service';
import { getClientIp } from '../utils/ip';
import type { CreateVaultFolderInput, UpdateVaultFolderInput } from '../schemas/vaultFolder.schemas';

export async function create(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const { name, scope, parentId, teamId } = req.body as CreateVaultFolderInput;
  const result = await vaultFolderService.createFolder(
    req.user.userId, name, scope, parentId, teamId, req.user.tenantId
  );
  auditService.log({
    userId: req.user.userId, action: 'CREATE_FOLDER',
    targetType: 'VaultFolder', targetId: result.id,
    details: { name, scope, teamId: teamId ?? null },
    ipAddress: getClientIp(req),
  });
  res.status(201).json(result);
}

export async function update(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const data = req.body as UpdateVaultFolderInput;
  const result = await vaultFolderService.updateFolder(
    req.user.userId, req.params.id as string, data, req.user.tenantId
  );
  auditService.log({
    userId: req.user.userId, action: 'UPDATE_FOLDER',
    targetType: 'VaultFolder', targetId: req.params.id as string,
    details: { fields: Object.keys(data) },
    ipAddress: getClientIp(req),
  });
  res.json(result);
}

export async function remove(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const result = await vaultFolderService.deleteFolder(
    req.user.userId, req.params.id as string, req.user.tenantId
  );
  auditService.log({
    userId: req.user.userId, action: 'DELETE_FOLDER',
    targetType: 'VaultFolder', targetId: req.params.id as string,
    ipAddress: getClientIp(req),
  });
  res.json(result);
}

export async function list(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const result = await vaultFolderService.getFolderTree(req.user.userId, req.user.tenantId);
  res.json(result);
}
