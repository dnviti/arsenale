/**
 * Background service worker — handles ALL API calls to Arsenale servers,
 * bypassing CORS entirely. Popup/options pages communicate via chrome.runtime.sendMessage.
 */

import {
  getAccounts,
  getActiveAccount,
  setActiveAccountId,
  addAccount,
  updateAccount,
  removeAccount,
  touchAccount,
} from './lib/accountStore';
import type {
  BackgroundMessage,
  BackgroundResponse,
  HealthCheckResult,
  LoginResponse,
  LoginResult,
  PendingAccount,
} from './types';

// ── Token refresh alarm ────────────────────────────────────────────────
const REFRESH_ALARM = 'token-refresh';
const REFRESH_INTERVAL_MINUTES = 10;

chrome.alarms.create(REFRESH_ALARM, { periodInMinutes: REFRESH_INTERVAL_MINUTES });

chrome.alarms.onAlarm.addListener(async (alarm) => {
  // Per-account refresh alarms are named "token-refresh-{accountId}"
  if (alarm.name.startsWith('token-refresh-')) {
    const accountId = alarm.name.replace('token-refresh-', '');
    const result = await refreshTokenForAccount(accountId);
    if (!result.success) {
      // Mark account as session expired and show badge
      await updateAccount({ id: accountId, sessionExpired: true });
      chrome.action.setBadgeText({ text: '!' });
      chrome.action.setBadgeBackgroundColor({ color: '#ef4444' });
    }
    return;
  }

  // Fallback: periodic refresh for active account
  if (alarm.name === REFRESH_ALARM) {
    const account = await getActiveAccount();
    if (!account || account.sessionExpired) return;
    const result = await refreshTokenForAccount(account.id);
    if (!result.success) {
      await updateAccount({ id: account.id, sessionExpired: true });
      chrome.action.setBadgeText({ text: '!' });
      chrome.action.setBadgeBackgroundColor({ color: '#ef4444' });
    }
  }
});

// ── Message handler ────────────────────────────────────────────────────
chrome.runtime.onMessage.addListener(
  (message: BackgroundMessage, _sender, sendResponse: (response: BackgroundResponse) => void) => {
    handleMessage(message).then(sendResponse);
    // Return true to indicate we will respond asynchronously
    return true;
  },
);

async function handleMessage(message: BackgroundMessage): Promise<BackgroundResponse> {
  switch (message.type) {
    case 'HEALTH_CHECK':
      return handleHealthCheck(message.serverUrl);
    case 'LOGIN':
      return handleLogin(message.serverUrl, message.email, message.password);
    case 'VERIFY_TOTP':
      return handleVerifyTotp(message.serverUrl, message.tempToken, message.code, message.pendingAccount);
    case 'REQUEST_SMS_CODE':
      return handleRequestSmsCode(message.serverUrl, message.tempToken);
    case 'VERIFY_SMS':
      return handleVerifySms(message.serverUrl, message.tempToken, message.code, message.pendingAccount);
    case 'REQUEST_WEBAUTHN_OPTIONS':
      return handleRequestWebAuthnOptions(message.serverUrl, message.tempToken);
    case 'VERIFY_WEBAUTHN':
      return handleVerifyWebAuthn(message.serverUrl, message.tempToken, message.credential, message.pendingAccount);
    case 'SWITCH_TENANT':
      return handleSwitchTenant(message.accountId, message.tenantId);
    case 'LOGOUT_ACCOUNT':
      return handleLogoutAccount(message.accountId);
    case 'API_REQUEST':
      return handleApiRequest(message.accountId, message.method, message.path, message.body);
    case 'REFRESH_TOKEN':
      return refreshTokenForAccount(message.accountId);
    case 'GET_ACCOUNTS':
      return handleGetAccounts();
    case 'SET_ACTIVE_ACCOUNT':
      return handleSetActiveAccount(message.accountId);
    case 'REMOVE_ACCOUNT':
      return handleRemoveAccount(message.accountId);
    case 'UPDATE_ACCOUNT':
      return handleUpdateAccount(message.account);
    default:
      return { success: false, error: 'Unknown message type' };
  }
}

// ── Handlers ───────────────────────────────────────────────────────────

async function handleHealthCheck(serverUrl: string): Promise<BackgroundResponse<HealthCheckResult>> {
  try {
    const url = normalizeUrl(serverUrl);
    const res = await fetch(`${url}/api/health`, { method: 'GET' });
    if (!res.ok) return { success: false, error: `Server responded with ${String(res.status)}` };
    const data = (await res.json()) as HealthCheckResult;
    return { success: true, data };
  } catch (err) {
    return { success: false, error: formatError(err) };
  }
}

