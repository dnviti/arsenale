import { config } from '../config';

const LEVELS = {
  error: 0,
  warn: 1,
  info: 2,
  debug: 3,
} as const;

type LogLevel = keyof typeof LEVELS;

const currentLevel: number = LEVELS[config.logLevel] ?? LEVELS.info;

export const logger = {
  error: (...args: unknown[]) => {
    if (currentLevel >= LEVELS.error) console.error(...args);
  },
  warn: (...args: unknown[]) => {
    if (currentLevel >= LEVELS.warn) console.warn(...args);
  },
  info: (...args: unknown[]) => {
    if (currentLevel >= LEVELS.info) console.log(...args);
  },
  debug: (...args: unknown[]) => {
    if (currentLevel >= LEVELS.debug) console.debug(...args);
  },
};

/**
 * Maps LOG_LEVEL to guacamole-lite's log level string.
 * guacamole-lite levels: QUIET, ERRORS, NORMAL, VERBOSE, DEBUG
 */
export function toGuacamoleLogLevel(level: LogLevel): string {
  switch (level) {
    case 'error': return 'ERRORS';
    case 'warn':  return 'ERRORS';
    case 'info':  return 'NORMAL';
    case 'debug': return 'DEBUG';
    default:      return 'NORMAL';
  }
}
