package syncprofiles

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

func scanProfile(row interface{ Scan(dest ...any) error }) (syncProfileResponse, error) {
	var (
		item              syncProfileResponse
		configText        string
		cronExpression    sql.NullString
		teamID            sql.NullString
		lastSyncAt        sql.NullTime
		lastSyncStatus    sql.NullString
		lastSyncDetails   sql.NullString
		encryptedAPIToken sql.NullString
	)
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.TenantID,
		&item.Provider,
		&configText,
		&cronExpression,
		&item.Enabled,
		&teamID,
		&lastSyncAt,
		&lastSyncStatus,
		&lastSyncDetails,
		&item.CreatedByID,
		&item.CreatedAt,
		&item.UpdatedAt,
		&encryptedAPIToken,
	); err != nil {
		return syncProfileResponse{}, err
	}
	if err := json.Unmarshal([]byte(configText), &item.Config); err != nil {
		return syncProfileResponse{}, fmt.Errorf("decode sync profile config: %w", err)
	}
	normalizeConfig(&item.Config)
	if cronExpression.Valid {
		item.CronExpression = &cronExpression.String
	}
	if teamID.Valid {
		item.TeamID = &teamID.String
	}
	if lastSyncAt.Valid {
		item.LastSyncAt = &lastSyncAt.Time
	}
	if lastSyncStatus.Valid {
		item.LastSyncStatus = &lastSyncStatus.String
	}
	if lastSyncDetails.Valid {
		item.LastSyncDetails = json.RawMessage(lastSyncDetails.String)
	}
	item.HasAPIToken = encryptedAPIToken.Valid && encryptedAPIToken.String != ""
	return item, nil
}
