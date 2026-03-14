# Database

> Auto-generated on 2026-03-14 by `/docs create database`.
> Source of truth is the codebase. Run `/docs update database` after code changes.

## Overview

- **Provider**: PostgreSQL 16
- **ORM**: Prisma (with `prisma-client` generator)
- **Schema location**: `server/prisma/schema.prisma`
- **Connection**: Configured via `DATABASE_URL` environment variable
- **Migrations**: Automatically applied on server start via `prisma migrate deploy`

<!-- manual-start -->
<!-- manual-end -->

## Entity-Relationship Summary

The database models a multi-tenant remote access management system:

- **Users** own **Connections** (SSH/RDP/VNC), organize them in **Folders**, and can **share** them with other users via **SharedConnection**.
- **Tenants** represent organizations. Users join tenants through **TenantMember** with roles (Owner/Admin/Member).
- **Teams** exist within tenants. Users join teams through **TeamMember** with roles (Admin/Editor/Viewer).
- Connections can be routed through **Gateways** (GUACD, SSH Bastion, or Managed SSH). Managed gateways have **ManagedGatewayInstances** (containers).
- **GatewayTemplates** provide reusable gateway configurations for quick deployment.
- Each tenant has an optional **SshKeyPair** for managed SSH gateways with auto-rotation.
- **VaultSecrets** store encrypted credentials (Login, SSH Key, Certificate, API Key, Secure Note) with versioning (**VaultSecretVersion**) and sharing (**SharedSecret**, **ExternalSecretShare**).
- Secrets can be organized in **VaultFolders** scoped to personal, team, or tenant level.
- **TenantVaultMember** tracks tenant-level vault key distribution.
- **ActiveSession** tracks live SSH/RDP/VNC sessions with heartbeats and idle detection.
- **SessionRecording** stores metadata for recorded sessions (asciicast for SSH, .guac for RDP/VNC).
- **AuditLog** records 100+ distinct action types with optional geo-location enrichment.
- **Notification** delivers in-app notifications for sharing and secret events.
- **RefreshToken** manages JWT refresh token families with rotation and reuse detection.
- **OAuthAccount** links external identity providers (Google, Microsoft, GitHub, OIDC, SAML).
- **WebAuthnCredential** stores FIDO2/passkey credentials for passwordless MFA.
- **OpenTab** persists the user's open connection tabs server-side.
- **AppConfig** stores key-value application settings (e.g., self-signup toggle).

<!-- manual-start -->
<!-- manual-end -->

## Models

