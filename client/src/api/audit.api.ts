import api from './client';

export type AuditAction =
  | 'LOGIN' | 'LOGIN_OAUTH' | 'LOGIN_TOTP' | 'LOGIN_FAILURE' | 'LOGOUT' | 'REGISTER'
  | 'VAULT_UNLOCK' | 'VAULT_LOCK' | 'VAULT_SETUP'
  | 'CREATE_CONNECTION' | 'UPDATE_CONNECTION' | 'DELETE_CONNECTION'
  | 'SHARE_CONNECTION' | 'UNSHARE_CONNECTION' | 'UPDATE_SHARE_PERMISSION'
  | 'CREATE_FOLDER' | 'UPDATE_FOLDER' | 'DELETE_FOLDER'
  | 'PASSWORD_CHANGE' | 'PROFILE_UPDATE'
  | 'TOTP_ENABLE' | 'TOTP_DISABLE'
  | 'OAUTH_LINK' | 'OAUTH_UNLINK'
  | 'PASSWORD_REVEAL';

export interface AuditLogEntry {
  id: string;
  action: AuditAction;
  targetType: string | null;
  targetId: string | null;
  details: Record<string, unknown> | null;
  ipAddress: string | null;
  createdAt: string;
}

export interface AuditLogResponse {
  data: AuditLogEntry[];
  total: number;
  page: number;
  limit: number;
  totalPages: number;
}

export interface AuditLogParams {
  page?: number;
  limit?: number;
  action?: AuditAction;
  startDate?: string;
  endDate?: string;
}

export async function getAuditLogs(params: AuditLogParams = {}): Promise<AuditLogResponse> {
  const res = await api.get('/audit', { params });
  return res.data;
}
