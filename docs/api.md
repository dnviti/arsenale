# API Reference

> Auto-generated on 2026-03-14 by `/docs create api`.
> Source of truth is the codebase. Run `/docs update api` after code changes.

## Overview

All REST endpoints are mounted under `/api`. The server runs on port 3001 (configurable via `PORT`).

| Route Group | Base Path | Auth Required | Description |
|-------------|-----------|---------------|-------------|
| Health | `/api/health`, `/api/ready` | No | Health and readiness probes |
| Auth | `/api/auth` | Mixed | Registration, login, MFA, token refresh |
| OAuth | `/api/auth/oauth` | Mixed | OAuth provider flows (Google, Microsoft, GitHub, OIDC) |
| SAML | `/api/auth/saml` | Mixed | SAML 2.0 SSO |
| Vault | `/api/vault` | Yes | Vault lock/unlock, MFA unlock, auto-lock |
| Connections | `/api/connections` | Yes | Connection CRUD, favorites |
| Folders | `/api/folders` | Yes | Folder CRUD |
| Sharing | `/api/connections` | Yes | Connection sharing management |
| Import/Export | `/api/connections` | Yes | Connection import/export |
| Sessions | `/api/sessions` | Yes | RDP/VNC/SSH session lifecycle, admin monitoring |
| User | `/api/user` | Yes | Profile, settings, identity verification |
| 2FA (TOTP) | `/api/user/2fa` | Yes | TOTP setup/verify/disable |
| 2FA (SMS) | `/api/user/2fa/sms` | Yes | SMS MFA setup/verify/disable |
| 2FA (WebAuthn) | `/api/user/2fa/webauthn` | Yes | Passkey registration/management |
| Files | `/api/files` | Yes | RDP drive file management |
| Audit | `/api/audit` | Yes | Audit log queries |
| Notifications | `/api/notifications` | Yes | In-app notification management |
| Tenants | `/api/tenants` | Yes | Tenant CRUD, user management |
| Teams | `/api/teams` | Yes | Team CRUD, member management |
| Admin | `/api/admin` | Yes (Admin) | Email config, app settings |
| Gateways | `/api/gateways` | Yes (Tenant) | Gateway CRUD, SSH keys, orchestration |
| Tabs | `/api/tabs` | Yes | Tab state persistence |
| Secrets | `/api/secrets` | Yes | Vault secrets CRUD, versioning, sharing |
| Public Share | `/api/share` | No | External secret access (public) |
| Recordings | `/api/recordings` | Yes | Session recording management |
| GeoIP | `/api/geoip` | Yes | IP geolocation lookup |
| LDAP | `/api/ldap` | Yes (Admin) | LDAP integration status, test, and manual sync |
| Sync | `/api/sync` | Yes (Admin) | External sync profiles (NetBox), CRUD and manual trigger |

<!-- manual-start -->
<!-- manual-end -->

## Authentication

Most endpoints require a JWT Bearer token in the `Authorization` header:

```
Authorization: Bearer <access_token>
```

Public endpoints (no auth required): `/api/health`, `/api/ready`, `/api/auth/config`, `/api/auth/register`, `/api/auth/login`, `/api/auth/verify-email`, `/api/auth/resend-verification`, `/api/auth/forgot-password`, `/api/auth/reset-password/*`, `/api/auth/refresh`, `/api/auth/logout`, `/api/auth/verify-totp`, `/api/auth/verify-sms`, `/api/auth/request-sms-code`, `/api/auth/verify-webauthn`, `/api/auth/request-webauthn-options`, `/api/auth/mfa-setup/*`, `/api/auth/oauth/*`, `/api/auth/saml/*`, `/api/share/:token/*`.

CSRF-protected endpoints (require `X-CSRF-Token` header): `/api/auth/refresh`, `/api/auth/logout`, `/api/auth/switch-tenant`.