async function handleLogin(
  serverUrl: string,
  email: string,
  password: string,
): Promise<BackgroundResponse<LoginResponse>> {
  try {
    const url = normalizeUrl(serverUrl);
    const res = await fetch(`${url}/api/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) {
      const body = await res.text();
      let errorMsg = body || `Login failed with ${String(res.status)}`;
      try {
        const parsed = JSON.parse(body) as { error?: string };
        if (parsed.error) errorMsg = parsed.error;
      } catch { /* use raw body */ }
      return { success: false, error: errorMsg };
    }
    const data = (await res.json()) as LoginResponse;

    // If MFA is required or setup is needed, return the challenge info
    if ('requiresMFA' in data || 'mfaSetupRequired' in data) {
      return { success: true, data };
    }

    // Full success — create account entry
    const loginData = data as LoginResult;
    const account = await addAccount({
      label: loginData.user.name || loginData.user.email,
      serverUrl: url,
      userId: loginData.user.id,
      email: loginData.user.email,
      accessToken: loginData.accessToken,
      refreshToken: loginData.refreshToken,
      tenantId: loginData.user.tenantId,
      tenantName: loginData.user.tenantName,
    });

    // Schedule per-account token refresh
    scheduleRefreshAlarm(account.id, loginData.accessToken);

    // Clear any session expired badge
    clearBadgeIfNoExpired();

    return { success: true, data: { ...loginData, accountId: account.id } as unknown as LoginResponse };
  } catch (err) {
    return { success: false, error: formatError(err) };
  }
}

/** Complete MFA with a TOTP code and create the account entry. */
async function handleVerifyTotp(
  serverUrl: string,
  tempToken: string,
  code: string,
  pendingAccount: PendingAccount,
): Promise<BackgroundResponse<LoginResult>> {
  try {
    const url = normalizeUrl(serverUrl);
    const res = await fetch(`${url}/api/auth/verify-totp`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tempToken, code }),
    });
    if (!res.ok) {
      const body = await res.text();
      let errorMsg = body || `TOTP verification failed (${String(res.status)})`;
      try {
        const parsed = JSON.parse(body) as { error?: string };
        if (parsed.error) errorMsg = parsed.error;
      } catch { /* use raw body */ }
      return { success: false, error: errorMsg };
    }
    const data = (await res.json()) as LoginResult;
    return await createAccountFromMfa(url, pendingAccount, data);
  } catch (err) {
    return { success: false, error: formatError(err) };
  }
}

/** Request an SMS code for MFA. */
async function handleRequestSmsCode(
  serverUrl: string,
  tempToken: string,
): Promise<BackgroundResponse> {
  try {
    const url = normalizeUrl(serverUrl);
    const res = await fetch(`${url}/api/auth/request-sms-code`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tempToken }),
    });
    if (!res.ok) {
      const body = await res.text();
      return { success: false, error: body || `SMS request failed (${String(res.status)})` };
    }
    return { success: true };
  } catch (err) {
    return { success: false, error: formatError(err) };
  }
}

/** Complete MFA with an SMS code and create the account entry. */
async function handleVerifySms(
  serverUrl: string,
  tempToken: string,
  code: string,
  pendingAccount: PendingAccount,
): Promise<BackgroundResponse<LoginResult>> {
  try {
    const url = normalizeUrl(serverUrl);
    const res = await fetch(`${url}/api/auth/verify-sms`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tempToken, code }),
    });
    if (!res.ok) {
      const body = await res.text();
      let errorMsg = body || `SMS verification failed (${String(res.status)})`;
      try {
        const parsed = JSON.parse(body) as { error?: string };
        if (parsed.error) errorMsg = parsed.error;
      } catch { /* use raw body */ }
      return { success: false, error: errorMsg };
    }
    const data = (await res.json()) as LoginResult;
    return await createAccountFromMfa(url, pendingAccount, data);
  } catch (err) {
    return { success: false, error: formatError(err) };
  }
}

/** Request WebAuthn assertion options. */
async function handleRequestWebAuthnOptions(
  serverUrl: string,
  tempToken: string,
): Promise<BackgroundResponse<Record<string, unknown>>> {
  try {
    const url = normalizeUrl(serverUrl);
    const res = await fetch(`${url}/api/auth/request-webauthn-options`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tempToken }),
    });
    if (!res.ok) {
      const body = await res.text();
      return { success: false, error: body || `WebAuthn options request failed (${String(res.status)})` };
    }
    const data = (await res.json()) as Record<string, unknown>;
    return { success: true, data };
  } catch (err) {
    return { success: false, error: formatError(err) };
  }
}

/** Complete MFA with a WebAuthn credential and create the account entry. */
async function handleVerifyWebAuthn(
  serverUrl: string,
  tempToken: string,
  credential: Record<string, unknown>,
  pendingAccount: PendingAccount,
): Promise<BackgroundResponse<LoginResult>> {
  try {
    const url = normalizeUrl(serverUrl);
    const res = await fetch(`${url}/api/auth/verify-webauthn`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tempToken, credential }),
    });
    if (!res.ok) {
      const body = await res.text();
      let errorMsg = body || `WebAuthn verification failed (${String(res.status)})`;
      try {
        const parsed = JSON.parse(body) as { error?: string };
        if (parsed.error) errorMsg = parsed.error;
      } catch { /* use raw body */ }
      return { success: false, error: errorMsg };
    }
    const data = (await res.json()) as LoginResult;
    return await createAccountFromMfa(url, pendingAccount, data);
  } catch (err) {
    return { success: false, error: formatError(err) };
  }
}

/** Switch tenant for an existing account. */
async function handleSwitchTenant(
  accountId: string,
  tenantId: string,
): Promise<BackgroundResponse> {
  try {
    const accounts = await getAccounts();
    const account = accounts.find((a) => a.id === accountId);
    if (!account) return { success: false, error: 'Account not found' };

    const res = await fetch(`${account.serverUrl}/api/auth/switch-tenant`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${account.accessToken}`,
      },
      body: JSON.stringify({ tenantId }),
    });

    if (!res.ok) {
      const body = await res.text();
      return { success: false, error: body || `Tenant switch failed (${String(res.status)})` };
    }

    const data = (await res.json()) as { accessToken: string; refreshToken: string; user: LoginResult['user'] };
    await updateAccount({
      id: accountId,
      accessToken: data.accessToken,
      refreshToken: data.refreshToken,
      tenantId: data.user.tenantId,
      tenantName: data.user.tenantName,
    });

    scheduleRefreshAlarm(accountId, data.accessToken);
    return { success: true, data };
  } catch (err) {
    return { success: false, error: formatError(err) };
  }
}

