import prisma from '../lib/prisma';
import { config } from '../config';
import { logger } from '../utils/logger';

const CACHE_TTL_MS = 30_000;
let cache: { selfSignupEnabled: boolean; expiresAt: number } | null = null;

export async function getSelfSignupEnabled(): Promise<boolean> {
  const now = Date.now();
  if (cache && cache.expiresAt > now) {
    return cache.selfSignupEnabled;
  }

  try {
    const row = await prisma.appConfig.findUnique({
      where: { key: 'selfSignupEnabled' },
    });

    const value = row ? row.value === 'true' : config.selfSignupEnabled;
    cache = { selfSignupEnabled: value, expiresAt: now + CACHE_TTL_MS };
    return value;
  } catch (err) {
    logger.error('Failed to read AppConfig selfSignupEnabled:', err);
    return config.selfSignupEnabled;
  }
}

export async function setSelfSignupEnabled(enabled: boolean): Promise<void> {
  await prisma.appConfig.upsert({
    where: { key: 'selfSignupEnabled' },
    update: { value: String(enabled) },
    create: { key: 'selfSignupEnabled', value: String(enabled) },
  });
  cache = { selfSignupEnabled: enabled, expiresAt: Date.now() + CACHE_TTL_MS };
}

export async function getPublicConfig(): Promise<{ selfSignupEnabled: boolean }> {
  return { selfSignupEnabled: await getSelfSignupEnabled() };
}