Tenant-scoped endpoints require the user to have an active tenant membership (set via JWT claims after login or tenant switch). Admin-only endpoints additionally require `ADMIN` or `OWNER` tenant role.

<!-- manual-start -->
<!-- manual-end -->

## Health & Readiness

### `GET /api/health`

Health check. Always returns 200.

**Auth**: No | **Response**: `{ "status": "ok" }`

### `GET /api/ready`

Readiness probe. Checks database and guacd connectivity.

**Auth**: No | **Response**: `{ "status": "ready"|"not_ready", "checks": { "database": {...}, "guacd": {...} } }`

<!-- manual-start -->
<!-- manual-end -->

## Auth

### `GET /api/auth/config`

Returns public authentication configuration (enabled OAuth providers, self-signup status, email verification requirement).

**Auth**: No

### `POST /api/auth/register`

Register a new user account. Rate limited: 5 per hour per IP.

**Auth**: No | **Body**: `{ email, password }` | **Response**: `{ message, recoveryKey?, requiresVerification? }`

### `GET /api/auth/verify-email?token=<token>`

Verify email address using the token sent by email.

**Auth**: No

### `POST /api/auth/resend-verification`

Resend email verification link.

**Auth**: No | **Body**: `{ email }`

### `POST /api/auth/login`

Login with email/password. Rate limited per IP. Returns tokens or MFA challenge.

**Auth**: No | **Body**: `{ email, password }` | **Response**: `{ accessToken, user, csrfToken }` or `{ requiresMfa, mfaMethods[], pendingToken }`

### `POST /api/auth/verify-totp`

Verify TOTP code during MFA challenge.

**Auth**: No | **Body**: `{ pendingToken, code }` | **Response**: `{ accessToken, user, csrfToken }`

### `POST /api/auth/request-sms-code`

Request SMS OTP during MFA challenge. Rate limited.

**Auth**: No | **Body**: `{ pendingToken }`

### `POST /api/auth/verify-sms`

Verify SMS OTP during MFA challenge.

**Auth**: No | **Body**: `{ pendingToken, code }` | **Response**: `{ accessToken, user, csrfToken }`

### `POST /api/auth/request-webauthn-options`

Get WebAuthn authentication options during MFA challenge.

**Auth**: No | **Body**: `{ pendingToken }` | **Response**: `{ options }`

### `POST /api/auth/verify-webauthn`

Verify WebAuthn assertion during MFA challenge.

**Auth**: No | **Body**: `{ pendingToken, credential }` | **Response**: `{ accessToken, user, csrfToken }`

### `POST /api/auth/mfa-setup/init`

Initialize mandatory MFA setup during first login.

**Auth**: No | **Body**: `{ pendingToken, method }` | **Response**: Method-specific setup data

### `POST /api/auth/mfa-setup/verify`

Complete mandatory MFA setup verification.

**Auth**: No | **Body**: `{ pendingToken, method, code|credential }` | **Response**: `{ accessToken, user, csrfToken }`

### `POST /api/auth/forgot-password`

Request password reset email. Rate limited.

**Auth**: No | **Body**: `{ email }`

### `POST /api/auth/reset-password/validate`

Validate a password reset token.

**Auth**: No | **Body**: `{ token }` | **Response**: `{ valid, requiresSms? }`

### `POST /api/auth/reset-password/request-sms`

Request SMS verification during password reset.

**Auth**: No | **Body**: `{ token }`

### `POST /api/auth/reset-password/complete`

Complete password reset with new password.

**Auth**: No | **Body**: `{ token, newPassword, smsCode?, recoveryKey? }` | **Response**: `{ message, recoveryKey? }`

### `POST /api/auth/refresh`

Refresh access token using httpOnly cookie. CSRF-protected.

**Auth**: Cookie | **Response**: `{ accessToken, csrfToken, user }`

### `POST /api/auth/logout`

Logout and revoke refresh token. CSRF-protected.

