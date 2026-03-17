---
title: Getting Started
description: Installation, prerequisites, and first run guide for Arsenale
generated-by: ctdf-docs
generated-at: 2026-03-16T19:30:00Z
source-files:
  - README.md
  - CLAUDE.md
  - package.json
  - .env.example
  - compose.dev.yml
---

# Getting Started

## Prerequisites

| Requirement | Version | Purpose |
|------------|---------|---------|
| **Node.js** | 22+ | Runtime for server and build tools |
| **npm** | 9+ | Package manager (workspaces support) |
| **Docker** or **Podman** | Latest | PostgreSQL database and guacd daemon |
| **Git** | Latest | Source control |

## Installation

### 1. Clone and Install

```bash
git clone https://github.com/dnviti/arsenale.git
cd arsenale
npm install
```

### 2. Configure Environment

```bash
cp .env.example .env
```

The `.env` file lives at the **monorepo root**, not inside `server/`. Key variables to review:

| Variable | Default | Purpose |
|----------|---------|---------|
| `DATABASE_URL` | `postgresql://arsenale:arsenale@localhost:5432/arsenale` | PostgreSQL connection |
| `JWT_SECRET` | `dev-secret-change-me` | JWT signing key |
| `GUACD_HOST` | `localhost` | Guacamole daemon host |
| `GUACD_PORT` | `4822` | Guacamole daemon port |
| `GUACAMOLE_SECRET` | `dev-guac-secret` | Guacamole token encryption |
| `CLIENT_URL` | `http://localhost:3000` | CORS origin |
| `VAULT_TTL_MINUTES` | `30` | Vault auto-lock timeout |

See [Configuration](configuration.md) for the full variable reference (120+ variables).

### 3. Start Development Stack

```bash
npm run predev && npm run dev
```

This single command:
1. **`predev`**: Starts Docker containers (PostgreSQL + guacenc), generates Prisma client
2. **`dev`**: Runs Express server (:3001) and Vite client (:3000) concurrently

Database migrations run automatically on server startup — no manual migration step needed.

### 4. Access the Application

| URL | Service |
|-----|---------|
| http://localhost:3000 | Web client (Vite dev server, proxies API calls) |
| http://localhost:3001 | Express API (direct access) |
| http://localhost:3002 | Guacamole WebSocket (RDP/VNC) |

Register a new account at the login page. The first user becomes the tenant owner.

## Development Workflow

```mermaid
flowchart LR
    PreDev["npm run predev<br/>Docker + Prisma"] --> Dev["npm run dev<br/>Server + Client"]
    Dev --> Code["Edit code<br/>Hot reload"]
    Code --> Verify["npm run verify<br/>Type + Lint + Build"]
    Verify --> Commit["Commit"]
```

### Running Services Individually

```bash
npm run dev:server    # Express on :3001 (tsx watch, hot reload)
npm run dev:client    # Vite on :3000 (proxies /api→:3001, /socket.io→:3001)
```

### Docker Containers

```bash
npm run docker:dev        # Start PostgreSQL + guacenc
npm run docker:dev:down   # Stop dev containers
```

The dev stack (`compose.dev.yml`) provides:
- **PostgreSQL 16** on port 5432 (credentials: `arsenale/arsenale`)
- **guacenc** on port 3003 (recording conversion service)

### Database Operations

```bash
npm run db:generate    # Regenerate Prisma client types after schema changes
npm run db:push        # Push schema changes directly (no migration file)
npm run db:migrate     # Create and run migration files
```

Migrations run automatically on server start via `prisma migrate deploy`. Use `db:push` only for rapid prototyping; use `db:migrate` for changes that need to be tracked.

## Verification

Before committing, run the full verification pipeline:

```bash
npm run verify
```

This executes in sequence:
1. **typecheck** — TypeScript type checking (both workspaces, no emit)
2. **lint** — ESLint (both workspaces via root flat config)
3. **sast** — `npm audit` (dependency vulnerability scan)
4. **test** — Unit tests (Vitest)
5. **build** — Server (tsc) + Client (Vite build)

## First Steps After Installation

1. **Register** — Create an account at the login page
2. **Unlock Vault** — Enter your password to unlock the credential vault
3. **Create a Connection** — Add an SSH, RDP, or VNC connection with host/port/credentials
4. **Connect** — Click the connection to open a terminal (SSH) or remote desktop (RDP/VNC)

## Alternative: Makefile

```bash
make full-stack    # Install + run server + client
make server-dev    # Server only with tsx watch
make client-dev    # Client only with Vite
make prisma-studio # Open Prisma Studio UI
```