### User

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | Unique identifier |
| email | String | Unique | User email address |
| username | String? | Optional | Display name |
| avatarData | String? | Optional | Base64-encoded avatar image |
| passwordHash | String? | Optional | Argon2 hashed password (null for OAuth-only users) |
| vaultSalt | String? | Optional | Salt for vault key derivation |
| encryptedVaultKey | String? | Optional | AES-256-GCM encrypted master key |
| vaultKeyIV | String? | Optional | Vault key initialization vector |
| vaultKeyTag | String? | Optional | Vault key auth tag |
| vaultSetupComplete | Boolean | Default: true | Whether vault encryption is configured |
| sshDefaults | Json? | Optional | Default SSH terminal settings |
| rdpDefaults | Json? | Optional | Default RDP connection settings |
| totpSecret | String? | Optional | Legacy TOTP secret (deprecated) |
| encryptedTotpSecret | String? | Optional | Encrypted TOTP secret |
| totpSecretIV | String? | Optional | TOTP secret IV |
| totpSecretTag | String? | Optional | TOTP secret auth tag |
| totpEnabled | Boolean | Default: false | TOTP 2FA enabled |
| phoneNumber | String? | Optional | Phone number for SMS MFA |
| phoneVerified | Boolean | Default: false | Phone number verified |
| smsMfaEnabled | Boolean | Default: false | SMS MFA enabled |
| smsOtpHash | String? | Optional | Current SMS OTP hash |
| smsOtpExpiresAt | DateTime? | Optional | SMS OTP expiry |
| webauthnEnabled | Boolean | Default: false | WebAuthn/passkey MFA enabled |
| vaultAutoLockMinutes | Int? | Optional | User's vault auto-lock preference |
| domainName | String? | Optional | Windows/AD domain name |
| domainUsername | String? | Optional | Domain username |
| encryptedDomainPassword | String? | Optional | Encrypted domain password |
| domainPasswordIV | String? | Optional | Domain password IV |
| domainPasswordTag | String? | Optional | Domain password auth tag |
| enabled | Boolean | Default: true | Account enabled/disabled |
| emailVerified | Boolean | Default: false | Email verification status |
| emailVerifyToken | String? | Unique | Email verification token |
| emailVerifyExpiry | DateTime? | Optional | Verification token expiry |
| pendingEmail | String? | Optional | Pending email change address |
| emailChangeCodeOldHash | String? | Optional | Email change OTP for old address |
| emailChangeCodeNewHash | String? | Optional | Email change OTP for new address |
| emailChangeExpiry | DateTime? | Optional | Email change expiry |
| passwordResetTokenHash | String? | Unique | Password reset token hash |
| passwordResetExpiry | DateTime? | Optional | Reset token expiry |
| encryptedVaultRecoveryKey | String? | Optional | Encrypted vault recovery key |
| vaultRecoveryKeyIV | String? | Optional | Recovery key IV |
| vaultRecoveryKeyTag | String? | Optional | Recovery key auth tag |
| vaultRecoveryKeySalt | String? | Optional | Recovery key derivation salt |
| failedLoginAttempts | Int | Default: 0 | Failed login counter for lockout |
| lockedUntil | DateTime? | Optional | Account lockout expiry |
| createdAt | DateTime | Auto | Creation timestamp |
| updatedAt | DateTime | Auto | Last update timestamp |

**Relations**: tenantMemberships (TenantMember[]), connections (Connection[]), folders (Folder[]), sharedWithMe (SharedConnection[]), sharedByMe (SharedConnection[]), refreshTokens (RefreshToken[]), oauthAccounts (OAuthAccount[]), auditLogs (AuditLog[]), notifications (Notification[]), teamMembers (TeamMember[]), gatewaysCreated (Gateway[]), openTabs (OpenTab[]), vaultSecrets (VaultSecret[]), webauthnCredentials (WebAuthnCredential[]), activeSessions (ActiveSession[]), sessionRecordings (SessionRecording[])

<!-- manual-start -->
<!-- manual-end -->

### Tenant

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | Unique identifier |
| name | String | Required | Organization name |
| slug | String | Unique | URL-safe identifier |
| hasTenantVaultKey | Boolean | Default: false | Whether tenant vault is initialized |
| mfaRequired | Boolean | Default: false | Mandatory MFA policy |
| vaultAutoLockMaxMinutes | Int? | Optional | Maximum vault auto-lock for members |
| defaultSessionTimeoutSeconds | Int | Default: 3600 | Default session inactivity timeout |
| createdAt | DateTime | Auto | Creation timestamp |
| updatedAt | DateTime | Auto | Last update timestamp |

**Relations**: members (TenantMember[]), teams (Team[]), gateways (Gateway[]), gatewayTemplates (GatewayTemplate[]), sshKeyPair (SshKeyPair?), vaultSecrets (VaultSecret[]), tenantVaultMembers (TenantVaultMember[])

<!-- manual-start -->
<!-- manual-end -->

### Team

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | Unique identifier |
| name | String | Required | Team name |
| description | String? | Optional | Team description |
| tenantId | String | FK -> Tenant | Parent tenant |
| createdAt | DateTime | Auto | Creation timestamp |
| updatedAt | DateTime | Auto | Last update timestamp |

**Unique constraint**: `[tenantId, name]`

**Relations**: tenant (Tenant), members (TeamMember[]), connections (Connection[]), folders (Folder[]), vaultSecrets (VaultSecret[]), vaultFolders (VaultFolder[])

