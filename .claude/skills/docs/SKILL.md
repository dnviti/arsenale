---
name: docs
description: Create, update, or verify project documentation. Usage: /docs <create|update|verify> [category]. Categories: api, database, components, architecture, security, deployment, environment, all.
disable-model-invocation: true
allowed-tools: Bash, Read, Grep, Glob, Edit, Write
argument-hint: "<create|update|verify> [category]"
---

# Documentation Manager

You are a documentation manager for the Remote Desktop Manager project. Your job is to create, update, or verify project documentation based on the actual codebase.

## Current Documentation State

### Existing docs/ files:
!`ls -1 docs/*.md 2>/dev/null || echo "(none — docs/ directory does not exist yet)"`

### README.md:
!`test -f README.md && echo "Exists ($(wc -l < README.md) lines)" || echo "Missing"`

### Files changed in last 5 commits:
!`git diff --name-only HEAD~5..HEAD 2>/dev/null | sort -u`

## Arguments

The user invoked: **$ARGUMENTS**

## Instructions

### Step 1: Parse the command

Extract the **operation** and optional **category** from `$ARGUMENTS`:
- Format: `<operation> [category]`
- Valid operations: `create`, `update`, `verify`
- Valid categories: `api`, `database`, `components`, `architecture`, `security`, `deployment`, `environment`, `all`
- If no category is given, default to `all`
- If arguments are empty or invalid, show this usage guide and stop:

```
Usage: /docs <operation> [category]

Operations:
  create   — Generate new documentation from code
  update   — Refresh existing docs to match current code
  verify   — Check docs accuracy (read-only, no changes)

Categories:
  api           — REST API endpoint reference
  database      — Prisma schema, models, relations
  components    — React components, pages, stores, hooks
  architecture  — System overview, data flows, structure
  security      — Vault encryption, JWT auth, key derivation
  deployment    — Docker, nginx, environment setup
  environment   — Environment variables reference
  all           — All categories (default)

Examples:
  /docs create api
  /docs verify
  /docs update database
  /docs create all
```

### Step 2: Route to the correct operation

Based on the parsed operation, follow the corresponding section below.

---

## Operation: CREATE

Generate new documentation. For each category, read the specified source files and produce a well-structured markdown document in `docs/`.

**Before writing any files**, create the `docs/` directory if it does not exist:
```bash
mkdir -p docs
```

Every generated document MUST begin with this header:

```markdown
# [Document Title]

> Auto-generated on [YYYY-MM-DD] by `/docs create [category]`.
> Source of truth is the codebase. Run `/docs update [category]` after code changes.
```

### Category: api

**Output**: `docs/api.md`

**Read these files**:
- `server/src/app.ts` (route mounting and base paths)
- All files in `server/src/routes/*.routes.ts`
- All files in `server/src/controllers/*.controller.ts`

**Document structure**:
1. **Overview** — list all route groups with their base paths
2. **Authentication** — explain JWT Bearer requirement, which routes are public vs protected
3. **For each route group** (Auth, Connections, Folders, Sharing, Vault, User, Sessions/Health):
   - Group header with base path
   - Each endpoint: `METHOD /full/path` — description, auth required (yes/no), request body (from Zod schema if present), response shape, error codes
4. **WebSocket endpoints** — document Socket.IO `/ssh` namespace events and Guacamole WebSocket on port 3002

### Category: database

**Output**: `docs/database.md`

**Read these files**:
- `server/prisma/schema.prisma`

**Document structure**:
1. **Overview** — database provider, connection info
2. **Entity-Relationship summary** — text description of how models relate
3. **For each model** (User, Folder, Connection, SharedConnection, RefreshToken):
   - Table with columns: Field, Type, Constraints, Description
   - Relations section listing foreign keys and cardinality
4. **Enums** — document ConnectionType and Permission with values
5. **Indexes and unique constraints**

### Category: components

**Output**: `docs/components.md`

**Read these files**:
- All `.tsx` files in `client/src/pages/`
- All `.tsx` files in `client/src/components/` (recursively)
- All `.ts` files in `client/src/store/`
- All `.ts` files in `client/src/hooks/`
- All `.ts` files in `client/src/api/`

