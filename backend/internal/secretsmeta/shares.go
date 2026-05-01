package secretsmeta

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/credentialresolver"
)

type secretShareItem struct {
	ID         string `json:"id"`
	UserID     string `json:"userId"`
	Email      string `json:"email"`
	Permission string `json:"permission"`
	CreatedAt  string `json:"createdAt"`
}

type externalShareListItem struct {
	ID             string `json:"id"`
	SecretName     string `json:"secretName"`
	SecretType     string `json:"secretType"`
	HasPin         bool   `json:"hasPin"`
	ExpiresAt      string `json:"expiresAt"`
	MaxAccessCount *int   `json:"maxAccessCount"`
	AccessCount    int    `json:"accessCount"`
	IsRevoked      bool   `json:"isRevoked"`
	CreatedAt      string `json:"createdAt"`
}

func (s Service) HandleListShares(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	items, err := s.LoadShares(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.handleResolverError(w, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, items)
}

func (s Service) HandleListExternalShares(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	items, err := s.LoadExternalShares(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"))
	if err != nil {
		var reqErr *credentialresolver.RequestError
		if errors.As(err, &reqErr) {
			app.ErrorJSON(w, http.StatusForbidden, "You do not have permission to view shares for this secret")
			return
		}
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	app.WriteJSON(w, http.StatusOK, items)
}

func (s Service) LoadShares(ctx context.Context, userID, tenantID, secretID string) ([]secretShareItem, error) {
	access, err := s.resolver().RequireManageSecret(ctx, userID, secretID, tenantID)
	if err != nil {
		return nil, err
	}
	if access.TeamID != nil && access.TeamRole != "TEAM_ADMIN" {
		return nil, &credentialresolver.RequestError{Status: http.StatusForbidden, Message: "Only team admins can view team secret shares"}
	}
	if s.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}

	rows, err := s.DB.Query(ctx, `
SELECT
	ss.id,
	u.id,
	u.email,
	ss.permission::text,
	ss."createdAt"
FROM "SharedSecret" ss
JOIN "User" u ON u.id = ss."sharedWithUserId"
WHERE ss."secretId" = $1
ORDER BY ss."createdAt" ASC
`, secretID)
	if err != nil {
		return nil, fmt.Errorf("list secret shares: %w", err)
	}
	defer rows.Close()

	items := make([]secretShareItem, 0)
	for rows.Next() {
		var (
			item      secretShareItem
			createdAt time.Time
		)
		if err := rows.Scan(&item.ID, &item.UserID, &item.Email, &item.Permission, &createdAt); err != nil {
			return nil, fmt.Errorf("scan secret share: %w", err)
		}
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate secret shares: %w", err)
	}

	return items, nil
}

func (s Service) LoadExternalShares(ctx context.Context, userID, tenantID, secretID string) ([]externalShareListItem, error) {
	if _, err := s.resolver().RequireManageSecret(ctx, userID, secretID, tenantID); err != nil {
		return nil, err
	}
	if s.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}

	rows, err := s.DB.Query(ctx, `
SELECT
	id,
	"secretName",
	"secretType",
	"hasPin",
	"expiresAt",
	"maxAccessCount",
	"accessCount",
	"isRevoked",
	"createdAt"
FROM "ExternalSecretShare"
WHERE "secretId" = $1
ORDER BY "createdAt" DESC
`, secretID)
	if err != nil {
		return nil, fmt.Errorf("list external secret shares: %w", err)
	}
	defer rows.Close()

	items := make([]externalShareListItem, 0)
	for rows.Next() {
		var (
			item      externalShareListItem
			expiresAt time.Time
			createdAt time.Time
			maxAccess sql.NullInt32
		)
		if err := rows.Scan(
			&item.ID,
			&item.SecretName,
			&item.SecretType,
			&item.HasPin,
			&expiresAt,
			&maxAccess,
			&item.AccessCount,
			&item.IsRevoked,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan external secret share: %w", err)
		}
		if maxAccess.Valid {
			value := int(maxAccess.Int32)
			item.MaxAccessCount = &value
		}
		item.ExpiresAt = expiresAt.UTC().Format(time.RFC3339Nano)
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate external secret shares: %w", err)
	}

	return items, nil
}