<!-- manual-start -->
<!-- manual-end -->

### TeamMember

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | Unique identifier |
| teamId | String | FK -> Team (cascade delete) | Parent team |
| userId | String | FK -> User (cascade delete) | Member user |
| role | TeamRole | Enum | Member's role in team |
| encryptedTeamVaultKey | String? | Optional | Encrypted team vault key for this member |
| teamVaultKeyIV | String? | Optional | Team vault key IV |
| teamVaultKeyTag | String? | Optional | Team vault key auth tag |
| joinedAt | DateTime | Auto | Join timestamp |

**Unique constraint**: `[teamId, userId]`

<!-- manual-start -->
<!-- manual-end -->

### Connection

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | Unique identifier |
| name | String | Required | Display name |
| type | ConnectionType | Enum | SSH, RDP, or VNC |
| host | String | Required | Target hostname/IP |
| port | Int | Required | Target port |
| folderId | String? | FK -> Folder (set null) | Parent folder |
| teamId | String? | FK -> Team | Owning team |
| encryptedUsername | String? | Optional | AES-256-GCM encrypted username |
| usernameIV | String? | Optional | Username IV |
| usernameTag | String? | Optional | Username auth tag |
| encryptedPassword | String? | Optional | Encrypted password |
| passwordIV | String? | Optional | Password IV |
| passwordTag | String? | Optional | Password auth tag |
| encryptedDomain | String? | Optional | Encrypted Windows domain |
| domainIV | String? | Optional | Domain IV |
| domainTag | String? | Optional | Domain auth tag |
| credentialSecretId | String? | FK -> VaultSecret (set null) | Linked keychain secret |
| description | String? | Optional | Connection notes |
| isFavorite | Boolean | Default: false | Favorited by owner |
| enableDrive | Boolean | Default: false | Enable RDP drive redirection |
| sshTerminalConfig | Json? | Optional | Per-connection SSH terminal settings |
| rdpSettings | Json? | Optional | Per-connection RDP settings |
| vncSettings | Json? | Optional | Per-connection VNC settings |
| defaultCredentialMode | String? | Optional | Default credential mode (saved/domain/manual) |
| userId | String | FK -> User | Owner |
| gatewayId | String? | FK -> Gateway (set null) | Assigned gateway |
| createdAt | DateTime | Auto | Creation timestamp |
| updatedAt | DateTime | Auto | Last update timestamp |

**Relations**: folder (Folder?), team (Team?), user (User), gateway (Gateway?), credentialSecret (VaultSecret?), shares (SharedConnection[]), openTabs (OpenTab[]), activeSessions (ActiveSession[]), sessionRecordings (SessionRecording[])

<!-- manual-start -->
<!-- manual-end -->

### SharedConnection

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | Unique identifier |
| connectionId | String | FK -> Connection (cascade) | Shared connection |
| sharedWithUserId | String | FK -> User | Recipient |
| sharedByUserId | String | FK -> User | Sharer |
| permission | Permission | Enum | READ_ONLY or FULL_ACCESS |
| encryptedUsername | String? | Optional | Re-encrypted credentials for recipient |
| usernameIV, usernameTag | String? | Optional | |
| encryptedPassword | String? | Optional | |
| passwordIV, passwordTag | String? | Optional | |
| encryptedDomain | String? | Optional | |
| domainIV, domainTag | String? | Optional | |
| createdAt | DateTime | Auto | Share timestamp |

**Unique constraint**: `[connectionId, sharedWithUserId]`

<!-- manual-start -->
<!-- manual-end -->

### Folder

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | Unique identifier |
| name | String | Required | Folder name |
| parentId | String? | FK -> Folder (self-relation) | Parent folder for nesting |
| userId | String | FK -> User | Owner |
| teamId | String? | FK -> Team | Owning team |
| sortOrder | Int | Default: 0 | Display order |
| createdAt | DateTime | Auto | Creation timestamp |
| updatedAt | DateTime | Auto | Last update timestamp |

