package syncprofiles

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (s Service) TestConnection(ctx context.Context, profileID, tenantID string) (syncTestResult, error) {
	runtime, err := s.loadProfileRuntime(ctx, profileID, tenantID)
	if err != nil {
		return syncTestResult{}, err
	}
	apiToken, err := decryptValue(s.ServerEncryptionKey, runtime.EncryptedAPIToken, runtime.APITokenIV, runtime.APITokenTag)
	if err != nil {
		return syncTestResult{OK: false, Error: "failed to decrypt sync API token"}, nil
	}
	ok, message := testNetBoxConnection(runtime.Profile.Config, apiToken)
	result := syncTestResult{OK: ok}
	if message != "" {
		result.Error = message
	}
	return result, nil
}

func (s Service) TriggerSync(ctx context.Context, userID, tenantID, profileID string, dryRun bool) (triggerSyncResponse, error) {
	runtime, err := s.loadProfileRuntime(ctx, profileID, tenantID)
	if err != nil {
		return triggerSyncResponse{}, err
	}

	apiToken, err := decryptValue(s.ServerEncryptionKey, runtime.EncryptedAPIToken, runtime.APITokenIV, runtime.APITokenTag)
	if err != nil {
		return triggerSyncResponse{}, fmt.Errorf("decrypt sync api token: %w", err)
	}

	syncLogID, err := s.beginSyncRun(ctx, userID, runtime.Profile.ID, dryRun)
	if err != nil {
		return triggerSyncResponse{}, err
	}

	plan, err := s.buildSyncPlan(ctx, runtime.Profile, apiToken)
	if err != nil {
		if dryRun {
			plan = discoveryErrorPlan(runtime.Profile, err)
			logDetails := map[string]any{
				"dryRun":   true,
				"toCreate": len(plan.ToCreate),
				"toUpdate": len(plan.ToUpdate),
				"toSkip":   len(plan.ToSkip),
				"errors":   len(plan.Errors),
			}
			profileDetails := map[string]any{
				"dryRun": true,
				"errors": []string{err.Error()},
			}
			if err := s.completeSyncRun(ctx, userID, syncLogID, runtime.Profile.ID, "ERROR", logDetails, profileDetails); err != nil {
				return triggerSyncResponse{}, err
			}
			return triggerSyncResponse{Plan: plan}, nil
		}
		if recordErr := s.failSyncRun(ctx, userID, syncLogID, runtime.Profile.ID, err); recordErr != nil {
			return triggerSyncResponse{}, fmt.Errorf("sync failed: %w (recording error failed: %v)", err, recordErr)
		}
		return triggerSyncResponse{}, err
	}

	if dryRun {
		logDetails := map[string]any{
			"dryRun":   true,
			"toCreate": len(plan.ToCreate),
			"toUpdate": len(plan.ToUpdate),
			"toSkip":   len(plan.ToSkip),
			"errors":   len(plan.Errors),
		}
		if err := s.completeSyncRun(ctx, userID, syncLogID, runtime.Profile.ID, "SUCCESS", logDetails, map[string]any{"dryRun": true}); err != nil {
			return triggerSyncResponse{}, err
		}
		return triggerSyncResponse{Plan: plan}, nil
	}

	result := s.executeSyncPlan(ctx, plan, runtime.Profile)
	status := "SUCCESS"
	if result.Failed > 0 {
		status = "PARTIAL"
	}
	logDetails := map[string]any{
		"created": result.Created,
		"updated": result.Updated,
		"skipped": result.Skipped,
		"failed":  result.Failed,
		"errors":  result.Errors,
	}
	profileDetails := map[string]any{
		"created": result.Created,
		"updated": result.Updated,
		"skipped": result.Skipped,
		"failed":  result.Failed,
	}
	if err := s.completeSyncRun(ctx, userID, syncLogID, runtime.Profile.ID, status, logDetails, profileDetails); err != nil {
		return triggerSyncResponse{}, err
	}
	return triggerSyncResponse{Plan: plan, Result: &result}, nil
}

