# Components

> Auto-generated on 2026-03-14 by `/docs create components`.
> Source of truth is the codebase. Run `/docs update components` after code changes.

## Overview

The client is built with:

- **React 19** with TypeScript
- **Vite** — build tool and dev server
- **Material-UI (MUI) v6** — component library and theming
- **Zustand** — state management (14 stores with localStorage persistence for UI preferences)
- **Axios** — HTTP client with JWT auto-refresh
- **Socket.IO Client** — real-time SSH terminals, notifications, gateway monitoring
- **XTerm.js** — SSH terminal emulation
- **guacamole-common-js** — RDP/VNC remote desktop rendering

**Total**: 10 pages, 73 components, 14 stores, 4 hooks, 25 API modules.

<!-- manual-start -->
<!-- manual-end -->

## Pages

| Page | Route | Purpose | Key Stores |
|------|-------|---------|------------|
| `LoginPage` | `/login` | Multi-step login: email/password, MFA challenge (TOTP/SMS/WebAuthn), forced MFA setup, tenant selection | authStore, vaultStore |
| `RegisterPage` | `/register` | User registration with email verification and recovery key display | authStore |
| `DashboardPage` | `/` | Main app shell — fetches connections, restores tabs, renders MainLayout | connectionsStore, tabsStore |
| `ConnectionViewerPage` | `/viewer/:id` | Standalone popup window for a single connection (SSH/RDP/VNC) with auth bootstrap | tabsStore, authStore |
| `RecordingPlayerPage` | `/recordings/:id` | Standalone popup player for session recordings (asciicast or .guac) | authStore |
| `PublicSharePage` | `/share/:token` | Unauthenticated page for externally shared secrets (optional PIN) | — |
| `OAuthCallbackPage` | `/oauth/callback` | Handles OAuth redirects, extracts tokens, redirects to dashboard or vault setup | authStore |
| `VaultSetupPage` | `/vault-setup` | Post-OAuth vault password setup for OAuth-only users | authStore |
| `ForgotPasswordPage` | `/forgot-password` | Password reset email request form | — |
| `ResetPasswordPage` | `/reset-password` | Multi-step password reset (token validation, optional SMS, new password) | — |

<!-- manual-start -->
<!-- manual-end -->

## Components

### Layout (`client/src/components/Layout/`)

| Component | Purpose |
|-----------|---------|
| `MainLayout` | Top-level layout: sidebar, tab bar, connection viewers, and all full-screen dialog mount points. Manages open/close state for all dialogs. |
| `TenantSwitcher` | Sidebar dropdown for switching between tenant organizations |
| `NotificationBell` | AppBar bell icon with unread badge and notification dropdown list |

### Sidebar (`client/src/components/Sidebar/`)

| Component | Purpose |
|-----------|---------|
| `ConnectionTree` | Main sidebar — connection tree with folders, favorites section, recents, shared connections, search, drag-and-drop, and context menus |
| `TeamConnectionSection` | Sidebar section showing team connections grouped by team with folder support |
| `treeHelpers` | Helper functions for building tree node structures from flat connection/folder data |

### Tabs (`client/src/components/Tabs/`)

| Component | Purpose |
|-----------|---------|
| `TabBar` | Horizontal tab bar with close buttons, active indicator, pop-out action, and context menu |
| `TabPanel` | Content panel that renders the appropriate viewer (SSH terminal, RDP, VNC) for the active tab |

### Terminal / SSH (`client/src/components/Terminal/`, `client/src/components/SSH/`)

| Component | Purpose |
|-----------|---------|
| `SshTerminal` | XTerm.js-based SSH terminal with Socket.IO connection, resize handling, search addon, and SFTP browser integration |
| `SftpBrowser` | In-session SFTP file browser panel (navigate, upload, download, delete, rename, mkdir) |
| `SftpTransferQueue` | SFTP transfer progress queue showing active/completed/failed file transfers |

### RDP (`client/src/components/RDP/`)

