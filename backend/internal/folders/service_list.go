package folders

import (
	"context"
	"database/sql"
	"fmt"
)

func (s Service) ListFolders(ctx context.Context, userID, tenantID string) (listResponse, error) {
	personalRows, err := s.DB.Query(ctx, `
SELECT id, name, "parentId", "sortOrder", "createdAt", "updatedAt"
FROM "Folder"
WHERE "userId" = $1 AND "teamId" IS NULL
ORDER BY "sortOrder" ASC, name ASC
`, userID)
	if err != nil {
		return listResponse{}, fmt.Errorf("list personal folders: %w", err)
	}
	defer personalRows.Close()

	personal := make([]folderResponse, 0)
	for personalRows.Next() {
		var folder folderResponse
		var parentID sql.NullString
		if err := personalRows.Scan(&folder.ID, &folder.Name, &parentID, &folder.SortOrder, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return listResponse{}, fmt.Errorf("scan personal folder: %w", err)
		}
		if parentID.Valid {
			folder.ParentID = &parentID.String
		}
		folder.Scope = "private"
		personal = append(personal, folder)
	}
	if err := personalRows.Err(); err != nil {
		return listResponse{}, fmt.Errorf("iterate personal folders: %w", err)
	}

	teamRows, err := s.DB.Query(ctx, `
SELECT f.id, f.name, f."parentId", f."sortOrder", f."teamId", t.name, f."createdAt", f."updatedAt"
FROM "Folder" f
JOIN "TeamMember" tm ON tm."teamId" = f."teamId"
JOIN "Team" t ON t.id = f."teamId"
WHERE tm."userId" = $1
  AND (tm."expiresAt" IS NULL OR tm."expiresAt" > NOW())
  AND ($2 = '' OR t."tenantId" = $2)
ORDER BY f."sortOrder" ASC, f.name ASC
`, userID, tenantID)
	if err != nil {
		return listResponse{}, fmt.Errorf("list team folders: %w", err)
	}
	defer teamRows.Close()

	team := make([]folderResponse, 0)
	for teamRows.Next() {
		var folder folderResponse
		var parentID, teamID, teamName sql.NullString
		if err := teamRows.Scan(&folder.ID, &folder.Name, &parentID, &folder.SortOrder, &teamID, &teamName, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return listResponse{}, fmt.Errorf("scan team folder: %w", err)
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
		team = append(team, folder)
	}
	if err := teamRows.Err(); err != nil {
		return listResponse{}, fmt.Errorf("iterate team folders: %w", err)
	}

	return normalizeListResponse(listResponse{Personal: personal, Team: team}), nil
}
