import { Response, NextFunction } from 'express';
import { z } from 'zod';
import { AuthRequest } from '../types';
import * as auditService from '../services/audit.service';
import { AppError } from '../middleware/error.middleware';

const VALID_ACTIONS = [
  'LOGIN', 'LOGIN_OAUTH', 'LOGIN_TOTP', 'LOGIN_FAILURE', 'LOGOUT', 'REGISTER',
  'VAULT_UNLOCK', 'VAULT_LOCK', 'VAULT_SETUP',
  'CREATE_CONNECTION', 'UPDATE_CONNECTION', 'DELETE_CONNECTION',
  'SHARE_CONNECTION', 'UNSHARE_CONNECTION', 'UPDATE_SHARE_PERMISSION',
  'CREATE_FOLDER', 'UPDATE_FOLDER', 'DELETE_FOLDER',
  'PASSWORD_CHANGE', 'PROFILE_UPDATE',
  'TOTP_ENABLE', 'TOTP_DISABLE',
  'OAUTH_LINK', 'OAUTH_UNLINK',
  'PASSWORD_REVEAL',
] as const;

const querySchema = z.object({
  page: z.coerce.number().int().min(1).default(1),
  limit: z.coerce.number().int().min(1).max(100).default(50),
  action: z.enum(VALID_ACTIONS).optional(),
  startDate: z.coerce.date().optional(),
  endDate: z.coerce.date().optional(),
});

export async function list(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    const query = querySchema.parse(req.query);
    const result = await auditService.getAuditLogs({
      userId: req.user!.userId,
      ...query,
    });
    res.json(result);
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.issues[0].message, 400));
    next(err);
  }
}
