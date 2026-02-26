import dotenv from 'dotenv';
import path from 'path';

dotenv.config({ path: path.resolve(__dirname, '../../.env') });

export const config = {
  port: parseInt(process.env.PORT || '3001', 10),
  guacamoleWsPort: parseInt(process.env.GUACAMOLE_WS_PORT || '3002', 10),
  jwtSecret: process.env.JWT_SECRET || 'dev-secret-change-me',
  jwtExpiresIn: process.env.JWT_EXPIRES_IN || '15m',
  jwtRefreshExpiresIn: process.env.JWT_REFRESH_EXPIRES_IN || '7d',
  guacdHost: process.env.GUACD_HOST || 'localhost',
  guacdPort: parseInt(process.env.GUACD_PORT || '4822', 10),
  guacamoleSecret: process.env.GUACAMOLE_SECRET || 'dev-guac-secret',
  vaultTtlMinutes: parseInt(process.env.VAULT_TTL_MINUTES || '30', 10),
  nodeEnv: process.env.NODE_ENV || 'development',
  logLevel: (process.env.LOG_LEVEL || 'info') as 'error' | 'warn' | 'info' | 'debug',
  driveBasePath: process.env.DRIVE_BASE_PATH || path.resolve(__dirname, '../../data/drive'),
  fileUploadMaxSize: parseInt(process.env.FILE_UPLOAD_MAX_SIZE || String(10 * 1024 * 1024), 10),
  userDriveQuota: parseInt(process.env.USER_DRIVE_QUOTA || String(100 * 1024 * 1024), 10),
};
