---
title: Troubleshooting
description: Common errors, debugging tips, and frequently asked questions
generated-by: ctdf-docs
generated-at: 2026-03-16T19:30:00Z
source-files:
  - server/src/index.ts
  - server/src/middleware/error.middleware.ts
  - server/src/middleware/auth.middleware.ts
  - server/src/services/vault.service.ts
  - server/src/services/session.service.ts
  - client/src/api/client.ts
  - client/src/hooks/useAutoReconnect.ts
  - compose.dev.yml
  - .env.example
---

# Troubleshooting

## Startup Issues

### Database Connection Failed

**Symptom**: Server crashes with `Can't reach database server at localhost:5432`

**Cause**: PostgreSQL container not running.

**Fix**:
```bash
npm run docker:dev          # Start PostgreSQL container
# Verify it's healthy:
docker ps | grep postgres   # Should show "healthy"
```

### Prisma Migration Error

**Symptom**: `Error: P3009: migrate found failed migrations`

**Cause**: A previous migration partially applied.

**Fix**:
```bash
# Reset and re-apply (development only)
npx prisma migrate reset --force -w server
npm run db:generate
```

### Port Already in Use

**Symptom**: `EADDRINUSE: address already in use :::3001`

**Cause**: Another process (or previous crashed server) occupies the port.

**Fix**:
```bash
# Find and kill the process
lsof -ti:3001 | xargs kill -9
# Or use a different port in .env
PORT=3005
```

### Prisma Client Not Generated

**Symptom**: `Cannot find module '@prisma/client'` or type errors in generated types

**Fix**:
```bash
npm run db:generate
```

This runs automatically during `npm run predev`, but may need manual execution after `git pull` with schema changes.

## Authentication Issues

### JWT Token Expired / 401 Errors

**Symptom**: API calls return 401 even though user is logged in.

**Cause**: Access token expired and auto-refresh failed.

**What happens**: The Axios interceptor in `client/src/api/client.ts` catches 401 responses and attempts `POST /api/auth/refresh`. If the refresh token is also expired (default 7d), the user is logged out.

**Fix**: Log in again. To adjust token lifetimes:
```bash
JWT_EXPIRES_IN=30m            # Access token (default: 15m)
JWT_REFRESH_EXPIRES_IN=30d    # Refresh token (default: 7d)
```

### Token Hijack Detection

**Symptom**: User gets logged out with "Token binding mismatch" in server logs.

**Cause**: The auth middleware (`server/src/middleware/auth.middleware.ts`) binds JWT tokens to the client's IP address and User-Agent hash. If either changes (VPN switch, browser update), the token is rejected.

**Fix**: Log in again from the new network/browser. This is a security feature.

### OAuth Callback Fails

**Symptom**: OAuth login redirects back to login page without completing.

**Cause**: `CLIENT_URL` in `.env` doesn't match the actual browser URL.

**Fix**: Ensure `CLIENT_URL` matches exactly (including protocol and port):
```bash
CLIENT_URL=http://localhost:3000   # Development
CLIENT_URL=https://arsenale.example.com  # Production
```

### CORS Errors

**Symptom**: Browser console shows `Access-Control-Allow-Origin` errors.

**Cause**: The Express CORS middleware (`server/src/app.ts`) only allows the origin specified in `CLIENT_URL`.

**Fix**: Set `CLIENT_URL` to match the exact origin the browser uses.

## Vault Issues

### Vault Locked After Inactivity

**Symptom**: Credentials show as encrypted, operations require vault unlock.

**Cause**: Vault master key TTL expired (default 30 minutes). The server holds the decrypted master key in memory with a configurable TTL.

**Fix**: Unlock vault again (password or MFA). Adjust timeout:
```bash
VAULT_TTL_MINUTES=60          # Increase TTL
```

Tenant admins can also set a maximum auto-lock value that overrides per-user preferences.

### Cannot Decrypt Credentials

**Symptom**: "Decryption failed" errors when accessing connections.

**Cause**: Vault key corruption or server encryption key mismatch.

**Check**:
1. Verify `SERVER_ENCRYPTION_KEY` in `.env` hasn't changed since credentials were stored
2. In development, if `SERVER_ENCRYPTION_KEY` is absent, a deterministic key is auto-generated — this works only if `DATABASE_URL` hasn't changed

## Connection Issues

### SSH Connection Timeout

**Symptom**: SSH terminal shows "Connection timeout" or stays blank.

**Possible causes**:
1. Target host unreachable from server
2. `ALLOW_LOCAL_NETWORK=false` blocks private IP ranges
3. SSH gateway configuration incorrect

**Debug**:
```bash
# Test from server container
docker exec -it arsenale-server-1 ping <target-host>
# Check if local network access is enabled
grep ALLOW_LOCAL_NETWORK .env
```

### RDP/VNC Black Screen

