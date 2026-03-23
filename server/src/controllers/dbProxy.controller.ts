import { Response, NextFunction } from 'express';
import { AuthRequest, assertAuthenticated } from '../types';
import { getConnection } from '../services/connection.service';
import * as dbSessionService from '../services/dbSession.service';
import * as auditService from '../services/audit.service';
import { AppError } from '../middleware/error.middleware';
import { getClientIp } from '../utils/ip';

// ---- Database proxy session creation ----

export async function createSession(req: AuthRequest, res: Response, next: NextFunction) {
  let connectionId: string | undefined;

  try {
    assertAuthenticated(req);
    const { connectionId: connId, username, password } = req.body as {
      connectionId: string;
      username?: string;
      password?: string;
    };
    connectionId = connId;

    if (!connectionId) {
      throw new AppError('connectionId is required', 400);
    }

    // Validate the user can access this connection
    const conn = await getConnection(req.user.userId, connectionId, req.user.tenantId);
    if (conn.type !== 'DATABASE') {
      throw new AppError('Not a DATABASE connection', 400);
    }

    const result = await dbSessionService.createSession({
      userId: req.user.userId,
      connectionId,
      tenantId: req.user.tenantId,
      ipAddress: getClientIp(req) ?? undefined,
      overrideUsername: username,
      overridePassword: password,
    });

    res.json(result);
  } catch (err) {
    const errorMessage = err instanceof Error ? err.message : 'Unknown error';
    auditService.log({
      userId: req.user?.userId,
      action: 'SESSION_ERROR',
      targetType: 'Connection',
      targetId: connectionId,
      details: { protocol: 'DATABASE', error: errorMessage },
      ipAddress: getClientIp(req),
    });
    next(err);
  }
}

// ---- Database proxy session end ----

export async function endSession(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const sessionId = req.params.sessionId as string;
  await dbSessionService.endSession(req.user.userId, sessionId);
  res.json({ ok: true });
}

// ---- Database session heartbeat ----

export async function heartbeat(req: AuthRequest, res: Response) {
  assertAuthenticated(req);
  const sessionId = req.params.sessionId as string;
  await dbSessionService.heartbeat(sessionId, req.user.userId);
  res.json({ ok: true });
}

// ---- Execute SQL query ----

export async function executeQuery(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    assertAuthenticated(req);
    const sessionId = req.params.sessionId as string;
    const { sql } = req.body as { sql: string };

    if (!sql || typeof sql !== 'string') {
      throw new AppError('sql is required', 400);
    }

    // tenantId is guaranteed by requireTenant middleware on this route
    const tenantId = req.user.tenantId as string;

    const result = await dbSessionService.executeQuery({
      userId: req.user.userId,
      tenantId,
      tenantRole: req.user.tenantRole,
      sessionId,
      sql,
      ipAddress: getClientIp(req) ?? undefined,
    });

    res.json(result);
  } catch (err) {
    next(err);
  }
}

// ---- Get database schema ----

export async function getSchema(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    assertAuthenticated(req);
    const sessionId = req.params.sessionId as string;
    const tenantId = req.user.tenantId as string;
    const schema = await dbSessionService.getSchema(req.user.userId, sessionId, tenantId);
    res.json(schema);
  } catch (err) {
    next(err);
  }
}

// ---- Get execution plan (EXPLAIN) ----

export async function getExecutionPlan(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    assertAuthenticated(req);
    const sessionId = req.params.sessionId as string;
    const { sql } = req.body as { sql: string };

    if (!sql || typeof sql !== 'string') {
      throw new AppError('sql is required', 400);
    }

    const tenantId = req.user.tenantId as string;
    const result = await dbSessionService.getExecutionPlan({
      userId: req.user.userId,
      tenantId,
      tenantRole: req.user.tenantRole,
      sessionId,
      sql,
      ipAddress: getClientIp(req) ?? undefined,
    });

    res.json(result);
  } catch (err) {
    next(err);
  }
}

// ---- Database introspection ----

const VALID_INTROSPECTION_TYPES = new Set([
  'indexes', 'statistics', 'foreign_keys', 'table_schema', 'row_count', 'database_version',
]);

export async function introspectDatabase(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    assertAuthenticated(req);
    const sessionId = req.params.sessionId as string;
    const { type, target } = req.body as { type: string; target?: string };

    if (!type || typeof type !== 'string') {
      throw new AppError('type is required', 400);
    }
    if (!VALID_INTROSPECTION_TYPES.has(type)) {
      throw new AppError(`Invalid introspection type: ${type}`, 400);
    }
    // target is required for all types except database_version
    if (type !== 'database_version' && (!target || typeof target !== 'string')) {
      throw new AppError('target is required for this introspection type', 400);
    }

    const tenantId = req.user.tenantId as string;
    const result = await dbSessionService.introspectDatabase({
      userId: req.user.userId,
      tenantId,
      tenantRole: req.user.tenantRole,
      sessionId,
      type: type as Parameters<typeof dbSessionService.introspectDatabase>[0]['type'],
      target: target ?? '',
      ipAddress: getClientIp(req) ?? undefined,
    });

    res.json(result);
  } catch (err) {
    next(err);
  }
}

// ---- Query history ----

export async function getQueryHistory(req: AuthRequest, res: Response, next: NextFunction) {
  try {
    assertAuthenticated(req);
    const sessionId = req.params.sessionId as string;
    const limit = req.query.limit ? parseInt(req.query.limit as string, 10) : undefined;
    const search = (req.query.search as string) || undefined;

    const history = await dbSessionService.getQueryHistory({
      userId: req.user.userId,
      sessionId,
      limit,
      search,
    });

    res.json(history);
  } catch (err) {
    next(err);
  }
}