| Component | Purpose |
|-----------|---------|
| `RdpViewer` | Guacamole-based RDP viewer with clipboard sync, dynamic scaling, toolbar, and drive redirection |
| `FileBrowser` | In-session RDP file browser for the virtual drive (upload, download, delete, create folder) |

### VNC (`client/src/components/VNC/`)

| Component | Purpose |
|-----------|---------|
| `VncViewer` | Guacamole-based VNC viewer with clipboard sync, scaling, and toolbar |

### Dialogs (`client/src/components/Dialogs/`)

All full-screen dialogs use the MUI `Dialog` component with `fullScreen` prop and `Slide` transition, rendered from `MainLayout` to preserve active sessions.

| Component | Purpose |
|-----------|---------|
| `SettingsDialog` | Full-screen settings with tabbed sections (profile, security, terminal, RDP, VNC, gateway, tenant, teams, audit) |
| `AuditLogDialog` | Full-screen personal audit log with filtering, pagination, and geo-location |
| `ConnectionAuditLogDialog` | Full-screen audit log scoped to a specific connection |
| `KeychainDialog` | Full-screen secrets/keychain manager (list, create, edit, share, external share) |
| `ConnectionDialog` | Create/edit connection dialog (SSH, RDP, VNC) with host, port, credentials, gateway selection, and per-connection settings |
| `FolderDialog` | Create/rename folder dialog |
| `ShareDialog` | Manage connection sharing (add/remove users, change permissions) |
| `ShareFolderDialog` | Batch-share all connections in a folder |
| `ImportDialog` | Import connections from CSV/JSON/mRemoteNG/RDP files |
| `ExportDialog` | Export connections to CSV or JSON (with optional credentials) |
| `ConnectAsDialog` | Choose credential mode (saved, domain, manual) before opening a connection |
| `CreateUserDialog` | Tenant admin: create a new user with email, password, and role |
| `UserProfileDialog` | View tenant user's profile, teams, and admin actions (change email/password, MFA status) |
| `InviteDialog` | Tenant admin: invite a user by email with a role |
| `TeamDialog` | Create or edit a team (name, description, members, roles) |

### Keychain (`client/src/components/Keychain/`)

| Component | Purpose |
|-----------|---------|
| `SecretListPanel` | Left panel — filterable, sortable list of secrets with scope/type badges |
| `SecretDetailView` | Right panel — full secret data, metadata, tags, shares, and external shares |
| `SecretDialog` | Create/edit secret (Login, SSH Key, Certificate, API Key, Secure Note) |
| `SecretVersionHistory` | Version history with diff viewing and restore capability |
| `SecretPicker` | Autocomplete picker to select a keychain secret (used in ConnectionDialog) |
| `ShareSecretDialog` | Share a secret with another user (internal sharing with permissions) |
| `ExternalShareDialog` | Create external share link (expiry, max accesses, optional PIN) |

### Settings (`client/src/components/Settings/`)

| Component | Purpose |
|-----------|---------|
| `ProfileSection` | Username, email, avatar upload |
| `ChangePasswordSection` | Password change with identity verification |
| `TwoFactorSection` | TOTP 2FA setup/disable with QR code |
| `SmsMfaSection` | SMS MFA setup — phone, verification, enable/disable |
| `WebAuthnSection` | WebAuthn/passkey management — register, rename, remove |
| `LinkedAccountsSection` | OAuth linked accounts — link/unlink providers |
| `TerminalSettingsSection` | SSH terminal defaults (theme, font, cursor) |
| `RdpSettingsSection` | RDP defaults (color depth, resize, clipboard, audio, etc.) |
| `VncSettingsSection` | VNC defaults (color depth, cursor, resize, clipboard) |
| `ConnectionDefaultsSection` | Default credential mode setting |
| `VaultAutoLockSection` | Vault auto-lock timer (with tenant maximum enforcement) |
| `DomainProfileSection` | Windows/AD domain profile (domain, username, password) |
| `TenantSection` | Tenant management — name, MFA policy, session timeout, user management |
| `TenantAuditLogSection` | Tenant-wide audit log with user filter, geo map, table/timeline views |
| `TeamSection` | Team management — CRUD teams, manage members and roles |
| `GatewaySection` | Gateway management — CRUD, SSH keys, health tests, orchestration tabs |
| `EmailProviderSection` | Email provider status and test-send (admin) |
| `SelfSignupSection` | Toggle self-signup on/off (admin, respects env-lock) |

