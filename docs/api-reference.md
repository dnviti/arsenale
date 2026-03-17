---
title: API Reference
description: Complete REST API endpoint reference and WebSocket namespace documentation
generated-by: ctdf-docs
generated-at: 2026-03-16T19:30:00Z
source-files:
  - server/src/routes/auth.routes.ts
  - server/src/routes/oauth.routes.ts
  - server/src/routes/saml.routes.ts
  - server/src/routes/vault.routes.ts
  - server/src/routes/connections.routes.ts
  - server/src/routes/folders.routes.ts
  - server/src/routes/sessions.routes.ts
  - server/src/routes/user.routes.ts
  - server/src/routes/twofa.routes.ts
  - server/src/routes/smsMfa.routes.ts
  - server/src/routes/webauthn.routes.ts
  - server/src/routes/secrets.routes.ts
  - server/src/routes/vault-folders.routes.ts
  - server/src/routes/sharing.routes.ts
  - server/src/routes/externalShare.routes.ts
  - server/src/routes/audit.routes.ts
  - server/src/routes/notifications.routes.ts
  - server/src/routes/tenants.routes.ts
  - server/src/routes/teams.routes.ts
  - server/src/routes/gateways.routes.ts
  - server/src/routes/admin.routes.ts
  - server/src/routes/files.routes.ts
  - server/src/routes/tabs.routes.ts
  - server/src/routes/recordings.routes.ts
  - server/src/routes/geoip.routes.ts
  - server/src/routes/ldap.routes.ts
  - server/src/routes/sync.routes.ts
  - server/src/routes/externalVault.routes.ts
  - server/src/routes/accessPolicy.routes.ts
  - server/src/routes/importExport.routes.ts
  - server/src/routes/health.routes.ts
  - server/src/socket/index.ts
  - server/src/socket/ssh.handler.ts
---

# API Reference

All endpoints are prefixed with `/api`. Authentication is via JWT Bearer token unless noted otherwise. State-changing requests require a `X-CSRF-Token` header.

## Authentication

### Local Auth — `/api/auth`

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/config` | No | Public auth configuration (enabled providers, MFA, signup) |
| POST | `/register` | No | Register new user (email, username, password) |
| GET | `/verify-email` | No | Email verification callback (token in query) |
| POST | `/resend-verification` | No | Resend verification email |
| POST | `/login` | No | Login with email + password |
| POST | `/verify-totp` | No | TOTP verification (after login) |
| POST | `/request-sms-code` | No | Request SMS code (after login) |
| POST | `/verify-sms` | No | SMS code verification |
| POST | `/request-webauthn-options` | No | WebAuthn challenge generation |
| POST | `/verify-webauthn` | No | WebAuthn assertion verification |
| POST | `/mfa-setup/init` | No | MFA enrollment initiation (first-time setup) |
| POST | `/mfa-setup/verify` | No | MFA enrollment verification |
| POST | `/forgot-password` | No | Request password reset email |
| POST | `/reset-password/validate` | No | Validate reset token |
| POST | `/reset-password/request-sms` | No | SMS verification for password reset |
| POST | `/reset-password/complete` | No | Complete password reset |
| POST | `/refresh` | No | Refresh access token (uses refresh token cookie) |
| POST | `/logout` | Yes | Revoke tokens |
| POST | `/switch-tenant` | Yes | Switch tenant context |

### OAuth — `/api/auth/oauth`

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/providers` | No | List available OAuth providers |
| POST | `/link-code` | Yes | Generate account linking code |
| GET | `/link/:provider` | Yes | Initiate account linking |
| POST | `/exchange-code` | No | Exchange authorization code for tokens |
| GET | `/accounts` | Yes | List linked OAuth accounts |
| DELETE | `/link/:provider` | Yes | Unlink OAuth account |
| POST | `/vault-setup` | Yes | Setup vault after first OAuth login |
| GET | `/:provider` | No | Initiate OAuth flow |
| GET | `/:provider/callback` | No | OAuth callback handler |

