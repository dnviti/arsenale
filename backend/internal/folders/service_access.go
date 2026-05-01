package folders

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) resolveAccess(ctx context.Context, userID, tenantID, folderID string) (accessResult, error) {
	if folder, err := s.loadPersonalFolder(ctx, folderID, userID); err == nil {
		return accessResult{Folder: folder}, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return accessResult{}, err
	}

	folder, role, err := s.loadTeamFolder(ctx, folderID, userID, tenantID)
	if err != nil {
		return accessResult{}, err
	}
	if !canManageTeam(role) {
		return accessResult{}, pgx.ErrNoRows
	}
	return accessResult{Folder: folder, TeamRole: &role}, nil
}

func (s Service) loadPersonalFolder(ctx context.Context, folderID, userID string) (folderResponse, error) {
	var folder folderResponse
	var parentID sql.NullString
	if err := s.DB.QueryRow(ctx, `
SELECT id, name, "parentId", "sortOrder", "createdAt", "updatedAt"
FROM "Folder"
WHERE id = $1 AND "userId" = $2 AND "teamId" IS NULL
`, folderID, userID).Scan(&folder.ID, &folder.Name, &parentID, &folder.SortOrder, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
		return folderResponse{}, err
	}
	if parentID.Valid {
		folder.ParentID = &parentID.String
	}
	folder.Scope = "private"
	return folder, nil
}

func (s Service) loadTeamFolder(ctx context.Context, folderID, userID, tenantID string) (folderResponse, string, error) {
	var folder folderResponse
	var parentID, teamID, teamName sql.NullString
	var role string
	err := s.DB.QueryRow(ctx, `
SELECT f.id, f.name, f."parentId", f."sortOrder", f."teamId", t.name, f."createdAt", f."updatedAt", tm.role::text
FROM "Folder" f
JOIN "Team" t ON t.id = f."teamId"
JOIN "TeamMember" tm ON tm."teamId" = f."teamId"
WHERE f.id = $1
  AND tm."userId" = $2
  AND (tm."expiresAt" IS NULL OR tm."expiresAt" > NOW())
  AND ($3 = '' OR t."tenantId" = $3)
`, folderID, userID, tenantID).Scan(
		&folder.ID,
		&folder.Name,
		&parentID,
		&folder.SortOrder,
		&teamID,
		&teamName,
		&folder.CreatedAt,
		&folder.UpdatedAt,
		&role,
	)
	if err != nil {
		return folderResponse{}, "", err
	}
	if parentID.Valid {
		folder.ParentID = &parentID.String
	}
	if teamID.Valid {
		folder.TeamID = &teamID.String
	}
	if teamName.Valid {
		folder.TeamName = &teamName.String
	}
	folder.Scope = "team"
	return folder, role, nil
}

func (s Service) ensureParentExists(ctx context.Context, userID, teamID string, parentID *string) error {
	if parentID == nil || strings.TrimSpace(*parentID) == "" {
		return nil
	}
	query := `SELECT 1 FROM "Folder" WHERE id = $1 AND "userId" = $2 AND "teamId" IS NULL`
	args := []any{strings.TrimSpace(*parentID), userID}
	if teamID != "" {
		query = `SELECT 1 FROM "Folder" WHERE id = $1 AND "teamId" = $2`
		args = []any{strings.TrimSpace(*parentID), teamID}
	}
	var ok int
	if err := s.DB.QueryRow(ctx, query, args...).Scan(&ok); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &requestError{status: http.StatusNotFound, message: "Parent folder not found"}
		}
		return fmt.Errorf("check parent folder: %w", err)
	}
	return nil
}

func (s Service) requireTeamRole(ctx context.Context, userID, tenantID, teamID string, manage bool) (string, error) {
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
			if manage {
				return "", &requestError{status: http.StatusForbidden, message: "Insufficient team role to create folders"}
			}
			return "", pgx.ErrNoRows
		}
		return "", fmt.Errorf("load team role: %w", err)
	}
	if manage && !canManageTeam(role) {
		return "", &requestError{status: http.StatusForbidden, message: "Insufficient team role to create folders"}
	}
	return role, nil
}

func canManageTeam(role string) bool {
	switch role {
	case "TEAM_ADMIN", "TEAM_EDITOR":
		return true
	default:
		return false
	}
}

func presentFields(payload updatePayload) []string {
	fields := make([]string, 0, 2)
	if payload.Name.Present {
		fields = append(fields, "name")
	}
	if payload.ParentID.Present {
		fields = append(fields, "parentId")
	}
	return fields
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func (s Service) lookupTeamName(ctx context.Context, teamID string) (*string, error) {
	var name string
	if err := s.DB.QueryRow(ctx, `SELECT name FROM "Team" WHERE id = $1`, teamID).Scan(&name); err != nil {
		return nil, err
	}
	return &name, nil
}

func (s Service) insertAuditLog(ctx context.Context, userID, action, targetID string, details map[string]any, ip *string) error {
	var payload any
	if details != nil {
		payload = details
	}
	_, err := s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details, "ipAddress")
VALUES ($1, $2, $3::"AuditAction", 'Folder', $4, $5, $6)
`, uuid.NewString(), userID, action, targetID, payload, ip)
	return err
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
	if host, _, err := net.SplitHostPort(value); err == nil {
		return host
	}
	return strings.TrimSpace(value)
}