**Auth**: Cookie

### `POST /api/auth/switch-tenant`

Switch active tenant context. CSRF-protected.

**Auth**: Yes | **Body**: `{ tenantId }` | **Response**: `{ accessToken, csrfToken, user }`

<!-- manual-start -->
<!-- manual-end -->

## OAuth

### `GET /api/auth/oauth/providers`

List available OAuth providers.

**Auth**: No | **Response**: `{ providers: [{ provider, name, enabled }] }`

### `GET /api/auth/oauth/:provider`

Initiate OAuth flow (redirect to provider). Providers: `google`, `microsoft`, `github`, `oidc`.

**Auth**: No

### `GET /api/auth/oauth/:provider/callback`

OAuth callback handler. Redirects to client with tokens.

**Auth**: No

### `GET /api/auth/oauth/link/:provider`

Initiate OAuth account linking (uses JWT from query param).

**Auth**: JWT in query | **Query**: `?token=<jwt>`

### `GET /api/auth/oauth/accounts`

List linked OAuth accounts.

**Auth**: Yes | **Response**: `[{ provider, providerEmail, createdAt }]`

### `DELETE /api/auth/oauth/link/:provider`

Unlink an OAuth account.

**Auth**: Yes

### `POST /api/auth/oauth/vault-setup`

Set vault password for OAuth-only users.

**Auth**: Yes | **Body**: `{ password }`

<!-- manual-start -->
<!-- manual-end -->

## SAML

### `GET /api/auth/saml/metadata`

SAML Service Provider metadata XML.

**Auth**: No

### `GET /api/auth/saml`

Initiate SAML login (redirect to IdP).

**Auth**: No

### `GET /api/auth/saml/link`

Initiate SAML account linking (JWT from query param).

**Auth**: JWT in query

### `POST /api/auth/saml/callback`

SAML ACS callback (POST with URL-encoded body from IdP).

**Auth**: No

<!-- manual-start -->
<!-- manual-end -->

## Vault

All endpoints require authentication.

### `POST /api/vault/unlock`

Unlock vault with password.

**Body**: `{ password }` | **Response**: `{ unlocked: true }`

### `POST /api/vault/lock`

Lock vault (soft lock — preserves MFA recovery).

### `GET /api/vault/status`

Get vault lock status and available MFA unlock methods.

**Response**: `{ unlocked, mfaUnlockAvailable, mfaUnlockMethods[] }`

### `POST /api/vault/reveal-password`

Reveal a connection's decrypted password.

**Body**: `{ connectionId }` | **Response**: `{ password }`

### `POST /api/vault/unlock-mfa/totp`

Unlock vault using TOTP code (requires prior password unlock in session).

**Body**: `{ code }`

### `POST /api/vault/unlock-mfa/webauthn-options`

Get WebAuthn options for vault MFA unlock.

### `POST /api/vault/unlock-mfa/webauthn`

Unlock vault with WebAuthn credential.

**Body**: `{ credential }`

### `POST /api/vault/unlock-mfa/request-sms`

Request SMS code for vault MFA unlock.

### `POST /api/vault/unlock-mfa/sms`

Unlock vault with SMS code.

**Body**: `{ code }`

### `GET /api/vault/auto-lock`

Get vault auto-lock preference.

**Response**: `{ autoLockMinutes, tenantMaxMinutes? }`

### `PUT /api/vault/auto-lock`

Set vault auto-lock preference.

**Body**: `{ minutes }` (0 = never, null = server default)

<!-- manual-start -->
<!-- manual-end -->

## Connections

All endpoints require authentication.

### `GET /api/connections`

List all connections (own + shared + team).

**Response**: `{ own: [...], shared: [...], team: [...] }`

### `POST /api/connections`

Create a new connection.

**Body**: `{ name, type, host, port, username?, password?, domain?, folderId?, teamId?, description?, sshTerminalConfig?, rdpSettings?, vncSettings?, gatewayId?, enableDrive?, defaultCredentialMode?, credentialSecretId? }`