### SAML — `/api/auth/saml`

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/metadata` | No | SP metadata XML |
| GET | `/` | No | Initiate SAML login |
| GET | `/link` | Yes | Initiate SAML account linking |
| POST | `/callback` | No | SAML ACS callback |

## Vault — `/api/vault`

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/unlock` | Unlock vault with password |
| POST | `/lock` | Lock vault |
| GET | `/status` | Get vault lock status and MFA options |
| POST | `/reveal-password` | Reveal decrypted connection password |
| POST | `/unlock-mfa/totp` | Unlock vault with TOTP |
| POST | `/unlock-mfa/webauthn-options` | Get WebAuthn challenge for vault unlock |
| POST | `/unlock-mfa/webauthn` | Unlock vault with WebAuthn assertion |
| POST | `/unlock-mfa/request-sms` | Request SMS code for vault unlock |
| POST | `/unlock-mfa/sms` | Unlock vault with SMS code |
| GET | `/auto-lock` | Get auto-lock preference (minutes) |
| PUT | `/auto-lock` | Set auto-lock preference |

## Connections — `/api/connections`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | List all connections (own, shared, team) |
| POST | `/` | Create connection (SSH/RDP/VNC) |
| GET | `/:id` | Get connection details |
| PUT | `/:id` | Update connection |
| DELETE | `/:id` | Delete connection |
| PATCH | `/:id/favorite` | Toggle favorite status |
| POST | `/:id/share` | Share connection with user |
| DELETE | `/:id/share/:userId` | Remove share |
| PUT | `/:id/share/:userId` | Update share permission |
| GET | `/:id/shares` | List shares for connection |
| POST | `/batch-share` | Share multiple connections at once |
| POST | `/export` | Export connections as JSON |
| POST | `/import` | Import connections from JSON |

## Folders — `/api/folders`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | List all folders (hierarchical) |
| POST | `/` | Create folder |
| PUT | `/:id` | Update folder (name, parent) |
| DELETE | `/:id` | Delete folder |

## Sessions — `/api/sessions`

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/rdp` | Create RDP session (returns Guacamole token) |
| POST | `/rdp/:sessionId/heartbeat` | Keep RDP session alive |
| POST | `/rdp/:sessionId/end` | End RDP session |
| POST | `/vnc` | Create VNC session |
| POST | `/vnc/:sessionId/heartbeat` | Keep VNC session alive |
| POST | `/vnc/:sessionId/end` | End VNC session |
| POST | `/ssh` | Validate SSH access (for Socket.IO connection) |
| GET | `/active` | List active sessions (admin) |
| GET | `/count` | Session count (admin) |
| GET | `/count/gateway` | Session count by gateway (admin) |
| POST | `/:sessionId/terminate` | Force terminate session (admin) |

## User Management — `/api/user`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/search` | Search users (tenant scoped) |
| GET | `/profile` | Get user profile |
| PUT | `/profile` | Update profile |
| PUT | `/password` | Change password |
| PUT | `/ssh-defaults` | Set SSH terminal defaults |
| PUT | `/rdp-defaults` | Set RDP defaults |
| POST | `/avatar` | Upload avatar |
| GET | `/domain-profile` | Get domain profile |
| PUT | `/domain-profile` | Update domain profile |
| DELETE | `/domain-profile` | Clear domain profile |
| POST | `/email-change/initiate` | Start email change flow |
| POST | `/email-change/confirm` | Confirm email change |
| POST | `/password-change/initiate` | Start password change (with verification) |
| POST | `/identity/initiate` | Request identity verification code |
| POST | `/identity/confirm` | Confirm identity verification |

### TOTP — `/api/user/2fa`

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/setup` | Generate TOTP secret + QR code |
| POST | `/verify` | Enable TOTP with verification code |
| POST | `/disable` | Disable TOTP |
| GET | `/status` | Get TOTP enrollment status |

### SMS MFA — `/api/user/2fa/sms`

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/setup-phone` | Register phone number |
| POST | `/verify-phone` | Verify phone with OTP |
| POST | `/enable` | Enable SMS MFA |
| POST | `/send-disable-code` | Request disable code |
| POST | `/disable` | Disable SMS MFA |
| GET | `/status` | Get SMS MFA status |

