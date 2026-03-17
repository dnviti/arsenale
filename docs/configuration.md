---
title: Configuration
description: Environment variables, config files, and feature flags reference
generated-by: ctdf-docs
generated-at: 2026-03-16T19:30:00Z
source-files:
  - .env.example
  - server/src/index.ts
  - server/src/app.ts
  - server/src/config/passport.ts
  - server/prisma.config.ts
  - client/vite.config.ts
  - compose.yml
  - compose.dev.yml
---

# Configuration

All environment variables are defined in a single `.env` file at the **monorepo root**. Never create a separate `server/.env` — Prisma CLI commands resolve the `.env` path to `../.env` via `server/prisma.config.ts`.

## Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `NODE_ENV` | `development` | Environment mode (`development`, `production`) |
| `PORT` | `3001` | Express server port |
| `CLIENT_URL` | `http://localhost:3000` | Client origin for CORS and OAuth redirects |
| `TRUST_PROXY` | `false` | Express trust proxy (`false`, `true`, number, or CIDR list) |
| `ALLOW_LOCAL_NETWORK` | `true` | Allow connections to private IP ranges (10.x, 172.16-31.x, 192.168.x) |

## Database

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgresql://arsenale:arsenale@localhost:5432/arsenale` | PostgreSQL connection string |
| `POSTGRES_USER` | `arsenale` | Database user (Docker) |
| `POSTGRES_PASSWORD` | `arsenale` | Database password (Docker) |
| `POSTGRES_DB` | `arsenale` | Database name (Docker) |

## Authentication — JWT

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | `dev-secret-change-me` | Signing key for access tokens |
| `JWT_EXPIRES_IN` | `15m` | Access token lifetime |
| `JWT_REFRESH_EXPIRES_IN` | `7d` | Refresh token lifetime |

## Authentication — OAuth

### Google

| Variable | Description |
|----------|-------------|
| `GOOGLE_CLIENT_ID` | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret |

### Microsoft

| Variable | Description |
|----------|-------------|
| `MICROSOFT_CLIENT_ID` | Microsoft/Azure AD client ID |
| `MICROSOFT_CLIENT_SECRET` | Microsoft/Azure AD client secret |
| `MICROSOFT_TENANT_ID` | Azure AD tenant (default: `common`) |

### GitHub

| Variable | Description |
|----------|-------------|
| `GITHUB_CLIENT_ID` | GitHub OAuth app client ID |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth app client secret |

### Generic OIDC

| Variable | Description |
|----------|-------------|
| `OIDC_ISSUER_URL` | OIDC discovery URL (e.g., Authentik, Keycloak, Authelia, Zitadel) |
| `OIDC_CLIENT_ID` | OIDC client ID |
| `OIDC_CLIENT_SECRET` | OIDC client secret |
| `OIDC_DISPLAY_NAME` | Button label in UI |

### SAML 2.0

| Variable | Description |
|----------|-------------|
| `SAML_ENTRY_POINT` | IdP SSO URL |
| `SAML_ISSUER` | SP entity ID |
| `SAML_CERT` | IdP signing certificate (PEM, no headers) |
| `SAML_PRIVATE_KEY` | SP private key for signing/decryption |
| `SAML_DISPLAY_NAME` | Button label in UI |
| `SAML_WANT_ASSERTIONS_SIGNED` | Require signed assertions (default: `true`) |

### LDAP

| Variable | Default | Description |
|----------|---------|-------------|
| `LDAP_URL` | — | LDAP server URL (e.g., `ldap://freeipa.local:389`) |
| `LDAP_BIND_DN` | — | Bind DN for search |
| `LDAP_BIND_PASSWORD` | — | Bind password |
| `LDAP_BASE_DN` | — | Search base |
| `LDAP_USER_FILTER` | `(uid={{username}})` | User search filter |
| `LDAP_GROUP_BASE_DN` | — | Group search base |
| `LDAP_GROUP_FILTER` | `(member={{dn}})` | Group membership filter |
| `LDAP_GROUP_ROLE_MAPPING` | — | JSON: `{"cn=admins,...":"ADMIN"}` |
| `LDAP_SYNC_CRON` | `0 */6 * * *` | Sync schedule |
| `LDAP_AUTO_PROVISION` | `true` | Auto-create users on first login |
| `LDAP_DEFAULT_TENANT_ID` | — | Tenant for auto-provisioned users |

