import crypto from 'crypto';
import prisma from '../lib/prisma';
import { encryptWithServerKey, decryptWithServerKey } from './crypto.service';
import { config, setSystemSecret } from '../config';
import { readSecret } from '../utils/secrets';
import { publish } from '../utils/cacheClient';
import { logger } from '../utils/logger';

// ---------------------------------------------------------------------------
// Secret definitions — each entry defines a system secret that is automatically
// generated, stored encrypted in the DB, and kept in memory at runtime.
// ---------------------------------------------------------------------------

const SYSTEM_SECRET_DEFS = [
  {
    name: 'jwt_secret',
    bytes: 64,
    envFallback: 'JWT_SECRET',
    configKey: 'jwtSecret' as const,
    distribute: false,
    target: null as string | null,
    rotationDays: 90,
    description: 'JWT signing secret for authentication tokens',
  },
  {
    name: 'guacamole_secret',
    bytes: 32,
    envFallback: 'GUACAMOLE_SECRET',
    configKey: 'guacamoleSecret' as const,
    distribute: false,
    target: null as string | null,
    rotationDays: 90,
    description: 'Encryption key for RDP/VNC session tokens',
  },
  {
    name: 'guacenc_auth_token',
    bytes: 32,
    envFallback: 'GUACENC_AUTH_TOKEN',
    configKey: 'guacencAuthToken' as const,
    distribute: true,
    target: 'guacenc',
    rotationDays: 90,
    description: 'Bearer auth token for the video conversion service',
  },
] as const;

type SecretConfigKey = typeof SYSTEM_SECRET_DEFS[number]['configKey'];

// In-memory cache: name -> { current plaintext, previous plaintext | null }
const secretCache = new Map<string, { current: string; previous: string | null }>();

// ---------------------------------------------------------------------------
// ensureSystemSecrets — called once on startup
// ---------------------------------------------------------------------------

export async function ensureSystemSecrets(): Promise<void> {
  logger.info('[system-secrets] Initializing auto-managed secrets...');

  for (const def of SYSTEM_SECRET_DEFS) {
    const externalValue = readSecret(def.name, def.envFallback);
    const dbRow = await prisma.systemSecret.findUnique({ where: { name: def.name } });

    let currentValue: string;
    let previousValue: string | null = null;

    if (externalValue && !dbRow) {
      // External value provided, no DB row yet — store encrypted in DB
      const encrypted = encryptWithServerKey(externalValue);
      await prisma.systemSecret.create({
        data: {
          name: def.name,
          encryptedValue: encrypted.ciphertext,
          valueIV: encrypted.iv,
          valueTag: encrypted.tag,
          autoRotate: true,
          rotationIntervalDays: def.rotationDays,
          distributed: def.distribute,
          targetService: def.target,
        },
      });
      currentValue = externalValue;
      logger.info(`[system-secrets] Stored external secret "${def.name}" in DB`);
    } else if (externalValue && dbRow) {
      // External value takes precedence — update DB if different
      const dbValue = decryptWithServerKey({
        ciphertext: dbRow.encryptedValue,
        iv: dbRow.valueIV,
        tag: dbRow.valueTag,
      });
      if (dbValue !== externalValue) {
        const encrypted = encryptWithServerKey(externalValue);
        await prisma.systemSecret.update({
          where: { name: def.name },
          data: {
            encryptedValue: encrypted.ciphertext,
            valueIV: encrypted.iv,
            valueTag: encrypted.tag,
          },
        });
        logger.info(`[system-secrets] Updated "${def.name}" from external source`);
      }
      currentValue = externalValue;
      // Preserve previous version if it exists
      if (dbRow.previousEncryptedValue && dbRow.previousValueIV && dbRow.previousValueTag) {
        previousValue = decryptWithServerKey({
          ciphertext: dbRow.previousEncryptedValue,
          iv: dbRow.previousValueIV,
          tag: dbRow.previousValueTag,
        });
      }
    } else if (!externalValue && !dbRow) {
      // No external, no DB — auto-generate
      currentValue = crypto.randomBytes(def.bytes).toString('hex');
      const encrypted = encryptWithServerKey(currentValue);
      await prisma.systemSecret.create({
        data: {
          name: def.name,
          encryptedValue: encrypted.ciphertext,
          valueIV: encrypted.iv,
          valueTag: encrypted.tag,
          autoRotate: true,
          rotationIntervalDays: def.rotationDays,
          distributed: def.distribute,
          targetService: def.target,
        },
      });
      logger.info(`[system-secrets] Auto-generated secret "${def.name}"`);
    } else {
      // No external, DB row exists — decrypt from DB
      currentValue = decryptWithServerKey({
        ciphertext: dbRow!.encryptedValue,
        iv: dbRow!.valueIV,
        tag: dbRow!.valueTag,
      });
      if (dbRow!.previousEncryptedValue && dbRow!.previousValueIV && dbRow!.previousValueTag) {
        previousValue = decryptWithServerKey({
          ciphertext: dbRow!.previousEncryptedValue,
          iv: dbRow!.previousValueIV,
          tag: dbRow!.previousValueTag,
        });
      }
      logger.info(`[system-secrets] Loaded secret "${def.name}" from DB (v${dbRow!.currentVersion})`);
    }

    // Populate in-memory cache
    secretCache.set(def.name, { current: currentValue, previous: previousValue });

    // Push to runtime config
    setSystemSecret(def.configKey, currentValue);
  }

  // Publish distributed secrets to sidecar services
  await publishDistributedSecrets();

  logger.info(`[system-secrets] ${SYSTEM_SECRET_DEFS.length} secret(s) initialized`);
}