/** Logout: revoke refresh token on the server and remove the account locally. */
async function handleLogoutAccount(accountId: string): Promise<BackgroundResponse> {
  try {
    const accounts = await getAccounts();
    const account = accounts.find((a) => a.id === accountId);
    if (!account) return { success: false, error: 'Account not found' };

    // Best-effort server logout — send refresh token in body (extension pattern)
    try {
      await fetch(`${account.serverUrl}/api/auth/logout`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${account.accessToken}`,
        },
        body: JSON.stringify({ refreshToken: account.refreshToken }),
      });
    } catch {
      // Server logout is best-effort; continue with local cleanup
    }

    // Cancel the per-account refresh alarm
    chrome.alarms.clear(`token-refresh-${accountId}`);

    await removeAccount(accountId);
    clearBadgeIfNoExpired();
    return { success: true };
  } catch (err) {
    return { success: false, error: formatError(err) };
  }
}

async function handleApiRequest(
  accountId: string,
  method: string,
  path: string,
  body?: unknown,
): Promise<BackgroundResponse> {
  try {
    const accounts = await getAccounts();
    const account = accounts.find((a) => a.id === accountId);
    if (!account) return { success: false, error: 'Account not found' };

    const url = `${account.serverUrl}${path.startsWith('/') ? path : `/${path}`}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${account.accessToken}`,
    };

    const res = await fetch(url, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    // Auto-refresh on 401
    if (res.status === 401) {
      const refreshResult = await refreshTokenForAccount(accountId);
      if (!refreshResult.success) return refreshResult;

      // Retry the request with new token
      const refreshedAccounts = await getAccounts();
      const refreshedAccount = refreshedAccounts.find((a) => a.id === accountId);
      if (!refreshedAccount) return { success: false, error: 'Account not found after refresh' };

      headers['Authorization'] = `Bearer ${refreshedAccount.accessToken}`;
      const retryRes = await fetch(url, { method, headers, body: body ? JSON.stringify(body) : undefined });
      const retryData: unknown = await retryRes.json().catch(() => null);
      await touchAccount(accountId);
      return retryRes.ok
        ? { success: true, data: retryData }
        : { success: false, error: `Request failed with ${String(retryRes.status)}` };
    }

    const data: unknown = await res.json().catch(() => null);
    await touchAccount(accountId);
    return res.ok
      ? { success: true, data }
      : { success: false, error: `Request failed with ${String(res.status)}` };
  } catch (err) {
    return { success: false, error: formatError(err) };
  }
}

async function refreshTokenForAccount(accountId: string): Promise<BackgroundResponse> {
  try {
    const accounts = await getAccounts();
    const account = accounts.find((a) => a.id === accountId);
    if (!account) return { success: false, error: 'Account not found' };

    const res = await fetch(`${account.serverUrl}/api/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken: account.refreshToken }),
    });

    if (!res.ok) {
      // Mark session as expired on 401
      if (res.status === 401) {
        await updateAccount({ id: accountId, sessionExpired: true });
        chrome.action.setBadgeText({ text: '!' });
        chrome.action.setBadgeBackgroundColor({ color: '#ef4444' });
      }
      return { success: false, error: 'Token refresh failed' };
    }

    const data = (await res.json()) as { accessToken: string; refreshToken: string };
    await updateAccount({
      id: accountId,
      accessToken: data.accessToken,
      refreshToken: data.refreshToken,
      sessionExpired: false,
    });

    // Reschedule the per-account alarm based on new token expiry
    scheduleRefreshAlarm(accountId, data.accessToken);

    return { success: true };
  } catch (err) {
    return { success: false, error: formatError(err) };
  }
}

