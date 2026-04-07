package connections

import (
	"context"
	"errors"
	"fmt"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/jackc/pgx/v5"
)

func (s Service) DeleteConnection(ctx context.Context, claims authn.Claims, connectionID string, ip *string) error {
	access, err := s.resolveAccess(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return err
	}
	if access.AccessType == "shared" {
		return pgx.ErrNoRows
	}
	if access.AccessType == "team" && (access.Connection.TeamRole == nil || !canManageTeam(*access.Connection.TeamRole)) {
		return pgx.ErrNoRows
	}

	command, err := s.DB.Exec(ctx, `DELETE FROM "Connection" WHERE id = $1`, connectionID)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "DELETE_CONNECTION", connectionID, nil, ip)
	return nil
}

func (s Service) ToggleFavorite(ctx context.Context, claims authn.Claims, connectionID string, ip *string) (map[string]any, error) {
	access, err := s.resolveAccess(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return nil, err
	}
	switch access.AccessType {
	case "shared":
		return nil, &requestError{status: 403, message: "Cannot favorite shared connections"}
	case "team":
		if access.Connection.TeamRole == nil || !canManageTeam(*access.Connection.TeamRole) {
			return nil, &requestError{status: 403, message: "Viewers cannot toggle favorites on team connections"}
		}
	}

	var isFavorite bool
	if err := s.DB.QueryRow(ctx, `UPDATE "Connection" SET "isFavorite" = NOT "isFavorite" WHERE id = $1 RETURNING "isFavorite"`, connectionID).Scan(&isFavorite); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("toggle favorite: %w", err)
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "CONNECTION_FAVORITE", connectionID, map[string]any{"isFavorite": isFavorite}, ip)
	return map[string]any{"id": connectionID, "isFavorite": isFavorite}, nil
}
