import prisma, { Prisma, ConnectionType } from '../lib/prisma';
import { encrypt, decrypt, getMasterKey } from './crypto.service';
import { AppError } from '../middleware/error.middleware';
import { resolveTeamKey } from './team.service';
import * as permissionService from './permission.service';
import { ROLE_HIERARCHY } from './permission.service';
import { tenantScopedTeamFilter } from '../utils/tenantScope';

function requireMasterKey(userId: string): Buffer {
  const key = getMasterKey(userId);
  if (!key) throw new AppError('Vault is locked. Please unlock it first.', 403);
  return key;
}

export interface CreateConnectionInput {
  name: string;
  type: ConnectionType;
  host: string;
  port: number;
  username: string;
  password: string;
  description?: string;
  folderId?: string;
  teamId?: string;
  enableDrive?: boolean;
  gatewayId?: string | null;
  sshTerminalConfig?: Prisma.InputJsonValue | null;
  rdpSettings?: Prisma.InputJsonValue | null;
}

export interface UpdateConnectionInput {
  name?: string;
  type?: ConnectionType;
  host?: string;
  port?: number;
  username?: string;
  password?: string;
  description?: string | null;
  folderId?: string | null;
  enableDrive?: boolean;
  gatewayId?: string | null;
  sshTerminalConfig?: Prisma.InputJsonValue | null;
  rdpSettings?: Prisma.InputJsonValue | null;
}

export async function createConnection(userId: string, input: CreateConnectionInput, tenantId?: string | null) {
  let encryptionKey: Buffer;

  if (input.teamId) {
    const perm = await permissionService.canManageTeamResource(userId, input.teamId, 'TEAM_EDITOR', tenantId);
    if (!perm.allowed) throw new AppError('Insufficient team role to create connections', 403);
    encryptionKey = await resolveTeamKey(input.teamId, userId);
  } else {
    encryptionKey = requireMasterKey(userId);
  }

  const encUsername = encrypt(input.username, encryptionKey);
  const encPassword = encrypt(input.password, encryptionKey);

  const connection = await prisma.connection.create({
    data: {
      name: input.name,
      type: input.type,
      host: input.host,
      port: input.port,
      folderId: input.folderId || null,
      teamId: input.teamId || null,
      encryptedUsername: encUsername.ciphertext,
      usernameIV: encUsername.iv,
      usernameTag: encUsername.tag,
      encryptedPassword: encPassword.ciphertext,
      passwordIV: encPassword.iv,
      passwordTag: encPassword.tag,
      description: input.description || null,
      enableDrive: input.enableDrive ?? false,
      gatewayId: input.gatewayId || null,
      sshTerminalConfig: input.sshTerminalConfig ?? undefined,
      rdpSettings: input.rdpSettings ?? undefined,
      userId,
    },
  });

  return {
    id: connection.id,
    name: connection.name,
    type: connection.type,
    host: connection.host,
    port: connection.port,
    folderId: connection.folderId,
    teamId: connection.teamId,
    description: connection.description,
    enableDrive: connection.enableDrive,
    sshTerminalConfig: connection.sshTerminalConfig,
    rdpSettings: connection.rdpSettings,
    createdAt: connection.createdAt,
    updatedAt: connection.updatedAt,
  };
}

export async function updateConnection(
  userId: string,
  connectionId: string,
  input: UpdateConnectionInput,
  tenantId?: string | null
) {
  const access = await permissionService.canManageConnection(userId, connectionId, tenantId);
  if (!access.allowed) throw new AppError('Connection not found', 404);

  const connection = access.connection;
  const encryptionKey = await permissionService.resolveEncryptionKey(userId, connection.teamId);

  const data: Record<string, unknown> = {};

  if (input.name !== undefined) data.name = input.name;
  if (input.type !== undefined) data.type = input.type;
  if (input.host !== undefined) data.host = input.host;
  if (input.port !== undefined) data.port = input.port;
  if (input.description !== undefined) data.description = input.description;
  if (input.folderId !== undefined) data.folderId = input.folderId;
  if (input.enableDrive !== undefined) data.enableDrive = input.enableDrive;
  if (input.gatewayId !== undefined) data.gatewayId = input.gatewayId;
  if (input.sshTerminalConfig !== undefined) data.sshTerminalConfig = input.sshTerminalConfig;
  if (input.rdpSettings !== undefined) data.rdpSettings = input.rdpSettings;

  if (input.username !== undefined) {
    const enc = encrypt(input.username, encryptionKey);
    data.encryptedUsername = enc.ciphertext;
    data.usernameIV = enc.iv;
    data.usernameTag = enc.tag;
  }

  if (input.password !== undefined) {
    const enc = encrypt(input.password, encryptionKey);
    data.encryptedPassword = enc.ciphertext;
    data.passwordIV = enc.iv;
    data.passwordTag = enc.tag;
  }

  const updated = await prisma.connection.update({
    where: { id: connectionId },
    data,
  });

  return {
    id: updated.id,
    name: updated.name,
    type: updated.type,
    host: updated.host,
    port: updated.port,
    folderId: updated.folderId,
    teamId: updated.teamId,
    description: updated.description,
    enableDrive: updated.enableDrive,
    sshTerminalConfig: updated.sshTerminalConfig,
    rdpSettings: updated.rdpSettings,
    createdAt: updated.createdAt,
    updatedAt: updated.updatedAt,
  };
}

