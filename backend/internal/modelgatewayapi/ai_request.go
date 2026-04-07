package modelgatewayapi

import (
	"net"
	"net/http"
	"strings"
)

func requestIP(r *http.Request) string {
	candidates := []string{
		strings.TrimSpace(r.Header.Get("X-Real-IP")),
		firstForwardedValue(r.Header.Get("X-Forwarded-For")),
		strings.TrimSpace(r.RemoteAddr),
	}
	for _, candidate := range candidates {
		candidate = stripPort(candidate)
		candidate = strings.TrimPrefix(candidate, "::ffff:")
		if candidate != "" {
			return candidate
		}
	}
	return ""
}

func stripPort(value string) string {
	host, _, err := net.SplitHostPort(value)
	if err == nil {
		return host
	}
	return value
}

func firstForwardedValue(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	return strings.TrimSpace(parts[0])
}
