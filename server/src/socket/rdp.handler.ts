import { Router, Response, NextFunction } from 'express';
import path from 'path';
import { AuthRequest } from '../types';
import { authenticate } from '../middleware/auth.middleware';
import { getConnection, getConnectionCredentials } from '../services/connection.service';
import { generateGuacamoleToken } from '../services/rdp.service';
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
    const conn = await getConnection(req.user!.userId, connectionId);

    if (conn.type !== 'RDP') {
      throw new AppError('Not an RDP connection', 400);
    }

    let username: string;
    let password: string;
    if (overrideUser && overridePass) {
      username = overrideUser;
      password = overridePass;
    } else {
      const creds = await getConnectionCredentials(req.user!.userId, connectionId);
      username = creds.username;
      password = creds.password;
    }

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
    });

    res.json({ token, enableDrive });
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.errors[0].message, 400));
    next(err);
  }
});

router.post('/ssh', async (req: AuthRequest, res: Response, next: NextFunction) => {
  try {
    const { connectionId } = sessionSchema.parse(req.body);
    const conn = await getConnection(req.user!.userId, connectionId);

    if (conn.type !== 'SSH') {
      throw new AppError('Not an SSH connection', 400);
    }

    // SSH sessions are handled via Socket.io, we just validate access here
    res.json({ connectionId, type: 'SSH' });
  } catch (err) {
    if (err instanceof z.ZodError) return next(new AppError(err.errors[0].message, 400));
    next(err);
  }
});

export default router;
