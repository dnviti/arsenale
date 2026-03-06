# Deployment

> Auto-generated on 2026-03-01 by `/docs create deployment`.
> Source of truth is the codebase. Run `/docs update deployment` after code changes.

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Node.js | 22.x | Server and client builds |
| npm | 10.x+ | Package management (workspaces) |
| Docker | 20.x+ | Container runtime |
| Docker Compose | v2 | Multi-container orchestration |

<!-- manual-start -->
<!-- manual-end -->

## Development Setup

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

Default values work out of the box for development. Key defaults:

| Variable | Default | Notes |
|----------|---------|-------|
| `DATABASE_URL` | `postgresql://arsenale:arsenale_password@127.0.0.1:5432/arsenale` | Uses `127.0.0.1` (not `localhost`) to avoid IPv6 issues on Windows |
| `JWT_SECRET` | `change-me-in-production` | Fine for development |
| `GUACD_HOST` | `localhost` | Docker-exposed guacd |
| `EMAIL_PROVIDER` | `smtp` | With empty `SMTP_HOST`, verification links are logged to console |
| `SMS_PROVIDER` | _(empty)_ | OTP codes logged to console in dev mode |

### 3. Start Development

```bash
npm run predev  # Starts Docker containers (guacd + postgres), generates Prisma client, syncs DB schema
npm run dev     # Starts server (:3001) and client (:3000) concurrently
```

Or in one command:

```bash
npm run predev && npm run dev
```

The `predev` script handles:
1. `docker compose -f docker-compose.dev.yml up -d --wait` — starts guacd and PostgreSQL
2. `npm run db:generate` — generates Prisma client types
3. `npm run db:push` — syncs schema to database (no migration files)

### Development Docker Containers

| Service | Image | Exposed Port | Purpose |
|---------|-------|-------------|---------|
| guacd | `guacamole/guacd` | 4822 | Guacamole daemon for RDP |
| postgres | `postgres:16` | 5432 | Database (user: `arsenale`, password: `arsenale_password`) |

Data persistence: PostgreSQL uses named volume `pgdata_dev`. guacd drive files stored at `./data/drive`.

<!-- manual-start -->
<!-- manual-end -->

## Production Deployment

### 1. Configure Secrets

```bash
cp .env.production.example .env.production
```

Generate strong secrets:

```bash
# Generate each secret separately
openssl rand -base64 32  # For JWT_SECRET
openssl rand -base64 32  # For GUACAMOLE_SECRET
openssl rand -base64 32  # For POSTGRES_PASSWORD
```

Required production variables:

| Variable | Description |
|----------|-------------|
| `POSTGRES_PASSWORD` | PostgreSQL database password |
| `JWT_SECRET` | JWT signing secret (≥32 bytes) |
| `GUACAMOLE_SECRET` | Guacamole token encryption key |
| `VAULT_TTL_MINUTES` | Vault session TTL (default: 30) |

### 2. Deploy with Docker Compose

```bash
docker compose --env-file .env.production up -d --build
```

### Production Docker Topology

```
┌─────────────────────────────────────────────────────┐
│                  Docker Network                      │
│                                                      │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐        │
│  │ postgres │   │  guacd   │   │  server  │        │
│  │ (PG 16)  │   │          │   │ :3001    │        │
│  │          │◄──┤          │◄──┤ :3002    │        │
│  │ internal │   │ internal │   │ exposed  │        │
│  └──────────┘   └──────────┘   └──────────┘        │
│       ▲                              ▲               │
│       │ healthcheck                  │               │
│       │                              │               │
│  ┌──────────────────────────────────────────┐       │
│  │              client (nginx)              │       │
│  │              :8080 → host :3000           │       │
│  └──────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────┘
```

| Service | Image | Ports | Dependencies |
|---------|-------|-------|-------------|
| postgres | `postgres:16` | Internal only | — |
| guacd | `guacamole/guacd` | Internal only | — |
| server | Custom (`server/Dockerfile`) | 3001:3001 | postgres (healthy), guacd (started) |
| client | Custom (`client/Dockerfile`) | 3000:8080 | server |

### Service Details