### WebAuthn — `/api/user/2fa/webauthn`

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/registration-options` | Get registration challenge |
| POST | `/register` | Register credential |
| GET | `/credentials` | List registered credentials |
| DELETE | `/credentials/:id` | Remove credential |
| PATCH | `/credentials/:id` | Rename credential |
| GET | `/status` | Get WebAuthn status |

## Secrets — `/api/secrets`

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/tenant-vault/init` | Initialize tenant vault |
| POST | `/tenant-vault/distribute` | Distribute tenant vault key to members |
| GET | `/tenant-vault/status` | Check tenant vault status |
| DELETE | `/external-shares/:shareId` | Revoke external share |
| GET | `/` | List secrets (filtered by scope, type, search, folder) |
| POST | `/` | Create secret (LOGIN/SSH_KEY/CERTIFICATE/API_KEY/SECURE_NOTE) |
| GET | `/:id` | Get secret with decrypted payload |
| PUT | `/:id` | Update secret |
| DELETE | `/:id` | Delete secret |
| GET | `/:id/versions` | List version history |
| GET | `/:id/versions/:version/data` | Get specific version data |
| POST | `/:id/versions/:version/restore` | Restore to specific version |
| POST | `/:id/share` | Share secret with user |
| DELETE | `/:id/share/:userId` | Remove share |
| PUT | `/:id/share/:userId` | Update share permission |
| GET | `/:id/shares` | List shares |
| POST | `/:id/external-shares` | Create external share (token + optional PIN) |
| GET | `/:id/external-shares` | List external shares |

## Vault Folders — `/api/vault-folders`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | List vault folders |
| POST | `/` | Create vault folder |
| PUT | `/:id` | Update vault folder |
| DELETE | `/:id` | Delete vault folder |

## Public Share — `/api/share` (No Auth)

| Method | Endpoint | Rate Limited | Description |
|--------|----------|-------------|-------------|
| GET | `/:token/info` | Yes | Get share metadata (expiry, access count) |
| POST | `/:token` | Yes | Access share (with optional PIN) |

## Audit — `/api/audit`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | Personal audit logs (with action, date, target filters) |
| GET | `/countries` | All countries in audit logs |
| GET | `/gateways` | All gateways in audit logs |
| GET | `/tenant` | Tenant-scoped audit logs (admin) |
| GET | `/tenant/gateways` | Tenant gateway audit (admin) |
| GET | `/tenant/countries` | Tenant access countries (admin) |
| GET | `/tenant/geo-summary` | Tenant geo summary (admin) |
| GET | `/connection/:connectionId` | Connection audit logs |
| GET | `/connection/:connectionId/users` | Users who accessed connection |

## Notifications — `/api/notifications`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | List notifications (limit 50) |
| PUT | `/read-all` | Mark all as read |
| PUT | `/:id/read` | Mark single as read |
| DELETE | `/:id` | Delete notification |

## Tenants — `/api/tenants`

| Method | Endpoint | Role | Description |
|--------|----------|------|-------------|
| POST | `/` | Any | Create tenant |
| GET | `/mine/all` | Any | List user's tenants |
| GET | `/mine` | Any | Get current tenant details |
| PUT | `/:id` | ADMIN+ | Update tenant settings |
| DELETE | `/:id` | OWNER | Delete tenant |
| GET | `/:id/mfa-stats` | ADMIN+ | MFA adoption statistics |
| GET | `/:id/users` | ADMIN+ | List tenant users |
| GET | `/:id/users/:userId/profile` | ADMIN+ | View user profile |
| POST | `/:id/invite` | ADMIN+ | Invite user by email |
| PUT | `/:id/users/:userId` | ADMIN+ | Update user role |
| DELETE | `/:id/users/:userId` | ADMIN+ | Remove user |
| POST | `/:id/users` | ADMIN+ | Create user directly |
| PATCH | `/:id/users/:userId/enabled` | ADMIN+ | Toggle user enabled |
| PATCH | `/:id/users/:userId/expiry` | ADMIN+ | Set membership expiry |
| PUT | `/:id/users/:userId/email` | ADMIN+ | Change user email |
| PUT | `/:id/users/:userId/password` | ADMIN+ | Change user password |
| GET | `/:id/ip-allowlist` | ADMIN+ | Get IP allowlist |
| PUT | `/:id/ip-allowlist` | ADMIN+ | Update IP allowlist |

## Teams — `/api/teams`

| Method | Endpoint | Role | Description |
|--------|----------|------|-------------|
| POST | `/` | OPERATOR+ | Create team |
| GET | `/` | Any | List teams |
| GET | `/:id` | Member | Get team details |
| PUT | `/:id` | TEAM_ADMIN | Update team |
| DELETE | `/:id` | TEAM_ADMIN | Delete team |
| GET | `/:id/members` | Member | List members |
| POST | `/:id/members` | TEAM_ADMIN | Add member |
| PUT | `/:id/members/:userId` | TEAM_ADMIN | Update member role |
| DELETE | `/:id/members/:userId` | TEAM_ADMIN | Remove member |
| PATCH | `/:id/members/:userId/expiry` | TEAM_ADMIN | Set member expiry |