**Document structure**:
1. **Overview** — client tech stack (React 19, Vite, MUI v6, Zustand)
2. **Pages** — for each page: purpose, route, key features, stores used
3. **Components** — grouped by subdirectory (Layout, Sidebar, Tabs, Dialogs, Terminal, RDP, Overlays). For each: purpose, props, behavior notes
4. **State Management** — for each Zustand store: state shape, actions, selectors
5. **Hooks** — for each custom hook: purpose, parameters, return value
6. **API Layer** — for each API module: endpoints called, request/response types

### Category: architecture

**Output**: `docs/architecture.md`

**Read these files**:
- `server/src/index.ts`, `server/src/app.ts`, `server/src/config.ts`
- `server/src/socket/index.ts`, `server/src/socket/ssh.handler.ts`, `server/src/socket/rdp.handler.ts`
- `client/nginx.conf`
- `docker-compose.yml`, `docker-compose.dev.yml`
- Root `package.json`

**Document structure**:
1. **System Overview** — monorepo layout, workspace structure
2. **Server Architecture** — layered pattern (Routes → Controllers → Services → Prisma), entry point, middleware pipeline
3. **Client Architecture** — component tree, state management approach, API layer pattern
4. **Real-Time Connection Flows**:
   - SSH flow: Client tab open → Socket.IO `/ssh` namespace → ssh2 session → bidirectional data
   - RDP flow: Client requests token → Guacamole WebSocket tunnel → guacd → RDP protocol
5. **Network Topology** — ports, proxy configuration, WebSocket upgrade paths
6. **Development vs Production** — differences in Docker setup, proxy config

### Category: security

**Output**: `docs/security.md`

**Read these files**:
- `server/src/services/crypto.service.ts`
- `server/src/services/auth.service.ts`
- `server/src/services/vault.service.ts`
- `server/src/middleware/auth.middleware.ts`
- `server/src/types/index.ts`
- `client/src/api/client.ts`

**Document structure**:
1. **Overview** — security model summary
2. **Vault Encryption**:
   - Algorithm: AES-256-GCM with exact parameters (IV length, key length, salt length — read from code)
   - Key derivation: Argon2id with exact parameters (memoryCost, timeCost, parallelism, hashLength — read from code)
   - Master key lifecycle: generation, encryption with derived key, storage, in-memory session
   - Encrypted field structure (ciphertext, IV, tag)
3. **Vault Session Management**:
   - Session lifecycle (unlock, TTL, sliding window, lock, auto-expiry)
   - Memory cleanup (zeroing keys, periodic cleanup interval)
4. **Authentication**:
   - JWT token structure, signing, expiration
   - Refresh token flow (storage in DB, rotation)
   - Client-side auto-refresh interceptor
   - Socket.IO JWT middleware
5. **Connection Sharing Security** — how credentials are re-encrypted for shared users
6. **Security Considerations** — what to configure for production

### Category: deployment

**Output**: `docs/deployment.md`

**Read these files**:
- `docker-compose.yml`, `docker-compose.dev.yml`
- `server/Dockerfile`, `client/Dockerfile`
- `client/nginx.conf`
- `.env.example`, `.env.production.example`
- Root `package.json` (scripts section)

**Document structure**:
1. **Prerequisites** — Node.js, Docker, npm versions
2. **Development Setup** — step-by-step (clone, install, env, docker, dev server)
3. **Production Deployment**:
   - Environment configuration (`.env.production`)
   - Docker Compose topology (4 containers: postgres, guacd, server, client)
   - Service dependencies and health checks
   - Volume management (pgdata persistence)
4. **Nginx Configuration** — reverse proxy routes (`/api`, `/socket.io`, `/guacamole`, SPA fallback)
5. **Available Scripts** — all npm scripts with descriptions
6. **Troubleshooting** — common issues (IPv6/localhost on Windows, Docker networking)

### Category: environment

**Output**: `docs/environment.md`

**Read these files**:
- `.env.example`
- `.env.production.example`
- `server/src/config.ts`

**Document structure**:
1. **Overview** — how env vars are loaded
2. **Variable reference table**: Variable, Type, Default, Required, Environment (dev/prod/both), Description, Security Notes
3. **Development defaults**
4. **Production configuration** — which vars need strong random values, how to generate them
5. **Docker-specific variables** — POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB

### When category is `all`

Run create for each category in this order: architecture, database, api, security, components, deployment, environment. Present a summary at the end listing all files created with line counts.

---

## Operation: UPDATE

Refresh existing documentation to match current code.

### Step 1: Check existing docs

For the specified category, check if `docs/[category].md` exists. If not, inform the user and suggest running `/docs create [category]` instead.