### `GET /api/connections/:id`

Get a single connection.

### `PUT /api/connections/:id`

Update a connection.

### `DELETE /api/connections/:id`

Delete a connection.

### `PATCH /api/connections/:id/favorite`

Toggle favorite status.

<!-- manual-start -->
<!-- manual-end -->

## Connection Sharing

All endpoints require authentication.

### `POST /api/connections/:id/share`

Share a connection with a user.

**Body**: `{ userId, permission }`

### `POST /api/connections/batch-share`

Share multiple connections at once.

**Body**: `{ connectionIds, userId, permission }`

### `DELETE /api/connections/:id/share/:userId`

Revoke sharing from a user.

### `PUT /api/connections/:id/share/:userId`

Update share permission.

**Body**: `{ permission }`

### `GET /api/connections/:id/shares`

List all shares for a connection.

<!-- manual-start -->
<!-- manual-end -->

## Import/Export

All endpoints require authentication. Mounted under `/api/connections`.

### `POST /api/connections/export`

Export connections to CSV or JSON.

**Body**: `{ connectionIds, format, includeCredentials? }`

### `POST /api/connections/import`

Import connections from CSV, JSON, mRemoteNG, or RDP file format.

**Body**: `{ data, format?, folderId? }`

<!-- manual-start -->
<!-- manual-end -->

## Folders

All endpoints require authentication.

### `GET /api/folders`

List all folders (tree structure).

### `POST /api/folders`

Create a folder.

**Body**: `{ name, parentId?, teamId? }`

### `PUT /api/folders/:id`

Update a folder.

**Body**: `{ name?, parentId?, sortOrder? }`

### `DELETE /api/folders/:id`

Delete a folder (connections moved to root).

<!-- manual-start -->
<!-- manual-end -->

## Sessions

All endpoints require authentication.

### `POST /api/sessions/rdp`

Create an RDP session. Returns encrypted Guacamole token.

**Body**: `{ connectionId, credentialMode?, username?, password? }` | **Response**: `{ token, wsUrl, sessionId }`

### `POST /api/sessions/vnc`

Create a VNC session. Same pattern as RDP.

**Body**: `{ connectionId, credentialMode?, username?, password? }` | **Response**: `{ token, wsUrl, sessionId }`

### `POST /api/sessions/ssh`

Validate SSH access (does not create a session — SSH sessions are created via Socket.IO).

**Body**: `{ connectionId }`

### `POST /api/sessions/rdp/:sessionId/heartbeat`

Send heartbeat for an RDP/VNC session.

### `POST /api/sessions/rdp/:sessionId/end`

End an RDP/VNC session.

### `POST /api/sessions/vnc/:sessionId/heartbeat`

Send heartbeat for a VNC session.

### `POST /api/sessions/vnc/:sessionId/end`

End a VNC session.

### `GET /api/sessions/active`

List active sessions (admin, tenant-scoped).

**Auth**: Admin | **Response**: `[{ id, userId, connectionId, protocol, status, ... }]`

### `GET /api/sessions/count`

Get active session count (admin, tenant-scoped).

**Auth**: Admin

### `GET /api/sessions/count/gateway`

Get session count grouped by gateway (admin, tenant-scoped).

**Auth**: Admin

### `POST /api/sessions/:sessionId/terminate`

Terminate an active session (admin).

**Auth**: Admin

<!-- manual-start -->
<!-- manual-end -->

## User

All endpoints require authentication.

### `GET /api/user/profile`

Get current user's profile.

### `PUT /api/user/profile`

Update profile (username, avatar).

### `PUT /api/user/password`

Change password.

**Body**: `{ currentPassword, newPassword }`

### `PUT /api/user/ssh-defaults`

Update default SSH terminal settings.

**Body**: `{ theme?, fontFamily?, fontSize?, cursorStyle? }`