// ---------------------------------------------------------------------------
// Getters
// ---------------------------------------------------------------------------

export function getSecretValue(name: string): string {
  const entry = secretCache.get(name);
  if (!entry) {
    throw new Error(`System secret "${name}" not found in cache — was ensureSystemSecrets() called?`);
  }
  return entry.current;
}

export function getSecretValueSync(name: string): { current: string; previous: string | null } {
  const entry = secretCache.get(name);
  if (!entry) {
    throw new Error(`System secret "${name}" not found in cache — was ensureSystemSecrets() called?`);
  }
  return entry;
}

// ---------------------------------------------------------------------------
// Rotation
// ---------------------------------------------------------------------------

export async function rotateSecret(name: string): Promise<void> {
  const def = SYSTEM_SECRET_DEFS.find((d) => d.name === name);
  if (!def) throw new Error(`Unknown system secret: "${name}"`);

  const dbRow = await prisma.systemSecret.findUnique({ where: { name } });
  if (!dbRow) throw new Error(`System secret "${name}" not found in DB`);

  // Generate new value
  const newValue = crypto.randomBytes(def.bytes).toString('hex');

  // Read current value (becomes "previous")
  const oldValue = decryptWithServerKey({
    ciphertext: dbRow.encryptedValue,
    iv: dbRow.valueIV,
    tag: dbRow.valueTag,
  });

  // Encrypt new and old values
  const encryptedNew = encryptWithServerKey(newValue);
  const encryptedOld = encryptWithServerKey(oldValue);

  // Update DB: new → current, old current → previous
  await prisma.systemSecret.update({
    where: { name },
    data: {
      encryptedValue: encryptedNew.ciphertext,
      valueIV: encryptedNew.iv,
      valueTag: encryptedNew.tag,
      previousEncryptedValue: encryptedOld.ciphertext,
      previousValueIV: encryptedOld.iv,
      previousValueTag: encryptedOld.tag,
      currentVersion: dbRow.currentVersion + 1,
      rotatedAt: new Date(),
    },
  });

  // Update in-memory cache
  secretCache.set(name, { current: newValue, previous: oldValue });

  // Update runtime config
  setSystemSecret(def.configKey, newValue);

  // Publish to distributed services if applicable
  if (def.distribute && def.target) {
    const payload = JSON.stringify({
      name: def.name,
      value: newValue,
      version: dbRow.currentVersion + 1,
      rotatedAt: new Date().toISOString(),
    });
    await publish(`system:secret:${def.target}`, payload);
    logger.info(`[system-secrets] Published rotated secret "${name}" to [REDACTED] channel`);
  }

  logger.info(`[system-secrets] Rotated secret "${name}" to v${dbRow.currentVersion + 1}`);
}

export async function processSecretRotations(): Promise<void> {
  const secrets = await prisma.systemSecret.findMany({
    where: { autoRotate: true },
  });

  for (const secret of secrets) {
    const daysSinceRotation = secret.rotatedAt
      ? (Date.now() - secret.rotatedAt.getTime()) / (1000 * 60 * 60 * 24)
      : (Date.now() - secret.createdAt.getTime()) / (1000 * 60 * 60 * 24);

    if (daysSinceRotation >= secret.rotationIntervalDays) {
      try {
        await rotateSecret(secret.name);
        logger.info(`[system-secrets] Auto-rotated secret "${secret.name}" (${Math.floor(daysSinceRotation)}d since last rotation)`);
      } catch (err) {
        logger.error(
          `[system-secrets] Failed to auto-rotate secret "${secret.name}":`,
          err instanceof Error ? err.message : 'Unknown error',
        );
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Distribution (publish secrets to sidecar services via gocache)
// ---------------------------------------------------------------------------

export async function publishDistributedSecrets(): Promise<void> {
  for (const def of SYSTEM_SECRET_DEFS) {
    if (!def.distribute || !def.target) continue;

    const entry = secretCache.get(def.name);
    if (!entry) continue;

    const dbRow = await prisma.systemSecret.findUnique({ where: { name: def.name } });
    if (!dbRow) continue;

    const payload = JSON.stringify({
      name: def.name,
      value: entry.current,
      version: dbRow.currentVersion,
      rotatedAt: dbRow.rotatedAt?.toISOString() ?? null,
    });

    await publish(`system:secret:${def.target}`, payload);
    logger.info(`[system-secrets] Published secret "${def.name}" to [REDACTED] channel`);
  }
}

// ---------------------------------------------------------------------------
// Display (for initial setup wizard only)
// ---------------------------------------------------------------------------

export async function getAllSecretsForDisplay(): Promise<Array<{ name: string; value: string; description: string }>> {
  const results: Array<{ name: string; value: string; description: string }> = [];

  for (const def of SYSTEM_SECRET_DEFS) {
    const entry = secretCache.get(def.name);
    if (!entry) continue;

    results.push({
      name: def.envFallback,
      value: entry.current,
      description: def.description,
    });
  }

  return results;
}