### Gateway / Orchestration (`client/src/components/gateway/`, `client/src/components/orchestration/`)

| Component | Purpose |
|-----------|---------|
| `GatewayDialog` | Create/edit gateway (GUACD, SSH Bastion, Managed SSH) with connection test |
| `GatewayTemplateDialog` | Create/edit gateway template with auto-scaling and LB defaults |
| `GatewayTemplateSection` | Gateway templates list with create/edit/delete/deploy actions |
| `OrchestrationSection` | Settings section wrapper for orchestration dashboard |
| `SessionDashboard` | Active sessions with filtering, counts per gateway, terminate actions |
| `GatewayInstanceList` | Managed container instances with status, health, restart, log viewing |
| `ScalingControls` | Auto-scaling configuration (enable/disable, min/max, sessions-per-instance) |
| `ContainerLogDialog` | Container logs for a managed gateway instance |
| `SessionTimeoutConfig` | Gateway inactivity session timeout configuration |

### Recording (`client/src/components/Recording/`)

| Component | Purpose |
|-----------|---------|
| `RecordingsDialog` | Full-screen dialog listing session recordings with filter, delete, and playback |
| `RecordingPlayerDialog` | Opens recording player in a popup window |
| `GuacPlayer` | Guacamole session recording player (RDP/VNC replay with playback controls) |
| `SshPlayer` | SSH terminal recording player (asciinema-style with speed/seek) |

### Audit (`client/src/components/Audit/`)

| Component | Purpose |
|-----------|---------|
| `IpGeoCell` | Table cell with IP address, country flag, and geo info tooltip |
| `GeoIpDialog` | Detailed geo-IP location dialog for an audit entry |
| `AuditGeoMap` | Interactive map visualization of audit log geo-locations |

### Overlays (`client/src/components/Overlays/`)

| Component | Purpose |
|-----------|---------|
| `VaultLockedOverlay` | Overlay when vault is locked — password unlock, MFA unlock options (TOTP, SMS, WebAuthn) |

### Shared (`client/src/components/shared/`, `client/src/components/common/`)

| Component | Purpose |
|-----------|---------|
| `FloatingToolbar` | Floating action toolbar over active RDP/VNC sessions (fullscreen, clipboard, screenshot) |
| `IdentityVerification` | Reusable identity verification flow (email OTP, TOTP, SMS, WebAuthn, password) for sensitive operations |

### Root-Level Components

| Component | Purpose |
|-----------|---------|
| `OAuthButtons` | Row of OAuth login/link buttons based on server-provided provider config |
| `UserPicker` | Autocomplete user search for share/invite dialogs |

<!-- manual-start -->
<!-- manual-end -->

## State Management

### `authStore` (`client/src/store/authStore.ts`)

| Field | Type | Description |
|-------|------|-------------|
| `accessToken` | string \| null | JWT access token |
| `csrfToken` | string \| null | CSRF token for auth endpoints |
| `user` | object \| null | User identity (id, email, username, avatarData, tenantId, tenantRole, domainName) |
| `isAuthenticated` | boolean | Authentication status |

**Actions**: `setAuth`, `setAccessToken`, `setCsrfToken`, `updateUser`, `fetchDomainProfile`, `logout`

### `connectionsStore` (`client/src/store/connectionsStore.ts`)

| Field | Type | Description |
|-------|------|-------------|
| `ownConnections` | Connection[] | User's own connections |
| `sharedConnections` | Connection[] | Connections shared with user |
| `teamConnections` | Connection[] | Team connections |
| `folders` | Folder[] | User's folders |
| `teamFolders` | Folder[] | Team folders |
| `loading` | boolean | Loading state |

