# Architecture

> Auto-generated on 2026-03-01 by `/docs create architecture`.
> Source of truth is the codebase. Run `/docs update architecture` after code changes.

## System Overview

Arsenale is a **monorepo** using npm workspaces with two packages:

```
arsenale/
├── server/          # Express + TypeScript backend
├── client/          # React 19 + Vite frontend
├── package.json     # Root workspace config
├── docker-compose.yml           # Production stack
└── docker-compose.dev.yml       # Dev containers (guacd + postgres)
```

<!-- manual-start -->
<!-- manual-end -->

## Server Architecture

**Entry point**: `server/src/index.ts`

The server follows a **layered architecture**:

```
Routes → Controllers → Services → Prisma ORM → PostgreSQL
```

### Startup Sequence

1. Run startup migrations (mark legacy users as email-verified and vault-setup-complete)
2. Create HTTP server from Express app
3. Attach Socket.IO (SSH terminal + notifications)
4. Start Guacamole WebSocket server (`guacamole-lite`) on port 3002
5. Listen on configured port (default 3001)

### Express App (`server/src/app.ts`)

**Middleware pipeline**:
1. CORS (origin: `http://localhost:3000`, credentials enabled)
2. JSON body parser (500kb limit)
3. Passport initialization (OAuth strategies)
4. Route mounting
5. Error handler

**Route mounting**:

| Base Path | Module | Description |
|-----------|--------|-------------|
| `/api/auth` | `oauth.routes` | OAuth login/callback/link |
| `/api/auth` | `auth.routes` | Local auth (register, login, MFA) |
| `/api/vault` | `vault.routes` | Vault unlock/lock/status |
| `/api/connections` | `connections.routes` | CRUD connections |
| `/api/folders` | `folders.routes` | CRUD folders |
| `/api/connections` | `sharing.routes` | Share/unshare connections |
| `/api/sessions` | `rdp.handler` | RDP/SSH session tokens |
| `/api/user` | `user.routes` | Profile, settings, avatar |
| `/api/user/2fa` | `twofa.routes` | TOTP setup/verify |
| `/api/user/2fa/sms` | `smsMfa.routes` | SMS MFA setup/verify |
| `/api/files` | `files.routes` | User drive file management |
| `/api/audit` | `audit.routes` | Audit log queries |
| `/api/notifications` | `notification.routes` | Notification management |
| `/api/tenants` | `tenant.routes` | Multi-tenant organization |
| `/api/teams` | `team.routes` | Team management |
| `/api/admin` | `admin.routes` | Admin email status/test |
| `/api/health` | (inline) | Health check (`{ status: 'ok' }`) |

<!-- manual-start -->
<!-- manual-end -->

## Client Architecture

**Tech stack**: React 19, Vite, Material-UI (MUI) v6, Zustand, Axios

### Component Structure

```
client/src/
├── pages/           # Route-level components (10 pages)
├── components/      # UI components grouped by feature
│   ├── Layout/      # MainLayout, NotificationBell
│   ├── Sidebar/     # ConnectionTree, TeamConnectionSection
│   ├── Tabs/        # TabBar, TabPanel
│   ├── Dialogs/     # ConnectionDialog, ShareDialog, etc.
│   ├── Terminal/    # SshTerminal (XTerm.js)
│   ├── RDP/         # RdpViewer (Guacamole), FileBrowser
│   ├── SSH/         # SftpBrowser, SftpTransferQueue
│   ├── Settings/    # Terminal, RDP, 2FA, SMS, OAuth, Email sections
│   ├── Overlays/    # VaultLockedOverlay
│   └── shared/      # FloatingToolbar
├── store/           # 12 Zustand stores
├── hooks/           # useAuth, useSocket, useSftpTransfers
└── api/             # 17 Axios API modules
```

### State Management

Zustand stores with selective localStorage persistence:
- `authStore` — tokens and user identity (`arsenale-auth`)
- `uiPreferencesStore` — panel states, sidebar, view modes (`arsenale-ui-preferences`)
- `themeStore` — dark/light mode (`arsenale-theme`)
- Other stores (connections, vault, tabs, etc.) are session-only

### API Layer

Centralized Axios client (`client/src/api/client.ts`):
- Base URL: `/api`
- Request interceptor: attaches JWT `Authorization: Bearer` header
- Response interceptor: automatic token refresh on 401, then retry

