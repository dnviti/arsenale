package syncprofiles

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) GetLogs(ctx context.Context, profileID, tenantID string, page, limit int) (syncLogsResponse, error) {
	if _, err := uuid.Parse(strings.TrimSpace(profileID)); err != nil {
		return syncLogsResponse{}, &requestError{status: http.StatusBadRequest, message: "invalid sync profile id"}
	}
	var exists bool
	if err := s.DB.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM "SyncProfile" WHERE id = $1 AND "tenantId" = $2)`, profileID, tenantID).Scan(&exists); err != nil {
		return syncLogsResponse{}, fmt.Errorf("check sync profile: %w", err)
	}
	if !exists {
		return syncLogsResponse{}, pgx.ErrNoRows
	}

	offset := (page - 1) * limit
	rows, err := s.DB.Query(ctx, `
SELECT id, "syncProfileId", status::text, "startedAt", "completedAt",
       CASE WHEN details IS NULL THEN NULL ELSE details::text END, "triggeredBy"
FROM "SyncLog"
WHERE "syncProfileId" = $1
ORDER BY "startedAt" DESC
OFFSET $2 LIMIT $3
`, profileID, offset, limit)
	if err != nil {
		return syncLogsResponse{}, fmt.Errorf("list sync logs: %w", err)
	}
	defer rows.Close()

	logs := make([]syncLogEntry, 0)
	for rows.Next() {
		var (
			item        syncLogEntry
			completedAt sql.NullTime
			details     sql.NullString
		)
		if err := rows.Scan(&item.ID, &item.SyncProfileID, &item.Status, &item.StartedAt, &completedAt, &details, &item.TriggeredBy); err != nil {
			return syncLogsResponse{}, fmt.Errorf("scan sync log: %w", err)
		}
		if completedAt.Valid {
			item.CompletedAt = &completedAt.Time
		}
		if details.Valid {
			item.Details = json.RawMessage(details.String)
		}
		logs = append(logs, item)
	}
	if err := rows.Err(); err != nil {
		return syncLogsResponse{}, fmt.Errorf("iterate sync logs: %w", err)
	}

	var total int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*)::int FROM "SyncLog" WHERE "syncProfileId" = $1`, profileID).Scan(&total); err != nil {
		return syncLogsResponse{}, fmt.Errorf("count sync logs: %w", err)
	}
	return syncLogsResponse{Logs: logs, Total: total, Page: page, Limit: limit}, nil
}
