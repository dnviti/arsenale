import prisma from '../lib/prisma';
import { logger } from '../utils/logger';
import type { DiscoveredDevice, SyncPlan, SyncResult, ConflictStrategy } from './types';

const log = logger.child('sync:engine');

export async function buildSyncPlan(
  profileId: string,
  devices: DiscoveredDevice[],
  conflictStrategy: ConflictStrategy,
): Promise<SyncPlan> {
  const plan: SyncPlan = {
    toCreate: [],
    toUpdate: [],
    toSkip: [],
    errors: [],
  };

  // Load existing synced connections for this profile
  const existing = await prisma.connection.findMany({
    where: { syncProfileId: profileId },
    select: { id: true, externalId: true, name: true, host: true, port: true, type: true },
  });

  const existingMap = new Map(
    existing.filter((c): c is typeof c & { externalId: string } => Boolean(c.externalId)).map((c) => [c.externalId, c]),
  );

  for (const device of devices) {
    if (!device.host) {
      plan.errors.push({ device, error: 'No IP address resolved' });
      continue;
    }

    const existingConn = existingMap.get(device.externalId);

    if (!existingConn) {
      plan.toCreate.push(device);
      continue;
    }

    if (conflictStrategy === 'skip') {
      plan.toSkip.push({ device, reason: 'Connection already exists (skip strategy)' });
      continue;
    }

    // Check what changed
    const changes: string[] = [];
    if (existingConn.name !== device.name) changes.push(`name: "${existingConn.name}" → "${device.name}"`);
    if (existingConn.host !== device.host) changes.push(`host: "${existingConn.host}" → "${device.host}"`);
    if (existingConn.port !== device.port) changes.push(`port: ${existingConn.port} → ${device.port}`);
    if (existingConn.type !== device.protocol) changes.push(`protocol: ${existingConn.type} → ${device.protocol}`);

    if (changes.length === 0) {
      plan.toSkip.push({ device, reason: 'No changes detected' });
      continue;
    }

    if (conflictStrategy === 'update' || conflictStrategy === 'overwrite') {
      plan.toUpdate.push({ device, connectionId: existingConn.id, changes });
    }
  }

  log.info(
    `Sync plan: create=${plan.toCreate.length} update=${plan.toUpdate.length} ` +
    `skip=${plan.toSkip.length} errors=${plan.errors.length}`,
  );

  return plan;
}

async function getOrCreateFolder(
  name: string,
  userId: string,
  teamId: string | null,
  parentId: string | null,
): Promise<string> {
  const where = teamId
    ? { name, teamId, parentId: parentId ?? null }
    : { name, userId, teamId: null, parentId: parentId ?? null };

  const existing = await prisma.folder.findFirst({ where, select: { id: true } });
  if (existing) return existing.id;

  const folder = await prisma.folder.create({
    data: {
      name,
      userId,
      teamId: teamId ?? undefined,
      parentId: parentId ?? undefined,
    },
  });

  return folder.id;
}

async function resolveFolderId(
  device: DiscoveredDevice,
  userId: string,
  teamId: string | null,
): Promise<string | undefined> {
  if (!device.siteName) return undefined;

  const siteId = await getOrCreateFolder(device.siteName, userId, teamId, null);

  if (device.rackName) {
    return getOrCreateFolder(device.rackName, userId, teamId, siteId);
  }

  return siteId;
}

export async function executeSyncPlan(
  plan: SyncPlan,
  userId: string,
  profileId: string,
  teamId: string | null,
): Promise<SyncResult> {
  const result: SyncResult = {
    created: 0,
    updated: 0,
    skipped: plan.toSkip.length,
    failed: plan.errors.length,
    errors: plan.errors.map((e) => ({
      externalId: e.device.externalId,
      name: e.device.name,
      error: e.error,
    })),
  };

  // Process creates
  for (const device of plan.toCreate) {
    try {
      const folderId = await resolveFolderId(device, userId, teamId);

      await prisma.connection.create({
        data: {
          name: device.name,
          type: device.protocol,
          host: device.host,
          port: device.port,
          description: device.description,
          userId,
          teamId: teamId ?? undefined,
          folderId,
          syncProfileId: profileId,
          externalId: device.externalId,
        },
      });
      result.created++;
    } catch (err) {
      log.error(`Failed to create connection for "${device.name}":`, (err as Error).message);
      result.failed++;
      result.errors.push({
        externalId: device.externalId,
        name: device.name,
        error: (err as Error).message,
      });
    }
  }

  // Process updates
  for (const entry of plan.toUpdate) {
    try {
      const folderId = await resolveFolderId(entry.device, userId, teamId);

      await prisma.connection.update({
        where: { id: entry.connectionId },
        data: {
          name: entry.device.name,
          type: entry.device.protocol,
          host: entry.device.host,
          port: entry.device.port,
          description: entry.device.description,
          folderId,
        },
      });
      result.updated++;
    } catch (err) {
      log.error(`Failed to update connection for "${entry.device.name}":`, (err as Error).message);
      result.failed++;
      result.errors.push({
        externalId: entry.device.externalId,
        name: entry.device.name,
        error: (err as Error).message,
      });
    }
  }

  log.info(
    `Sync complete: created=${result.created} updated=${result.updated} ` +
    `skipped=${result.skipped} failed=${result.failed}`,
  );

  return result;
}
