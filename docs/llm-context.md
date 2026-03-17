---
title: LLM Context
description: Consolidated single-file context for LLM and bot consumption
generated-by: ctdf-docs
generated-at: 2026-03-16T19:30:00Z
source-files:
  - CLAUDE.md
  - README.md
  - server/src/index.ts
  - server/src/app.ts
  - server/prisma/schema.prisma
  - client/src/App.tsx
  - client/vite.config.ts
  - package.json
---

# LLM Context — Arsenale

## Project Summary

Arsenale is an enterprise Privileged Access Management (PAM) platform. It provides browser-based SSH, RDP, and VNC access with encrypted credential storage, multi-tenancy, audit logging, and gateway orchestration.

**Stack**: Express 5 + React 19 + PostgreSQL 16 + Prisma 7 + Socket.IO + Guacamole + Docker

**Monorepo workspaces**: `server/`, `client/`, `tunnel-agent/`, `clients/browser-extensions/`

---

## Architecture

**Server** (Express, TypeScript, CommonJS):
- Layered: Routes (32) → Controllers (30) → Services (53) → Prisma ORM → PostgreSQL
- Entry: `server/src/index.ts` — auto-migrates DB, creates HTTP + Socket.IO + Guacamole WS servers
- App: `server/src/app.ts` — Helmet, CORS, CSRF, Passport, 32 route mounts under `/api`
- 19 middleware files (auth, CSRF, 7 rate limiters, tenant/team RBAC)
- 5 Socket.IO handlers (SSH terminal, notifications, gateway monitor, tunnel)
- Scheduled jobs: key rotation, LDAP sync, cleanup, health monitoring, auto-scaling

**Client** (React 19, Vite 7, MUI v7, Zustand):
- 15 Zustand stores, 12 custom hooks, 31 API modules, 10 pages, 100+ components
- Full-screen dialog pattern (15 dialogs) — overlays preserve active sessions
- PWA with offline support, keyboard lock API, DLP browser hardening
- Real-time: Socket.IO for SSH terminals, WebSocket for RDP/VNC via Guacamole

**Browser Extension** (Chrome Manifest V3):
- Service worker handles API calls (bypasses CORS), token refresh via chrome.alarms
- Popup: account switcher, vault status, connections, keychain
- Content scripts: form detection and credential autofill
- Multi-account with AES-GCM encrypted token storage

---

## Key API Endpoints (150+)

| Domain | Prefix | Key Operations |
|--------|--------|---------------|
| Auth | `/api/auth` | Login, register, OAuth, SAML, MFA (TOTP/SMS/WebAuthn), refresh |
| Vault | `/api/vault` | Unlock/lock, MFA unlock, password reveal, auto-lock |
| Connections | `/api/connections` | CRUD, sharing, batch share, import/export, favorites |
| Sessions | `/api/sessions` | SSH/RDP/VNC session lifecycle, monitoring, terminate |
| Secrets | `/api/secrets` | CRUD (5 types), versioning, sharing, external shares, tenant vault |
| Users | `/api/user` | Profile, password, email, SSH/RDP defaults, domain profile, MFA setup |
| Tenants | `/api/tenants` | Multi-tenant management, users, roles, IP allowlist |
| Teams | `/api/teams` | Team CRUD, members, roles, expiry |
| Gateways | `/api/gateways` | CRUD, SSH keys, orchestration, templates, tunnels, scaling |
| Audit | `/api/audit` | Personal/tenant logs, geo analysis, connection/gateway audit |
| Admin | `/api/admin` | Email config, app config, auth providers |
| Health | `/api/health`, `/api/ready` | Health/readiness probes |

---

## Database Models (25+)

**Core**: User, Connection (SSH/RDP/VNC), Folder, SharedConnection
**Auth**: OAuthAccount, RefreshToken, WebAuthnCredential
**Tenancy**: Tenant, TenantMember, TenantVaultMember, Team, TeamMember
**Vault**: VaultSecret (5 types, 3 scopes), VaultSecretVersion, VaultFolder, ExternalSecretShare, SharedSecret
**Sessions**: ActiveSession (ACTIVE/IDLE/CLOSED), SessionRecording
**Gateways**: Gateway (3 types), GatewayTemplate, ManagedGatewayInstance, SshKeyPair
**Monitoring**: AuditLog (120+ actions), Notification, AccessPolicy (ABAC)
**Integration**: SyncProfile (NetBox), SyncLog, ExternalVaultProvider, AppConfig

**Key enums**: TenantRole (7 levels), TeamRole (3 levels), ConnectionType, SecretType, SecretScope, SessionProtocol, GatewayType, AuditAction (120+)

---

## Security

- **Encryption**: AES-256-GCM at rest, Argon2id key derivation, per-user master key with TTL
- **Auth**: JWT with token binding (IP+UA hash), refresh token rotation, OAuth/SAML/LDAP
- **MFA**: TOTP, WebAuthn/Passkeys, SMS
- **RBAC**: 7 tenant roles (OWNER→GUEST), 3 team roles
- **ABAC**: Time windows, MFA step-up, trusted device requirements
- **DLP**: Clipboard, download, upload, print restrictions
- **Rate limiting**: Login, registration, SMS, OAuth, vault unlock, identity verification
- **Audit**: 120+ action types, geo-IP enrichment, impossible travel detection
- **Network**: IP allowlist, CSRF protection, Helmet security headers

---

## Development Commands

```bash
npm run predev && npm run dev  # Full dev setup (Docker + server + client)
npm run dev:server             # Express on :3001 (tsx watch)
npm run dev:client             # Vite on :3000 (proxies to :3001)
npm run verify                 # typecheck → lint → audit → test → build
npm run db:generate            # Regenerate Prisma client
npm run db:migrate             # Run database migrations
npm run docker:prod            # Production Docker stack
```

---

## Key Patterns

1. **Full-screen dialogs** over navigation — never create page routes for overlay UI
2. **API errors**: Use `extractApiError(err, fallback)` from `client/src/utils/apiError.ts`
3. **UI preferences**: Persist via `uiPreferencesStore` (Zustand + localStorage), never raw localStorage
4. **File naming**: `*.routes.ts`, `*.controller.ts`, `*.service.ts`, `*Store.ts`, `*.api.ts`, `use*.ts`
5. **Environment**: Single `.env` at monorepo root, Prisma resolves via `server/prisma.config.ts`
6. **Real-time**: Socket.IO for SSH (`/ssh` namespace), native WebSocket for RDP/VNC (Guacamole on :3002)

---

## Configuration

120+ environment variables. Key categories:
- Database (`DATABASE_URL`), JWT (`JWT_SECRET`, `JWT_EXPIRES_IN`)
- Guacamole (`GUACD_HOST`, `GUACAMOLE_SECRET`), Vault (`VAULT_TTL_MINUTES`)
- OAuth (Google, Microsoft, GitHub, OIDC), SAML, LDAP
- Email (SMTP, SendGrid, SES, Resend, Mailgun), SMS (Twilio, SNS, Vonage)
- Orchestration (`ORCHESTRATOR_TYPE`: docker/podman/kubernetes)
- Logging (`LOG_LEVEL`, `LOG_FORMAT`), GeoIP, recordings, files

See `docs/configuration.md` for the full reference.
