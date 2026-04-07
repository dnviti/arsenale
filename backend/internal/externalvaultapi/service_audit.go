package externalvaultapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
)

func requireTenantAdmin(claims authn.Claims) *requestError {
	if strings.TrimSpace(claims.TenantID) == "" {
		return &requestError{status: http.StatusForbidden, message: "Tenant membership required"}
	}
	switch strings.ToUpper(strings.TrimSpace(claims.TenantRole)) {
	case "OWNER", "ADMIN":
		return nil
	default:
		return &requestError{status: http.StatusForbidden, message: "Insufficient tenant role"}
	}
}

func (s Service) insertAuditLog(ctx context.Context, userID, action, targetID string, details map[string]any) error {
	_, err := s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details)
VALUES ($1, $2, $3::"AuditAction", 'ExternalVaultProvider', $4, $5)
`, uuid.NewString(), userID, action, targetID, details)
	return err
}