### WebAuthn / Passkeys

| Variable | Default | Description |
|----------|---------|-------------|
| `WEBAUTHN_RP_ID` | `localhost` | Relying party ID (domain) |
| `WEBAUTHN_RP_ORIGIN` | `http://localhost:3000` | Expected origin |
| `WEBAUTHN_RP_NAME` | `Arsenale` | Display name |

## Vault & Security

| Variable | Default | Description |
|----------|---------|-------------|
| `VAULT_TTL_MINUTES` | `30` | Master key in-memory TTL |
| `SERVER_ENCRYPTION_KEY` | — | 32-byte hex key for server-side encryption (auto-generated in dev) |
| `IMPOSSIBLE_TRAVEL_SPEED_KMH` | `900` | Threshold for impossible travel detection |
| `ALLOW_EXTERNAL_SHARING` | `false` | Enable cross-tenant secret sharing |

### Rate Limiting & Account Lockout

| Variable | Default | Description |
|----------|---------|-------------|
| `LOGIN_RATE_LIMIT_WINDOW_MS` | `900000` | Login attempt window (15 min) |
| `LOGIN_RATE_LIMIT_MAX` | `10` | Max login attempts per window |
| `ACCOUNT_LOCKOUT_THRESHOLD` | `5` | Failed attempts before lockout |
| `ACCOUNT_LOCKOUT_DURATION_MS` | `1800000` | Lockout duration (30 min) |

### Session Limits

| Variable | Default | Description |
|----------|---------|-------------|
| `MAX_CONCURRENT_SESSIONS` | `10` | Max concurrent remote sessions per user |
| `ABSOLUTE_SESSION_TIMEOUT_SECONDS` | `43200` | Session absolute timeout (12 hours) |

## Guacamole (RDP/VNC)

| Variable | Default | Description |
|----------|---------|-------------|
| `GUACD_HOST` | `localhost` | guacd daemon host |
| `GUACD_PORT` | `4822` | guacd daemon port |
| `GUACAMOLE_SECRET` | `dev-guac-secret` | Token encryption key for guacamole-lite |
| `GUACAMOLE_WS_PORT` | `3002` | Guacamole WebSocket server port |

## Email

| Variable | Default | Description |
|----------|---------|-------------|
| `EMAIL_PROVIDER` | `smtp` | Provider: `smtp`, `sendgrid`, `ses`, `resend`, `mailgun` |
| `EMAIL_FROM` | — | Sender address |
| `EMAIL_VERIFY_REQUIRED` | `false` | Require email verification before login |
| `SELF_SIGNUP_ENABLED` | `true` | Allow self-registration |

### SMTP

| Variable | Description |
|----------|-------------|
| `SMTP_HOST` | SMTP server host |
| `SMTP_PORT` | SMTP port (587 for TLS) |
| `SMTP_USER` | SMTP username |
| `SMTP_PASS` | SMTP password |
| `SMTP_SECURE` | Use TLS (`true`/`false`) |

### Cloud Providers

| Provider | Variables |
|----------|-----------|
| SendGrid | `SENDGRID_API_KEY` |
| AWS SES | `AWS_SES_REGION`, `AWS_SES_ACCESS_KEY_ID`, `AWS_SES_SECRET_ACCESS_KEY` |
| Resend | `RESEND_API_KEY` |
| Mailgun | `MAILGUN_API_KEY`, `MAILGUN_DOMAIN` |

## SMS (MFA)

