package sessionrecording

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func StartAsciicastRecording(ctx context.Context, db *pgxpool.Pool, recordingRoot, userID, connectionID, protocol, gatewayDir, sessionID string, width, height int) (Reference, error) {
	if db == nil {
		return Reference{}, fmt.Errorf("database is unavailable")
	}

	now := time.Now().UTC()
	plan, err := buildRecordingPlan(recordingRoot, userID, connectionID, protocol, "cast", gatewayDir, now)
	if err != nil {
		return Reference{}, err
	}
	if err := os.MkdirAll(plan.HostDir, 0o777); err != nil {
		return Reference{}, fmt.Errorf("create recording directory: %w", err)
	}
	_ = os.Chmod(plan.HostDir, 0o777)
	if parentDir := filepath.Dir(plan.HostDir); parentDir != "" && parentDir != "." {
		_ = os.Chmod(parentDir, 0o777)
	}

	if width <= 0 {
		width = defaultCastCols
	}
	if height <= 0 {
		height = defaultCastRows
	}

	header := map[string]any{
		"version":   2,
		"width":     width,
		"height":    height,
		"timestamp": now.Unix(),
	}
	if err := writeAsciicastHeader(plan.HostPath, header); err != nil {
		return Reference{}, err
	}

	recordingID := uuid.NewString()
	if _, err := db.Exec(ctx, `
INSERT INTO "SessionRecording" (
	id, "sessionId", "userId", "connectionId", protocol, "filePath", width, height, format, status
) VALUES (
	$1, NULLIF($2, ''), $3, $4, $5::"SessionProtocol", $6, $7, $8, 'asciicast', 'RECORDING'::"RecordingStatus"
)
`, recordingID, strings.TrimSpace(sessionID), userID, connectionID, protocol, plan.HostPath, width, height); err != nil {
		_ = os.Remove(plan.HostPath)
		return Reference{}, fmt.Errorf("insert session recording: %w", err)
	}

	insertRecordingAudit(ctx, db, recordingID, userID, connectionID, protocol)

	return Reference{
		ID:         recordingID,
		FilePath:   plan.HostPath,
		StartedAt:  now,
		Width:      width,
		Height:     height,
		Format:     "asciicast",
		Protocol:   strings.ToUpper(strings.TrimSpace(protocol)),
		Connection: strings.TrimSpace(connectionID),
	}, nil
}

func DeleteRecording(ctx context.Context, db *pgxpool.Pool, ref Reference) error {
	if db == nil || strings.TrimSpace(ref.ID) == "" {
		return nil
	}
	if _, err := db.Exec(ctx, `DELETE FROM "SessionRecording" WHERE id = $1`, ref.ID); err != nil {
		return fmt.Errorf("delete session recording: %w", err)
	}
	if strings.TrimSpace(ref.FilePath) != "" {
		_ = os.Remove(ref.FilePath)
	}
	return nil
}
