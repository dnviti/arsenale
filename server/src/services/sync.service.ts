import prisma from '../lib/prisma';
import { logger } from '../utils/logger';
import { AppError } from '../middleware/error.middleware';
import { encryptWithServerKey, decryptWithServerKey } from './crypto.service';
import { createSyncProvider } from '../sync';
import { buildSyncPlan, executeSyncPlan } from '../sync/engine';
import * as syncScheduler from './syncScheduler.service';
import type { SyncProfileConfig, SyncProviderConfig, SyncPlan, SyncResult } from '../sync/types';
import type { EncryptedField } from '../types';
import type { SyncProvider } from '../lib/prisma';

const log = logger.child('sync');

export interface CreateSyncProfileInput {
  name: string;
  provider: SyncProvider;
  url: string;
  apiToken: string;
  filters?: Record<string, string>;
  platformMapping?: Record<string, string>;
  defaultProtocol?: string;
  defaultPort?: Record<string, number>;
  conflictStrategy?: string;
  cronExpression?: string;
  teamId?: string;
}

export interface UpdateSyncProfileInput {
  name?: string;
  url?: string;
  apiToken?: string;
  filters?: Record<string, string>;
  platformMapping?: Record<string, string>;
  defaultProtocol?: string;
  defaultPort?: Record<string, number>;
  conflictStrategy?: string;
  cronExpression?: string | null;
  enabled?: boolean;
  teamId?: string | null;
}

export async function createSyncProfile(userId: string, tenantId: string, input: CreateSyncProfileInput) {
  const encrypted = encryptWithServerKey(input.apiToken);

  const config: SyncProfileConfig = {
    url: input.url,
    filters: input.filters ?? {},
    platformMapping: (input.platformMapping ?? {}) as SyncProfileConfig['platformMapping'],
    defaultProtocol: (input.defaultProtocol as SyncProfileConfig['defaultProtocol']) ?? 'SSH',
    defaultPort: (input.defaultPort ?? {}) as SyncProfileConfig['defaultPort'],
    conflictStrategy: (input.conflictStrategy as SyncProfileConfig['conflictStrategy']) ?? 'update',
  };

  const profile = await prisma.syncProfile.create({
    data: {
      name: input.name,
      tenantId,
      provider: input.provider,
      config: JSON.parse(JSON.stringify(config)),
      encryptedApiToken: encrypted.ciphertext,
      apiTokenIV: encrypted.iv,
      apiTokenTag: encrypted.tag,
      cronExpression: input.cronExpression,
      teamId: input.teamId,
      createdById: userId,
    },
  });

  if (input.cronExpression) {
    syncScheduler.registerSyncJob(profile.id, input.cronExpression);
  }

  log.info(`Created sync profile "${input.name}" (${profile.id}) for tenant ${tenantId}`);
  return sanitizeProfile(profile);
}

export async function updateSyncProfile(userId: string, profileId: string, tenantId: string, input: UpdateSyncProfileInput) {
  const profile = await prisma.syncProfile.findFirst({
    where: { id: profileId, tenantId },
  });
  if (!profile) throw new AppError('Sync profile not found', 404);

  const currentConfig = profile.config as unknown as SyncProfileConfig;

  // Build updated config
  const newConfig: SyncProfileConfig = {
    url: input.url ?? currentConfig.url,
    filters: input.filters ?? currentConfig.filters,
    platformMapping: (input.platformMapping as SyncProfileConfig['platformMapping']) ?? currentConfig.platformMapping,
    defaultProtocol: (input.defaultProtocol as SyncProfileConfig['defaultProtocol']) ?? currentConfig.defaultProtocol,
    defaultPort: (input.defaultPort as SyncProfileConfig['defaultPort']) ?? currentConfig.defaultPort,
    conflictStrategy: (input.conflictStrategy as SyncProfileConfig['conflictStrategy']) ?? currentConfig.conflictStrategy,
  };

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const data: Record<string, any> = {
    config: JSON.parse(JSON.stringify(newConfig)),
    updatedAt: new Date(),
  };

  if (input.name !== undefined) data.name = input.name;
  if (input.enabled !== undefined) data.enabled = input.enabled;
  if (input.cronExpression !== undefined) data.cronExpression = input.cronExpression;
  if (input.teamId !== undefined) data.teamId = input.teamId;

  if (input.apiToken) {
    const encrypted = encryptWithServerKey(input.apiToken);
    data.encryptedApiToken = encrypted.ciphertext;
    data.apiTokenIV = encrypted.iv;
    data.apiTokenTag = encrypted.tag;
  }

  const updated = await prisma.syncProfile.update({
    where: { id: profileId },
    data,
  });

  // Reschedule cron if changed
  syncScheduler.unregisterSyncJob(profileId);
  if (updated.cronExpression && updated.enabled) {
    syncScheduler.registerSyncJob(profileId, updated.cronExpression);
  }

  log.info(`Updated sync profile "${updated.name}" (${profileId})`);
  return sanitizeProfile(updated);
}

export async function deleteSyncProfile(profileId: string, tenantId: string) {
  const profile = await prisma.syncProfile.findFirst({
    where: { id: profileId, tenantId },
  });
  if (!profile) throw new AppError('Sync profile not found', 404);

  syncScheduler.unregisterSyncJob(profileId);

  await prisma.syncLog.deleteMany({ where: { syncProfileId: profileId } });
  await prisma.syncProfile.delete({ where: { id: profileId } });

  log.info(`Deleted sync profile "${profile.name}" (${profileId})`);
}

