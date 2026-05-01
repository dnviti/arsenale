package syncprofiles

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) beginSyncRun(ctx context.Context, userID, profileID string, dryRun bool) (string, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin sync run: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	syncLogID := uuid.NewString()
	if _, err := tx.Exec(ctx, `
INSERT INTO "SyncLog" (id, "syncProfileId", status, "startedAt", "triggeredBy")
VALUES ($1, $2, 'RUNNING'::"SyncStatus", NOW(), $3)
`, syncLogID, profileID, userID); err != nil {
		return "", fmt.Errorf("insert sync run log: %w", err)
	}

	if _, err := tx.Exec(ctx, `
UPDATE "SyncProfile"
SET "lastSyncAt" = NOW(),
    "lastSyncStatus" = 'RUNNING'::"SyncStatus",
    "updatedAt" = NOW()
WHERE id = $1
`, profileID); err != nil {
		return "", fmt.Errorf("update sync profile running status: %w", err)
	}

	if err := insertAuditLog(ctx, tx, userID, "SYNC_START", profileID, map[string]any{"dryRun": dryRun}); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit sync run start: %w", err)
	}
	return syncLogID, nil
}

func (s Service) completeSyncRun(ctx context.Context, userID, syncLogID, profileID, status string, logDetails, profileDetails map[string]any) error {
	logDetailsJSON, err := json.Marshal(logDetails)
	if err != nil {
		return fmt.Errorf("marshal sync log details: %w", err)
	}
	profileDetailsJSON, err := json.Marshal(profileDetails)
	if err != nil {
		return fmt.Errorf("marshal sync profile details: %w", err)
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin sync completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
UPDATE "SyncLog"
SET status = $2::"SyncStatus",
    "completedAt" = NOW(),
    details = $3::jsonb
WHERE id = $1
`, syncLogID, status, string(logDetailsJSON)); err != nil {
		return fmt.Errorf("update sync log completion: %w", err)
	}

	if _, err := tx.Exec(ctx, `
UPDATE "SyncProfile"
SET "lastSyncStatus" = $2::"SyncStatus",
    "lastSyncDetails" = $3::jsonb,
    "updatedAt" = NOW()
WHERE id = $1
`, profileID, status, string(profileDetailsJSON)); err != nil {
		return fmt.Errorf("update sync profile completion: %w", err)
	}

	auditAction := "SYNC_COMPLETE"
	if status != "SUCCESS" {
		auditAction = "SYNC_ERROR"
	}
	if err := insertAuditLog(ctx, tx, userID, auditAction, profileID, logDetails); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sync completion: %w", err)
	}
	return nil
}

func (s Service) failSyncRun(ctx context.Context, userID, syncLogID, profileID string, cause error) error {
	details := map[string]any{"error": cause.Error()}
	return s.completeSyncRun(ctx, userID, syncLogID, profileID, "ERROR", details, details)
}

func (s Service) executeSyncPlan(ctx context.Context, plan syncPlan, profile syncProfileResponse) syncResultResponse {
	result := syncResultResponse{
		Skipped: len(plan.ToSkip),
		Failed:  len(plan.Errors),
		Errors:  make([]syncResultError, 0, len(plan.Errors)),
	}
	for _, entry := range plan.Errors {
		result.Errors = append(result.Errors, syncResultError{
			ExternalID: entry.Device.ExternalID,
			Name:       entry.Device.Name,
			Error:      entry.Error,
		})
	}

	for _, device := range plan.ToCreate {
		if err := s.createSyncedConnection(ctx, profile, device); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, syncResultError{
				ExternalID: device.ExternalID,
				Name:       device.Name,
				Error:      err.Error(),
			})
			continue
		}
		result.Created++
	}

	for _, entry := range plan.ToUpdate {
		if err := s.updateSyncedConnection(ctx, profile, entry); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, syncResultError{
				ExternalID: entry.Device.ExternalID,
				Name:       entry.Device.Name,
				Error:      err.Error(),
			})
			continue
		}
		result.Updated++
	}

	return result
}

func (s Service) createSyncedConnection(ctx context.Context, profile syncProfileResponse, device discoveredDevice) error {
	folderID, err := s.resolveSyncFolderID(ctx, device, profile.CreatedByID, profile.TeamID)
	if err != nil {
		return err
	}

	if _, err := s.DB.Exec(ctx, `
INSERT INTO "Connection" (
	id, "name", type, host, port, "folderId", "teamId", description, "userId",
	"syncProfileId", "externalId", "createdAt", "updatedAt"
)
VALUES (
	$1, $2, $3::"ConnectionType", $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW()
)
`, uuid.NewString(), strings.TrimSpace(device.Name), device.Protocol, strings.TrimSpace(device.Host), device.Port,
		nullableString(folderID), nullableString(profile.TeamID), nullableString(normalizeOptionalStringPtr(device.Description)),
		profile.CreatedByID, profile.ID, device.ExternalID); err != nil {
		return fmt.Errorf("create synced connection: %w", err)
	}
	return nil
}

func (s Service) updateSyncedConnection(ctx context.Context, profile syncProfileResponse, entry syncPlanUpdateItem) error {
	folderID, err := s.resolveSyncFolderID(ctx, entry.Device, profile.CreatedByID, profile.TeamID)
	if err != nil {
		return err
	}

	commandTag, err := s.DB.Exec(ctx, `
UPDATE "Connection"
SET "name" = $2,
    type = $3::"ConnectionType",
    host = $4,
    port = $5,
    description = $6,
    "folderId" = $7,
    "updatedAt" = NOW()
WHERE id = $1
`, entry.ConnectionID, strings.TrimSpace(entry.Device.Name), entry.Device.Protocol, strings.TrimSpace(entry.Device.Host), entry.Device.Port,
		nullableString(normalizeOptionalStringPtr(entry.Device.Description)), nullableString(folderID))
	if err != nil {
		return fmt.Errorf("update synced connection: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (s Service) resolveSyncFolderID(ctx context.Context, device discoveredDevice, userID string, teamID *string) (*string, error) {
	if strings.TrimSpace(device.SiteName) == "" {
		return nil, nil
	}

	siteID, err := s.getOrCreateSyncFolder(ctx, device.SiteName, userID, teamID, nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(device.RackName) == "" {
		return &siteID, nil
	}

	rackID, err := s.getOrCreateSyncFolder(ctx, device.RackName, userID, teamID, &siteID)
	if err != nil {
		return nil, err
	}
	return &rackID, nil
}

func (s Service) getOrCreateSyncFolder(ctx context.Context, name, userID string, teamID, parentID *string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("folder name is required")
	}

	var folderID string
	var err error
	if teamID != nil {
		err = s.DB.QueryRow(ctx, `
SELECT id
FROM "Folder"
WHERE name = $1
  AND "teamId" = $2
  AND "parentId" IS NOT DISTINCT FROM $3
LIMIT 1
`, trimmed, *teamID, nullableString(parentID)).Scan(&folderID)
	} else {
		err = s.DB.QueryRow(ctx, `
SELECT id
FROM "Folder"
WHERE name = $1
  AND "userId" = $2
  AND "teamId" IS NULL
  AND "parentId" IS NOT DISTINCT FROM $3
LIMIT 1
`, trimmed, userID, nullableString(parentID)).Scan(&folderID)
	}
	if err == nil {
		return folderID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("find sync folder: %w", err)
	}

	folderID = uuid.NewString()
	if _, err := s.DB.Exec(ctx, `
INSERT INTO "Folder" (id, name, "parentId", "userId", "teamId", "sortOrder", "createdAt", "updatedAt")
VALUES ($1, $2, $3, $4, $5, 0, NOW(), NOW())
`, folderID, trimmed, nullableString(parentID), userID, nullableString(teamID)); err != nil {
		return "", fmt.Errorf("create sync folder: %w", err)
	}
	return folderID, nil
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

func normalizeOptionalStringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