## Gateways — `/api/gateways`

| Method | Endpoint | Role | Description |
|--------|----------|------|-------------|
| GET | `/` | Any | List gateways |
| POST | `/` | OPERATOR+ | Create gateway |
| PUT | `/:id` | OPERATOR+ | Update gateway |
| DELETE | `/:id` | OPERATOR+ | Delete gateway |
| POST | `/:id/test` | Any | Test gateway connectivity |
| POST | `/ssh-keypair` | OPERATOR+ | Generate SSH keypair |
| GET | `/ssh-keypair` | OPERATOR+ | Get public key |
| GET | `/ssh-keypair/private` | OPERATOR+ | Download private key |
| POST | `/ssh-keypair/rotate` | OPERATOR+ | Rotate SSH keypair |
| PATCH | `/ssh-keypair/rotation` | OPERATOR+ | Update rotation policy |
| GET | `/ssh-keypair/rotation` | OPERATOR+ | Get rotation status |
| POST | `/:id/push-key` | OPERATOR+ | Push SSH key to gateway |
| POST | `/:id/deploy` | OPERATOR+ | Deploy managed gateway |
| DELETE | `/:id/deploy` | OPERATOR+ | Undeploy managed gateway |
| POST | `/:id/scale` | OPERATOR+ | Scale managed instances |
| GET | `/:id/instances` | OPERATOR+ | List instances |
| POST | `/:id/instances/:instanceId/restart` | OPERATOR+ | Restart instance |
| GET | `/:id/instances/:instanceId/logs` | OPERATOR+ | Get instance logs |
| GET | `/:id/scaling` | OPERATOR+ | Get scaling status |
| PUT | `/:id/scaling` | OPERATOR+ | Update scaling config |
| GET | `/templates` | OPERATOR+ | List templates |
| POST | `/templates` | OPERATOR+ | Create template |
| PUT | `/templates/:templateId` | OPERATOR+ | Update template |
| DELETE | `/templates/:templateId` | OPERATOR+ | Delete template |
| POST | `/templates/:templateId/deploy` | OPERATOR+ | Deploy from template |
| POST | `/:id/tunnel-token` | OPERATOR+ | Generate tunnel token |
| DELETE | `/:id/tunnel-token` | OPERATOR+ | Revoke tunnel token |
| POST | `/:id/tunnel-disconnect` | OPERATOR+ | Force disconnect tunnel |
| GET | `/:id/tunnel-events` | OPERATOR+ | Get tunnel events |
| GET | `/:id/tunnel-metrics` | OPERATOR+ | Get tunnel metrics |
| GET | `/tunnel-overview` | ADMIN+ | Tunnel fleet overview |

## Admin — `/api/admin`

| Method | Endpoint | Role | Description |
|--------|----------|------|-------------|
| GET | `/email/status` | ADMIN+ | Email configuration status |
| POST | `/email/test` | ADMIN+ | Send test email |
| GET | `/app-config` | ADMIN+ | Get application config |
| PUT | `/app-config/self-signup` | ADMIN+ | Enable/disable self-signup |
| GET | `/auth-providers` | ADMIN+ | Get auth provider details |

## Files — `/api/files`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | List uploaded files |
| GET | `/:name` | Download file |
| POST | `/` | Upload file (quota enforced) |
| DELETE | `/:name` | Delete file |

## Tabs — `/api/tabs`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | Get persisted open tabs |
| PUT | `/` | Sync tab state |
| DELETE | `/` | Clear all tabs |

## Recordings — `/api/recordings`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/` | List recordings (filtered) |
| GET | `/:id` | Get recording metadata |
| GET | `/:id/stream` | Stream asciicast recording |
| GET | `/:id/analyze` | Analyze recording content |
| GET | `/:id/video` | Export recording as video |
| DELETE | `/:id` | Delete recording |

## Other Endpoints

### GeoIP — `/api/geoip`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/:ip` | Lookup IP geolocation |

### LDAP — `/api/ldap`

