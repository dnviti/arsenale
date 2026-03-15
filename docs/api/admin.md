# Admin Endpoints

> Auto-generated on 2026-03-15 by /docs create api.
> Source of truth is the codebase. Run /docs update api after code changes.

## Tenants

All endpoints require authentication.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/tenants` | User | Create a new tenant |
| `GET` | `/api/tenants/mine/all` | User | List all tenant memberships |
| `GET` | `/api/tenants/mine` | Tenant | Get current tenant details |
| `PUT` | `/api/tenants/:id` | Admin | Update tenant (name, MFA policy, session timeout) |
| `DELETE` | `/api/tenants/:id` | Owner | Delete tenant |
| `GET` | `/api/tenants/:id/mfa-stats` | Admin | Get MFA compliance stats |
| `GET` | `/api/tenants/:id/users` | Tenant | List tenant users |
| `GET` | `/api/tenants/:id/users/:userId/profile` | Tenant | Get user profile details |
| `POST` | `/api/tenants/:id/invite` | Admin | Invite user by email |
| `POST` | `/api/tenants/:id/users` | Admin | Create a new user in tenant |
| `PUT` | `/api/tenants/:id/users/:userId` | Admin | Update user role |
| `DELETE` | `/api/tenants/:id/users/:userId` | Admin | Remove user from tenant |
| `PATCH` | `/api/tenants/:id/users/:userId/enabled` | Admin | Enable/disable user account |
| `PATCH` | `/api/tenants/:id/users/:userId/expiry` | Admin | Update membership expiry date |
| `PUT` | `/api/tenants/:id/users/:userId/email` | Admin | Admin change user email |
| `PUT` | `/api/tenants/:id/users/:userId/password` | Admin | Admin change user password |
| `GET` | `/api/tenants/:id/ip-allowlist` | Admin | Get tenant IP allowlist |
| `PUT` | `/api/tenants/:id/ip-allowlist` | Admin | Update tenant IP allowlist |

<!-- manual-start -->
<!-- manual-end -->

## Teams

All endpoints require authentication and tenant membership.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/teams` | Tenant | Create a team |
| `GET` | `/api/teams` | Tenant | List teams |
| `GET` | `/api/teams/:id` | Team Member | Get team details |
| `PUT` | `/api/teams/:id` | Team Admin | Update team |
| `DELETE` | `/api/teams/:id` | Team Admin | Delete team |
| `GET` | `/api/teams/:id/members` | Team Member | List members |
| `POST` | `/api/teams/:id/members` | Team Admin | Add member |
| `PUT` | `/api/teams/:id/members/:userId` | Team Admin | Update member role |
| `DELETE` | `/api/teams/:id/members/:userId` | Team Admin | Remove member |
| `PATCH` | `/api/teams/:id/members/:userId/expiry` | Team Admin | Update member expiry |

<!-- manual-start -->
<!-- manual-end -->

## Admin

All endpoints require authentication with Admin tenant role.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/admin/email/status` | Get email provider configuration status |
| `POST` | `/api/admin/email/test` | Send test email |
| `GET` | `/api/admin/app-config` | Get app configuration (self-signup, etc.) |
| `PUT` | `/api/admin/app-config/self-signup` | Toggle self-signup |

<!-- manual-start -->
<!-- manual-end -->

## Gateways

All endpoints require authentication and tenant membership. Most require Operator role.

### Gateway CRUD

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/gateways` | Tenant | List gateways |
| `POST` | `/api/gateways` | Operator | Create gateway |
| `PUT` | `/api/gateways/:id` | Operator | Update gateway |
| `DELETE` | `/api/gateways/:id` | Operator | Delete gateway |
| `POST` | `/api/gateways/:id/test` | Tenant | Test gateway connectivity |

### SSH Key Pair Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/gateways/ssh-keypair` | Generate SSH key pair |
| `GET` | `/api/gateways/ssh-keypair` | Get public key |
| `GET` | `/api/gateways/ssh-keypair/private` | Download private key |
| `POST` | `/api/gateways/ssh-keypair/rotate` | Rotate key pair |
| `PATCH` | `/api/gateways/ssh-keypair/rotation` | Update rotation policy |
| `GET` | `/api/gateways/ssh-keypair/rotation` | Get rotation status |
| `POST` | `/api/gateways/:id/push-key` | Push public key to gateway |