| Variable | Default | Description |
|----------|---------|-------------|
| `SMS_PROVIDER` | — | Provider: `twilio`, `sns`, `vonage` |

| Provider | Variables |
|----------|-----------|
| Twilio | `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_FROM_NUMBER` |
| AWS SNS | `AWS_SNS_REGION`, `AWS_SNS_ACCESS_KEY_ID`, `AWS_SNS_SECRET_ACCESS_KEY` |
| Vonage | `VONAGE_API_KEY`, `VONAGE_API_SECRET`, `VONAGE_FROM_NUMBER` |

## Logging & Monitoring

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | `error`, `warn`, `info`, `verbose`, `debug` |
| `LOG_FORMAT` | `text` | `text` or `json` |
| `LOG_TIMESTAMPS` | `true` | Include timestamps |
| `LOG_HTTP_REQUESTS` | `false` | Log HTTP requests |
| `LOG_GUACAMOLE` | `false` | Log Guacamole traffic |
| `GEOIP_DB_PATH` | — | Path to MaxMind GeoLite2 database |

## Files & Recordings

| Variable | Default | Description |
|----------|---------|-------------|
| `DRIVE_BASE_PATH` | `./drive` | File storage base directory |
| `FILE_UPLOAD_MAX_SIZE` | `52428800` | Max upload size (50MB) |
| `USER_DRIVE_QUOTA` | `104857600` | Per-user quota (100MB) |
| `RECORDING_ENABLED` | `false` | Enable session recording |
| `RECORDING_PATH` | `./recordings` | Recording storage path |
| `RECORDING_RETENTION_DAYS` | `30` | Auto-cleanup after N days |
| `GUACENC_SERVICE_URL` | — | Guacenc video conversion endpoint |
| `ASCIICAST_CONVERTER_URL` | — | SSH recording conversion endpoint |

## Container Orchestration

| Variable | Default | Description |
|----------|---------|-------------|
| `ORCHESTRATOR_TYPE` | — | `docker`, `podman`, `kubernetes`, or auto-detect |
| `DOCKER_SOCKET_PATH` | `/var/run/docker.sock` | Docker socket path |
| `DOCKER_NETWORK` | `arsenale_net` | Docker network for managed instances |
| `PODMAN_SOCKET_PATH` | — | Podman socket path |
| `ORCHESTRATOR_K8S_NAMESPACE` | `arsenale` | Kubernetes namespace |
| `ORCHESTRATOR_SSH_GATEWAY_IMAGE` | `ghcr.io/dnviti/arsenale/ssh-gateway:latest` | SSH gateway container image |
| `ORCHESTRATOR_GUACD_IMAGE` | `guacamole/guacd:1.6.0` | guacd container image |

## SSH Gateway & Tunnels

| Variable | Default | Description |
|----------|---------|-------------|
| `SSH_GATEWAY_PORT` | `2222` | SSH gateway listen port |
| `SSH_AUTHORIZED_KEYS` | — | Authorized keys file path |
| `GATEWAY_API_TOKEN` | — | API authentication token |
| `KEY_ROTATION_CRON` | — | SSH key rotation schedule |
| `KEY_ROTATION_ADVANCE_DAYS` | — | Days before expiry to rotate |

## Config Files

| File | Purpose |
|------|---------|
| `.env` | Environment variables (monorepo root) |
| `eslint.config.mjs` | ESLint flat config (TypeScript + security + React) |
| `server/tsconfig.json` | Server TypeScript (ES2022, CommonJS) |
| `client/tsconfig.json` | Client TypeScript (ES2022, ESNext, react-jsx) |
| `client/vite.config.ts` | Vite config with proxy, PWA, and chunk splitting |
| `server/prisma.config.ts` | Prisma config (resolves .env to monorepo root) |
| `server/prisma/schema.prisma` | Database schema |
| `compose.dev.yml` | Development Docker stack |
| `compose.yml` | Production Docker stack |
