package folders

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) CreateFolder(ctx context.Context, claims authn.Claims, payload createPayload, ip *string) (folderResponse, error) {
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return folderResponse{}, &requestError{status: http.StatusBadRequest, message: "name is required"}
	}

	var teamID any
	scope := "private"
	if payload.TeamID != nil && strings.TrimSpace(*payload.TeamID) != "" {
		normalized := strings.TrimSpace(*payload.TeamID)
		role, err := s.requireTeamRole(ctx, claims.UserID, claims.TenantID, normalized, true)
		if err != nil {
			return folderResponse{}, err
		}
		_ = role
		teamID = normalized
		scope = "team"
		if err := s.ensureParentExists(ctx, claims.UserID, normalized, payload.ParentID); err != nil {
			return folderResponse{}, err
		}
	} else if err := s.ensureParentExists(ctx, claims.UserID, "", payload.ParentID); err != nil {
		return folderResponse{}, err
	}

	now := time.Now()
	folderID := uuid.NewString()
	var folder folderResponse
	var parentID, createdTeamID sql.NullString
	if err := s.DB.QueryRow(ctx, `
INSERT INTO "Folder" (id, name, "parentId", "userId", "teamId", "sortOrder", "createdAt", "updatedAt")
VALUES ($1, $2, $3, $4, $5, 0, $6, $7)
RETURNING id, name, "parentId", "sortOrder", "teamId", "createdAt", "updatedAt"
`, folderID, name, nullableString(payload.ParentID), claims.UserID, teamID, now, now).Scan(
		&folder.ID,
		&folder.Name,
		&parentID,
		&folder.SortOrder,
		&createdTeamID,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	); err != nil {
		return folderResponse{}, fmt.Errorf("create folder: %w", err)
	}
	if parentID.Valid {
		folder.ParentID = &parentID.String
	}
	if createdTeamID.Valid {
		folder.TeamID = &createdTeamID.String
	}
	folder.Scope = scope
	if folder.TeamID != nil {
		teamName, err := s.lookupTeamName(ctx, *folder.TeamID)
		if err == nil {
			folder.TeamName = teamName
		}
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "CREATE_FOLDER", folder.ID, map[string]any{
		"name":   folder.Name,
		"teamId": folder.TeamID,
	}, ip)
	return folder, nil
}

func (s Service) UpdateFolder(ctx context.Context, claims authn.Claims, folderID string, payload updatePayload, ip *string) (folderResponse, error) {
	access, err := s.resolveAccess(ctx, claims.UserID, claims.TenantID, folderID)
	if err != nil {
		return folderResponse{}, err
	}

	var updates []string
	var args []any
	addUpdate := func(column string, value any) {
		updates = append(updates, fmt.Sprintf(`%s = $%d`, column, len(args)+1))
		args = append(args, value)
	}

	if payload.Name.Present {
		if payload.Name.Value == nil || strings.TrimSpace(*payload.Name.Value) == "" {
			return folderResponse{}, &requestError{status: http.StatusBadRequest, message: "name is required"}
		}
		addUpdate(`name`, strings.TrimSpace(*payload.Name.Value))
	}
	if payload.ParentID.Present {
		if payload.ParentID.Value != nil && strings.TrimSpace(*payload.ParentID.Value) == folderID {
			return folderResponse{}, &requestError{status: http.StatusBadRequest, message: "A folder cannot be its own parent"}
		}
		parentID := payload.ParentID.Value
		teamID := ""
		if access.Folder.TeamID != nil {
			teamID = *access.Folder.TeamID
		}
		if err := s.ensureParentExists(ctx, claims.UserID, teamID, parentID); err != nil {
			return folderResponse{}, err
		}
		addUpdate(`"parentId"`, nullableString(parentID))
	}

	if len(updates) == 0 {
		return access.Folder, nil
	}

	addUpdate(`"updatedAt"`, time.Now())
	args = append(args, folderID)
	query := fmt.Sprintf(`UPDATE "Folder" SET %s WHERE id = $%d`, strings.Join(updates, ", "), len(args))
	if _, err := s.DB.Exec(ctx, query, args...); err != nil {
		return folderResponse{}, fmt.Errorf("update folder: %w", err)
	}

	updated, err := s.resolveAccess(ctx, claims.UserID, claims.TenantID, folderID)
	if err != nil {
		return folderResponse{}, err
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "UPDATE_FOLDER", folderID, map[string]any{
		"fields": presentFields(payload),
	}, ip)
	return updated.Folder, nil
}

func (s Service) DeleteFolder(ctx context.Context, claims authn.Claims, folderID string, ip *string) error {
	access, err := s.resolveAccess(ctx, claims.UserID, claims.TenantID, folderID)
	if err != nil {
		return err
	}

	if access.Folder.TeamID != nil {
		if _, err := s.DB.Exec(ctx, `UPDATE "Connection" SET "folderId" = NULL WHERE "folderId" = $1 AND "teamId" = $2`, folderID, *access.Folder.TeamID); err != nil {
			return fmt.Errorf("detach team folder connections: %w", err)
		}
		if _, err := s.DB.Exec(ctx, `UPDATE "Folder" SET "parentId" = $1 WHERE "parentId" = $2 AND "teamId" = $3`, nullableString(access.Folder.ParentID), folderID, *access.Folder.TeamID); err != nil {
			return fmt.Errorf("reparent team child folders: %w", err)
		}
	} else {
		if _, err := s.DB.Exec(ctx, `UPDATE "Connection" SET "folderId" = NULL WHERE "folderId" = $1 AND "userId" = $2`, folderID, claims.UserID); err != nil {
			return fmt.Errorf("detach personal folder connections: %w", err)
		}
		if _, err := s.DB.Exec(ctx, `UPDATE "Folder" SET "parentId" = $1 WHERE "parentId" = $2 AND "userId" = $3 AND "teamId" IS NULL`, nullableString(access.Folder.ParentID), folderID, claims.UserID); err != nil {
			return fmt.Errorf("reparent personal child folders: %w", err)
		}
	}

	command, err := s.DB.Exec(ctx, `DELETE FROM "Folder" WHERE id = $1`, folderID)
	if err != nil {
		return fmt.Errorf("delete folder: %w", err)
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "DELETE_FOLDER", folderID, nil, ip)
	return nil
}
