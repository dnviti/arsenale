/**
 * AWS Secrets Manager adapter.
 *
 * Auth methods:
 *   IAM_ACCESS_KEY — { accessKeyId, secretAccessKey, region }
 *   IAM_ROLE       — { region, roleArn? }   (uses environment/IRSA credentials)
 *
 * Secrets are addressed by name or ARN. Versioning via AWSCURRENT / AWSPREVIOUS
 * staging labels is supported through the secretPath syntax:
 *   "my-secret"                → latest (AWSCURRENT)
 *   "my-secret#AWSPREVIOUS"   → previous version
 */

import { createHmac, createHash } from 'node:crypto';
import { decryptWithServerKey } from '../crypto.service';
import { AppError } from '../../middleware/error.middleware';
import type { VaultAdapter, VaultProviderRow } from './types';

const REQUEST_TIMEOUT_MS = 10_000;

// ---------- SigV4 signing (minimal, no SDK) ----------

interface AwsCreds {
  accessKeyId: string;
  secretAccessKey: string;
  region: string;
}

function sha256(data: string): string {
  return createHash('sha256').update(data).digest('hex');
}

function hmacSha256(key: Buffer | string, data: string): Buffer {
  return createHmac('sha256', key).update(data).digest();
}

function sigV4Sign(
  creds: AwsCreds,
  method: string,
  url: URL,
  headers: Record<string, string>,
  body: string,
): Record<string, string> {
  const service = 'secretsmanager';
  const now = new Date();
  const dateStamp = now.toISOString().replace(/[-:T]/g, '').slice(0, 8);
  const amzDate = `${dateStamp}T${now.toISOString().replace(/[-:T]/g, '').slice(8, 14)}Z`;

  headers['x-amz-date'] = amzDate;

  const signedHeaderNames = Object.keys(headers).sort().join(';');
  const canonicalHeaders = Object.keys(headers)
    .sort()
    .map((k) => `${k}:${headers[k].trim()}`)
    .join('\n') + '\n';

  const canonicalRequest = [
    method,
    url.pathname || '/',
    url.search.replace(/^\?/, ''),
    canonicalHeaders,
    signedHeaderNames,
    sha256(body),
  ].join('\n');

  const credentialScope = `${dateStamp}/${creds.region}/${service}/aws4_request`;
  const stringToSign = ['AWS4-HMAC-SHA256', amzDate, credentialScope, sha256(canonicalRequest)].join('\n');

  let signingKey: Buffer = hmacSha256(`AWS4${creds.secretAccessKey}`, dateStamp);
  signingKey = hmacSha256(signingKey, creds.region);
  signingKey = hmacSha256(signingKey, service);
  signingKey = hmacSha256(signingKey, 'aws4_request');

  const signature = hmacSha256(signingKey, stringToSign).toString('hex');

  headers['Authorization'] = `AWS4-HMAC-SHA256 Credential=${creds.accessKeyId}/${credentialScope}, SignedHeaders=${signedHeaderNames}, Signature=${signature}`;

  return headers;
}

// ---------- Helpers ----------

function parsePayload(provider: VaultProviderRow): AwsCreds {
  const json = decryptWithServerKey({
    ciphertext: provider.encryptedAuthPayload,
    iv: provider.authPayloadIV,
    tag: provider.authPayloadTag,
  });

  let payload: Record<string, unknown>;
  try {
    payload = JSON.parse(json) as Record<string, unknown>;
  } catch {
    throw new AppError('Failed to parse AWS auth payload — credentials may be corrupted', 500);
  }

  if (provider.authMethod === 'IAM_ROLE') {
    // When using IAM_ROLE, access keys come from the environment (IRSA / instance profile).
    // We still need a region.
    const region = (payload.region as string) || provider.serverUrl || 'us-east-1';
    const accessKeyId = process.env.AWS_ACCESS_KEY_ID ?? '';
    const secretAccessKey = process.env.AWS_SECRET_ACCESS_KEY ?? '';
    if (!accessKeyId || !secretAccessKey) {
      throw new AppError('IAM_ROLE auth requires AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY in the environment', 400);
    }
    return { accessKeyId, secretAccessKey, region };
  }

  // IAM_ACCESS_KEY
  const accessKeyId = payload.accessKeyId as string;
  const secretAccessKey = payload.secretAccessKey as string;
  const region = (payload.region as string) || 'us-east-1';
  if (!accessKeyId || !secretAccessKey) {
    throw new AppError('AWS auth payload must contain accessKeyId and secretAccessKey', 400);
  }
  return { accessKeyId, secretAccessKey, region };
}