export async function deleteConnection(userId: string, connectionId: string, tenantId?: string | null) {
  const access = await permissionService.canManageConnection(userId, connectionId, tenantId);
  if (!access.allowed) throw new AppError('Connection not found', 404);

  await prisma.connection.delete({ where: { id: connectionId } });
  return { deleted: true };
}

export async function getConnection(userId: string, connectionId: string, tenantId?: string | null) {
  const access = await permissionService.canViewConnection(userId, connectionId, tenantId);
  if (!access.allowed) throw new AppError('Connection not found', 404);

  const connection = access.connection;

  if (access.accessType === 'owner') {
    return {
      id: connection.id,
      name: connection.name,
      type: connection.type,
      host: connection.host,
      port: connection.port,
      folderId: connection.folderId,
      teamId: connection.teamId,
      description: connection.description,
      enableDrive: connection.enableDrive,
      sshTerminalConfig: connection.sshTerminalConfig,
      rdpSettings: connection.rdpSettings,
      gatewayId: connection.gatewayId,
      gateway: connection.gateway,
      isOwner: true,
      scope: 'private' as const,
      createdAt: connection.createdAt,
      updatedAt: connection.updatedAt,
    };
  }

  if (access.accessType === 'team') {
    return {
      id: connection.id,
      name: connection.name,
      type: connection.type,
      host: connection.host,
      port: connection.port,
      folderId: connection.folderId,
      teamId: connection.teamId,
      description: connection.description,
      enableDrive: connection.enableDrive,
      sshTerminalConfig: connection.sshTerminalConfig,
      rdpSettings: connection.rdpSettings,
      gatewayId: connection.gatewayId,
      gateway: connection.gateway,
      isOwner: false,
      scope: 'team' as const,
      teamRole: access.teamRole,
      createdAt: connection.createdAt,
      updatedAt: connection.updatedAt,
    };
  }

  // Shared
  const shared = await prisma.sharedConnection.findFirst({
    where: { connectionId, sharedWithUserId: userId },
  });
  return {
    id: connection.id,
    name: connection.name,
    type: connection.type,
    host: connection.host,
    port: connection.port,
    folderId: null,
    teamId: null,
    description: connection.description,
    enableDrive: connection.enableDrive,
    sshTerminalConfig: connection.sshTerminalConfig,
    rdpSettings: connection.rdpSettings,
    gatewayId: connection.gatewayId,
    gateway: connection.gateway,
    isOwner: false,
    scope: 'shared' as const,
    permission: shared?.permission,
    createdAt: connection.createdAt,
    updatedAt: connection.updatedAt,
  };
}

