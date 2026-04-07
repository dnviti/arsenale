package sessions

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/sessionrecording"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func scanSessionRecord(row pgx.Row) (*sessionRecord, error) {
	var record sessionRecord
	if err := row.Scan(
		&record.ID,
		&record.UserID,
		&record.ConnectionID,
		&record.Protocol,
		&record.GatewayID,
		&record.InstanceID,
		&record.IPAddress,
		&record.StartedAt,
		&record.Status,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("scan session record: %w", err)
	}
	return &record, nil
}

func loadGatewayName(ctx context.Context, tx pgx.Tx, gatewayID string) (string, error) {
	row := tx.QueryRow(ctx, `SELECT name FROM "Gateway" WHERE id = $1`, gatewayID)
	var gatewayName string
	if err := row.Scan(&gatewayName); err != nil {
		return "", err
	}
	return gatewayName, nil
}

func lookupRecordingID(ctx context.Context, tx pgx.Tx, sessionID string) (string, error) {
	row := tx.QueryRow(
		ctx,
		`SELECT id
		   FROM "SessionRecording"
		  WHERE "sessionId" = $1
		  ORDER BY "createdAt" DESC
		  LIMIT 1`,
		sessionID,
	)
	var recordingID string
	if err := row.Scan(&recordingID); err != nil {
		return "", err
	}
	return recordingID, nil
}

func insertAuditLog(ctx context.Context, tx pgx.Tx, params auditLogParams) error {
	_, err := tx.Exec(
		ctx,
		`INSERT INTO "AuditLog" (
			 id, "userId", action, "targetType", "targetId", details, "ipAddress", "gatewayId", "geoCoords", flags
		 ) VALUES (
			 $1, $2, $3::"AuditAction", $4, $5, $6::jsonb, $7, $8, ARRAY[]::double precision[], ARRAY[]::text[]
		 )`,
		uuid.NewString(),
		nilIfEmpty(params.UserID),
		params.Action,
		nilIfEmpty(params.TargetType),
		nilIfEmpty(params.TargetID),
		string(params.Details),
		params.IPAddress,
		params.GatewayID,
	)
	return err
}

func formatDuration(ms int64) string {
	seconds := ms / 1000
	minutes := seconds / 60
	hours := minutes / 60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes%60, seconds%60)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds%60)
	}
	return fmt.Sprintf("%ds", seconds)
}

func nilIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringToPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringPtrValue(value *string) any {
	if value == nil || *value == "" {
		return nil
	}
	return *value
}

func shouldAutoCompleteRecording(protocol string) bool {
	switch strings.ToUpper(strings.TrimSpace(protocol)) {
	case "SSH":
		return true
	default:
		return false
	}
}

func completeSessionRecordings(ctx context.Context, db *pgxpool.Pool, recordingIDs []string) error {
	if db == nil || len(recordingIDs) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(recordingIDs))
	for _, recordingID := range recordingIDs {
		recordingID = strings.TrimSpace(recordingID)
		if recordingID == "" {
			continue
		}
		if _, ok := seen[recordingID]; ok {
			continue
		}
		seen[recordingID] = struct{}{}
		if err := sessionrecording.CompleteRecording(ctx, db, recordingID); err != nil {
			return fmt.Errorf("complete session recording %s: %w", recordingID, err)
		}
	}
	return nil
}

func emptyToNil(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}