function parseSecretPath(secretPath: string): { secretId: string; versionStage?: string } {
  const [secretId, versionStage] = secretPath.split('#', 2);
  return { secretId, versionStage: versionStage || undefined };
}

async function fetchSecret(creds: AwsCreds, secretId: string, versionStage?: string): Promise<Record<string, string>> {
  const endpoint = `https://secretsmanager.${creds.region}.amazonaws.com`;
  const url = new URL(endpoint);

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const bodyObj: Record<string, any> = { SecretId: secretId };
  if (versionStage) bodyObj.VersionStage = versionStage;
  const body = JSON.stringify(bodyObj);

  const headers: Record<string, string> = {
    'content-type': 'application/x-amz-json-1.1',
    'x-amz-target': 'secretsmanager.GetSecretValue',
    host: url.host,
  };

  sigV4Sign(creds, 'POST', url, headers, body);

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

  let resp: Response;
  try {
    resp = await fetch(endpoint, { method: 'POST', headers, body, signal: controller.signal });
  } catch (err) {
    if ((err as Error).name === 'AbortError') {
      throw new AppError(`AWS Secrets Manager request timed out after ${REQUEST_TIMEOUT_MS}ms`, 504);
    }
    throw err;
  } finally {
    clearTimeout(timeout);
  }

  if (!resp.ok) {
    const text = await resp.text().catch(() => '');
    throw new AppError(`AWS Secrets Manager API error (${resp.status}): ${text.slice(0, 200)}`, 502);
  }

  const result = (await resp.json()) as { SecretString?: string; SecretBinary?: string };
  if (!result.SecretString) {
    throw new AppError(`Secret "${secretId}" has no string value (binary secrets are not supported)`, 502);
  }

  try {
    return JSON.parse(result.SecretString) as Record<string, string>;
  } catch {
    // Plain-text secret — return as { value: "<raw>" }
    return { value: result.SecretString };
  }
}

// ---------- Cache ----------

interface CachedSecret { data: Record<string, string>; expiresAt: number }
const secretCache = new Map<string, CachedSecret>();

setInterval(() => {
  const now = Date.now();
  for (const [key, entry] of secretCache.entries()) {
    if (entry.expiresAt < now) secretCache.delete(key);
  }
}, 60_000);

// ---------- Adapter ----------

export const awsAdapter: VaultAdapter = {
  async readSecret(provider, secretPath) {
    const cacheKey = `${provider.id}:${secretPath}`;
    const cached = secretCache.get(cacheKey);
    if (cached && cached.expiresAt > Date.now()) return cached.data;

    const creds = parsePayload(provider);
    const { secretId, versionStage } = parseSecretPath(secretPath);
    const data = await fetchSecret(creds, secretId, versionStage);

    if (provider.cacheTtlSeconds > 0) {
      secretCache.set(cacheKey, { data, expiresAt: Date.now() + provider.cacheTtlSeconds * 1000 });
    }
    return data;
  },

  async testConnection(provider, secretPath) {
    try {
      const data = await this.readSecret(provider, secretPath);
      return { success: true, keys: Object.keys(data) };
    } catch (err) {
      return { success: false, error: err instanceof Error ? err.message : 'Unknown error' };
    }
  },
};

export function invalidateAwsCaches(providerId: string): void {
  for (const key of secretCache.keys()) {
    if (key.startsWith(`${providerId}:`)) secretCache.delete(key);
  }
}
