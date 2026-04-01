package connections

import (
	"context"
	"net"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

type hostPolicy struct {
	allowLoopback     bool
	allowLocalNetwork bool
}

func validateConnectionHost(ctx context.Context, host string) error {
	normalized := strings.TrimSpace(strings.ToLower(host))
	if normalized == "" {
		return nil
	}
	if strings.HasPrefix(normalized, "[") && strings.HasSuffix(normalized, "]") {
		normalized = normalized[1 : len(normalized)-1]
	}

	policy := loadHostPolicy()
	if normalized == "localhost" && !policy.allowLoopback {
		return blockedHostError(policy)
	}

	localAddrs := loadLocalInterfaceAddresses()
	if addr, err := netip.ParseAddr(normalized); err == nil {
		if isForbiddenConnectionAddr(addr, localAddrs, policy) {
			return blockedHostError(policy)
		}
		return nil
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, normalized)
	if err != nil {
		return nil
	}
	for _, item := range addrs {
		if addr, ok := netip.AddrFromSlice(item.IP); ok {
			if isForbiddenConnectionAddr(addr, localAddrs, policy) {
				return blockedHostError(policy)
			}
		}
	}
	return nil
}

func loadHostPolicy() hostPolicy {
	return hostPolicy{
		allowLoopback:     readBoolEnv("ALLOW_LOOPBACK", false),
		allowLocalNetwork: readBoolEnv("ALLOW_LOCAL_NETWORK", true),
	}
}

func readBoolEnv(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func blockedHostError(policy hostPolicy) error {
	return &requestError{status: 400, message: blockedHostMessage(policy)}
}

func blockedHostMessage(policy hostPolicy) string {
	if policy.allowLoopback && policy.allowLocalNetwork {
		return "Connections to wildcard, link-local, metadata, or server interface addresses are not allowed"
	}
	if policy.allowLoopback {
		return "Connections to local network, wildcard, link-local, metadata, or server interface addresses are not allowed"
	}
	if policy.allowLocalNetwork {
		return "Connections to loopback, wildcard, link-local, metadata, or server interface addresses are not allowed"
	}
	return "Connections to loopback, local network, wildcard, link-local, metadata, or server interface addresses are not allowed"
}

func loadLocalInterfaceAddresses() map[string]struct{} {
	result := make(map[string]struct{})
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return result
	}
	for _, item := range addrs {
		switch value := item.(type) {
		case *net.IPNet:
			if addr, ok := netip.AddrFromSlice(value.IP); ok {
				result[addr.Unmap().String()] = struct{}{}
			}
		case *net.IPAddr:
			if addr, ok := netip.AddrFromSlice(value.IP); ok {
				result[addr.Unmap().String()] = struct{}{}
			}
		}
	}
	return result
}

func isForbiddenConnectionAddr(addr netip.Addr, localAddrs map[string]struct{}, policy hostPolicy) bool {
	addr = addr.Unmap()
	if !addr.IsValid() {
		return false
	}
	if addr.IsUnspecified() {
		return true
	}
	if addr.IsLoopback() {
		return !policy.allowLoopback
	}

	if addr.Is4() {
		value := addr.As4()
		if !policy.allowLocalNetwork {
			if value[0] == 10 {
				return true
			}
			if value[0] == 172 && value[1] >= 16 && value[1] <= 31 {
				return true
			}
			if value[0] == 192 && value[1] == 168 {
				return true
			}
		}
		if value[0] == 169 && value[1] == 254 {
			return true
		}
	}

	if addr.Is6() {
		if addr.IsLinkLocalUnicast() {
			return true
		}
		if !policy.allowLocalNetwork && isIPv6UniqueLocal(addr) {
			return true
		}
	}

	if _, ok := localAddrs[addr.String()]; ok {
		if policy.allowLoopback && addr.IsLoopback() {
			return false
		}
		return true
	}

	return false
}

func isIPv6UniqueLocal(addr netip.Addr) bool {
	if !addr.Is6() {
		return false
	}
	value := addr.As16()
	return value[0]&0xfe == 0xfc
}