### `PUT /api/user/rdp-defaults`

Update default RDP settings.

**Body**: Partial RDP settings object.

### `POST /api/user/avatar`

Upload avatar image.

**Body**: Base64 image data.

### `GET /api/user/search`

Search users by email/username (tenant-scoped).

**Auth**: Tenant member | **Query**: `?q=<search>`

### `GET /api/user/domain-profile`

Get Windows/AD domain profile.

### `PUT /api/user/domain-profile`

Update domain profile.

**Body**: `{ domainName, domainUsername, password? }`

### `DELETE /api/user/domain-profile`

Clear domain profile.

### `POST /api/user/email-change/initiate`

Initiate email change (sends OTP to old and new address). Rate limited.

**Body**: `{ newEmail, password }`

### `POST /api/user/email-change/confirm`

Confirm email change with both OTP codes.

**Body**: `{ oldCode, newCode }`

### `POST /api/user/password-change/initiate`

Initiate password change with identity verification. Rate limited.

**Body**: `{ currentPassword, newPassword }`

### `POST /api/user/identity/initiate`

Initiate identity verification for sensitive operations. Rate limited.

**Body**: `{ password }`

### `POST /api/user/identity/confirm`

Confirm identity verification.

**Body**: `{ code }`

<!-- manual-start -->
<!-- manual-end -->

## Two-Factor Authentication

### TOTP (`/api/user/2fa`)

All endpoints require authentication.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/user/2fa/setup` | Generate TOTP secret and QR code |
| `POST` | `/api/user/2fa/verify` | Verify TOTP code and enable 2FA |
| `POST` | `/api/user/2fa/disable` | Disable TOTP 2FA |
| `GET` | `/api/user/2fa/status` | Get TOTP enabled status |

### SMS MFA (`/api/user/2fa/sms`)

All endpoints require authentication.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/user/2fa/sms/setup-phone` | Set phone number and send verification code. Rate limited. |
| `POST` | `/api/user/2fa/sms/verify-phone` | Verify phone number with code |
| `POST` | `/api/user/2fa/sms/enable` | Enable SMS MFA |
| `POST` | `/api/user/2fa/sms/send-disable-code` | Send disable confirmation code. Rate limited. |
| `POST` | `/api/user/2fa/sms/disable` | Disable SMS MFA with code |
| `GET` | `/api/user/2fa/sms/status` | Get SMS MFA status |

### WebAuthn / Passkeys (`/api/user/2fa/webauthn`)

All endpoints require authentication.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/user/2fa/webauthn/registration-options` | Get registration options for a new credential |
| `POST` | `/api/user/2fa/webauthn/register` | Register a new WebAuthn credential |
| `GET` | `/api/user/2fa/webauthn/credentials` | List registered credentials |
| `DELETE` | `/api/user/2fa/webauthn/credentials/:id` | Remove a credential |
| `PATCH` | `/api/user/2fa/webauthn/credentials/:id` | Rename a credential |
| `GET` | `/api/user/2fa/webauthn/status` | Get WebAuthn enabled status |

<!-- manual-start -->
<!-- manual-end -->

## Files

All endpoints require authentication. Used for RDP drive redirection file management.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/files` | List files in user's drive |
| `GET` | `/api/files/:name` | Download a file |
| `POST` | `/api/files` | Upload a file (multipart, quota checked) |
| `DELETE` | `/api/files/:name` | Delete a file |

<!-- manual-start -->
<!-- manual-end -->

## Audit

