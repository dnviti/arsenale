import { Server as HttpServer } from 'http';
import { Server } from 'socket.io';
import { config } from '../config';
import { verifyJwt } from '../utils/jwt';
import { setupSshHandler } from './ssh.handler';
import { setupNotificationHandler } from './notification.handler';
import { setupGatewayMonitorHandler } from './gatewayMonitor.handler';

export function setupSocketIO(httpServer: HttpServer): Server {
  const io = new Server(httpServer, {
    cors: {
      origin: [config.clientUrl],
      methods: ['GET', 'POST'],
    },
  });

  // Server-level auth middleware: reject unauthenticated connections
  // before they reach any namespace-specific middleware
  io.use((socket, next) => {
    const token = socket.handshake.auth.token;
    if (!token) return next(new Error('Authentication required'));

    try {
      verifyJwt(token);
      next();
    } catch {
      next(new Error('Authentication required'));
    }
  });

  setupSshHandler(io);
  setupNotificationHandler(io);
  setupGatewayMonitorHandler(io);

  return io;
}
