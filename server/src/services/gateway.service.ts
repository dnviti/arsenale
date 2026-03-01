import net from 'net';
import prisma from '../lib/prisma';
import type { GatewayType } from '../lib/prisma';
import { AppError } from '../middleware/error.middleware';
import { encrypt, decrypt, getMasterKey } from './crypto.service';

export interface CreateGatewayInput {
  name: string;
  type: GatewayType;
  host: string;
  port: number;
  description?: string;
  isDefault?: boolean;
  username?: string;
  password?: string;
}

export interface UpdateGatewayInput {
  name?: string;
  host?: string;
  port?: number;
  description?: string | null;
  isDefault?: boolean;
  username?: string;
  password?: string;
}

// Fields returned for public gateway responses (no credential columns)
const publicSelect = {
  id: true,
  name: true,
  type: true,
  host: true,
  port: true,
  description: true,
  isDefault: true,
  tenantId: true,
  createdById: true,
  createdAt: true,
  updatedAt: true,
} as const;

function requireMasterKey(userId: string): Buffer {
  const key = getMasterKey(userId);
  if (!key) throw new AppError('Vault is locked. Please unlock it first.', 403);
  return key;
}

export async function listGateways(tenantId: string) {
  return prisma.gateway.findMany({
    where: { tenantId },
    select: publicSelect,
    orderBy: [{ type: 'asc' }, { name: 'asc' }],
  });
}

export async function createGateway(
  userId: string,
  tenantId: string,
  input: CreateGatewayInput,
) {
  const encData: Record<string, string | null> = {
    encryptedUsername: null,
    usernameIV: null,
    usernameTag: null,
    encryptedPassword: null,
    passwordIV: null,
    passwordTag: null,
  };

  if (input.type === 'SSH_BASTION') {
    if (input.username || input.password) {
      const masterKey = requireMasterKey(userId);
      if (input.username) {
        const enc = encrypt(input.username, masterKey);
        encData.encryptedUsername = enc.ciphertext;
        encData.usernameIV = enc.iv;
        encData.usernameTag = enc.tag;
      }
      if (input.password) {
        const enc = encrypt(input.password, masterKey);
        encData.encryptedPassword = enc.ciphertext;
        encData.passwordIV = enc.iv;
        encData.passwordTag = enc.tag;
      }
    }
  } else if (input.username || input.password) {
    throw new AppError('Credentials can only be set for SSH_BASTION gateways', 400);
  }

  const gateway = await prisma.$transaction(async (tx) => {
    if (input.isDefault) {
      await tx.gateway.updateMany({
        where: { tenantId, type: input.type, isDefault: true },
        data: { isDefault: false },
      });
    }

    return tx.gateway.create({
      data: {
        name: input.name,
        type: input.type,
        host: input.host,
        port: input.port,
        description: input.description ?? null,
        isDefault: input.isDefault ?? false,
        tenantId,
        createdById: userId,
        ...encData,
      },
      select: publicSelect,
    });
  });

  return gateway;
}

export async function updateGateway(
  userId: string,
  tenantId: string,
  gatewayId: string,
  input: UpdateGatewayInput,
) {
  const existing = await prisma.gateway.findFirst({
    where: { id: gatewayId, tenantId },
  });
  if (!existing) throw new AppError('Gateway not found', 404);

  const data: Record<string, unknown> = {};
  if (input.name !== undefined) data.name = input.name;
  if (input.host !== undefined) data.host = input.host;
  if (input.port !== undefined) data.port = input.port;
  if (input.description !== undefined) data.description = input.description;

  if (input.username !== undefined || input.password !== undefined) {
    if (existing.type !== 'SSH_BASTION') {
      throw new AppError('Credentials can only be set for SSH_BASTION gateways', 400);
    }
    const masterKey = requireMasterKey(userId);
    if (input.username !== undefined) {
      const enc = encrypt(input.username, masterKey);
      data.encryptedUsername = enc.ciphertext;
      data.usernameIV = enc.iv;
      data.usernameTag = enc.tag;
    }
    if (input.password !== undefined) {
      const enc = encrypt(input.password, masterKey);
      data.encryptedPassword = enc.ciphertext;
      data.passwordIV = enc.iv;
      data.passwordTag = enc.tag;
    }
  }

  const updated = await prisma.$transaction(async (tx) => {
    if (input.isDefault === true && !existing.isDefault) {
      await tx.gateway.updateMany({
        where: { tenantId, type: existing.type, isDefault: true, id: { not: gatewayId } },
        data: { isDefault: false },
      });
      data.isDefault = true;
    } else if (input.isDefault === false) {
      data.isDefault = false;
    }

    return tx.gateway.update({
      where: { id: gatewayId },
      data,
      select: publicSelect,
    });
  });

  return updated;
}

export async function deleteGateway(tenantId: string, gatewayId: string) {
  const existing = await prisma.gateway.findFirst({
    where: { id: gatewayId, tenantId },
  });
  if (!existing) throw new AppError('Gateway not found', 404);

  const connectionCount = await prisma.connection.count({
    where: { gatewayId },
  });
  if (connectionCount > 0) {
    throw new AppError(
      `Cannot delete gateway: ${connectionCount} connection(s) are using it. Reassign or remove those connections first.`,
      409,
    );
  }

  await prisma.gateway.delete({ where: { id: gatewayId } });
  return { deleted: true };
}

export async function getGatewayCredentials(
  userId: string,
  tenantId: string,
  gatewayId: string,
): Promise<{ username: string | null; password: string | null }> {
  const gateway = await prisma.gateway.findFirst({
    where: { id: gatewayId, tenantId },
  });
  if (!gateway) throw new AppError('Gateway not found', 404);
  if (gateway.type !== 'SSH_BASTION') {
    return { username: null, password: null };
  }

  const masterKey = requireMasterKey(userId);

  const username =
    gateway.encryptedUsername && gateway.usernameIV && gateway.usernameTag
      ? decrypt(
          { ciphertext: gateway.encryptedUsername, iv: gateway.usernameIV, tag: gateway.usernameTag },
          masterKey,
        )
      : null;

  const password =
    gateway.encryptedPassword && gateway.passwordIV && gateway.passwordTag
      ? decrypt(
          { ciphertext: gateway.encryptedPassword, iv: gateway.passwordIV, tag: gateway.passwordTag },
          masterKey,
        )
      : null;

  return { username, password };
}

export async function testGatewayConnectivity(
  tenantId: string,
  gatewayId: string,
): Promise<{ reachable: boolean; latencyMs: number | null; error: string | null }> {
  const gateway = await prisma.gateway.findFirst({
    where: { id: gatewayId, tenantId },
    select: { host: true, port: true },
  });
  if (!gateway) throw new AppError('Gateway not found', 404);

  const TIMEOUT_MS = 5000;
  const start = Date.now();

  return new Promise((resolve) => {
    const socket = new net.Socket();
    let settled = false;

    const finish = (reachable: boolean, error: string | null) => {
      if (settled) return;
      settled = true;
      socket.destroy();
      resolve({
        reachable,
        latencyMs: reachable ? Date.now() - start : null,
        error,
      });
    };

    socket.setTimeout(TIMEOUT_MS);
    socket.on('connect', () => finish(true, null));
    socket.on('timeout', () => finish(false, 'Connection timed out'));
    socket.on('error', (err) => finish(false, err.message));

    socket.connect(gateway.port, gateway.host);
  });
}