async function handleGetAccounts(): Promise<BackgroundResponse> {
  const accounts = await getAccounts();
  return { success: true, data: accounts };
}

async function handleSetActiveAccount(accountId: string): Promise<BackgroundResponse> {
  await setActiveAccountId(accountId);
  await touchAccount(accountId);
  return { success: true };
}

async function handleRemoveAccount(accountId: string): Promise<BackgroundResponse> {
  chrome.alarms.clear(`token-refresh-${accountId}`);
  await removeAccount(accountId);
  return { success: true };
}

async function handleUpdateAccount(
  partial: { id: string } & Record<string, unknown>,
): Promise<BackgroundResponse> {
  const result = await updateAccount(partial as Parameters<typeof updateAccount>[0]);
  return result ? { success: true, data: result } : { success: false, error: 'Account not found' };
}

// ── Shared helpers ─────────────────────────────────────────────────────

/** After MFA verification succeeds, create the account entry and schedule refresh. */
async function createAccountFromMfa(
  serverUrl: string,
  pendingAccount: PendingAccount,
  data: LoginResult,
): Promise<BackgroundResponse<LoginResult>> {
  const account = await addAccount({
    label: data.user.name || data.user.email,
    serverUrl,
    userId: data.user.id,
    email: pendingAccount.email,
    accessToken: data.accessToken,
    refreshToken: data.refreshToken,
    tenantId: data.user.tenantId,
    tenantName: data.user.tenantName,
  });

  scheduleRefreshAlarm(account.id, data.accessToken);
  clearBadgeIfNoExpired();

  return { success: true, data: { ...data, accountId: account.id } as unknown as LoginResult };
}

/**
 * Parse a JWT access token and schedule a chrome.alarms alarm 60s before expiry.
 */
function scheduleRefreshAlarm(accountId: string, accessToken: string): void {
  try {
    const payload = accessToken.split('.')[1];
    if (!payload) return;
    const decoded = JSON.parse(atob(payload)) as { exp?: number };
    if (!decoded.exp) return;

    const expiryMs = decoded.exp * 1000;
    const fireAt = expiryMs - 60_000; // 60s before expiry
    const delayMs = Math.max(fireAt - Date.now(), 5_000); // at least 5s

    chrome.alarms.create(`token-refresh-${accountId}`, {
      delayInMinutes: delayMs / 60_000,
    });
  } catch {
    // If token parsing fails, fall back to the periodic alarm
  }
}

/** Clear the error badge if no accounts have expired sessions. */
async function clearBadgeIfNoExpired(): Promise<void> {
  const accounts = await getAccounts();
  const hasExpired = accounts.some((a) => a.sessionExpired);
  if (!hasExpired) {
    chrome.action.setBadgeText({ text: '' });
  }
}

// ── Utilities ──────────────────────────────────────────────────────────

function normalizeUrl(url: string): string {
  let normalized = url.trim();
  // Strip trailing slash
  while (normalized.endsWith('/')) {
    normalized = normalized.slice(0, -1);
  }
  // Ensure protocol
  if (!normalized.startsWith('http://') && !normalized.startsWith('https://')) {
    normalized = `https://${normalized}`;
  }
  return normalized;
}

function formatError(err: unknown): string {
  if (err instanceof Error) return err.message;
  return String(err);
}