All endpoints require authentication.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/audit` | User | List personal audit logs (paginated, filterable) |
| `GET` | `/api/audit/countries` | User | List distinct countries in user's logs |
| `GET` | `/api/audit/gateways` | User | List distinct gateways in user's logs |
| `GET` | `/api/audit/tenant` | Admin | List tenant-wide audit logs |
| `GET` | `/api/audit/tenant/countries` | Admin | List distinct countries in tenant logs |
| `GET` | `/api/audit/tenant/gateways` | Admin | List distinct gateways in tenant logs |
| `GET` | `/api/audit/tenant/geo-summary` | Admin | Geographic summary with coordinates |
| `GET` | `/api/audit/connection/:connectionId` | User | Connection-scoped audit logs |
| `GET` | `/api/audit/connection/:connectionId/users` | User | Distinct users in connection logs |

<!-- manual-start -->
<!-- manual-end -->

## Notifications

All endpoints require authentication.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/notifications` | List notifications (paginated) |
| `PUT` | `/api/notifications/read-all` | Mark all as read |
| `PUT` | `/api/notifications/:id/read` | Mark one as read |
| `DELETE` | `/api/notifications/:id` | Delete a notification |

<!-- manual-start -->
<!-- manual-end -->

## Tenants

All endpoints require authentication.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/tenants` | User | Create a new tenant |
| `GET` | `/api/tenants/mine/all` | User | List all tenant memberships |
| `GET` | `/api/tenants/mine` | Tenant | Get current tenant details |
| `PUT` | `/api/tenants/:id` | Admin | Update tenant (name, MFA policy, session timeout) |
| `DELETE` | `/api/tenants/:id` | Owner | Delete tenant |
| `GET` | `/api/tenants/:id/mfa-stats` | Admin | Get MFA compliance stats |
| `GET` | `/api/tenants/:id/users` | Tenant | List tenant users |
| `GET` | `/api/tenants/:id/users/:userId/profile` | Tenant | Get user profile details |
| `POST` | `/api/tenants/:id/invite` | Admin | Invite user by email |
| `POST` | `/api/tenants/:id/users` | Admin | Create a new user in tenant |
| `PUT` | `/api/tenants/:id/users/:userId` | Admin | Update user role |
| `DELETE` | `/api/tenants/:id/users/:userId` | Admin | Remove user from tenant |
| `PATCH` | `/api/tenants/:id/users/:userId/enabled` | Admin | Enable/disable user account |
| `PUT` | `/api/tenants/:id/users/:userId/email` | Admin | Admin change user email |
| `PUT` | `/api/tenants/:id/users/:userId/password` | Admin | Admin change user password |

<!-- manual-start -->
<!-- manual-end -->

## Teams

All endpoints require authentication and tenant membership.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/teams` | Tenant | Create a team |
| `GET` | `/api/teams` | Tenant | List teams |
| `GET` | `/api/teams/:id` | Team Member | Get team details |
| `PUT` | `/api/teams/:id` | Team Admin | Update team |
| `DELETE` | `/api/teams/:id` | Team Admin | Delete team |
| `GET` | `/api/teams/:id/members` | Team Member | List members |
| `POST` | `/api/teams/:id/members` | Team Admin | Add member |
| `PUT` | `/api/teams/:id/members/:userId` | Team Admin | Update member role |
| `DELETE` | `/api/teams/:id/members/:userId` | Team Admin | Remove member |

<!-- manual-start -->
<!-- manual-end -->

## Admin

All endpoints require authentication with Admin tenant role.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/admin/email/status` | Get email provider configuration status |
| `POST` | `/api/admin/email/test` | Send test email |
| `GET` | `/api/admin/app-config` | Get app configuration (self-signup, etc.) |
| `PUT` | `/api/admin/app-config/self-signup` | Toggle self-signup |

<!-- manual-start -->
<!-- manual-end -->

## Gateways

All endpoints require authentication and tenant membership. Most require Admin role.

### Gateway CRUD

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/gateways` | Tenant | List gateways |
| `POST` | `/api/gateways` | Admin | Create gateway |
| `PUT` | `/api/gateways/:id` | Admin | Update gateway |
| `DELETE` | `/api/gateways/:id` | Admin | Delete gateway |
| `POST` | `/api/gateways/:id/test` | Tenant | Test gateway connectivity |

