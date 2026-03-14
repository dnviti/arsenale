# Security

> Auto-generated on 2026-03-14 by `/docs create security`.
> Source of truth is the codebase. Run `/docs update security` after code changes.

## Overview

Arsenale employs a defense-in-depth security model:

1. **Vault encryption** — all credentials encrypted at rest with AES-256-GCM, user-derived keys via Argon2id
2. **JWT authentication** — short-lived access tokens with httpOnly refresh token cookies and CSRF protection
3. **Multi-factor authentication** — TOTP, SMS OTP, and WebAuthn/FIDO2 passkeys
4. **Tenant isolation** — multi-tenant RBAC with per-tenant policies
5. **Audit logging** — 100+ action types with IP and geo-location tracking
6. **Rate limiting** — per-IP login throttling and account lockout
7. **Security headers** — Helmet with strict CSP, HSTS, and frame protection

<!-- manual-start -->
<!-- manual-end -->

## Vault Encryption

### Algorithm

- **Cipher**: AES-256-GCM (authenticated encryption)
- **IV length**: 16 bytes (randomly generated per encryption)
- **Key length**: 32 bytes (256 bits)
- **Salt length**: 32 bytes (for key derivation)
- **Auth tag**: Included with every ciphertext for integrity verification

Source: `server/src/services/crypto.service.ts` — constants `ALGORITHM`, `IV_LENGTH`, `KEY_LENGTH`, `SALT_LENGTH`.

### Key Derivation

Master keys are derived from the user's password using Argon2id:

| Parameter | Value |
|-----------|-------|
| **Algorithm** | argon2id |
| **Memory cost** | 65,536 KiB (64 MB) |
| **Time cost** | 3 iterations |
| **Parallelism** | 1 |
| **Hash length** | 32 bytes (256 bits) |

Source: `crypto.service.ts` `deriveKeyFromPassword()` function.

### Master Key Lifecycle

1. **Registration**: A random 32-byte master key is generated (`crypto.randomBytes(KEY_LENGTH)`)
2. **Derivation**: The user's password is combined with a random 32-byte salt via Argon2id to produce a derived key
3. **Encryption**: The master key is encrypted with the derived key using AES-256-GCM
4. **Storage**: The encrypted master key (`encryptedVaultKey`), IV (`vaultKeyIV`), auth tag (`vaultKeyTag`), and salt (`vaultSalt`) are stored in the `User` record
5. **Unlock**: When the user enters their password, the derived key is recreated from the salt, and the master key is decrypted
6. **Session**: The decrypted master key is held in-memory in the vault session store with a configurable TTL

### Encrypted Field Structure

All encrypted data is stored as an `EncryptedField`:

```typescript
interface EncryptedField {
  ciphertext: string;  // hex-encoded AES-256-GCM ciphertext
  iv: string;          // hex-encoded 16-byte initialization vector
  tag: string;         // hex-encoded GCM authentication tag
}
```

In the database, these are stored as three separate columns (e.g., `encryptedUsername`, `usernameIV`, `usernameTag`).

### Recovery Key

During registration, a recovery key is generated (`crypto.randomBytes(32).toString('base64url')`). The master key is encrypted with a key derived from the recovery key (using the same Argon2id parameters) and stored separately. This allows vault recovery during password reset without losing encrypted data.

<!-- manual-start -->
<!-- manual-end -->

## Vault Session Management

### Session Lifecycle

1. **Unlock**: User provides password (or MFA for re-unlock). Master key is decrypted and stored in the in-memory `vaultStore` Map.
2. **Active**: Every vault access resets the TTL (sliding window). Default TTL: 30 minutes (`VAULT_TTL_MINUTES`).
3. **Soft lock**: TTL expiry or manual lock clears the vault session but preserves the recovery entry for MFA re-unlock.
4. **Hard lock**: Logout or password change clears both the vault session AND the recovery entry.
5. **Auto-expiry**: A cleanup interval runs every 60 seconds, zeroing out expired master keys and deleting sessions.

