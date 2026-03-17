---
title: Documentation Index
description: Landing page, table of contents, and project summary for Arsenale
generated-by: ctdf-docs
generated-at: 2026-03-16T19:30:00Z
source-files:
  - README.md
  - CLAUDE.md
  - package.json
---

# Arsenale Documentation

Arsenale is an enterprise-grade Privileged Access Management (PAM) platform for managing SSH, RDP, and VNC connections with end-to-end encryption, multi-tenancy, and comprehensive audit logging.

## Quick Start

```bash
git clone https://github.com/dnviti/arsenale.git
cd arsenale
cp .env.example .env
npm install
npm run predev && npm run dev
```

- **Client**: http://localhost:3000
- **Server API**: http://localhost:3001
- **Guacamole WS**: http://localhost:3002

## Table of Contents

### Core Documentation

| Document | Description |
|----------|-------------|
| [Architecture](architecture.md) | System architecture, component interactions, data flow |
| [Getting Started](getting-started.md) | Installation, prerequisites, first run |
| [Configuration](configuration.md) | Environment variables, config files, feature flags |
| [API Reference](api-reference.md) | All REST endpoints, WebSocket namespaces |
| [Deployment](deployment.md) | Docker, CI/CD, production setup |
| [Development](development.md) | Contributing, local dev, testing, branch strategy |
| [Troubleshooting](troubleshooting.md) | Common errors, debugging, FAQ |
| [LLM Context](llm-context.md) | Consolidated single-file for LLM/bot consumption |

### Deep-Dive References

| Directory | Contents |
|-----------|----------|
| [api/](api/overview.md) | Detailed endpoint documentation per domain |
| [components/](components/overview.md) | Client-side component, store, and hook reference |
| [database/](database/overview.md) | Prisma schema, models, enums, relationships |
| [security/](security/authentication.md) | Authentication flows, encryption, access policies |
| [guides/](guides/zero-trust-tunnel-user-guide.md) | Zero-trust tunnel setup and implementation |

## Technology Stack

| Layer | Technology |
|-------|-----------|
| **Server** | Express 5, TypeScript, Prisma 7, PostgreSQL 16 |
| **Client** | React 19, Vite 7, MUI v7, Zustand, XTerm.js, Guacamole |
| **Auth** | JWT, OAuth 2.0 (Google/GitHub/Microsoft), SAML 2.0, LDAP, TOTP, WebAuthn, SMS |
| **Encryption** | AES-256-GCM at rest, Argon2id key derivation |
| **Real-Time** | Socket.IO (SSH terminal), Guacamole WebSocket (RDP/VNC) |
| **Infrastructure** | Docker/Podman/Kubernetes, GitHub Actions CI/CD |
| **Browser Extension** | Chrome Manifest V3, React popup/options, content script autofill |

## Key Features

- **SSH/RDP/VNC** — Browser-based remote access with session recording
- **Encrypted Vault** — AES-256-GCM credential storage with master key per user
- **Multi-Tenancy** — Tenant isolation with RBAC (Owner/Admin/Operator/Member/Consultant/Auditor/Guest)
- **Teams** — Team-scoped connections and secrets with role-based access
- **Secret Manager** — Login, SSH key, certificate, API key, and secure note storage with versioning
- **Sharing** — Connection and secret sharing with granular permissions
- **MFA** — TOTP, WebAuthn/Passkeys, SMS verification
- **Audit Logging** — 120+ action types with geo-IP enrichment and impossible travel detection
- **DLP** — Data Loss Prevention policies (clipboard, download, upload restrictions)
- **Gateway Orchestration** — Managed SSH gateways with auto-scaling and health monitoring
- **Zero-Trust Tunnels** — Token-based gateway registration with mTLS
- **ABAC Policies** — Time windows, MFA step-up, trusted device requirements
- **PWA** — Progressive Web App with offline support and app shortcuts
- **Browser Extension** — Chrome extension for credential autofill and keychain access
