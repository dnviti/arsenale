import { Router, Response, NextFunction } from 'express';
import path from 'path';
import prisma from '../lib/prisma';
import { AuthRequest, RdpSettings } from '../types';
import { authenticate } from '../middleware/auth.middleware';
import { getConnection, getConnectionCredentials } from '../services/connection.service';
import { generateGuacamoleToken, mergeRdpSettings } from '../services/rdp.service';
import { AppError } from '../middleware/error.middleware';
import { z } from 'zod';

const sessionSchema = z.object({
  connectionId: z.string().uuid(),
  username: z.string().min(1).optional(),
  password: z.string().min(1).optional(),
}).refine(
  (data) => (!data.username && !data.password) || (data.username && data.password),
  { message: 'Both username and password must be provided together' },
);

const router = Router();

router.use(authenticate);

router.post('/rdp', async (req: AuthRequest, res: Response, next: NextFunction) => {
  try {
    const { connectionId, username: overrideUser, password: overridePass } = sessionSchema.parse(req.body);
    const conn = await getConnection(req.user!.userId, connectionId, req.user!.tenantId);

    if (conn.type !== 'RDP') {
      throw new AppError('Not an RDP connection', 400);
    }

    let username: string;
    let password: string;
    if (overrideUser && overridePass) {
      username = overrideUser;
      password = overridePass;
    } else {
      const creds = await getConnectionCredentials(req.user!.userId, connectionId, req.user!.tenantId);
      username = creds.username;
      password = creds.password;
    }

    // Load user RDP defaults and connection RDP settings, then merge
    const user = await prisma.user.findUnique({
      where: { id: req.user!.userId },
      select: { rdpDefaults: true },
    });
    const userRdpDefaults = (user?.rdpDefaults as Partial<RdpSettings>) ?? null;
    const connRdpSettings = (conn.rdpSettings as Partial<RdpSettings>) ?? null;
    const mergedRdp = mergeRdpSettings(userRdpDefaults, connRdpSettings);

    const enableDrive = conn.enableDrive ?? false;
    const drivePath = enableDrive
      ? path.posix.join('/guacd-drive', req.user!.userId)
      : undefined;

    const token = generateGuacamoleToken({
      host: conn.host,
      port: conn.port,
      username,
      password,
      enableDrive,
      drivePath,
      rdpSettings: mergedRdp,
    });

    res.json({ token, enableDrive });
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.issues[0].message, 400));
    next(err);
  }
});

router.post('/ssh', async (req: AuthRequest, res: Response, next: NextFunction) => {
  try {
    const { connectionId } = sessionSchema.parse(req.body);
    const conn = await getConnection(req.user!.userId, connectionId, req.user!.tenantId);

    if (conn.type !== 'SSH') {
      throw new AppError('Not an SSH connection', 400);
    }

    // SSH sessions are handled via Socket.io, we just validate access here
    res.json({ connectionId, type: 'SSH' });
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.issues[0].message, 400));
    next(err);
  }
});

export default router;
