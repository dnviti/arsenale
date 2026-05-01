package gateways

import (
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func requireGatewayManager(w http.ResponseWriter, claims authn.Claims) bool {
	if claimsCanManageGateways(claims.TenantRole) {
		return true
	}
	app.ErrorJSON(w, http.StatusForbidden, "Insufficient permissions")
	return false
}

func claimsCanManageGateways(role string) bool {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case "OWNER", "ADMIN", "OPERATOR":
		return true
	default:
		return false
	}
}
