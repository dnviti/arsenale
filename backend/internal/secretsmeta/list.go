package secretsmeta

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

type listFilters struct {
	Scope      string
	Type       string
	TeamID     string
	FolderID   string
	Search     string
	Tags       []string
	IsFavorite *bool
}

type secretListItem struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    *string        `json:"description"`
	Type           string         `json:"type"`
	Scope          string         `json:"scope"`
	TeamID         *string        `json:"teamId"`
	TenantID       *string        `json:"tenantId"`
	FolderID       *string        `json:"folderId"`
	Metadata       map[string]any `json:"metadata"`
	Tags           []string       `json:"tags"`
	IsFavorite     bool           `json:"isFavorite"`
	PwnedCount     int            `json:"pwnedCount"`
	ExpiresAt      *time.Time     `json:"expiresAt"`
	CurrentVersion int            `json:"currentVersion"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func (s Service) HandleList(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	filters, err := parseListFilters(r)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	items, err := s.LoadList(r.Context(), claims.UserID, claims.TenantID, filters)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	app.WriteJSON(w, http.StatusOK, items)
}

func parseListFilters(r *http.Request) (listFilters, error) {
	query := r.URL.Query()
	filters := listFilters{
		Scope:    strings.TrimSpace(query.Get("scope")),
		Type:     strings.TrimSpace(query.Get("type")),
		TeamID:   strings.TrimSpace(query.Get("teamId")),
		FolderID: strings.TrimSpace(query.Get("folderId")),
		Search:   strings.TrimSpace(query.Get("search")),
	}

	if filters.Scope != "" {
		switch filters.Scope {
		case "PERSONAL", "TEAM", "TENANT":
		default:
			return listFilters{}, fmt.Errorf("scope must be PERSONAL, TEAM, or TENANT")
		}
	}

	if filters.Type != "" {
		switch filters.Type {
		case "LOGIN", "SSH_KEY", "CERTIFICATE", "API_KEY", "SECURE_NOTE":
		default:
			return listFilters{}, fmt.Errorf("type must be LOGIN, SSH_KEY, CERTIFICATE, API_KEY, or SECURE_NOTE")
		}
	}

	if raw := strings.TrimSpace(query.Get("isFavorite")); raw != "" {
		switch raw {
		case "true":
			value := true
			filters.IsFavorite = &value
		case "false":
			value := false
			filters.IsFavorite = &value
		default:
			return listFilters{}, fmt.Errorf("isFavorite must be true or false")
		}
	}

	if rawTags := strings.TrimSpace(query.Get("tags")); rawTags != "" {
		for _, item := range strings.Split(rawTags, ",") {
			tag := strings.TrimSpace(item)
			if tag != "" {
				filters.Tags = append(filters.Tags, tag)
			}
		}
	}

	return filters, nil
}

func (s Service) LoadList(ctx context.Context, userID, tenantID string, filters listFilters) ([]secretListItem, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}

	query, args, err := buildListQuery(userID, tenantID, filters)
	if err != nil {
		return nil, err
	}

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	defer rows.Close()

	items := make([]secretListItem, 0)
	for rows.Next() {
		var (
			item      secretListItem
			metadata  []byte
			expiresAt sql.NullTime
		)
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Type,
			&item.Scope,
			&item.TeamID,
			&item.TenantID,
			&item.FolderID,
			&metadata,
			&item.Tags,
			&item.IsFavorite,
			&item.PwnedCount,
			&expiresAt,
			&item.CurrentVersion,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan secret list item: %w", err)
		}

		if len(metadata) > 0 && string(metadata) != "null" {
			if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
				return nil, fmt.Errorf("decode secret metadata: %w", err)
			}
		}
		if expiresAt.Valid {
			value := expiresAt.Time.UTC()
			item.ExpiresAt = &value
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate secret list: %w", err)
	}

	return items, nil
}

func buildListQuery(userID, tenantID string, filters listFilters) (string, []any, error) {
	selectClause := `
SELECT DISTINCT
	vs.id,
	vs.name,
	vs.description,
	vs.type::text,
	vs.scope::text,
	vs."teamId",
	vs."tenantId",
	vs."folderId",
	vs.metadata,
	vs.tags,
	vs."isFavorite",
	COALESCE(vs."pwnedCount", 0),
	vs."expiresAt",
	vs."currentVersion",
	vs."createdAt",
	vs."updatedAt"
FROM "VaultSecret" vs
`

	args := []any{userID}
	whereParts := []string{}

	switch filters.Scope {
	case "PERSONAL":
		whereParts = append(whereParts, `vs.scope = 'PERSONAL'::"SecretScope"`, `vs."userId" = $1`)
	case "TEAM":
		whereParts = append(whereParts,
			`vs.scope = 'TEAM'::"SecretScope"`,
			`EXISTS (
				SELECT 1
				FROM "TeamMember" tm
				WHERE tm."teamId" = vs."teamId"
				  AND tm."userId" = $1
			)`,
		)
		if filters.TeamID != "" {
			args = append(args, filters.TeamID)
			whereParts = append(whereParts, fmt.Sprintf(`vs."teamId" = $%d`, len(args)))
		}
	case "TENANT":
		if strings.TrimSpace(tenantID) == "" {
			return "", nil, fmt.Errorf("tenant context required")
		}
		args = append(args, tenantID)
		whereParts = append(whereParts, `vs.scope = 'TENANT'::"SecretScope"`, fmt.Sprintf(`vs."tenantId" = $%d`, len(args)))
	default:
		args = append(args, tenantID)
		whereParts = append(whereParts, `(
			(vs.scope = 'PERSONAL'::"SecretScope" AND vs."userId" = $1)
			OR (vs.scope = 'TEAM'::"SecretScope" AND EXISTS (
				SELECT 1
				FROM "TeamMember" tm
				WHERE tm."teamId" = vs."teamId"
				  AND tm."userId" = $1
			))
			OR ($2 <> '' AND vs.scope = 'TENANT'::"SecretScope" AND vs."tenantId" = $2)
			OR EXISTS (
				SELECT 1
				FROM "SharedSecret" ss
				WHERE ss."secretId" = vs.id
				  AND ss."sharedWithUserId" = $1
			)
		)`)
	}

	if filters.Type != "" {
		args = append(args, filters.Type)
		whereParts = append(whereParts, fmt.Sprintf(`vs.type = $%d::"SecretType"`, len(args)))
	}
	if filters.FolderID != "" {
		args = append(args, filters.FolderID)
		whereParts = append(whereParts, fmt.Sprintf(`vs."folderId" = $%d`, len(args)))
	}
	if filters.IsFavorite != nil {
		args = append(args, *filters.IsFavorite)
		whereParts = append(whereParts, fmt.Sprintf(`vs."isFavorite" = $%d`, len(args)))
	}
	if filters.Search != "" {
		args = append(args, "%"+filters.Search+"%")
		whereParts = append(whereParts, fmt.Sprintf(`vs.name ILIKE $%d`, len(args)))
	}
	if len(filters.Tags) > 0 {
		args = append(args, filters.Tags)
		whereParts = append(whereParts, fmt.Sprintf(`vs.tags && $%d::text[]`, len(args)))
	}

	query := selectClause
	if len(whereParts) > 0 {
		query += "WHERE " + strings.Join(whereParts, "\n  AND ")
	}
	query += "\nORDER BY vs.name ASC"

	return query, args, nil
}
