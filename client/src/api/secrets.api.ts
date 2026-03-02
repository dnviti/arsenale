import api from './client';

// --- Secret payload types (discriminated union) ---

export interface LoginData {
  type: 'LOGIN';
  username: string;
  password: string;
  url?: string;
  notes?: string;
}

export interface SshKeyData {
  type: 'SSH_KEY';
  username?: string;
  privateKey: string;
  publicKey?: string;
  passphrase?: string;
  algorithm?: string;
  notes?: string;
}

export interface CertificateData {
  type: 'CERTIFICATE';
  certificate: string;
  privateKey: string;
  chain?: string;
  passphrase?: string;
  expiresAt?: string;
  notes?: string;
}

export interface ApiKeyData {
  type: 'API_KEY';
  apiKey: string;
  endpoint?: string;
  headers?: Record<string, string>;
  notes?: string;
}

export interface SecureNoteData {
  type: 'SECURE_NOTE';
  content: string;
}

export type SecretPayload =
  | LoginData
  | SshKeyData
  | CertificateData
  | ApiKeyData
  | SecureNoteData;

export type SecretType = 'LOGIN' | 'SSH_KEY' | 'CERTIFICATE' | 'API_KEY' | 'SECURE_NOTE';
export type SecretScope = 'PERSONAL' | 'TEAM' | 'TENANT';

// --- List / detail shapes ---

export interface SecretListItem {
  id: string;
  name: string;
  description: string | null;
  type: SecretType;
  scope: SecretScope;
  teamId: string | null;
  tenantId: string | null;
  folderId: string | null;
  metadata: Record<string, unknown> | null;
  tags: string[];
  isFavorite: boolean;
  expiresAt: string | null;
  currentVersion: number;
  createdAt: string;
  updatedAt: string;
}

export interface SecretDetail extends SecretListItem {
  data: SecretPayload;
  shared?: boolean;
  permission?: 'READ_ONLY' | 'FULL_ACCESS';
}

// --- Input shapes ---

export interface CreateSecretInput {
  name: string;
  description?: string;
  type: SecretType;
  scope: SecretScope;
  teamId?: string;
  folderId?: string;
  data: SecretPayload;
  metadata?: Record<string, unknown>;
  tags?: string[];
  expiresAt?: string;
}

export interface UpdateSecretInput {
  name?: string;
  description?: string | null;
  data?: SecretPayload;
  metadata?: Record<string, unknown> | null;
  tags?: string[];
  folderId?: string | null;
  isFavorite?: boolean;
  expiresAt?: string | null;
  changeNote?: string;
}

// --- Version / share shapes ---

export interface SecretVersion {
  id: string;
  version: number;
  changedBy: string;
  changeNote: string | null;
  createdAt: string;
  changer?: { email: string; username: string | null };
}

export interface SecretShare {
  id: string;
  userId: string;
  email: string;
  permission: 'READ_ONLY' | 'FULL_ACCESS';
  createdAt: string;
}

export interface TenantVaultStatus {
  initialized: boolean;
  hasAccess: boolean;
}

// --- Filter shape ---

export interface SecretListFilters {
  scope?: SecretScope;
  type?: SecretType;
  teamId?: string;
  folderId?: string | null;
  search?: string;
  tags?: string[];
  isFavorite?: boolean;
}

// --- API functions ---

export async function listSecrets(filters?: SecretListFilters): Promise<SecretListItem[]> {
  const params: Record<string, string> = {};
  if (filters?.scope) params.scope = filters.scope;
  if (filters?.type) params.type = filters.type;
  if (filters?.teamId) params.teamId = filters.teamId;
  if (filters?.folderId !== undefined && filters.folderId !== null) params.folderId = filters.folderId;
  if (filters?.search) params.search = filters.search;
  if (filters?.isFavorite !== undefined) params.isFavorite = String(filters.isFavorite);
  if (filters?.tags?.length) params.tags = filters.tags.join(',');
  const res = await api.get('/secrets', { params });
  return res.data;
}

export async function getSecret(id: string): Promise<SecretDetail> {
  const res = await api.get(`/secrets/${id}`);
  return res.data;
}

export async function createSecret(input: CreateSecretInput): Promise<SecretListItem> {
  const res = await api.post('/secrets', input);
  return res.data;
}

export async function updateSecret(id: string, input: UpdateSecretInput): Promise<SecretListItem> {
  const res = await api.put(`/secrets/${id}`, input);
  return res.data;
}

export async function deleteSecret(id: string): Promise<{ deleted: true }> {
  const res = await api.delete(`/secrets/${id}`);
  return res.data;
}

export async function listVersions(id: string): Promise<SecretVersion[]> {
  const res = await api.get(`/secrets/${id}/versions`);
  return res.data;
}

export async function restoreVersion(id: string, version: number): Promise<SecretListItem> {
  const res = await api.post(`/secrets/${id}/versions/${version}/restore`);
  return res.data;
}

export async function shareSecret(
  id: string,
  target: { email?: string; userId?: string },
  permission: 'READ_ONLY' | 'FULL_ACCESS',
): Promise<SecretShare> {
  const res = await api.post(`/secrets/${id}/share`, { ...target, permission });
  return res.data;
}

export async function unshareSecret(id: string, userId: string): Promise<{ deleted: true }> {
  const res = await api.delete(`/secrets/${id}/share/${userId}`);
  return res.data;
}

export async function updateSharePermission(
  id: string,
  userId: string,
  permission: 'READ_ONLY' | 'FULL_ACCESS',
): Promise<SecretShare> {
  const res = await api.put(`/secrets/${id}/share/${userId}`, { permission });
  return res.data;
}

export async function listShares(id: string): Promise<SecretShare[]> {
  const res = await api.get(`/secrets/${id}/shares`);
  return res.data;
}

export async function initTenantVault(): Promise<{ initialized: true }> {
  const res = await api.post('/secrets/tenant-vault/init');
  return res.data;
}

export async function distributeTenantKey(targetUserId: string): Promise<{ distributed: true }> {
  const res = await api.post('/secrets/tenant-vault/distribute', { targetUserId });
  return res.data;
}

export async function getTenantVaultStatus(): Promise<TenantVaultStatus> {
  const res = await api.get('/secrets/tenant-vault/status');
  return res.data;
}