| Method | Endpoint | Role | Description |
|--------|----------|------|-------------|
| GET | `/status` | ADMIN+ | LDAP connection status |
| POST | `/test` | ADMIN+ | Test LDAP connection |
| POST | `/sync` | ADMIN+ | Trigger LDAP sync |

### Sync Profiles — `/api/sync-profiles`

| Method | Endpoint | Role | Description |
|--------|----------|------|-------------|
| POST | `/` | ADMIN+ | Create sync profile |
| GET | `/` | ADMIN+ | List sync profiles |
| GET | `/:id` | ADMIN+ | Get sync profile |
| PUT | `/:id` | ADMIN+ | Update sync profile |
| DELETE | `/:id` | ADMIN+ | Delete sync profile |
| POST | `/:id/test` | ADMIN+ | Test connection |
| POST | `/:id/sync` | ADMIN+ | Trigger sync |
| GET | `/:id/logs` | ADMIN+ | Get sync logs |

### External Vault Providers — `/api/vault-providers`

| Method | Endpoint | Role | Description |
|--------|----------|------|-------------|
| GET | `/` | ADMIN+ | List vault providers |
| POST | `/` | ADMIN+ | Create provider |
| GET | `/:providerId` | ADMIN+ | Get provider |
| PUT | `/:providerId` | ADMIN+ | Update provider |
| DELETE | `/:providerId` | ADMIN+ | Delete provider |
| POST | `/:providerId/test` | ADMIN+ | Test provider connectivity |

### Access Policies — `/api/access-policies`

| Method | Endpoint | Role | Description |
|--------|----------|------|-------------|
| GET | `/` | ADMIN+ | List ABAC policies |
| POST | `/` | ADMIN+ | Create policy |
| PUT | `/:id` | ADMIN+ | Update policy |
| DELETE | `/:id` | ADMIN+ | Delete policy |

### Health

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/api/health` | No | Health check (always returns ok) |
| GET | `/api/ready` | No | Readiness probe (checks DB + guacd) |

## WebSocket Namespaces

### SSH Terminal — `/ssh` (Socket.IO)

**Auth**: JWT token passed in handshake `auth.token`.

| Event (Client → Server) | Payload | Description |
|--------------------------|---------|-------------|
| `open` | `{ sessionId, connectionId }` | Open SSH session |
| `stdin` | `data: string` | Send terminal input |
| `resize` | `{ cols, rows }` | Resize terminal |
| `sftp:list` | `{ path }` | List SFTP directory |
| `sftp:upload:start` | `{ filename, remotePath, totalBytes }` | Begin chunked upload |
| `sftp:upload:chunk` | `{ transferId, data, offset }` | Upload chunk (64KB) |
| `sftp:upload:end` | `{ transferId }` | Finalize upload |
| `sftp:download:start` | `{ remotePath }` | Begin file download |
| `sftp:delete` | `{ path }` | Delete file |
| `close` | — | Close SSH session |

| Event (Server → Client) | Payload | Description |
|--------------------------|---------|-------------|
| `data` | `output: string` | Terminal output |
| `sftp:progress` | `{ transferId, bytesTransferred, totalBytes }` | Transfer progress |
| `sftp:transfer:complete` | `{ transferId }` | Transfer completed |
| `sftp:transfer:error` | `{ transferId, error }` | Transfer failed |
| `sftp:download:chunk` | `{ transferId, data, offset, totalBytes }` | Download data chunk |

### Gateway Monitor — `/gateway-monitor` (Socket.IO)

| Event (Server → Client) | Payload | Description |
|--------------------------|---------|-------------|
| `gateway:health` | `{ gatewayId, status, latency }` | Health status update |
| `instances:updated` | `{ gatewayId, instances[] }` | Instance state change |
| `scaling:updated` | `{ gatewayId, scalingStatus }` | Scaling status change |
| `gateway:updated` | `{ gatewayId, ...partial }` | Gateway config update |
| `tunnel:status` | `{ gatewayId, connected, agent }` | Tunnel connection status |

### Notifications — `/notifications` (Socket.IO)

| Event (Server → Client) | Payload | Description |
|--------------------------|---------|-------------|
| `notification` | `NotificationEntry` | New notification |

### Guacamole — `/guacamole` (WebSocket)

Native WebSocket on port 3002. Uses Guacamole protocol (not Socket.IO). Client passes encrypted token as query parameter. Server decrypts and forwards to guacd daemon.