export async function getSyncProfile(profileId: string, tenantId: string) {
  const profile = await prisma.syncProfile.findFirst({
    where: { id: profileId, tenantId },
  });
  if (!profile) throw new AppError('Sync profile not found', 404);
  return sanitizeProfile(profile);
}

export async function listSyncProfiles(tenantId: string) {
  const profiles = await prisma.syncProfile.findMany({
    where: { tenantId },
    orderBy: { createdAt: 'desc' },
  });
  return profiles.map(sanitizeProfile);
}

export async function testConnection(profileId: string, tenantId: string) {
  const profile = await prisma.syncProfile.findFirst({
    where: { id: profileId, tenantId },
  });
  if (!profile) throw new AppError('Sync profile not found', 404);

  const apiToken = decryptApiToken(profile);
  const config = profile.config as unknown as SyncProfileConfig;
  const provider = createSyncProvider(profile.provider);

  const providerConfig: SyncProviderConfig = {
    ...config,
    apiToken,
  };

  return provider.testConnection(providerConfig);
}

export async function triggerSync(
  userId: string,
  profileId: string,
  tenantId: string,
  dryRun: boolean,
): Promise<{ plan: SyncPlan; result?: SyncResult }> {
  const profile = await prisma.syncProfile.findFirst({
    where: { id: profileId, tenantId },
  });
  if (!profile) throw new AppError('Sync profile not found', 404);

  const config = profile.config as unknown as SyncProfileConfig;
  const apiToken = decryptApiToken(profile);
  const provider = createSyncProvider(profile.provider);

  const providerConfig: SyncProviderConfig = {
    ...config,
    apiToken,
  };

  // Create sync log entry
  const syncLog = await prisma.syncLog.create({
    data: {
      syncProfileId: profileId,
      status: 'RUNNING',
      triggeredBy: userId,
    },
  });

  // Update profile status
  await prisma.syncProfile.update({
    where: { id: profileId },
    data: { lastSyncAt: new Date(), lastSyncStatus: 'RUNNING' },
  });

  try {
    // Discover devices
    const devices = await provider.discoverDevices(providerConfig);

    // Build plan
    const plan = await buildSyncPlan(profileId, devices, config.conflictStrategy);

    if (dryRun) {
      const dryRunDetails = JSON.parse(JSON.stringify({
        dryRun: true,
        toCreate: plan.toCreate.length,
        toUpdate: plan.toUpdate.length,
        toSkip: plan.toSkip.length,
        errors: plan.errors.length,
      }));
      await prisma.syncLog.update({
        where: { id: syncLog.id },
        data: {
          status: 'SUCCESS',
          completedAt: new Date(),
          details: dryRunDetails,
        },
      });
      await prisma.syncProfile.update({
        where: { id: profileId },
        data: { lastSyncStatus: 'SUCCESS', lastSyncDetails: JSON.parse(JSON.stringify({ dryRun: true })) },
      });

      return { plan };
    }

    // Execute the plan
    const result = await executeSyncPlan(plan, profile.createdById, profileId, profile.teamId);
    const status = result.failed > 0 ? 'PARTIAL' : 'SUCCESS';

    const resultDetails = JSON.parse(JSON.stringify({
      created: result.created,
      updated: result.updated,
      skipped: result.skipped,
      failed: result.failed,
      errors: result.errors,
    }));

    await prisma.syncLog.update({
      where: { id: syncLog.id },
      data: {
        status,
        completedAt: new Date(),
        details: resultDetails,
      },
    });

    const profileDetails = JSON.parse(JSON.stringify({
      created: result.created,
      updated: result.updated,
      skipped: result.skipped,
      failed: result.failed,
    }));

    await prisma.syncProfile.update({
      where: { id: profileId },
      data: {
        lastSyncStatus: status,
        lastSyncDetails: profileDetails,
      },
    });

    return { plan, result };
  } catch (err) {
    const errorMessage = (err as Error).message;
    log.error(`Sync failed for profile ${profileId}:`, errorMessage);

    await prisma.syncLog.update({
      where: { id: syncLog.id },
      data: {
        status: 'ERROR',
        completedAt: new Date(),
        details: JSON.parse(JSON.stringify({ error: errorMessage })),
      },
    });

    await prisma.syncProfile.update({
      where: { id: profileId },
      data: {
        lastSyncStatus: 'ERROR',
        lastSyncDetails: JSON.parse(JSON.stringify({ error: errorMessage })),
      },
    });

    throw new AppError(`Sync failed: ${errorMessage}`, 500);
  }
}

export async function getSyncLogs(profileId: string, tenantId: string, page: number, limit: number) {
  // Verify profile belongs to tenant
  const profile = await prisma.syncProfile.findFirst({
    where: { id: profileId, tenantId },
    select: { id: true },
  });
  if (!profile) throw new AppError('Sync profile not found', 404);

  const [logs, total] = await Promise.all([
    prisma.syncLog.findMany({
      where: { syncProfileId: profileId },
      orderBy: { startedAt: 'desc' },
      skip: (page - 1) * limit,
      take: limit,
    }),
    prisma.syncLog.count({ where: { syncProfileId: profileId } }),
  ]);

  return { logs, total, page, limit };
}

// --- Helpers ---

function decryptApiToken(profile: { encryptedApiToken: string; apiTokenIV: string; apiTokenTag: string }): string {
  const encrypted: EncryptedField = {
    ciphertext: profile.encryptedApiToken,
    iv: profile.apiTokenIV,
    tag: profile.apiTokenTag,
  };
  return decryptWithServerKey(encrypted);
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function sanitizeProfile(profile: any) {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { encryptedApiToken, apiTokenIV, apiTokenTag, ...rest } = profile;
  return { ...rest, hasApiToken: Boolean(encryptedApiToken) };
}