**Actions**: `fetchConnections`, `fetchFolders`, `toggleFavorite`, `moveConnection`, `reset`

### `tabsStore` (`client/src/store/tabsStore.ts`)

| Field | Type | Description |
|-------|------|-------------|
| `tabs` | Tab[] | Open tabs (id, connection, active, credentials) |
| `activeTabId` | string \| null | Currently active tab |
| `recentTick` | number | Change counter for re-render triggers |

**Actions**: `openTab`, `closeTab`, `setActiveTab`, `restoreTabs`, `clearAll`. Auto-syncs to server with debounce.

### `vaultStore` (`client/src/store/vaultStore.ts`)

| Field | Type | Description |
|-------|------|-------------|
| `unlocked` | boolean | Vault unlock status |
| `initialized` | boolean | Whether initial status check completed |
| `mfaUnlockAvailable` | boolean | Whether MFA re-unlock is possible |
| `mfaUnlockMethods` | string[] | Available MFA methods for re-unlock |

**Actions**: `checkStatus`, `setUnlocked`, `startPolling`, `stopPolling`

### `uiPreferencesStore` (`client/src/store/uiPreferencesStore.ts`)

Persisted to localStorage via Zustand `persist` middleware (key: `arsenale-ui-preferences`). Namespaced by userId.

Key preferences: `rdpFileBrowserOpen`, `sshSftpBrowserOpen`, `sshSftpTransferQueueOpen`, `sidebarFavoritesOpen`, `sidebarRecentsOpen`, `sidebarSharedOpen`, `sidebarCompact`, `sidebarTeamSections`, `settingsActiveTab`, `keychainScopeFilter`, `keychainTypeFilter`, `keychainSortBy`, `orchestrationDashboardTab`, `orchestrationAutoRefresh`, `auditLog*`, `tenantAuditLog*`, `connAuditLog*`, `lastActiveTenantId`.

**Actions**: `set`, `toggle`, `toggleTeamSection`

### `tenantStore` (`client/src/store/tenantStore.ts`)

| Field | Type | Description |
|-------|------|-------------|
| `tenant` | Tenant \| null | Current tenant details |
| `users` | User[] | Tenant user list |
| `memberships` | Membership[] | User's tenant memberships |
| `loading`, `usersLoading` | boolean | Loading states |

**Actions**: `fetchTenant`, `fetchMemberships`, `switchTenant`, `createTenant`, `updateTenant`, `deleteTenant`, `fetchUsers`, `inviteUser`, `updateUserRole`, `removeUser`, `createUser`, `toggleUserEnabled`, `reset`

### `gatewayStore` (`client/src/store/gatewayStore.ts`)

| Field | Type | Description |
|-------|------|-------------|
| `gateways` | Gateway[] | Tenant gateways |
| `sshKeyPair` | KeyPair \| null | Tenant SSH key pair |
| `activeSessions` | Session[] | Active sessions |
| `sessionCount` | number | Total session count |
| `sessionCountByGateway` | object[] | Sessions per gateway |
| `scalingStatus` | object | Scaling status per gateway |
| `instances` | object | Instances per gateway |
| `templates` | Template[] | Gateway templates |

**Actions**: CRUD for gateways, SSH key pair management, session monitoring, orchestration (deploy, undeploy, scale, instances, scaling config, restart), templates, real-time updates (health, instances, scaling, gateway)

### `teamStore` (`client/src/store/teamStore.ts`)

**State**: `teams`, `selectedTeam`, `members`, loading flags.
**Actions**: CRUD for teams, member management, `reset`.

### `secretStore` (`client/src/store/secretStore.ts`)

**State**: `secrets`, `selectedSecret`, `filters`, `tenantVaultStatus`, `expiringCount`, loading/error.
**Actions**: CRUD for secrets, favorites, filters, tenant vault initialization, expiring count.

