package gatewayruntime

import "strings"

const (
	TypeGuacd      = "GUACD"
	TypeSSHBastion = "SSH_BASTION"
	TypeManagedSSH = "MANAGED_SSH"
	TypeDBProxy    = "DB_PROXY"
)

type Definition struct {
	Type                string
	Managed             bool
	PrimaryPort         int
	TunnelLocalHost     string
	StandaloneDirectory string
	ComposeService      string
}

var definitions = []Definition{
	{
		Type:                TypeGuacd,
		Managed:             true,
		PrimaryPort:         4822,
		TunnelLocalHost:     "127.0.0.1",
		StandaloneDirectory: "gateways/guacd",
		ComposeService:      "guacd",
	},
	{
		Type:                TypeSSHBastion,
		Managed:             false,
		PrimaryPort:         2222,
		TunnelLocalHost:     "127.0.0.1",
		StandaloneDirectory: "gateways/ssh-gateway",
		ComposeService:      "ssh-gateway",
	},
	{
		Type:                TypeManagedSSH,
		Managed:             true,
		PrimaryPort:         2222,
		TunnelLocalHost:     "127.0.0.1",
		StandaloneDirectory: "gateways/ssh-gateway",
		ComposeService:      "ssh-gateway",
	},
	{
		Type:                TypeDBProxy,
		Managed:             true,
		PrimaryPort:         5432,
		TunnelLocalHost:     "127.0.0.1",
		StandaloneDirectory: "gateways/db-proxy",
		ComposeService:      "db-proxy",
	},
}

func All() []Definition {
	out := make([]Definition, len(definitions))
	copy(out, definitions)
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

func TunnelLocalHost(gatewayType string) string {
	def, ok := Lookup(gatewayType)
	if !ok || strings.TrimSpace(def.TunnelLocalHost) == "" {
		return "127.0.0.1"
	}
	return def.TunnelLocalHost
}

func TunnelLocalPort(gatewayType string, configuredPort int) int {
	if configuredPort > 0 {
		return configuredPort
	}
	return PrimaryPort(gatewayType)
}
