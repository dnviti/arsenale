// Human-readable catalog of gateway types for the UI.
//
// This MIRRORS the backend source of truth in
// backend/pkg/gatewayruntime/definitions.go (also exposed at GET
// /api/gateways/types). Keep the two in sync when adding or editing a type.

export type GatewayTypeCode = "GUACD" | "SSH_BASTION" | "MANAGED_SSH" | "DB_PROXY";

export type GatewayDeploymentModel = "Arsenale-managed" | "Self-hosted";

export interface GatewayTypeInfo {
  type: GatewayTypeCode;
  /** Short label for badges and tables (e.g. "Remote Desktop"). */
  label: string;
  /** Full title for selectors (e.g. "Remote Desktop Gateway (Guacamole)"). */
  title: string;
  /** One line explaining what the gateway does. */
  summary: string;
  /** Fuller explanation of what the gateway does AND what gets deployed. */
  description: string;
  protocols: string[];
  /** Arsenale-managed (containerized, deployed by Arsenale) vs self-hosted. */
  deploymentModel: GatewayDeploymentModel;
  /** Whether the operator supplies connection credentials on the gateway. */
  requiresCredentials: boolean;
}

/** Display order, matching the backend catalog. */
export const GATEWAY_TYPE_ORDER: GatewayTypeCode[] = [
  "GUACD",
  "SSH_BASTION",
  "MANAGED_SSH",
  "DB_PROXY",
];

export const GATEWAY_TYPE_CATALOG: Record<GatewayTypeCode, GatewayTypeInfo> = {
  GUACD: {
    type: "GUACD",
    label: "Remote Desktop",
    title: "Remote Desktop Gateway (Guacamole)",
    summary: "Browser-based RDP and VNC access via Apache Guacamole.",
    description:
      "Arsenale deploys and manages a containerized guacd service that brokers RDP and VNC sessions to the browser. Connection credentials are injected per session from the vault — none are stored on the gateway.",
    protocols: ["RDP", "VNC"],
    deploymentModel: "Arsenale-managed",
    requiresCredentials: false,
  },
  SSH_BASTION: {
    type: "SSH_BASTION",
    label: "SSH Bastion",
    title: "SSH Bastion (Jump Host)",
    summary: "Reach SSH targets through an existing bastion you operate.",
    description:
      "Self-hosted: you point Arsenale at the host and port of an SSH bastion/jump host you already run — Arsenale does not deploy or scale it. Connection credentials (password or SSH key) are configured on the gateway. Single-instance only.",
    protocols: ["SSH"],
    deploymentModel: "Self-hosted",
    requiresCredentials: true,
  },
  MANAGED_SSH: {
    type: "MANAGED_SSH",
    label: "Managed SSH",
    title: "Managed SSH Gateway",
    summary: "Arsenale-managed SSH gateway using the server key pair.",
    description:
      "Arsenale deploys and manages a containerized SSH gateway (optionally auto-scaled as a managed group). It authenticates with the server's SSH key pair, so no per-gateway credentials are needed.",
    protocols: ["SSH"],
    deploymentModel: "Arsenale-managed",
    requiresCredentials: false,
  },
  DB_PROXY: {
    type: "DB_PROXY",
    label: "Database Proxy",
    title: "Database Proxy Gateway",
    summary: "Arsenale-managed proxy for database connections.",
    description:
      "Arsenale deploys and manages a containerized database proxy (optionally auto-scaled as a managed group) for database sessions such as PostgreSQL and MySQL. Database credentials are injected per session from the vault.",
    protocols: ["PostgreSQL", "MySQL"],
    deploymentModel: "Arsenale-managed",
    requiresCredentials: false,
  },
};

export function gatewayTypeInfo(type: string): GatewayTypeInfo | undefined {
  return GATEWAY_TYPE_CATALOG[type as GatewayTypeCode];
}

/** Short, friendly label for a gateway type; falls back to the raw code. */
export function gatewayTypeLabel(type: string): string {
  return gatewayTypeInfo(type)?.label ?? type;
}