**Relations**: parent (Folder?), children (Folder[]), user (User), team (Team?), connections (Connection[])

<!-- manual-start -->
<!-- manual-end -->

### Gateway

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | Unique identifier |
| name | String | Required | Display name |
| type | GatewayType | Enum | GUACD, SSH_BASTION, or MANAGED_SSH |
| host | String | Required | Gateway hostname |
| port | Int | Required | Gateway port |
| description | String? | Optional | Notes |
| isDefault | Boolean | Default: false | Default gateway for its type |
| tenantId | String | FK -> Tenant | Owning tenant |
| createdById | String | FK -> User | Creator |
| encryptedUsername | String? | Optional | Encrypted gateway credentials |
| usernameIV, usernameTag | String? | Optional | |
| encryptedPassword | String? | Optional | |
| passwordIV, passwordTag | String? | Optional | |
| encryptedSshKey | String? | Optional | |
| sshKeyIV, sshKeyTag | String? | Optional | |
| apiPort | Int? | Optional | Gateway API sidecar port |
| templateId | String? | FK -> GatewayTemplate (set null) | Source template |
| isManaged | Boolean | Default: false | Whether orchestrator manages containers |
| publishPorts | Boolean | Default: false | Expose container ports to host |
| lbStrategy | LoadBalancingStrategy | Default: ROUND_ROBIN | Load balancing strategy |
| desiredReplicas | Int | Default: 1 | Desired container count |
| autoScale | Boolean | Default: false | Auto-scaling enabled |
| minReplicas | Int | Default: 1 | Minimum replicas |
| maxReplicas | Int | Default: 5 | Maximum replicas |
| sessionsPerInstance | Int | Default: 10 | Scale threshold |
| scaleDownCooldownSeconds | Int | Default: 300 | Cooldown after scale-down |
| lastScaleAction | DateTime? | Optional | Last scaling event |
| inactivityTimeoutSeconds | Int | Default: 3600 | Session inactivity timeout |
| monitoringEnabled | Boolean | Default: true | Health monitoring active |
| monitorIntervalMs | Int | Default: 5000 | Health check interval |
| lastHealthStatus | GatewayHealthStatus | Default: UNKNOWN | Current health |
| lastCheckedAt | DateTime? | Optional | Last health check |
| lastLatencyMs | Int? | Optional | Last check latency |
| lastError | String? | Optional | Last error message |
| createdAt | DateTime | Auto | |
| updatedAt | DateTime | Auto | |

**Indexes**: `[tenantId]`, `[tenantId, type, isDefault]`

**Relations**: tenant (Tenant), createdBy (User), template (GatewayTemplate?), connections (Connection[]), activeSessions (ActiveSession[]), managedInstances (ManagedGatewayInstance[])

<!-- manual-start -->
<!-- manual-end -->

### GatewayTemplate

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | Unique identifier |
| name | String | Required | Template name |
| type | GatewayType | Enum | Gateway type |
| host | String | Required | Default host |
| port | Int | Required | Default port |
| description | String? | Optional | Notes |
| apiPort | Int? | Optional | API sidecar port |
| autoScale, minReplicas, maxReplicas, sessionsPerInstance, scaleDownCooldownSeconds | Various | | Auto-scaling defaults |
| monitoringEnabled, monitorIntervalMs, inactivityTimeoutSeconds | Various | | Monitoring defaults |
| publishPorts | Boolean | Default: false | Port publishing default |
| lbStrategy | LoadBalancingStrategy | Default: ROUND_ROBIN | LB strategy default |
| tenantId | String | FK -> Tenant | Owning tenant |
| createdById | String | FK -> User | Creator |
| createdAt | DateTime | Auto | |
| updatedAt | DateTime | Auto | |

**Index**: `[tenantId]`

**Relations**: tenant (Tenant), createdBy (User), gateways (Gateway[])

<!-- manual-start -->
<!-- manual-end -->

