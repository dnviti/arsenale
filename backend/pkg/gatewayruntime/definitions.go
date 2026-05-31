package gatewayruntime

import "strings"

const (
	TypeGuacd      = "GUACD"
	TypeSSHBastion = "SSH_BASTION"
	TypeManagedSSH = "MANAGED_SSH"
	TypeDBProxy    = "DB_PROXY"
)

// Deployment modes describe how a gateway is run.
const (
	// DeploymentSingleInstance is one gateway at a fixed address.
	DeploymentSingleInstance = "SINGLE_INSTANCE"
	// DeploymentManagedGroup is an Arsenale-orchestrated, optionally auto-scaled
	// pool of identical gateway instances behind one logical endpoint.
	DeploymentManagedGroup = "MANAGED_GROUP"
)

// Definition is the runtime contract for a gateway type. The first block of
// fields drives deployment; the second block (DisplayName … DeploymentModes) is
// the single source of truth for the human-readable catalog surfaced in the CLI
// (`arsenale gateway types`), the API (`GET /api/gateways/types`), and the web UI.
type Definition struct {
	Type                string
	Managed             bool
	PrimaryPort         int
	ListenerEnvVar      string
	TunnelLocalHost     string
	StandaloneDirectory string
	ComposeService      string

	// StableImage is the container image a managed type advertises in the
	// catalog (empty for self-hosted types). Deployment image selection itself
	// lives in the gateways runtime; this is for the human-readable catalog only.
	StableImage string

	// DisplayName is the friendly name shown to operators (e.g. "Remote Desktop
	// Gateway (Guacamole)") instead of the raw code.
	DisplayName string
	// Summary is a one-line description of what the gateway does.
	Summary string
	// Description explains what the gateway does AND what gets deployed.
	Description string
	// Protocols lists the protocols the gateway brokers (e.g. RDP, VNC, SSH).
	Protocols []string
	// RequiresCredentials is true when the operator must supply connection
	// credentials on the gateway itself (only SSH_BASTION). Managed types inject
	// credentials per session or use the server key pair.
	RequiresCredentials bool
	// DeploymentModes are the deployment modes this type allows.
	DeploymentModes []string
}

var definitions = []Definition{
	{
		Type:                TypeGuacd,
		Managed:             true,
		PrimaryPort:         4822,
		ListenerEnvVar:      "GUACD_PORT",
		TunnelLocalHost:     "127.0.0.1",
		StandaloneDirectory: "gateways/guacd",
		ComposeService:      "guacd",
		StableImage:         "ghcr.io/dnviti/arsenale/guacd:stable",
		DisplayName:         "Remote Desktop Gateway (Guacamole)",
		Summary:             "Browser-based RDP and VNC access via Apache Guacamole.",
		Description:         "Arsenale deploys and manages a containerized guacd service that brokers RDP and VNC sessions to the browser. Connection credentials are injected per session from the vault — none are stored on the gateway.",
		Protocols:           []string{"RDP", "VNC"},
		RequiresCredentials: false,
		DeploymentModes:     []string{DeploymentManagedGroup, DeploymentSingleInstance},
	},
	{
		Type:                TypeSSHBastion,
		Managed:             false,
		PrimaryPort:         2222,
		ListenerEnvVar:      "SSH_PORT",
		TunnelLocalHost:     "127.0.0.1",
		StandaloneDirectory: "gateways/ssh-gateway",
		ComposeService:      "ssh-gateway",
		DisplayName:         "SSH Bastion (Jump Host)",
		Summary:             "Reach SSH targets through an existing bastion you operate.",
		Description:         "Self-hosted: you point Arsenale at the host and port of an SSH bastion/jump host you already run — Arsenale does not deploy or scale it. Connection credentials (password or SSH key) are configured on the gateway. Single-instance only.",
		Protocols:           []string{"SSH"},
		RequiresCredentials: true,
		DeploymentModes:     []string{DeploymentSingleInstance},
	},
	{
		Type:                TypeManagedSSH,
		Managed:             true,
		PrimaryPort:         2222,
		ListenerEnvVar:      "SSH_PORT",
		TunnelLocalHost:     "127.0.0.1",
		StandaloneDirectory: "gateways/ssh-gateway",
		ComposeService:      "ssh-gateway",
		StableImage:         "ghcr.io/dnviti/arsenale/ssh-gateway:stable",
		DisplayName:         "Managed SSH Gateway",
		Summary:             "Arsenale-managed SSH gateway using the server key pair.",
		Description:         "Arsenale deploys and manages a containerized SSH gateway (optionally auto-scaled as a managed group). It authenticates with the server's SSH key pair, so no per-gateway credentials are needed.",
		Protocols:           []string{"SSH"},
		RequiresCredentials: false,
		DeploymentModes:     []string{DeploymentManagedGroup, DeploymentSingleInstance},
	},
	{
		Type:                TypeDBProxy,
		Managed:             true,
		PrimaryPort:         5432,
		ListenerEnvVar:      "DB_LISTEN_PORT",
		TunnelLocalHost:     "127.0.0.1",
		StandaloneDirectory: "gateways/db-proxy",
		ComposeService:      "db-proxy",
		StableImage:         "ghcr.io/dnviti/arsenale/db-proxy:stable",
		DisplayName:         "Database Proxy Gateway",
		Summary:             "Arsenale-managed proxy for database connections.",
		Description:         "Arsenale deploys and manages a containerized database proxy (optionally auto-scaled as a managed group) for database sessions such as PostgreSQL and MySQL. Database credentials are injected per session from the vault.",
		Protocols:           []string{"PostgreSQL", "MySQL"},
		RequiresCredentials: false,
		DeploymentModes:     []string{DeploymentManagedGroup, DeploymentSingleInstance},
	},
}