**postgres**:
- Health check: `pg_isready` every 5s with 5 retries
- Volume: `pgdata` (named, persistent)
- Environment from `.env.production`

**guacd**:
- Volume: `arsenale_drive` (shared with server for drive redirection)
- No health check (starts immediately)

**server**:
- Built from `server/Dockerfile` (Node 22 Alpine)
- Runs `prisma migrate deploy` on startup, then `node dist/index.js`
- Exposes ports 3001 (HTTP/Socket.IO) and 3002 (Guacamole WS)
- Volume: `arsenale_drive` at `/guacd-drive`
- Environment: `DATABASE_URL`, `GUACD_HOST=guacd`, `NODE_ENV=production`, secrets from `.env.production`

**client**:
- Multi-stage build: Node 22 Alpine (build) → Alpine 3.21 with nginx (runtime)
- Serves Vite build output from `/usr/share/nginx/html`
- nginx config from `client/nginx.conf`
- Exposes port 8080 (mapped to host 3000)

### Volume Management

| Volume | Mount Point | Purpose |
|--------|-------------|---------|
| `pgdata` | `/var/lib/postgresql/data` | PostgreSQL data persistence |
| `arsenale_drive` | `/guacd-drive` (server + guacd) | RDP drive redirection file storage |

<!-- manual-start -->
<!-- manual-end -->

## Nginx Configuration

Production nginx (`client/nginx.conf`) handles reverse proxying:

| Location | Target | Notes |
|----------|--------|-------|
| `/api` | `http://server:3001` | REST API + WebSocket upgrade support |
| `/socket.io` | `http://server:3001` | Socket.IO (SSH terminals, notifications) |
| `/guacamole` | `http://server:3002/` | Guacamole WebSocket (24h timeout) |
| `/` | Static files | SPA fallback (`try_files $uri $uri/ /index.html`) |

All proxy locations include WebSocket upgrade headers (`Upgrade`, `Connection`). The `/guacamole` location has extended timeouts (86400s) for long-lived RDP sessions.

<!-- manual-start -->
<!-- manual-end -->

## Available Scripts

| Script | Description |
|--------|-------------|
| `npm run predev` | Start dev Docker containers, generate Prisma, sync DB |
| `npm run dev` | Run server and client concurrently (hot reload) |
| `npm run dev:server` | Run server only (tsx watch on :3001) |
| `npm run dev:client` | Run client only (Vite on :3000) |
| `npm run build` | Build both server (tsc) and client (vite build) |
| `npm run docker:dev` | Start dev Docker containers |
| `npm run docker:dev:down` | Stop dev Docker containers |
| `npm run docker:prod` | Build and start production stack |
| `npm run db:generate` | Generate Prisma client types |
| `npm run db:push` | Sync Prisma schema to DB (no migration) |
| `npm run db:migrate` | Run Prisma migrations |
| `npm run typecheck` | TypeScript type-check (both workspaces) |
| `npm run lint` | ESLint check |
| `npm run lint:fix` | ESLint with auto-fix |
| `npm run sast` | npm audit (dependency vulnerability scan) |
| `npm run verify` | Full pipeline: typecheck → lint → audit → build |

<!-- manual-start -->
<!-- manual-end -->

## Troubleshooting

### IPv6 / localhost on Windows

PostgreSQL connection may fail with `localhost` on Windows due to IPv6 resolution. Use `127.0.0.1` instead:

```
DATABASE_URL=postgresql://arsenale:arsenale_password@127.0.0.1:5432/arsenale
```

### Docker Networking

- In development, containers expose ports to the host. The server runs on the host and connects to `localhost:4822` (guacd) and `127.0.0.1:5432` (postgres).
- In production, services communicate via Docker internal DNS names (`postgres`, `guacd`, `server`). No ports are exposed except server (3001) and client (80→3000).

### guacamole-lite Not Available

If `guacamole-lite` fails to load (native dependency issues), the server logs a warning and continues. RDP connections won't work, but SSH remains functional.

### Database Migrations

- Development: `npm run db:push` syncs schema directly (no migration files)
- Production: `prisma migrate deploy` runs on container startup. Create migrations with `npm run db:migrate` before deploying.

<!-- manual-start -->
<!-- manual-end -->
