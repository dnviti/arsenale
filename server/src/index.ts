import http from 'http';
import app from './app';
import { config } from './config';
import { setupSocketIO } from './socket';
import { logger, toGuacamoleLogLevel } from './utils/logger';

async function main() {
  const server = http.createServer(app);

  // Setup Socket.io for SSH
  setupSocketIO(server);

  // Setup guacamole-lite for RDP
  if (config.nodeEnv !== 'test') {
    try {
      const GuacamoleLite = require('guacamole-lite');
      const { getGuacamoleKey } = require('./services/rdp.service');
      const guacServer = new GuacamoleLite(
        { port: config.guacamoleWsPort },
        {
          host: config.guacdHost,
          port: config.guacdPort,
        },
        {
          crypt: {
            cypher: 'AES-256-CBC',
            key: getGuacamoleKey(),
          },
          log: {
            level: toGuacamoleLogLevel(config.logLevel),
          },
        }
      );

      guacServer.on('error', (clientConnection: unknown, error: unknown) => {
        logger.error(
          'Guacamole connection error:',
          error instanceof Error ? error.message : error
        );
      });

      logger.info(
        `Guacamole WebSocket server listening on port ${config.guacamoleWsPort}`
      );
    } catch (err) {
      logger.warn(
        'guacamole-lite not available. RDP connections will not work.',
        err instanceof Error ? err.message : err
      );
    }
  }

  server.listen(config.port, () => {
    logger.info(`Server running on port ${config.port}`);
    logger.info(`Environment: ${config.nodeEnv}`);
  });
}

main().catch((err) => logger.error(err));
