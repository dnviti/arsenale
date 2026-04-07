package gateways

import (
	"fmt"
	"net"
	"slices"
	"sort"
	"strings"
)

func buildManagedGatewayContainerName(record gatewayRecord, instanceIndex int) string {
	tenantSlug := sanitizeGatewayName(record.TenantID)
	if len(tenantSlug) > 8 {
		tenantSlug = tenantSlug[:8]
	}
	nameSlug := sanitizeGatewayName(record.Name)
	if len(nameSlug) > 32 {
		nameSlug = nameSlug[:32]
	}
	idSuffix := sanitizeGatewayName(record.ID)
	if len(idSuffix) > 8 {
		idSuffix = idSuffix[:8]
	}
	return fmt.Sprintf("arsenale-gw-%s-%s-%s-%d", tenantSlug, nameSlug, idSuffix, instanceIndex)
}

func sanitizeGatewayName(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "gateway"
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range raw {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "gateway"
	}
	return result
}

func findAvailableLoopbackPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("reserve loopback port: %w", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || addr.Port <= 0 {
		return 0, fmt.Errorf("allocate loopback port: unexpected address %T", listener.Addr())
	}
	return addr.Port, nil
}

func normalizedStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(result, value) {
			continue
		}
		result = append(result, value)
	}
	return result
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func inferInstanceHealth(status, health string) string {
	switch strings.ToLower(strings.TrimSpace(health)) {
	case "healthy", "unhealthy", "starting", "restarting":
		return strings.ToLower(strings.TrimSpace(health))
	}
	if strings.EqualFold(strings.TrimSpace(status), "running") {
		return "healthy"
	}
	return "unhealthy"
}

func inferInstanceStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "created", "configured":
		return "PROVISIONING"
	case "running":
		return "RUNNING"
	case "restarting", "paused", "exited", "dead", "stopped":
		return "STOPPED"
	default:
		return "ERROR"
	}
}

func inferPrimaryInstanceHost(record gatewayRecord, containerName string) string {
	if strings.TrimSpace(containerName) != "" {
		return strings.TrimSpace(containerName)
	}
	if strings.TrimSpace(record.Host) != "" {
		return strings.TrimSpace(record.Host)
	}
	return "localhost"
}

func managedGatewayAPIPort(record gatewayRecord, defaultGRPCPort int) *int {
	if !strings.EqualFold(strings.TrimSpace(record.Type), "MANAGED_SSH") {
		return nil
	}
	value := defaultGRPCPort
	return &value
}