func (s Service) buildSyncPlan(ctx context.Context, profile syncProfileResponse, apiToken string) (syncPlan, error) {
	devices, err := discoverNetBoxDevices(profile.Config, apiToken)
	if err != nil {
		return syncPlan{}, err
	}
	return s.buildPlan(ctx, profile.ID, devices, profile.Config.ConflictStrategy)
}

func discoveryErrorPlan(profile syncProfileResponse, cause error) syncPlan {
	return syncPlan{
		ToCreate: []discoveredDevice{},
		ToUpdate: []syncPlanUpdateItem{},
		ToSkip:   []syncPlanSkipItem{},
		Errors: []syncPlanErrorItem{
			{
				Device: discoveredDevice{
					ExternalID: "provider:discovery",
					Name:       profile.Name,
					Protocol:   profile.Provider,
					Metadata:   map[string]any{"provider": profile.Provider},
				},
				Error: cause.Error(),
			},
		},
	}
}

func (s Service) loadProfileRuntime(ctx context.Context, profileID, tenantID string) (syncProfileRuntime, error) {
	if _, err := uuid.Parse(strings.TrimSpace(profileID)); err != nil {
		return syncProfileRuntime{}, &requestError{status: http.StatusBadRequest, message: "invalid sync profile id"}
	}
	row := s.DB.QueryRow(ctx, `
SELECT id, name, "tenantId", provider::text, config::text, "cronExpression", enabled, "teamId",
       "lastSyncAt", "lastSyncStatus"::text,
       CASE WHEN "lastSyncDetails" IS NULL THEN NULL ELSE "lastSyncDetails"::text END,
       "createdById", "createdAt", "updatedAt", "encryptedApiToken",
       "apiTokenIV", "apiTokenTag"
FROM "SyncProfile"
WHERE id = $1 AND "tenantId" = $2
`, profileID, tenantID)

	var (
		profile           syncProfileResponse
		configText        string
		cronExpression    sql.NullString
		teamID            sql.NullString
		lastSyncAt        sql.NullTime
		lastSyncStatus    sql.NullString
		lastSyncDetails   sql.NullString
		encryptedAPIToken sql.NullString
		apiTokenIV        sql.NullString
		apiTokenTag       sql.NullString
	)
	if err := row.Scan(
		&profile.ID,
		&profile.Name,
		&profile.TenantID,
		&profile.Provider,
		&configText,
		&cronExpression,
		&profile.Enabled,
		&teamID,
		&lastSyncAt,
		&lastSyncStatus,
		&lastSyncDetails,
		&profile.CreatedByID,
		&profile.CreatedAt,
		&profile.UpdatedAt,
		&encryptedAPIToken,
		&apiTokenIV,
		&apiTokenTag,
	); err != nil {
		return syncProfileRuntime{}, err
	}
	if err := json.Unmarshal([]byte(configText), &profile.Config); err != nil {
		return syncProfileRuntime{}, fmt.Errorf("decode sync profile config: %w", err)
	}
	normalizeConfig(&profile.Config)
	if cronExpression.Valid {
		profile.CronExpression = &cronExpression.String
	}
	if teamID.Valid {
		profile.TeamID = &teamID.String
	}
	if lastSyncAt.Valid {
		profile.LastSyncAt = &lastSyncAt.Time
	}
	if lastSyncStatus.Valid {
		profile.LastSyncStatus = &lastSyncStatus.String
	}
	if lastSyncDetails.Valid {
		profile.LastSyncDetails = json.RawMessage(lastSyncDetails.String)
	}
	profile.HasAPIToken = encryptedAPIToken.Valid && encryptedAPIToken.String != ""
	return syncProfileRuntime{
		Profile:           profile,
		EncryptedAPIToken: encryptedAPIToken.String,
		APITokenIV:        apiTokenIV.String,
		APITokenTag:       apiTokenTag.String,
	}, nil
}
