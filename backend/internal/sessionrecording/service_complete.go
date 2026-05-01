package sessionrecording

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CompleteRecording(ctx context.Context, db *pgxpool.Pool, recordingID string) error {
	if db == nil || strings.TrimSpace(recordingID) == "" {
		return nil
	}

	recording, err := loadRecording(ctx, db, recordingID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return fmt.Errorf("load recording: %w", err)
	}
	if recording.Status != "RECORDING" {
		return nil
	}

	stat, err := os.Stat(recording.FilePath)
	if err != nil {
		if _, updateErr := db.Exec(
			ctx,
			`UPDATE "SessionRecording"
			 SET status = 'ERROR'::"RecordingStatus",
			     "completedAt" = $2
			 WHERE id = $1`,
			recording.ID,
			time.Now().UTC(),
		); updateErr != nil {
			return fmt.Errorf("mark recording error: %w", updateErr)
		}
		return nil
	}
	if err := ensureRecordingReadable(recording.FilePath, stat); err != nil {
		if _, updateErr := db.Exec(
			ctx,
			`UPDATE "SessionRecording"
			 SET status = 'ERROR'::"RecordingStatus",
			     "completedAt" = $2
			 WHERE id = $1`,
			recording.ID,
			time.Now().UTC(),
		); updateErr != nil {
			return fmt.Errorf("mark recording error after chmod failure: %w", updateErr)
		}
		return fmt.Errorf("ensure recording readable: %w", err)
	}

	completedAt := time.Now().UTC()
	duration := int(completedAt.Sub(recording.CreatedAt).Seconds())
	if duration < 0 {
		duration = 0
	}
	if _, err := db.Exec(
		ctx,
		`UPDATE "SessionRecording"
		 SET status = 'COMPLETE'::"RecordingStatus",
		     "fileSize" = $2,
		     duration = $3,
		     "completedAt" = $4
		 WHERE id = $1`,
		recording.ID,
		int(stat.Size()),
		duration,
		completedAt,
	); err != nil {
		return fmt.Errorf("complete recording: %w", err)
	}

	label := recording.Protocol
	if recording.ConnectionName != nil && *recording.ConnectionName != "" {
		label = *recording.ConnectionName
	}
	if _, err := db.Exec(
		ctx,
		`INSERT INTO "Notification" (id, "userId", type, message, "read", "relatedId", "createdAt")
		 VALUES ($1, $2, 'RECORDING_READY'::"NotificationType", $3, false, $4, NOW())`,
		uuid.NewString(),
		recording.UserID,
		fmt.Sprintf("Your %s session recording is ready", label),
		recording.ID,
	); err != nil {
		return fmt.Errorf("create recording notification: %w", err)
	}

	return nil
}

func insertRecordingAudit(ctx context.Context, db *pgxpool.Pool, recordingID, userID, connectionID, protocol string) {
	if db == nil {
		return
	}

	details, err := json.Marshal(map[string]any{
		"recordingId":  recordingID,
		"protocol":     protocol,
		"connectionId": connectionID,
	})
	if err != nil {
		return
	}

	_, _ = db.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details)
VALUES ($1, $2, 'RECORDING_START'::"AuditAction", 'Recording', $3, $4::jsonb)
`, uuid.NewString(), strings.TrimSpace(userID), recordingID, string(details))
}

func ensureRecordingReadable(filePath string, stat os.FileInfo) error {
	mode := stat.Mode().Perm()
	readableMode := mode | 0o044
	if readableMode == mode {
		return nil
	}
	return os.Chmod(filePath, readableMode)
}

func loadRecording(ctx context.Context, db *pgxpool.Pool, recordingID string) (*recordingRecord, error) {
	row := db.QueryRow(
		ctx,
		`SELECT sr.id, sr."userId", sr."connectionId", sr.protocol::text, sr."filePath",
		        sr.status::text, sr."createdAt", c.name
		 FROM "SessionRecording" sr
		 LEFT JOIN "Connection" c ON c.id = sr."connectionId"
		 WHERE sr.id = $1`,
		recordingID,
	)

	var record recordingRecord
	if err := row.Scan(
		&record.ID,
		&record.UserID,
		&record.ConnectionID,
		&record.Protocol,
		&record.FilePath,
		&record.Status,
		&record.CreatedAt,
		&record.ConnectionName,
	); err != nil {
		return nil, err
	}
	return &record, nil
}