### Memory Cleanup

- Master keys are zeroed (`buffer.fill(0)`) before deletion from the store
- The periodic cleanup interval (60s) catches both expired vault sessions, team vault sessions, tenant vault sessions, and recovery entries
- Team and tenant vault sessions are locked in cascade when the user's vault session expires

### Vault Recovery (MFA Re-unlock)

When the vault is unlocked with a password, the master key is also encrypted with the `SERVER_ENCRYPTION_KEY` and stored in the recovery store (`vaultRecoveryStore`). This allows MFA-based re-unlock after TTL expiry:

1. User's vault expires
2. User triggers MFA vault unlock (TOTP, SMS, or WebAuthn)
3. Server verifies MFA, retrieves the recovery entry, decrypts the master key
4. New vault session is created

The recovery entry has its own TTL matching `JWT_REFRESH_EXPIRES_IN` (default: 7 days).

### Auto-Lock Preference

Users can configure a custom vault auto-lock timer:
- `null` = use server default (VAULT_TTL_MINUTES)
- `0` = never auto-lock
- `> 0` = custom minutes

Tenant admins can enforce a maximum auto-lock duration (`vaultAutoLockMaxMinutes`), capping what users can set.

<!-- manual-start -->
<!-- manual-end -->

## Authentication

### JWT Token Structure

- **Access token**: Short-lived (default: 15 minutes, configurable via `JWT_EXPIRES_IN`)
  - Payload: `{ userId, email, tenantId?, tenantRole? }`
  - Signed with `JWT_SECRET` using HS256
- **Refresh token**: Long-lived (default: 7 days, configurable via `JWT_REFRESH_EXPIRES_IN`)
  - Stored as a UUID in the `RefreshToken` database table
  - Delivered via httpOnly, Secure (production), SameSite=strict cookie named `arsenale-rt`

### Refresh Token Rotation

Refresh tokens use a **family-based rotation** scheme with reuse detection:

1. Each login creates a new token family (random UUID)
2. On refresh, the old token is revoked and a new token is issued in the same family
3. If a revoked token is reused (potential theft), the entire token family is revoked
4. A 30-second grace period allows concurrent requests during rotation
5. Token reuse triggers an `REFRESH_TOKEN_REUSE` audit log entry

### CSRF Protection

State-changing auth endpoints (`/refresh`, `/logout`, `/switch-tenant`) require an `X-CSRF-Token` header matching the CSRF token delivered alongside the access token. The CSRF token is stored in a non-httpOnly cookie (`arsenale-csrf`) so the client JavaScript can read and include it.

### Client-Side Auto-Refresh

The Axios client interceptor (`client/src/api/client.ts`):

1. Attaches `Authorization: Bearer <jwt>` to every request
2. On 401 response, attempts to refresh the access token
3. Uses a **single-flight pattern**: only the first 401 triggers a refresh; subsequent concurrent 401s wait for the same promise
4. On refresh success, retries the original request with the new token
5. On refresh failure, calls `logout()` to clear all auth state

### Socket.IO JWT Middleware

Socket.IO namespaces (`/ssh`, `/notifications`, `/gateway-monitor`) authenticate via JWT in the handshake:

```typescript
sshNamespace.use((socket, next) => {
  const token = socket.handshake.auth.token;
  // verify JWT, attach payload to socket
});
```

### Rate Limiting and Account Lockout

| Protection | Threshold | Window |
|-----------|-----------|--------|
| Login rate limit | 5 attempts per IP | 15 minutes |
| Registration rate limit | 5 per IP | 1 hour |
| Account lockout | 10 consecutive failures | 30 minutes |
| SMS MFA rate limit | Configurable | Per-endpoint |
| Password reset rate limit | Configurable | Per-endpoint |
| External share access | 10 per IP | 1 minute |

Account lockout is tracked per-user (`failedLoginAttempts`, `lockedUntil` fields). Successful login resets the counter.

<!-- manual-start -->
<!-- manual-end -->

