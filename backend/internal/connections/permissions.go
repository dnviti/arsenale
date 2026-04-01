package connections

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/credentialresolver"
	"github.com/jackc/pgx/v5"
)

func (s Service) requireTeamRole(ctx context.Context, userID, tenantID, teamID string) (string, error) {
	var role string
	err := s.DB.QueryRow(ctx, `
SELECT tm.role::text
FROM "TeamMember" tm
JOIN "Team" t ON t.id = tm."teamId"
WHERE tm."teamId" = $1
  AND tm."userId" = $2
  AND (tm."expiresAt" IS NULL OR tm."expiresAt" > NOW())
  AND ($3 = '' OR t."tenantId" = $3)
`, teamID, userID, tenantID).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", &requestError{status: http.StatusForbidden, message: "Insufficient team role to create connections"}
		}
		return "", fmt.Errorf("load team role: %w", err)
	}
	if !canManageTeam(role) {
		return "", &requestError{status: http.StatusForbidden, message: "Insufficient team role to create connections"}
	}
	return role, nil
}

func (s Service) resolveConnectionEncryptionKey(ctx context.Context, userID string, teamID *string) ([]byte, error) {
	if teamID != nil && *teamID != "" {
		key, err := s.getTeamVaultKey(ctx, *teamID, userID)
		if err != nil {
			return nil, err
		}
		if len(key) == 0 {
			return nil, &requestError{status: http.StatusForbidden, message: "Vault is locked. Please unlock it first."}
		}
		return key, nil
	}

	key, err := s.getVaultKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(key) == 0 {
		return nil, &requestError{status: http.StatusForbidden, message: "Vault is locked. Please unlock it first."}
	}
	return key, nil
}

func (s Service) validateCredentialSecretReference(ctx context.Context, userID, tenantID, secretID, connectionType string) error {
	resolver := credentialresolver.Resolver{
		DB:        s.DB,
		Redis:     s.Redis,
		ServerKey: s.ServerEncryptionKey,
	}
	if err := resolver.ValidateConnectionSecretReference(ctx, userID, secretID, connectionType, tenantID); err != nil {
		var reqErr *credentialresolver.RequestError
		if errors.As(err, &reqErr) {
			return &requestError{status: reqErr.Status, message: reqErr.Message}
		}
		return err
	}
	return nil
}
