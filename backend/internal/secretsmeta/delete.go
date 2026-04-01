package secretsmeta

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandleDelete(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.DeleteSecret(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.handleResolverError(w, err)
		return
	}

	_ = s.insertAuditLog(r.Context(), claims.UserID, "SECRET_DELETE", r.PathValue("id"), nil, requestIP(r))
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) DeleteSecret(ctx context.Context, userID, tenantID, secretID string) (map[string]bool, error) {
	if _, err := s.resolver().RequireManageSecret(ctx, userID, secretID, tenantID); err != nil {
		return nil, err
	}
	if s.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}

	if _, err := s.DB.Exec(ctx, `DELETE FROM "VaultSecret" WHERE id = $1`, secretID); err != nil {
		return nil, fmt.Errorf("delete secret: %w", err)
	}

	return map[string]bool{"deleted": true}, nil
}