### SSH Key Pair Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/gateways/ssh-keypair` | Generate SSH key pair |
| `GET` | `/api/gateways/ssh-keypair` | Get public key |
| `GET` | `/api/gateways/ssh-keypair/private` | Download private key |
| `POST` | `/api/gateways/ssh-keypair/rotate` | Rotate key pair |
| `PATCH` | `/api/gateways/ssh-keypair/rotation` | Update rotation policy |
| `GET` | `/api/gateways/ssh-keypair/rotation` | Get rotation status |
| `POST` | `/api/gateways/:id/push-key` | Push public key to gateway |

### Gateway Templates

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/gateways/templates` | List templates |
| `POST` | `/api/gateways/templates` | Create template |
| `PUT` | `/api/gateways/templates/:templateId` | Update template |
| `DELETE` | `/api/gateways/templates/:templateId` | Delete template |
| `POST` | `/api/gateways/templates/:templateId/deploy` | Deploy gateway from template |

### Managed Gateway Lifecycle

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/gateways/:id/deploy` | Deploy managed gateway containers |
| `DELETE` | `/api/gateways/:id/deploy` | Undeploy managed gateway |
| `POST` | `/api/gateways/:id/scale` | Scale gateway replicas |
| `GET` | `/api/gateways/:id/instances` | List container instances |
| `POST` | `/api/gateways/:id/instances/:instanceId/restart` | Restart an instance |
| `GET` | `/api/gateways/:id/instances/:instanceId/logs` | Get instance logs |

### Auto-Scaling

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/gateways/:id/scaling` | Get scaling status |
| `PUT` | `/api/gateways/:id/scaling` | Update scaling config |

<!-- manual-start -->
<!-- manual-end -->

## Tabs

All endpoints require authentication.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/tabs` | Get persisted tabs |
| `PUT` | `/api/tabs` | Sync tab state to server |
| `DELETE` | `/api/tabs` | Clear all persisted tabs |

<!-- manual-start -->
<!-- manual-end -->

## Secrets (Keychain)

All endpoints require authentication.

### CRUD

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/secrets` | List secrets (filterable by scope, type, tags) |
| `POST` | `/api/secrets` | Create secret |
| `GET` | `/api/secrets/:id` | Get secret details |
| `PUT` | `/api/secrets/:id` | Update secret |
| `DELETE` | `/api/secrets/:id` | Delete secret |

### Versioning

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/secrets/:id/versions` | List versions |
| `GET` | `/api/secrets/:id/versions/:version/data` | Get version data |
| `POST` | `/api/secrets/:id/versions/:version/restore` | Restore a version |

### Sharing

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/secrets/:id/share` | Share secret with a user |
| `DELETE` | `/api/secrets/:id/share/:userId` | Revoke sharing |
| `PUT` | `/api/secrets/:id/share/:userId` | Update share permission |
| `GET` | `/api/secrets/:id/shares` | List shares |

### External Sharing

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/secrets/:id/external-shares` | Create external share link |
| `GET` | `/api/secrets/:id/external-shares` | List external shares |
| `DELETE` | `/api/secrets/external-shares/:shareId` | Revoke external share |

### Tenant Vault

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/secrets/tenant-vault/init` | Initialize tenant vault |
| `POST` | `/api/secrets/tenant-vault/distribute` | Distribute tenant vault key to members |
| `GET` | `/api/secrets/tenant-vault/status` | Get tenant vault status |

<!-- manual-start -->
<!-- manual-end -->

## Public Share

Public endpoints for accessing externally shared secrets. No authentication required.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/share/:token/info` | Get share info (name, type, expiry) |
| `POST` | `/api/share/:token` | Access shared secret (with optional PIN). Rate limited: 10/min. |

<!-- manual-start -->
<!-- manual-end -->

## Recordings

