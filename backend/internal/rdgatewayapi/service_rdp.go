package rdgatewayapi

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
)

func claimsCanManageRDGW(claims authn.Claims) bool {
	if strings.TrimSpace(claims.TenantID) == "" {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(claims.TenantRole)) {
	case "ADMIN", "OWNER":
		return true
	default:
		return false
	}
}

func claimsCanViewRDGWStatus(claims authn.Claims) bool {
	if strings.TrimSpace(claims.TenantID) == "" {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(claims.TenantRole)) {
	case "ADMIN", "OWNER", "OPERATOR":
		return true
	default:
		return false
	}
}

func generateRDPFile(params rdpFileParams) string {
	gatewayPort := params.GatewayPort
	if gatewayPort == 0 {
		gatewayPort = 443
	}

	lines := []string{
		fmt.Sprintf("full address:s:%s:%d", params.TargetHost, params.TargetPort),
		fmt.Sprintf("server port:i:%d", params.TargetPort),
		"use redirection server name:i:1",
		fmt.Sprintf("gatewayhostname:s:%s:%d", params.GatewayHostname, gatewayPort),
		"gatewayusagemethod:i:1",
		"gatewayprofileusagemethod:i:1",
		"gatewaybrokeringtype:i:0",
		"gatewaycredentialssource:i:0",
		fmt.Sprintf("screen mode id:i:%d", defaultInt(params.ScreenMode, 2)),
		fmt.Sprintf("desktopwidth:i:%d", defaultInt(params.DesktopWidth, 1920)),
		fmt.Sprintf("desktopheight:i:%d", defaultInt(params.DesktopHeight, 1080)),
		"session bpp:i:32",
		"smart sizing:i:1",
		"dynamic resolution:i:1",
		"displayconnectionbar:i:1",
		"redirectclipboard:i:1",
		"prompt for credentials on client:i:1",
		"promptcredentialonce:i:1",
		"authentication level:i:2",
		"negotiate security layer:i:1",
		"enablecredsspsupport:i:1",
		"compression:i:1",
		"bitmapcachepersistenable:i:1",
		"autoreconnection enabled:i:1",
		"autoreconnect max retries:i:3",
	}

	if strings.TrimSpace(params.Username) != "" {
		if strings.TrimSpace(params.Domain) != "" {
			lines = append(lines, fmt.Sprintf("username:s:%s\\%s", params.Domain, params.Username))
		} else {
			lines = append(lines, fmt.Sprintf("username:s:%s", params.Username))
		}
	}

	return strings.Join(lines, "\r\n") + "\r\n"
}

func sanitizeFilename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "connection"
	}
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '.' || r == '_' || r == '-':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	sanitized := builder.String()
	if sanitized == "" {
		return "connection"
	}
	return sanitized
}

func requestIP(r *http.Request) *string {
	for _, header := range []string{"X-Real-IP", "X-Forwarded-For"} {
		if value := strings.TrimSpace(r.Header.Get(header)); value != "" {
			if header == "X-Forwarded-For" {
				value = strings.TrimSpace(strings.Split(value, ",")[0])
			}
			host := stripPort(value)
			if host != "" {
				return &host
			}
		}
	}
	host := stripPort(r.RemoteAddr)
	if host == "" {
		return nil
	}
	return &host
}

func stripPort(value string) string {
	host, _, err := net.SplitHostPort(value)
	if err == nil {
		return host
	}
	return strings.TrimSpace(value)
}

func defaultInt(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}