### SshKeyPair

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | Unique identifier |
| tenantId | String | Unique, FK -> Tenant (cascade) | One per tenant |
| encryptedPrivateKey | String | Required | Server-encrypted private key |
| privateKeyIV | String | Required | |
| privateKeyTag | String | Required | |
| publicKey | String | Required | Public key (plaintext) |
| fingerprint | String | Required | Key fingerprint |
| algorithm | String | Default: "ed25519" | Key algorithm |
| expiresAt | DateTime? | Optional | Key expiry date |
| autoRotateEnabled | Boolean | Default: false | Auto-rotation enabled |
| rotationIntervalDays | Int | Default: 90 | Days between rotations |
| lastAutoRotatedAt | DateTime? | Optional | Last auto-rotation |
| createdAt | DateTime | Auto | |
| updatedAt | DateTime | Auto | |

<!-- manual-start -->
<!-- manual-end -->

### VaultSecret

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | Unique identifier |
| name | String | Required | Secret name |
| description | String? | Optional | Description |
| type | SecretType | Enum | LOGIN, SSH_KEY, CERTIFICATE, API_KEY, SECURE_NOTE |
| scope | SecretScope | Enum | PERSONAL, TEAM, TENANT |
| userId | String | FK -> User | Owner |
| teamId | String? | FK -> Team | Team scope |
| tenantId | String? | FK -> Tenant | Tenant scope |
| folderId | String? | FK -> VaultFolder (set null) | Parent folder |
| encryptedData | String | Required | AES-256-GCM encrypted payload |
| dataIV | String | Required | |
| dataTag | String | Required | |
| metadata | Json? | Optional | Additional metadata |
| tags | String[] | Default: [] | Searchable tags |
| isFavorite | Boolean | Default: false | Favorited |
| expiresAt | DateTime? | Optional | Secret expiry date |
| currentVersion | Int | Default: 1 | Current version number |
| createdAt | DateTime | Auto | |
| updatedAt | DateTime | Auto | |

**Indexes**: `[userId, scope]`, `[teamId]`, `[tenantId, scope]`, `[expiresAt]`

**Relations**: user (User), team (Team?), tenant (Tenant?), folder (VaultFolder?), versions (VaultSecretVersion[]), shares (SharedSecret[]), externalShares (ExternalSecretShare[]), connections (Connection[])

<!-- manual-start -->
<!-- manual-end -->

### VaultSecretVersion

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | |
| secretId | String | FK -> VaultSecret (cascade) | Parent secret |
| version | Int | Required | Version number |
| encryptedData | String | Required | Encrypted payload snapshot |
| dataIV | String | Required | |
| dataTag | String | Required | |
| changedBy | String | FK -> User | User who made the change |
| changeNote | String? | Optional | Version note |
| createdAt | DateTime | Auto | |

**Unique constraint**: `[secretId, version]` | **Index**: `[secretId]`

<!-- manual-start -->
<!-- manual-end -->

### SharedSecret

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | |
| secretId | String | FK -> VaultSecret (cascade) | Shared secret |
| sharedWithUserId | String | FK -> User | Recipient |
| sharedByUserId | String | FK -> User | Sharer |
| permission | Permission | Enum | READ_ONLY or FULL_ACCESS |
| encryptedData | String | Required | Re-encrypted for recipient's key |
| dataIV | String | Required | |
| dataTag | String | Required | |
| createdAt | DateTime | Auto | |

**Unique constraint**: `[secretId, sharedWithUserId]`

<!-- manual-start -->
<!-- manual-end -->

### ExternalSecretShare

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | |
| secretId | String | FK -> VaultSecret (cascade) | Source secret |
| createdByUserId | String | FK -> User | Creator |
| tokenHash | String | Unique | SHA-256 hash of access token |
| encryptedData | String | Required | Token-derived key encrypted payload |
| dataIV, dataTag | String | Required | |
| hasPin | Boolean | Default: false | PIN protection enabled |
| pinSalt | String? | Optional | Salt for PIN derivation |
| tokenSalt | String? | Optional | Salt for HKDF token derivation |
| expiresAt | DateTime | Required | Share expiry |
| maxAccessCount | Int? | Optional | Maximum access limit |
| accessCount | Int | Default: 0 | Current access count |
| secretType | SecretType | Enum | Type of shared secret |
| secretName | String | Required | Name snapshot |
| isRevoked | Boolean | Default: false | Manually revoked |
| createdAt | DateTime | Auto | |

