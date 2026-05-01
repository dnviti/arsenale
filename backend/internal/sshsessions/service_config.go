package sshsessions

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func normalizeCredentialMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "domain":
		return "domain"
	case "manual":
		return "manual"
	default:
		return "saved"
	}
}

func (s Service) client() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func defaultTerminalBrokerURL() string {
	if value := strings.TrimSpace(os.Getenv("TERMINAL_BROKER_URL")); value != "" {
		return value
	}
	return "http://terminal-broker:8090"
}

func defaultTunnelBrokerURL() string {
	if value := strings.TrimSpace(os.Getenv("GO_TUNNEL_BROKER_URL")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("TUNNEL_BROKER_URL")); value != "" {
		return value
	}
	return "http://tunnel-broker:8092"
}

func parseEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func parseEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func (s Service) gatewayRoutingMandatoryEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("GATEWAY_ROUTING_MODE")), "gateway-mandatory")
}
