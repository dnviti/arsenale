import { Server } from 'socket.io';
import prisma from '../lib/prisma';
import * as auditService from './audit.service';
import { formatDuration } from '../utils/format';
import { logger } from '../utils/logger';
import { config } from '../config';

let ioInstance: Server | null = null;

export function initSessionCleanup(io: Server): void {
  ioInstance = io;
}

export async function checkAndCloseInactiveSessions(): Promise<number> {
  try {
    const sessions = await prisma.activeSession.findMany({
      where: { status: { in: ['ACTIVE', 'IDLE'] } },
      include: {
        gateway: { select: { inactivityTimeoutSeconds: true } },
        user: { select: { tenant: { select: { defaultSessionTimeoutSeconds: true } } } },
      },
    });

    const now = Date.now();
    let closedCount = 0;

    for (const session of sessions) {
      const effectiveTimeout =
        session.gateway?.inactivityTimeoutSeconds ??
        session.user?.tenant?.defaultSessionTimeoutSeconds ??
        config.sessionInactivityTimeoutSeconds;

      const inactiveMs = now - session.lastActivityAt.getTime();
      if (inactiveMs < effectiveTimeout * 1000) continue;

      const durationMs = now - session.startedAt.getTime();

      // Mark CLOSED in DB first (before socket disconnect) to prevent
      // double audit logging — ssh.handler's endSessionBySocketId will
      // find the session already CLOSED and become a no-op.
      await prisma.activeSession.update({
        where: { id: session.id },
        data: { status: 'CLOSED', endedAt: new Date(now) },
      });

      auditService.log({
        userId: session.userId,
        action: 'SESSION_TIMEOUT',
        targetType: 'Connection',
        targetId: session.connectionId,
        details: {
          sessionId: session.id,
          protocol: session.protocol,
          durationMs,
          durationFormatted: formatDuration(durationMs),
          inactivitySeconds: Math.round(inactiveMs / 1000),
          effectiveTimeoutSeconds: effectiveTimeout,
        },
      });

      // For SSH sessions: force-disconnect the socket to trigger cleanup chain
      if (session.protocol === 'SSH' && session.socketId && ioInstance) {
        const socket = ioInstance.of('/ssh').sockets.get(session.socketId);
        if (socket) {
          socket.emit('session:timeout');
          socket.disconnect(true);
        }
      }

      closedCount++;
    }

    return closedCount;
  } catch (err) {
    logger.error('Session cleanup error:', err);
    return 0;
  }
}