**Indexes**: `[tokenHash]`, `[expiresAt]`

<!-- manual-start -->
<!-- manual-end -->

### ActiveSession

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | |
| userId | String | FK -> User (cascade) | Session owner |
| connectionId | String | FK -> Connection (cascade) | Target connection |
| gatewayId | String? | FK -> Gateway (set null) | Routing gateway |
| instanceId | String? | FK -> ManagedGatewayInstance (set null) | Specific container instance |
| protocol | SessionProtocol | Enum | SSH, RDP, or VNC |
| status | SessionStatus | Default: ACTIVE | ACTIVE, IDLE, or CLOSED |
| socketId | String? | Optional | Socket.IO socket ID (SSH) |
| guacTokenHash | String? | Optional | Guacamole token hash (RDP/VNC) |
| ipAddress | String? | Optional | Client IP address |
| startedAt | DateTime | Auto | Session start |
| lastActivityAt | DateTime | Auto | Last activity |
| endedAt | DateTime? | Optional | Session end |
| metadata | Json? | Optional | Host, port, credential source, routing info |

**Indexes**: `[userId, status]`, `[status]`, `[gatewayId, status]`, `[protocol, status]`, `[lastActivityAt]`, `[socketId]`, `[guacTokenHash]`, `[instanceId, status]`

<!-- manual-start -->
<!-- manual-end -->

### Other Models

#### RefreshToken

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | |
| token | String | Unique | Refresh token value |
| userId | String | FK -> User (cascade) | Token owner |
| tokenFamily | String | Indexed | Rotation family ID |
| revokedAt | DateTime? | Optional | Revocation timestamp |
| expiresAt | DateTime | Required | Token expiry |
| createdAt | DateTime | Auto | |

#### OAuthAccount

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | |
| userId | String | FK -> User (cascade), Indexed | |
| provider | AuthProvider | Enum | LOCAL, GOOGLE, MICROSOFT, GITHUB, OIDC, SAML |
| providerUserId | String | Required | External user ID |
| providerEmail | String? | Optional | External email |
| accessToken, refreshToken | String? | Optional | Stored OAuth tokens |
| samlAttributes | Json? | Optional | SAML assertion attributes |

**Unique constraint**: `[provider, providerUserId]`

#### WebAuthnCredential

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | |
| userId | String | FK -> User (cascade), Indexed | |
| credentialId | String | Unique | WebAuthn credential ID |
| publicKey | String | Required | COSE public key |
| counter | BigInt | Default: 0 | Signature counter |
| transports | String[] | Default: [] | Supported transports |
| deviceType | String? | Optional | Device type |
| backedUp | Boolean | Default: false | Backup eligible |
| friendlyName | String | Default: "Security Key" | Display name |
| aaguid | String? | Optional | Authenticator AAGUID |
| lastUsedAt | DateTime? | Optional | Last authentication |
| createdAt | DateTime | Auto | |

#### ManagedGatewayInstance

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | |
| gatewayId | String | FK -> Gateway (cascade) | Parent gateway |
| containerId | String | Unique | Container/pod ID |
| containerName | String | Required | Container name |
| host | String | Required | Container hostname |
| port | Int | Required | SSH port |
| apiPort | Int? | Optional | API sidecar port |
| status | ManagedInstanceStatus | Default: PROVISIONING | PROVISIONING, RUNNING, STOPPED, ERROR, REMOVING |
| orchestratorType | String | Required | docker, podman, or kubernetes |
| healthStatus | String? | Optional | Last health check result |
| lastHealthCheck | DateTime? | Optional | |
| errorMessage | String? | Optional | |
| consecutiveFailures | Int | Default: 0 | |
| createdAt, updatedAt | DateTime | Auto | |