### Gateway Templates

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/gateways/templates` | List templates |
| `POST` | `/api/gateways/templates` | Create template |
| `PUT` | `/api/gateways/templates/:templateId` | Update template |
| `DELETE` | `/api/gateways/templates/:templateId` | Delete template |
| `POST` | `/api/gateways/templates/:templateId/deploy` | Deploy gateway from template |

### Managed Gateway Lifecycle

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/gateways/:id/deploy` | Deploy managed gateway containers |
| `DELETE` | `/api/gateways/:id/deploy` | Undeploy managed gateway |
| `POST` | `/api/gateways/:id/scale` | Scale gateway replicas |
| `GET` | `/api/gateways/:id/instances` | List container instances |
| `POST` | `/api/gateways/:id/instances/:instanceId/restart` | Restart an instance |
| `GET` | `/api/gateways/:id/instances/:instanceId/logs` | Get instance logs |

### Auto-Scaling

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/gateways/:id/scaling` | Get scaling status |
| `PUT` | `/api/gateways/:id/scaling` | Update scaling config |

### Zero-Trust Tunnel Management

All tunnel endpoints require authentication and tenant membership with Operator role, except tunnel overview which requires Admin role.

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/gateways/tunnel-overview` | Admin | Fleet-wide tunnel status overview |
| `POST` | `/api/gateways/:id/tunnel-token` | Operator | Generate tunnel token for gateway |
| `DELETE` | `/api/gateways/:id/tunnel-token` | Operator | Revoke tunnel token |
| `POST` | `/api/gateways/:id/tunnel-disconnect` | Operator | Force disconnect an active tunnel |
| `GET` | `/api/gateways/:id/tunnel-events` | Operator | Get recent tunnel connect/disconnect events |
| `GET` | `/api/gateways/:id/tunnel-metrics` | Operator | Get live tunnel metrics |

#### `GET /api/gateways/tunnel-overview`

Returns fleet-wide tunnel status for all tunnel-enabled gateways in the tenant.

**Auth**: Admin

**Response**:
```json
{
  "total": 5,
  "connected": 3,
  "disconnected": 2,
  "avgRttMs": 42
}
```

| Field | Type | Description |
|-------|------|-------------|
| `total` | `number` | Total tunnel-enabled gateways |
| `connected` | `number` | Currently connected tunnels |
| `disconnected` | `number` | Tunnel-enabled but not connected |
| `avgRttMs` | `number \| null` | Average round-trip latency in ms (null if no data) |

#### `POST /api/gateways/:id/tunnel-token`

Generate a new tunnel token for a gateway. The plain token is returned only once and must be stored by the caller.

**Auth**: Operator