### `themeStore` (`client/src/store/themeStore.ts`)

**State**: `mode` ('light' | 'dark'). **Actions**: `toggle`.

### `rdpSettingsStore` / `terminalSettingsStore`

**State**: `userDefaults`, `loaded`, `loading`. **Actions**: `fetchDefaults`, `updateDefaults`.

### `notificationStore` (`client/src/store/notificationStore.ts`)

Ephemeral toast notifications. **State**: `notification` ({message, severity}). **Actions**: `notify`, `clear`.

### `notificationListStore` (`client/src/store/notificationListStore.ts`)

Server-persisted notifications. **State**: `notifications`, `unreadCount`, `total`, `loading`. **Actions**: `fetchNotifications`, `markAsRead`, `markAllAsRead`, `removeNotification`, `addNotification`, `reset`.

<!-- manual-start -->
<!-- manual-end -->

## Hooks

### `useAuth` (`client/src/hooks/useAuth.ts`)

Bootstraps authentication on mount. Refreshes access token from cookie if authenticated but token is missing. Redirects to login on failure.

**Returns**: `{ isAuthenticated: boolean, loading: boolean }`

### `useSocket` (`client/src/hooks/useSocket.ts`)

Creates and manages a Socket.IO connection to a given namespace with JWT auth.

**Parameters**: `namespace: string`, `options?: object`
**Returns**: `MutableRefObject<Socket | null>`

### `useSftpTransfers` (`client/src/hooks/useSftpTransfers.ts`)

Manages SFTP file transfer state — tracks uploads/downloads, progress events, chunked upload/download, cancel, clear.

**Returns**: `{ transfers: TransferItem[], uploadFile, downloadFile, cancelTransfer, clearCompleted }`

### `useGatewayMonitor` (`client/src/hooks/useGatewayMonitor.ts`)

Connects to `/gateway-monitor` Socket.IO namespace and applies real-time health, instance, scaling, and gateway update events to the gateway store.

**Returns**: void (side-effect hook)

<!-- manual-start -->
<!-- manual-end -->

## API Layer

25 API modules in `client/src/api/` provide typed Axios wrappers:

| Module | Description |
|--------|-------------|
| `client.ts` | Axios instance with JWT interceptor and auto-refresh |
| `auth.api.ts` | Login, register, MFA flows, refresh, logout, public config |
| `connections.api.ts` | Connection CRUD, favorites |
| `folders.api.ts` | Folder CRUD |
| `sharing.api.ts` | Connection sharing + RDP/SSH session creation |
| `vault.api.ts` | Vault lock/unlock, MFA unlock, auto-lock, password reveal |
| `user.api.ts` | Profile, settings, identity verification, domain profile |
| `audit.api.ts` | Personal, tenant, and connection audit logs, geo data |
| `gateway.api.ts` | Gateway CRUD, SSH keys, orchestration, templates, sessions |
| `tenant.api.ts` | Tenant CRUD, user management, MFA stats |
| `team.api.ts` | Team CRUD, member management |
| `secrets.api.ts` | Keychain CRUD, versioning, sharing, external shares, tenant vault |
| `recordings.api.ts` | Session recording listing, streaming, video export |
| `sessions.api.ts` | Session monitoring (active, count, terminate) |
| `oauth.api.ts` | OAuth providers, linked accounts, vault setup |
| `importExport.api.ts` | Connection import/export |
| `twofa.api.ts` | TOTP 2FA setup/verify/disable |
| `smsMfa.api.ts` | SMS MFA setup/verify/enable/disable |
| `webauthn.api.ts` | WebAuthn credential management |
| `passwordReset.api.ts` | Password reset flow |
| `admin.api.ts` | Admin config (email status, self-signup) |
| `tabs.api.ts` | Tab state persistence |
| `files.api.ts` | RDP drive file management |
| `email.api.ts` | Email verification resend |
| `notifications.api.ts` | Notification listing and management |

<!-- manual-start -->
<!-- manual-end -->
