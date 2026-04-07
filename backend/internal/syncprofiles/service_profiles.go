package syncprofiles

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) ListProfiles(ctx context.Context, tenantID string) ([]syncProfileResponse, error) {
	rows, err := s.DB.Query(ctx, `
SELECT id, name, "tenantId", provider::text, config::text, "cronExpression", enabled, "teamId",
       "lastSyncAt", "lastSyncStatus"::text,
       CASE WHEN "lastSyncDetails" IS NULL THEN NULL ELSE "lastSyncDetails"::text END,
       "createdById", "createdAt", "updatedAt", "encryptedApiToken"
FROM "SyncProfile"
WHERE "tenantId" = $1
ORDER BY "createdAt" DESC
`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list sync profiles: %w", err)
	}
	defer rows.Close()

	result := make([]syncProfileResponse, 0)
	for rows.Next() {
		item, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sync profiles: %w", err)
	}
	return result, nil
}

func (s Service) GetProfile(ctx context.Context, profileID, tenantID string) (syncProfileResponse, error) {
	if _, err := uuid.Parse(strings.TrimSpace(profileID)); err != nil {
		return syncProfileResponse{}, &requestError{status: http.StatusBadRequest, message: "invalid sync profile id"}
	}
	row := s.DB.QueryRow(ctx, `
SELECT id, name, "tenantId", provider::text, config::text, "cronExpression", enabled, "teamId",
       "lastSyncAt", "lastSyncStatus"::text,
       CASE WHEN "lastSyncDetails" IS NULL THEN NULL ELSE "lastSyncDetails"::text END,
       "createdById", "createdAt", "updatedAt", "encryptedApiToken"
FROM "SyncProfile"
WHERE id = $1 AND "tenantId" = $2
`, profileID, tenantID)
	return scanProfile(row)
}