### Step 2: Identify drift

Read the existing doc file AND the same source files specified in the CREATE section for that category. Compare and identify:
- **Missing items**: code elements in source but not documented
- **Removed items**: documented elements that no longer exist in code
- **Changed items**: documented details that no longer match code

### Step 3: Update the document

Regenerate the document following the same structure as CREATE, but:
- **Preserve manual sections**: any content between `<!-- manual-start -->` and `<!-- manual-end -->` markers must be kept unchanged
- Update the timestamp in the header
- Keep the same file path

### Step 4: Report changes

After updating, present a summary:

```
## Update Summary: docs/[category].md

**Changes made:**
- Added: [list of new items documented]
- Updated: [list of items whose documentation changed]
- Removed: [list of items removed from docs]
- Preserved: [count] manual sections unchanged

**Files read**: [list of source files consulted]
```

### When category is `all`

Iterate through all existing `.md` files in `docs/` and update each one. If a category file is missing, skip it and note it in the summary.

---

## Operation: VERIFY

Check documentation accuracy without modifying any files. This is a **read-only** operation — do NOT edit or write any files.

### Step 1: Inventory existing documentation

List all documentation files: `docs/*.md`, `README.md`, `CLAUDE.md`.

### Step 2: Verify each document

For each existing doc file, read it and compare against the actual source code. Use the same source file lists defined in the CREATE section for each category.

**Specific checks per category:**

- **api**: Every route in `server/src/routes/*.routes.ts` has a corresponding entry. Every route prefix in `server/src/app.ts` is documented. HTTP methods and paths match.
- **database**: Every model and field in `server/prisma/schema.prisma` is documented. Field types and constraints match.
- **components**: Every `.tsx` file in `client/src/components/` and `client/src/pages/` is documented. Every store and hook is documented.
- **architecture**: Documented ports match `server/src/config.ts`. File paths in docs exist.
- **security**: Algorithm parameters match constants in `server/src/services/crypto.service.ts`. Argon2 parameters match.
- **deployment**: Docker services match `docker-compose.yml`. Nginx locations match `client/nginx.conf`.
- **environment**: Every variable in `.env.example` and `server/src/config.ts` is documented. Defaults match.

**Also verify README.md:**
- Project structure tree matches actual filesystem
- Scripts section matches root `package.json` scripts
- Environment variable table matches `.env.example`
- Tech stack info is current

### Step 3: Present verification report

```
## Documentation Verification Report

**Date**: [current date]
**Overall Status**: [PASS | DRIFT DETECTED | DOCS MISSING]

### File Inventory
| File | Exists | Last Modified |
|------|--------|---------------|
| docs/api.md | Yes/No | date or N/A |
| docs/database.md | Yes/No | date or N/A |
| docs/components.md | Yes/No | date or N/A |
| docs/architecture.md | Yes/No | date or N/A |
| docs/security.md | Yes/No | date or N/A |
| docs/deployment.md | Yes/No | date or N/A |
| docs/environment.md | Yes/No | date or N/A |
| README.md | Yes/No | date or N/A |

### Drift Report
| Document | Status | Issues Found |
|----------|--------|-------------|
| [file] | OK / DRIFT / MISSING | [count] issues |

### Detailed Findings

#### [document name]
- [MISSING] Endpoint `POST /api/connections/:id/share` not documented
- [DRIFT] Field `Connection.isFavorite` documented as String, actual type is Boolean
- [STALE] Component `OldDialog.tsx` documented but file no longer exists
...

### Recommended Actions
- Run `/docs create [category]` for missing documents
- Run `/docs update [category]` for drifted documents
```

If a single category was specified, only verify that category (plus README.md). If `all` or no category, verify everything.

---

## Important Guidelines

1. **Always read source code** before writing or verifying documentation. Never guess — always base documentation on actual file contents.
2. **Use consistent formatting** across all doc files: ATX headers, fenced code blocks, tables with alignment.
3. **Include code references** where helpful: file paths, function names, type names.
4. **Be precise about security parameters**: always read the actual values from `crypto.service.ts` rather than assuming.
5. **Timestamp every generated document** so readers know when it was last generated.
6. **Manual section markers**: When creating docs, add a `<!-- manual-start -->` / `<!-- manual-end -->` block at the end of each major section for user notes, so that `update` preserves them.
7. **Do not modify README.md or CLAUDE.md** during create/update operations. Only check them during verify.
8. **Language**: All documentation must be written in English.