## Connection Sharing Security

When a connection is shared with another user, credentials are **re-encrypted** for the recipient:

1. The sharer's vault must be unlocked (master key in memory)
2. Connection credentials are decrypted with the sharer's master key
3. The recipient's public vault key is used to re-encrypt the credentials
4. The re-encrypted credentials are stored in the `SharedConnection` record
5. The recipient can only decrypt with their own master key when their vault is unlocked

This means the server never stores credentials in plaintext, and a compromised recipient cannot access the sharer's vault key.

The same re-encryption model applies to **secret sharing** (`SharedSecret`).

### External Sharing

External shares (shareable links for secrets) use a different key derivation:

1. A random token is generated and given to the creator
2. A key is derived from the token using **HKDF-SHA256** with the share ID as info and an optional salt
3. The secret data is encrypted with this derived key
4. Only the token hash (SHA-256) is stored in the database
5. Optional **PIN protection**: when enabled, the key is derived from `token + PIN` using Argon2id

<!-- manual-start -->
<!-- manual-end -->

## Server-Level Encryption

Some data must be decryptable by the server without user interaction (e.g., SSH key pairs for managed gateways). This uses a separate `SERVER_ENCRYPTION_KEY`:

- 32 bytes (64 hex characters)
- Required in production, auto-generated in development
- Uses the same AES-256-GCM algorithm
- Encrypts: SSH key pairs, vault recovery entries

**Important**: In development, the server encryption key is auto-generated on each startup, meaning SSH key pairs for managed gateways will not survive restarts.

<!-- manual-start -->
<!-- manual-end -->

## Guacamole Token Encryption

RDP/VNC session tokens for guacamole-lite are encrypted with AES-256-GCM:

- Key: `GUACAMOLE_SECRET` (separate from vault keys)
- Token contains: connection parameters (host, port, credentials), display settings, recording config
- The encrypted token is passed via the WebSocket URL
- guacamole-lite decrypts the token to establish the connection
- The server monkey-patches guacamole-lite's Crypt module to properly handle GCM auth tags

<!-- manual-start -->
<!-- manual-end -->

## Security Headers

Helmet middleware applies the following security headers:

| Header | Policy |
|--------|--------|
| Content-Security-Policy | `default-src 'self'`, restricted script/style/img/connect/font, `object-src 'none'`, `frame-ancestors 'none'` |
| Strict-Transport-Security | `max-age=31536000; includeSubDomains` |
| X-Frame-Options | `DENY` |
| Referrer-Policy | `strict-origin-when-cross-origin` |

<!-- manual-start -->
<!-- manual-end -->

## Security Considerations for Production

1. **Set strong secrets**: `JWT_SECRET`, `GUACAMOLE_SECRET`, `SERVER_ENCRYPTION_KEY` must be cryptographically random. Generate with `openssl rand -hex 32`.
2. **Enable HTTPS**: Use a reverse proxy (Caddy, Traefik, etc.) with TLS termination in front of the Nginx container.
3. **Configure `TRUST_PROXY`**: Set to the number of proxy hops for correct client IP resolution.
4. **Set `CLIENT_URL`**: Must match the actual production URL for CORS and OAuth redirects.
5. **Use strong database password**: Change default PostgreSQL credentials.
6. **Enable MFA policy**: Set `mfaRequired: true` on the tenant to enforce MFA for all members.
7. **Configure vault timeout**: Set `vaultAutoLockMaxMinutes` at the tenant level to cap vault session duration.
8. **Configure session timeout**: Set `defaultSessionTimeoutSeconds` to auto-close idle remote sessions.
9. **Review OAuth/SAML configuration**: Ensure callback URLs match the production domain.
10. **Enable audit logging**: Monitor the audit log for suspicious activity (login failures, token reuse).
11. **Configure GeoIP**: Set `GEOIP_DB_PATH` with a MaxMind GeoLite2 database for IP geolocation in audit logs.

<!-- manual-start -->
<!-- manual-end -->