**Response** (`201`):
```json
{
  "token": "tunneltok_abc123...",
  "tunnelEnabled": true,
  "tunnelConnected": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `token` | `string` | Plain-text tunnel token (shown only once) |
| `tunnelEnabled` | `boolean` | Whether tunnel is now enabled on the gateway |
| `tunnelConnected` | `boolean` | Whether a tunnel client is currently connected |

#### `DELETE /api/gateways/:id/tunnel-token`

Revoke the tunnel token for a gateway. Disconnects any active tunnel.

**Auth**: Operator

**Response**:
```json
{
  "revoked": true,
  "tunnelEnabled": false
}
```

#### `POST /api/gateways/:id/tunnel-disconnect`

Force disconnect an active tunnel without revoking the token. The tunnel client can reconnect using the same token.

**Auth**: Operator

**Response**:
```json
{
  "disconnected": true
}
```

**Errors**: `400` if tunnel is not currently connected.

#### `GET /api/gateways/:id/tunnel-events`

Returns the 20 most recent tunnel connect/disconnect audit events for a gateway.

**Auth**: Operator

**Response**:
```json
{
  "events": [
    {
      "action": "TUNNEL_CONNECT",
      "timestamp": "2026-03-15T10:30:00.000Z",
      "details": { "clientVersion": "1.0.0" },
      "ipAddress": "203.0.113.10"
    },
    {
      "action": "TUNNEL_DISCONNECT",
      "timestamp": "2026-03-15T09:15:00.000Z",
      "details": { "forced": true },
      "ipAddress": "203.0.113.10"
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `events[].action` | `string` | `TUNNEL_CONNECT` or `TUNNEL_DISCONNECT` |
| `events[].timestamp` | `string` | ISO 8601 timestamp |
| `events[].details` | `object \| null` | Safe audit details (clientVersion, forced) |
| `events[].ipAddress` | `string \| null` | Client IP address |

#### `GET /api/gateways/:id/tunnel-metrics`

Returns live metrics for an active tunnel connection. Returns `{ connected: false }` if no tunnel is connected.

**Auth**: Operator

**Response** (connected):
```json
{
  "connectedAt": "2026-03-15T08:00:00.000Z",
  "lastHeartbeat": "2026-03-15T10:30:00.000Z",
  "pingPongLatency": 35,
  "activeStreams": 2,
  "bytesTransferred": 1048576,
  "clientVersion": "1.0.0",
  "clientIp": "203.0.113.10"
}
```

**Response** (not connected):
```json
{
  "connected": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `connectedAt` | `string` | ISO 8601 connection start time |
| `lastHeartbeat` | `string \| undefined` | Last heartbeat timestamp |
| `pingPongLatency` | `number \| undefined` | WebSocket ping/pong latency in ms |
| `activeStreams` | `number` | Number of active multiplexed streams |
| `bytesTransferred` | `number` | Total bytes transferred through tunnel |
| `clientVersion` | `string \| undefined` | Tunnel client version string |
| `clientIp` | `string \| undefined` | Tunnel client IP address |

<!-- manual-start -->
<!-- manual-end -->

## Access Policies (ABAC)

All endpoints require authentication and tenant membership with Admin role. Mounted at `/api/access-policies`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/access-policies` | List all access policies |
| `POST` | `/api/access-policies` | Create an access policy |
| `PUT` | `/api/access-policies/:id` | Update an access policy |
| `DELETE` | `/api/access-policies/:id` | Delete an access policy |

### `GET /api/access-policies`

List all access policies for the current tenant.

**Response**: `AccessPolicy[]`

### `POST /api/access-policies`

Create a new access policy.

**Body**:
```json
{
  "targetType": "TENANT",
  "targetId": "uuid",
  "allowedTimeWindows": "09:00-17:00,20:00-22:00",
  "requireTrustedDevice": true,
  "requireMfaStepUp": false
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `targetType` | `"TENANT" \| "TEAM" \| "FOLDER"` | Yes | Scope of the policy |
| `targetId` | `string (UUID)` | Yes | ID of the tenant, team, or folder |
| `allowedTimeWindows` | `string \| null` | No | Comma-separated time windows in `HH:MM-HH:MM` format (24h) |
| `requireTrustedDevice` | `boolean` | No | Require a trusted/verified device |
| `requireMfaStepUp` | `boolean` | No | Require MFA step-up authentication |

**Response** (`201`): The created `AccessPolicy` object.

### `PUT /api/access-policies/:id`

Update an existing access policy. Only the fields provided are updated.

**Body**:
```json
{
  "allowedTimeWindows": "08:00-18:00",
  "requireTrustedDevice": false,
  "requireMfaStepUp": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `allowedTimeWindows` | `string \| null` | No | Comma-separated time windows in `HH:MM-HH:MM` format |
| `requireTrustedDevice` | `boolean` | No | Require a trusted/verified device |
| `requireMfaStepUp` | `boolean` | No | Require MFA step-up authentication |

**Response**: The updated `AccessPolicy` object.

### `DELETE /api/access-policies/:id`

Delete an access policy.

**Response**:
```json
{
  "deleted": true
}
```

<!-- manual-start -->
<!-- manual-end -->
