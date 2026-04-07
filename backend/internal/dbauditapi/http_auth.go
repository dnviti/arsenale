package dbauditapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/tenantauth"
	"github.com/google/uuid"
)

func requestIP(r *http.Request) *string {
	for _, header := range []string{"X-Real-IP", "X-Forwarded-For"} {
		if value := strings.TrimSpace(r.Header.Get(header)); value != "" {
			if header == "X-Forwarded-For" {
				value = strings.TrimSpace(strings.Split(value, ",")[0])
			}
			if value != "" {
				return &value
			}
		}
	}
	if value := strings.TrimSpace(r.RemoteAddr); value != "" {
		return &value
	}
	return nil
}

func writeError(w http.ResponseWriter, err error) {
	var reqErr *requestError
	if errors.As(err, &reqErr) {
		app.ErrorJSON(w, reqErr.status, reqErr.message)
		return
	}
	app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
}

func (s Service) insertAuditLog(ctx context.Context, userID, action, targetType, targetID string, details map[string]any, ip *string) error {
	if s.DB == nil || userID == "" {
		return nil
	}
	payload := "{}"
	if details != nil {
		encoded, err := json.Marshal(details)
		if err != nil {
			return err
		}
		payload = string(encoded)
	}
	_, err := s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details, "ipAddress")
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
`, uuid.NewString(), userID, action, targetType, targetID, payload, ip)
	return err
}

func (s Service) authorized(w http.ResponseWriter, r *http.Request, claims authn.Claims) bool {
	if claims.TenantID == "" {
		app.ErrorJSON(w, http.StatusForbidden, "Tenant context is required")
		return false
	}
	membership, err := s.TenantAuth.ResolveMembership(r.Context(), claims.UserID, claims.TenantID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return false
	}
	if membership == nil {
		app.ErrorJSON(w, http.StatusForbidden, "Forbidden")
		return false
	}
	switch membership.Role {
	case "ADMIN", "OWNER", "AUDITOR":
	default:
		app.ErrorJSON(w, http.StatusForbidden, "Forbidden")
		return false
	}
	if !membership.Permissions[tenantauth.CanViewAuditLog] {
		app.ErrorJSON(w, http.StatusForbidden, "Forbidden")
		return false
	}
	return true
}

func (s Service) authorizedWrite(w http.ResponseWriter, r *http.Request, claims authn.Claims) bool {
	if claims.TenantID == "" {
		app.ErrorJSON(w, http.StatusForbidden, "Tenant context is required")
		return false
	}
	membership, err := s.TenantAuth.ResolveMembership(r.Context(), claims.UserID, claims.TenantID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return false
	}
	if membership == nil {
		app.ErrorJSON(w, http.StatusForbidden, "Forbidden")
		return false
	}
	switch membership.Role {
	case "ADMIN", "OWNER":
		return true
	default:
		app.ErrorJSON(w, http.StatusForbidden, "Forbidden")
		return false
	}
}