export async function listConnections(userId: string, tenantId?: string | null) {
  // Personal connections (exclude team connections)
  const ownConnections = await prisma.connection.findMany({
    where: { userId, teamId: null },
    select: {
      id: true,
      name: true,
      type: true,
      host: true,
      port: true,
      folderId: true,
      description: true,
      isFavorite: true,
      enableDrive: true,
      sshTerminalConfig: true,
      rdpSettings: true,
      createdAt: true,
      updatedAt: true,
    },
    orderBy: { name: 'asc' },
  });

  // Shared connections
  const sharedConnections = await prisma.sharedConnection.findMany({
    where: { sharedWithUserId: userId },
    include: {
      connection: {
        select: {
          id: true,
          name: true,
          type: true,
          host: true,
          port: true,
          description: true,
          enableDrive: true,
          sshTerminalConfig: true,
          rdpSettings: true,
          createdAt: true,
          updatedAt: true,
        },
      },
      sharedBy: { select: { email: true } },
    },
  });

  // Team connections
  const userTeamMemberships = await prisma.teamMember.findMany({
    where: { userId, ...tenantScopedTeamFilter(tenantId) },
    select: { teamId: true, role: true, team: { select: { name: true } } },
  });

  let teamConnections: Array<Record<string, unknown>> = [];
  if (userTeamMemberships.length > 0) {
    const teamIdList = userTeamMemberships.map((m) => m.teamId);
    const teamNameMap = new Map(userTeamMemberships.map((m) => [m.teamId, m.team.name]));
    const teamRoleMap = new Map(userTeamMemberships.map((m) => [m.teamId, m.role]));

    const rawTeamConns = await prisma.connection.findMany({
      where: { teamId: { in: teamIdList } },
      select: {
        id: true,
        name: true,
        type: true,
        host: true,
        port: true,
        folderId: true,
        teamId: true,
        description: true,
        isFavorite: true,
        enableDrive: true,
        sshTerminalConfig: true,
        rdpSettings: true,
        createdAt: true,
        updatedAt: true,
      },
      orderBy: { name: 'asc' },
    });

    teamConnections = rawTeamConns.map((c) => ({
      ...c,
      teamName: teamNameMap.get(c.teamId!) ?? null,
      teamRole: teamRoleMap.get(c.teamId!) ?? null,
      isOwner: false,
      scope: 'team' as const,
    }));
  }

  return {
    own: ownConnections.map((c: (typeof ownConnections)[number]) => ({
      ...c,
      isOwner: true,
      scope: 'private' as const,
    })),
    shared: sharedConnections.map((s: (typeof sharedConnections)[number]) => ({
      ...s.connection,
      folderId: null,
      isOwner: false,
      isFavorite: false,
      permission: s.permission,
      sharedBy: s.sharedBy.email,
      scope: 'shared' as const,
    })),
    team: teamConnections,
  };
}

export async function getConnectionCredentials(
  userId: string,
  connectionId: string,
  tenantId?: string | null
): Promise<{ username: string; password: string }> {
  const access = await permissionService.canViewConnection(userId, connectionId, tenantId);
  if (!access.allowed) throw new AppError('Connection not found or credentials unavailable', 404);

  const connection = access.connection;

  if (access.accessType === 'owner') {
    const masterKey = requireMasterKey(userId);
    return {
      username: decrypt(
        { ciphertext: connection.encryptedUsername, iv: connection.usernameIV, tag: connection.usernameTag },
        masterKey
      ),
      password: decrypt(
        { ciphertext: connection.encryptedPassword, iv: connection.passwordIV, tag: connection.passwordTag },
        masterKey
      ),
    };
  }

  if (access.accessType === 'team') {
    const teamKey = await resolveTeamKey(connection.teamId!, userId);
    return {
      username: decrypt(
        { ciphertext: connection.encryptedUsername, iv: connection.usernameIV, tag: connection.usernameTag },
        teamKey
      ),
      password: decrypt(
        { ciphertext: connection.encryptedPassword, iv: connection.passwordIV, tag: connection.passwordTag },
        teamKey
      ),
    };
  }

  // Shared: decrypt from SharedConnection re-encrypted copy
  const masterKey = requireMasterKey(userId);
  const shared = await prisma.sharedConnection.findFirst({
    where: { connectionId, sharedWithUserId: userId },
  });

  if (shared?.encryptedUsername && shared.usernameIV && shared.usernameTag &&
      shared.encryptedPassword && shared.passwordIV && shared.passwordTag) {
    return {
      username: decrypt(
        { ciphertext: shared.encryptedUsername, iv: shared.usernameIV, tag: shared.usernameTag },
        masterKey
      ),
      password: decrypt(
        { ciphertext: shared.encryptedPassword, iv: shared.passwordIV, tag: shared.passwordTag },
        masterKey
      ),
    };
  }

  throw new AppError('Connection not found or credentials unavailable', 404);
}

export async function toggleFavorite(userId: string, connectionId: string, tenantId?: string | null) {
  const access = await permissionService.canViewConnection(userId, connectionId, tenantId);
  if (!access.allowed) throw new AppError('Connection not found', 404);

  if (access.accessType === 'shared') {
    throw new AppError('Cannot favorite shared connections', 403);
  }

  if (access.accessType === 'team') {
    if (ROLE_HIERARCHY[access.teamRole!] < ROLE_HIERARCHY['TEAM_EDITOR']) {
      throw new AppError('Viewers cannot toggle favorites on team connections', 403);
    }
  }

  const updated = await prisma.connection.update({
    where: { id: connectionId },
    data: { isFavorite: !access.connection.isFavorite },
  });

  return { id: updated.id, isFavorite: updated.isFavorite };
}