func All() []Definition {
	out := make([]Definition, len(definitions))
	copy(out, definitions)
	return out
}

// Types returns every gateway type code in stable order.
func Types() []string {
	out := make([]string, 0, len(definitions))
	for _, def := range definitions {
		out = append(out, def.Type)
	}
	return out
}

func NormalizeType(gatewayType string) string {
	return strings.ToUpper(strings.TrimSpace(gatewayType))
}

func Lookup(gatewayType string) (Definition, bool) {
	normalized := NormalizeType(gatewayType)
	for _, def := range definitions {
		if def.Type == normalized {
			return def, true
		}
	}
	return Definition{}, false
}

func IsAllowedType(gatewayType string) bool {
	_, ok := Lookup(gatewayType)
	return ok
}

func IsManagedType(gatewayType string) bool {
	def, ok := Lookup(gatewayType)
	return ok && def.Managed
}

func PrimaryPort(gatewayType string) int {
	def, ok := Lookup(gatewayType)
	if !ok {
		return 0
	}
	return def.PrimaryPort
}

func ListenerEnvVar(gatewayType string) string {
	def, ok := Lookup(gatewayType)
	if !ok {
		return ""
	}
	return def.ListenerEnvVar
}

// ComposeService returns the compose service name for a gateway type.
func ComposeService(gatewayType string) string {
	def, ok := Lookup(gatewayType)
	if !ok {
		return ""
	}
	return def.ComposeService
}

func TunnelLocalHost(gatewayType string) string {
	def, ok := Lookup(gatewayType)
	if !ok || strings.TrimSpace(def.TunnelLocalHost) == "" {
		return "127.0.0.1"
	}
	return def.TunnelLocalHost
}

func TunnelLocalPort(gatewayType string, configuredPort int) int {
	ports := TunnelLocalPortCandidates(gatewayType, configuredPort)
	if len(ports) == 0 {
		return 0
	}
	return ports[0]
}

func TunnelLocalPortCandidates(gatewayType string, configuredPort int) []int {
	ports := make([]int, 0, 2)
	def, ok := Lookup(gatewayType)
	if ok && def.PrimaryPort > 0 {
		ports = append(ports, def.PrimaryPort)
	}
	if configuredPort > 0 && (!ok || def.PrimaryPort != configuredPort) {
		ports = append(ports, configuredPort)
	}
	return ports
}

// Deployment models name, in plain language, who runs a gateway.
const (
	// DeploymentModelManaged: Arsenale deploys and runs the gateway container.
	DeploymentModelManaged = "Arsenale-managed"
	// DeploymentModelSelfHosted: the operator runs the gateway themselves.
	DeploymentModelSelfHosted = "Self-hosted"
)

// TypeInfo is the user-facing description of a gateway type, exposed by the API
// (GET /api/gateways/types) and the CLI (arsenale gateway types) and mirrored by
// the web UI.
type TypeInfo struct {
	Type                string   `json:"type"`
	DisplayName         string   `json:"displayName"`
	Summary             string   `json:"summary"`
	Description         string   `json:"description"`
	Protocols           []string `json:"protocols"`
	Managed             bool     `json:"managed"`
	DeploymentModel     string   `json:"deploymentModel"`
	DeploymentModes     []string `json:"deploymentModes"`
	DefaultPort         int      `json:"defaultPort"`
	RequiresCredentials bool     `json:"requiresCredentials"`
	Image               string   `json:"image,omitempty"`
}

func toTypeInfo(def Definition) TypeInfo {
	model := DeploymentModelSelfHosted
	image := ""
	if def.Managed {
		model = DeploymentModelManaged
		image = def.StableImage
	}
	return TypeInfo{
		Type:                def.Type,
		DisplayName:         def.DisplayName,
		Summary:             def.Summary,
		Description:         def.Description,
		Protocols:           append([]string(nil), def.Protocols...),
		Managed:             def.Managed,
		DeploymentModel:     model,
		DeploymentModes:     append([]string(nil), def.DeploymentModes...),
		DefaultPort:         def.PrimaryPort,
		RequiresCredentials: def.RequiresCredentials,
		Image:               image,
	}
}

// Catalog returns the user-facing description of every gateway type, in stable order.
func Catalog() []TypeInfo {
	out := make([]TypeInfo, 0, len(definitions))
	for _, def := range definitions {
		out = append(out, toTypeInfo(def))
	}
	return out
}

// Info returns the user-facing description of a single gateway type.
func Info(gatewayType string) (TypeInfo, bool) {
	def, ok := Lookup(gatewayType)
	if !ok {
		return TypeInfo{}, false
	}
	return toTypeInfo(def), true
}

// DisplayName returns the friendly name for a gateway type, falling back to the
// normalized code for unknown types.
func DisplayName(gatewayType string) string {
	def, ok := Lookup(gatewayType)
	if !ok || strings.TrimSpace(def.DisplayName) == "" {
		return NormalizeType(gatewayType)
	}
	return def.DisplayName
}