func (s Service) CreateProfile(ctx context.Context, claims authn.Claims, payload createPayload) (syncProfileResponse, error) {
	if len(s.ServerEncryptionKey) == 0 {
		return syncProfileResponse{}, fmt.Errorf("server encryption key is unavailable")
	}

	config, normalizedTeamID, err := s.validateCreatePayload(ctx, claims.TenantID, payload)
	if err != nil {
		return syncProfileResponse{}, err
	}
	encrypted, err := encryptValue(s.ServerEncryptionKey, payload.APIToken)
	if err != nil {
		return syncProfileResponse{}, fmt.Errorf("encrypt sync API token: %w", err)
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return syncProfileResponse{}, fmt.Errorf("marshal sync config: %w", err)
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return syncProfileResponse{}, fmt.Errorf("begin sync profile create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
INSERT INTO "SyncProfile" (
  id, name, "tenantId", provider, config, "encryptedApiToken", "apiTokenIV", "apiTokenTag",
  "cronExpression", enabled, "teamId", "createdById", "createdAt", "updatedAt"
)
VALUES (
  $1, $2, $3, $4::"SyncProvider", $5::jsonb, $6, $7, $8, $9, true, $10, $11, NOW(), NOW()
)
RETURNING id, name, "tenantId", provider::text, config::text, "cronExpression", enabled, "teamId",
          "lastSyncAt", "lastSyncStatus"::text,
          CASE WHEN "lastSyncDetails" IS NULL THEN NULL ELSE "lastSyncDetails"::text END,
          "createdById", "createdAt", "updatedAt", "encryptedApiToken"
`, uuid.NewString(), payload.Name, claims.TenantID, payload.Provider, string(configJSON), encrypted.Ciphertext, encrypted.IV, encrypted.Tag, normalizeCronExpression(payload.CronExpression), normalizedTeamID, claims.UserID)

	item, err := scanProfile(row)
	if err != nil {
		return syncProfileResponse{}, err
	}
	if err := insertAuditLog(ctx, tx, claims.UserID, "SYNC_PROFILE_CREATE", item.ID, map[string]any{
		"name":     item.Name,
		"provider": item.Provider,
	}); err != nil {
		return syncProfileResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return syncProfileResponse{}, fmt.Errorf("commit sync profile create: %w", err)
	}
	s.reconcileSchedule(item.ID, item.CronExpression, item.Enabled)
	return item, nil
}

func (s Service) UpdateProfile(ctx context.Context, claims authn.Claims, profileID string, payload updatePayload) (syncProfileResponse, error) {
	current, encryptedPresent, err := s.loadProfileRecord(ctx, profileID, claims.TenantID)
	if err != nil {
		return syncProfileResponse{}, err
	}

	updatedConfig := current.Config
	if payload.URL != nil {
		updatedConfig.URL = strings.TrimSpace(*payload.URL)
	}
	if payload.Filters != nil {
		updatedConfig.Filters = cloneStringMap(*payload.Filters)
	}
	if payload.PlatformMapping != nil {
		updatedConfig.PlatformMapping = cloneStringMap(*payload.PlatformMapping)
	}
	if payload.DefaultProtocol != nil {
		updatedConfig.DefaultProtocol = strings.TrimSpace(*payload.DefaultProtocol)
	}
	if payload.DefaultPort != nil {
		updatedConfig.DefaultPort = cloneIntMap(*payload.DefaultPort)
	}
	if payload.ConflictStrategy != nil {
		updatedConfig.ConflictStrategy = strings.TrimSpace(*payload.ConflictStrategy)
	}
	if err := validateConfig(updatedConfig); err != nil {
		return syncProfileResponse{}, err
	}

	var (
		nameValue      = current.Name
		enabled        = current.Enabled
		teamID         = current.TeamID
		cronExpression = current.CronExpression
	)
	if payload.Name != nil {
		nameValue = strings.TrimSpace(*payload.Name)
	}
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}
	if payload.TeamID.Present {
		teamID, err = s.normalizeTeamID(ctx, claims.TenantID, payload.TeamID.Value)
		if err != nil {
			return syncProfileResponse{}, err
		}
	}
	if payload.CronExpression.Present {
		cronExpression = normalizeCronExpression(payload.CronExpression.Value)
	}
	if len(nameValue) == 0 || len(nameValue) > 100 {
		return syncProfileResponse{}, &requestError{status: http.StatusBadRequest, message: "name must be between 1 and 100 characters"}
	}

	configJSON, err := json.Marshal(updatedConfig)
	if err != nil {
		return syncProfileResponse{}, fmt.Errorf("marshal sync config: %w", err)
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return syncProfileResponse{}, fmt.Errorf("begin sync profile update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	apiTokenCipher := ""
	apiTokenIV := ""
	apiTokenTag := ""
	hasNewToken := payload.APIToken != nil && strings.TrimSpace(*payload.APIToken) != ""
	if hasNewToken {
		if len(s.ServerEncryptionKey) == 0 {
			return syncProfileResponse{}, fmt.Errorf("server encryption key is unavailable")
		}
		encrypted, err := encryptValue(s.ServerEncryptionKey, strings.TrimSpace(*payload.APIToken))
		if err != nil {
			return syncProfileResponse{}, fmt.Errorf("encrypt sync API token: %w", err)
		}
		apiTokenCipher = encrypted.Ciphertext
		apiTokenIV = encrypted.IV
		apiTokenTag = encrypted.Tag
	}

	row := tx.QueryRow(ctx, `
UPDATE "SyncProfile"
SET name = $3,
    config = $4::jsonb,
    enabled = $5,
    "teamId" = $6,
    "cronExpression" = $7,
    "encryptedApiToken" = CASE WHEN $8 = '' THEN "encryptedApiToken" ELSE $8 END,
    "apiTokenIV" = CASE WHEN $9 = '' THEN "apiTokenIV" ELSE $9 END,
    "apiTokenTag" = CASE WHEN $10 = '' THEN "apiTokenTag" ELSE $10 END,
    "updatedAt" = NOW()
WHERE id = $1 AND "tenantId" = $2
RETURNING id, name, "tenantId", provider::text, config::text, "cronExpression", enabled, "teamId",
          "lastSyncAt", "lastSyncStatus"::text,
          CASE WHEN "lastSyncDetails" IS NULL THEN NULL ELSE "lastSyncDetails"::text END,
          "createdById", "createdAt", "updatedAt", "encryptedApiToken"
`, profileID, claims.TenantID, nameValue, string(configJSON), enabled, teamID, cronExpression, apiTokenCipher, apiTokenIV, apiTokenTag)

	item, err := scanProfile(row)
	if err != nil {
		return syncProfileResponse{}, err
	}
	if !hasNewToken {
		item.HasAPIToken = encryptedPresent
	}
	if err := insertAuditLog(ctx, tx, claims.UserID, "SYNC_PROFILE_UPDATE", item.ID, map[string]any{
		"name": item.Name,
	}); err != nil {
		return syncProfileResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return syncProfileResponse{}, fmt.Errorf("commit sync profile update: %w", err)
	}
	s.reconcileSchedule(item.ID, item.CronExpression, item.Enabled)
	return item, nil
}

func (s Service) DeleteProfile(ctx context.Context, claims authn.Claims, profileID string) error {
	if _, _, err := s.loadProfileRecord(ctx, profileID, claims.TenantID); err != nil {
		return err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin sync profile delete: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM "SyncLog" WHERE "syncProfileId" = $1`, profileID); err != nil {
		return fmt.Errorf("delete sync logs: %w", err)
	}
	commandTag, err := tx.Exec(ctx, `DELETE FROM "SyncProfile" WHERE id = $1 AND "tenantId" = $2`, profileID, claims.TenantID)
	if err != nil {
		return fmt.Errorf("delete sync profile: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	if err := insertAuditLog(ctx, tx, claims.UserID, "SYNC_PROFILE_DELETE", profileID, nil); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sync profile delete: %w", err)
	}
	s.unregisterSchedule(profileID)
	return nil
}