**Indexes**: `[gatewayId]`, `[status]`

#### SessionRecording

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | |
| sessionId | String? | Optional | Linked active session |
| userId | String | FK -> User (cascade) | |
| connectionId | String | FK -> Connection (cascade) | |
| protocol | SessionProtocol | Enum | SSH, RDP, or VNC |
| filePath | String | Required | Recording file path |
| fileSize | Int? | Optional | File size in bytes |
| duration | Int? | Optional | Duration in seconds |
| width, height | Int? | Optional | Terminal/display dimensions |
| format | String | Default: "asciicast" | asciicast or guac |
| status | RecordingStatus | Default: RECORDING | RECORDING, COMPLETE, ERROR |
| createdAt | DateTime | Auto | |
| completedAt | DateTime? | Optional | |

**Indexes**: `[userId, createdAt]`, `[sessionId]`, `[connectionId]`

#### Notification

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| id | String | PK, UUID | |
| userId | String | FK -> User (cascade) | |
| type | NotificationType | Enum | Event type |
| message | String | Required | Notification text |
| read | Boolean | Default: false | Read status |
| relatedId | String? | Optional | Related entity ID |
| createdAt | DateTime | Auto | |

**Indexes**: `[userId, read]`, `[userId, createdAt]`

#### OpenTab, TenantMember, TenantVaultMember, VaultFolder, AppConfig

- **OpenTab**: userId + connectionId (unique), sortOrder, isActive. Index: `[userId]`
- **TenantMember**: tenantId + userId (unique), role (TenantRole), isActive. Index: `[userId, isActive]`
- **TenantVaultMember**: tenantId + userId (unique), encryptedTenantVaultKey + IV + tag
- **VaultFolder**: self-referential tree, scoped to personal/team/tenant. Indexes: `[userId, scope]`, `[teamId]`, `[tenantId]`
- **AppConfig**: key (PK string), value, updatedAt

<!-- manual-start -->
<!-- manual-end -->

## Enums

| Enum | Values |
|------|--------|
| **ConnectionType** | `RDP`, `SSH`, `VNC` |
| **GatewayType** | `GUACD`, `SSH_BASTION`, `MANAGED_SSH` |
| **Permission** | `READ_ONLY`, `FULL_ACCESS` |
| **TenantRole** | `OWNER`, `ADMIN`, `MEMBER` |
| **TeamRole** | `TEAM_ADMIN`, `TEAM_EDITOR`, `TEAM_VIEWER` |
| **SecretType** | `LOGIN`, `SSH_KEY`, `CERTIFICATE`, `API_KEY`, `SECURE_NOTE` |
| **SecretScope** | `PERSONAL`, `TEAM`, `TENANT` |
| **SessionProtocol** | `SSH`, `RDP`, `VNC` |
| **SessionStatus** | `ACTIVE`, `IDLE`, `CLOSED` |
| **AuthProvider** | `LOCAL`, `GOOGLE`, `MICROSOFT`, `GITHUB`, `OIDC`, `SAML` |
| **GatewayHealthStatus** | `UNKNOWN`, `REACHABLE`, `UNREACHABLE` |
| **ManagedInstanceStatus** | `PROVISIONING`, `RUNNING`, `STOPPED`, `ERROR`, `REMOVING` |
| **LoadBalancingStrategy** | `ROUND_ROBIN`, `LEAST_CONNECTIONS` |
| **NotificationType** | `CONNECTION_SHARED`, `SHARE_PERMISSION_UPDATED`, `SHARE_REVOKED`, `SECRET_SHARED`, `SECRET_SHARE_REVOKED`, `SECRET_EXPIRING`, `SECRET_EXPIRED` |
| **RecordingStatus** | `RECORDING`, `COMPLETE`, `ERROR` |
| **AuditAction** | 100+ values — see `server/prisma/schema.prisma` for the full list |

<!-- manual-start -->
<!-- manual-end -->