<!-- manual-start -->
<!-- manual-end -->

## Real-Time Connection Flows

### SSH Flow

```
┌──────────┐    Socket.IO /ssh     ┌──────────┐      SSH2       ┌──────────┐
│  Client   │◄────────────────────►│  Server   │◄──────────────►│  Remote  │
│ (XTerm.js)│   session:start      │ (Node.js) │                │  Host    │
│           │   data (bidir)       │           │                │          │
│           │   resize             │           │                │          │
│           │   sftp:* events      │           │                │          │
└──────────┘                       └──────────┘                 └──────────┘
```

1. Client opens SSH tab → connects to Socket.IO `/ssh` namespace with JWT
2. Emits `session:start` with `connectionId` (and optional credential overrides)
3. Server authenticates via JWT middleware, retrieves connection from DB
4. Server decrypts credentials from vault, creates SSH2 connection
5. Bidirectional data flows: `data` events (terminal I/O), `resize` events
6. SFTP operations via `sftp:*` events (list, mkdir, delete, rename, upload, download)

### RDP Flow

```
┌──────────┐    HTTP POST          ┌──────────┐                 ┌──────────┐
│  Client   │───────────────────►  │  Server   │                │  guacd   │
│(Guacamole │  /api/sessions/rdp   │ (Node.js) │                │ (4822)   │
│  Common)  │◄── { token }         │           │                │          │
│           │                      │           │                │          │
│           │    WebSocket :3002   │guacamole- │   Guacamole    │          │
│           │◄────────────────────►│  lite     │◄──────────────►│          │──► Remote
└──────────┘                       └──────────┘                 └──────────┘    Host
```

1. Client requests RDP session via `POST /api/sessions/rdp` with `connectionId`
2. Server decrypts credentials, merges RDP settings (user defaults + connection overrides)
3. Server generates encrypted Guacamole token (AES-256-CBC with `GUACAMOLE_SECRET`)
4. Client connects to Guacamole WebSocket on port 3002 with the token
5. `guacamole-lite` decrypts token, connects to `guacd` daemon
6. `guacd` establishes RDP connection to remote host

<!-- manual-start -->
<!-- manual-end -->

## Network Topology

### Development

```
Browser ──► :3000 (Vite dev server)
              ├── /api/* ──────────► :3001 (Express server)
              ├── /socket.io/* ────► :3001 (Socket.IO)
              └── /guacamole/* ────► :3002 (guacamole-lite)

Docker:
  guacd ──► :4822
  postgres ──► :5432
```

Vite proxies `/api`, `/socket.io`, and `/guacamole` to the server in development.

### Production

```
Browser ──► :3000 (nginx)
              ├── /api/* ──────────► server:3001 (Express)
              ├── /socket.io/* ────► server:3001 (Socket.IO)
              ├── /guacamole/* ────► server:3002 (guacamole-lite)
              └── /* ──────────────► static files (SPA fallback)

Docker internal network:
  postgres (no exposed port)
  guacd (no exposed port)
  server :3001, :3002
  client (nginx) :8080 → mapped to host :3000
```

### Ports

| Port | Service | Description |
|------|---------|-------------|
| 3000 | Client | Vite dev server / nginx (production) |
| 3001 | Server | Express HTTP + Socket.IO |
| 3002 | Server | Guacamole WebSocket (`guacamole-lite`) |
| 4822 | guacd | Guacamole daemon (RDP protocol) |
| 5432 | PostgreSQL | Database |

<!-- manual-start -->
<!-- manual-end -->

## Development vs Production

| Aspect | Development | Production |
|--------|-------------|------------|
| **Containers** | guacd + postgres only | postgres + guacd + server + client |
| **Server** | `tsx watch` (hot reload) on host | Node.js in Docker container |
| **Client** | Vite dev server on host | nginx serving static build |
| **Proxy** | Vite proxy config | nginx reverse proxy |
| **Database** | Exposed on :5432, default credentials | Internal network, env-based credentials |
| **guacd** | Exposed on :4822 | Internal network only |
| **Volumes** | `./data/drive` bind mount | Named volumes (`pgdata`, `arsenale_drive`) |
| **Migrations** | `npm run db:push` (schema sync) | `prisma migrate deploy` on container start |

<!-- manual-start -->
<!-- manual-end -->