**Symptom**: RDP or VNC viewer shows black screen or disconnects immediately.

**Possible causes**:
1. guacd container not running or unhealthy
2. `GUACD_HOST` / `GUACD_PORT` misconfigured
3. `GUACAMOLE_SECRET` mismatch between server and guacamole-lite

**Debug**:
```bash
# Check guacd health
docker ps | grep guacd
# Test readiness probe
curl http://localhost:3001/api/ready
# Check guacamole WebSocket port
curl -I http://localhost:3002
```

### Session Limit Reached

**Symptom**: "Maximum concurrent sessions reached" error.

**Cause**: User exceeded `MAX_CONCURRENT_SESSIONS` (default: 10).

**Fix**: Close existing sessions or increase the limit:
```bash
MAX_CONCURRENT_SESSIONS=20
```

## Build & CI Issues

### TypeScript Type Errors After Schema Change

**Symptom**: `npm run typecheck` fails with Prisma-related type errors.

**Fix**: Regenerate the Prisma client:
```bash
npm run db:generate
```

### ESLint Security Warnings

**Symptom**: `npm run lint` reports `eslint-plugin-security` warnings.

**Context**: The security plugin flags potential vulnerabilities (e.g., `security/detect-object-injection`). Object injection detection is disabled globally in `eslint.config.mjs` because bracket notation is used extensively with validated inputs.

### npm Audit Failures

**Symptom**: `npm run sast` fails with critical vulnerabilities.

**Fix**: Update affected packages:
```bash
npm audit fix
# If automated fix not available, check if the vulnerability
# is in a dev dependency or affects your use case
```

## Docker Issues

### Container Runtime Detection

**Symptom**: `npm run docker:dev` fails with "Neither docker nor podman found".

**Cause**: The `scripts/container-runtime.sh` script auto-detects Docker or Podman.

**Fix**: Install Docker or Podman, or set the runtime manually:
```bash
# Check which is available
docker --version
podman --version
```

### Volume Permission Issues

**Symptom**: Server container fails to write to `/recordings` or `/guacd-drive`.

**Cause**: Container runs as non-root `appuser` but volumes are owned by root.

**Fix**: Ensure volume directories have correct permissions:
```bash
# For Docker
sudo chown -R 1000:1000 ./recordings ./drive
# For rootless Podman (maps to host user automatically)
# No action needed
```

### Database Data Loss on Restart

**Symptom**: Data disappears after `docker compose down`.

**Cause**: Using `-v` flag removes named volumes.

**Fix**: Never use `docker compose down -v` unless you intend to reset. Use `docker compose down` (without `-v`) to preserve data.

## Performance Issues

### Slow API Responses

**Debug**:
1. Enable HTTP request logging:
   ```bash
   LOG_HTTP_REQUESTS=true
   LOG_LEVEL=debug
   ```
2. Check database query performance in Prisma logs
3. Verify PostgreSQL container has sufficient resources

### Socket.IO Connection Drops

**Symptom**: SSH terminal disconnects intermittently.

**Cause**: Proxy timeout or WebSocket upgrade failure.

**Context**: The client uses `useAutoReconnect` hook with exponential backoff (base 1s, max 15s, 5 retries, 60s total timeout).

**Fix**:
- If behind a reverse proxy, ensure WebSocket upgrade is configured
- Check `ABSOLUTE_SESSION_TIMEOUT_SECONDS` (default: 43200 = 12h)

## FAQ

### Where is the `.env` file?

At the **monorepo root** (`/arsenale/.env`), not inside `server/`. The Prisma config at `server/prisma.config.ts` resolves it to `../.env`.

### How do I reset the database?

```bash
npx prisma migrate reset --force -w server
npm run db:generate
```

### How do I add a new API endpoint?

1. Create or edit `server/src/routes/<domain>.routes.ts`
2. Create or edit `server/src/controllers/<domain>.controller.ts`
3. Create or edit `server/src/services/<domain>.service.ts`
4. Mount the route in `server/src/app.ts`
5. Run `npm run verify` to ensure everything passes

### How do I add a new dialog?

Follow the full-screen dialog pattern (see [Development](development.md#full-screen-dialog-pattern)):
1. Create component with `Dialog fullScreen` + `SlideUp` transition
2. Add state in `MainLayout` (`const [dialogOpen, setDialogOpen] = useState(false)`)
3. Render at fragment root level in `MainLayout`, outside blur wrapper
4. Never create a new page route for overlay UI

### How do I add a new Zustand store?

1. Create `client/src/store/<name>Store.ts`
2. Follow existing pattern: `create<StateType>()(...)` with typed actions
3. Use `persist` middleware if state should survive page reloads
4. For UI preferences, add to existing `uiPreferencesStore` instead

### What's the `npm run verify` pipeline?

typecheck → lint → audit → test → build. All must pass before closing any task.