All endpoints require authentication.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/recordings` | List session recordings |
| `GET` | `/api/recordings/:id` | Get recording metadata |
| `GET` | `/api/recordings/:id/stream` | Stream recording file |
| `GET` | `/api/recordings/:id/analyze` | Analyze .guac recording (command extraction) |
| `GET` | `/api/recordings/:id/video` | Export recording as video (via guacenc sidecar) |
| `DELETE` | `/api/recordings/:id` | Delete a recording |

<!-- manual-start -->
<!-- manual-end -->

## GeoIP

All endpoints require authentication.

### `GET /api/geoip/:ip`

Lookup IP geolocation. Uses MaxMind GeoLite2 database if configured, falls back to ip-api.com with caching.

**Response**: `{ country, city, lat, lon, ... }`

<!-- manual-start -->
<!-- manual-end -->

## WebSocket Endpoints

### Socket.IO — SSH Terminal (`/ssh`)

Connected via Socket.IO at `/ssh` namespace. Authentication via `auth.token` in handshake.

**Client -> Server Events:**

| Event | Data | Description |
|-------|------|-------------|
| `session:start` | `{ connectionId, username?, password?, credentialMode? }` | Start SSH session |
| `data` | `string` | Terminal input (keystrokes) |
| `resize` | `{ cols, rows }` | Terminal resize |
| `session:heartbeat` | — | Explicit heartbeat |
| `sftp:list` | `{ path }` | List directory contents |
| `sftp:mkdir` | `{ path }` | Create directory |
| `sftp:delete` | `{ path }` | Delete file |
| `sftp:rmdir` | `{ path }` | Remove directory |
| `sftp:rename` | `{ oldPath, newPath }` | Rename file/directory |
| `sftp:upload:start` | `{ remotePath, fileSize, filename }` | Begin file upload |
| `sftp:upload:chunk` | `{ transferId, chunk }` | Upload data chunk |
| `sftp:upload:end` | `{ transferId }` | Complete upload |
| `sftp:download:start` | `{ remotePath, filename }` | Begin file download |
| `sftp:download:cancel` | `{ transferId }` | Cancel download |

**Server -> Client Events:**

| Event | Data | Description |
|-------|------|-------------|
| `session:ready` | — | SSH connection established |
| `session:error` | `{ message }` | Connection error |
| `session:closed` | — | Session ended |
| `data` | `string` | Terminal output |
| `sftp:progress` | `{ transferId, bytesTransferred, totalBytes, filename }` | Transfer progress |
| `sftp:transfer:complete` | `{ transferId }` | Transfer complete |
| `sftp:transfer:error` | `{ transferId, message }` | Transfer error |
| `sftp:download:chunk` | `{ transferId, chunk }` | Download data chunk |
| `sftp:download:complete` | `{ transferId }` | Download complete |

### Socket.IO — Notifications (`/notifications`)

Real-time notification delivery. Authentication via `auth.token` in handshake.

**Server -> Client Events:**

| Event | Data | Description |
|-------|------|-------------|
| `notification` | `{ id, type, message, relatedId }` | New notification |

### Socket.IO — Gateway Monitor (`/gateway-monitor`)

Real-time gateway health and instance updates. Authentication via `auth.token` in handshake.

**Server -> Client Events:**

| Event | Data | Description |
|-------|------|-------------|
| `health:update` | `{ gatewayId, status, latencyMs, ... }` | Gateway health change |
| `instances:update` | `{ gatewayId, instances[] }` | Instance status change |
| `scaling:update` | `{ gatewayId, scalingStatus }` | Scaling event |
| `gateway:update` | `{ gateway }` | Gateway config change |

### Guacamole WebSocket (port 3002)

Direct WebSocket connection at `/guacamole` (proxied via Nginx). Used for RDP and VNC sessions. Communicates using the Guacamole protocol with encrypted connection tokens.

<!-- manual-start -->
<!-- manual-end -->
