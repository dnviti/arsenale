/** Represents a configured Arsenale server account. */
export interface Account {
  /** Unique identifier (UUID v4). */
  id: string;
  /** User-visible label (e.g. "Production", "Home Lab"). */
  label: string;
  /** Base URL of the Arsenale server (e.g. "https://arsenale.example.com"). */
  serverUrl: string;
  /** User ID returned after authentication. */
  userId: string;
  /** User email. */
  email: string;
  /** Short-lived JWT access token. */
  accessToken: string;
  /** Refresh token for obtaining new access tokens. */
  refreshToken: string;
  /** Optional tenant ID for multi-tenant deployments. */
  tenantId?: string;
  /** Optional tenant display name. */
  tenantName?: string;
  /** ISO-8601 timestamp of last activity. */
  lastUsed: string;
  /** Whether the vault is currently unlocked for this account. */
  vaultUnlocked: boolean;
  /** Whether the session has expired (refresh failed with 401). */
  sessionExpired?: boolean;
}

/** Shape of the stored data in chrome.storage.local. */
export interface StorageSchema {
  /** All configured accounts. */
  accounts: Account[];
  /** ID of the currently active account (or null if none). */
  activeAccountId: string | null;
}

/** Messages sent from popup/options to the service worker. */
export type BackgroundMessage =
  | { type: 'API_REQUEST'; accountId: string; method: 'GET' | 'POST' | 'PUT' | 'DELETE'; path: string; body?: unknown }
  | { type: 'HEALTH_CHECK'; serverUrl: string }
  | { type: 'LOGIN'; serverUrl: string; email: string; password: string }
  | { type: 'VERIFY_TOTP'; serverUrl: string; tempToken: string; code: string; pendingAccount: PendingAccount }
  | { type: 'REQUEST_SMS_CODE'; serverUrl: string; tempToken: string }
  | { type: 'VERIFY_SMS'; serverUrl: string; tempToken: string; code: string; pendingAccount: PendingAccount }
  | { type: 'REQUEST_WEBAUTHN_OPTIONS'; serverUrl: string; tempToken: string }
  | { type: 'VERIFY_WEBAUTHN'; serverUrl: string; tempToken: string; credential: Record<string, unknown>; pendingAccount: PendingAccount }
  | { type: 'SWITCH_TENANT'; accountId: string; tenantId: string }
  | { type: 'LOGOUT_ACCOUNT'; accountId: string }
  | { type: 'REFRESH_TOKEN'; accountId: string }
  | { type: 'GET_ACCOUNTS' }
  | { type: 'SET_ACTIVE_ACCOUNT'; accountId: string }
  | { type: 'REMOVE_ACCOUNT'; accountId: string }
  | { type: 'UPDATE_ACCOUNT'; account: Partial<Account> & { id: string } };

/** Standardized response from the service worker. */
export interface BackgroundResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
}

/** Health check response from /api/health. */
export interface HealthCheckResult {
  status: string;
  version?: string;
}

/** Partial account info carried through the MFA flow before full account creation. */
export interface PendingAccount {
  serverUrl: string;
  email: string;
}

/** Tenant membership entry returned by the server. */
export interface TenantMembership {
  tenantId: string;
  name: string;
  slug: string;
  role: string;
  isActive: boolean;
}

/** Login response from /api/auth/login — full success (no MFA required). */
export interface LoginResult {
  accessToken: string;
  refreshToken: string;
  csrfToken?: string;
  user: {
    id: string;
    email: string;
    name: string;
    tenantId?: string;
    tenantName?: string;
  };
  tenantMemberships?: TenantMembership[];
}

/** Login response when MFA is required. */
export interface LoginMfaRequired {
  requiresMFA: true;
  requiresTOTP?: boolean;
  methods: string[];
  tempToken: string;
}

/** Login response when MFA setup is required before first login. */
export interface LoginMfaSetupRequired {
  mfaSetupRequired: true;
  tempToken: string;
}

/** Union type for all possible /api/auth/login responses. */
export type LoginResponse = LoginResult | LoginMfaRequired | LoginMfaSetupRequired;
