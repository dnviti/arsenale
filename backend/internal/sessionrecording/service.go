package sessionrecording

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultGatewayDir = "default"
	defaultCastCols   = 80
	defaultCastRows   = 24
)

var safePathComponentPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

type Reference struct {
	ID         string
	FilePath   string
	StartedAt  time.Time
	Width      int
	Height     int
	Format     string
	Protocol   string
	Connection string
}

type recordingPlan struct {
	HostPath   string
	HostDir    string
	GuacdPath  string
	GuacdDir   string
	GuacdName  string
	RecordedAt time.Time
}

type recordingRecord struct {
	ID             string
	UserID         string
	ConnectionID   string
	Protocol       string
	FilePath       string
	Status         string
	CreatedAt      time.Time
	ConnectionName *string
}

func TenantRecordingEnabled(ctx context.Context, db *pgxpool.Pool, tenantID string) (bool, error) {
	if db == nil || strings.TrimSpace(tenantID) == "" {
		return true, nil
	}

	var enabled bool
	if err := db.QueryRow(ctx, `SELECT "recordingEnabled" FROM "Tenant" WHERE id = $1`, tenantID).Scan(&enabled); err != nil {
		return false, fmt.Errorf("load tenant recording policy: %w", err)
	}
	return enabled, nil
}

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

func MetadataFromReference(ref Reference) map[string]any {
	return map[string]any{
		"id":        ref.ID,
		"filePath":  ref.FilePath,
		"startedAt": ref.StartedAt.UTC().Format(time.RFC3339Nano),
		"width":     ref.Width,
		"height":    ref.Height,
		"format":    ref.Format,
		"protocol":  ref.Protocol,
	}
}

func MetadataStringsFromReference(ref Reference) map[string]string {
	return map[string]string{
		"recordingId":        strings.TrimSpace(ref.ID),
		"recordingPath":      strings.TrimSpace(ref.FilePath),
		"recordingStartedAt": ref.StartedAt.UTC().Format(time.RFC3339Nano),
		"recordingWidth":     strconv.Itoa(ref.Width),
		"recordingHeight":    strconv.Itoa(ref.Height),
		"recordingFormat":    strings.TrimSpace(ref.Format),
		"recordingProtocol":  strings.TrimSpace(ref.Protocol),
	}
}

func ReferenceFromMetadata(metadata map[string]any) (Reference, bool) {
	raw, ok := metadata["recording"]
	if !ok {
		return Reference{}, false
	}
	payload, ok := raw.(map[string]any)
	if !ok {
		return Reference{}, false
	}

	startedAt, ok := parseTimeValue(payload["startedAt"])
	if !ok {
		return Reference{}, false
	}

	ref := Reference{
		ID:        stringify(payload["id"]),
		FilePath:  stringify(payload["filePath"]),
		StartedAt: startedAt,
		Width:     intValue(payload["width"], defaultCastCols),
		Height:    intValue(payload["height"], defaultCastRows),
		Format:    defaultString(stringify(payload["format"]), "asciicast"),
		Protocol:  strings.ToUpper(defaultString(stringify(payload["protocol"]), "")),
	}
	if strings.TrimSpace(ref.ID) == "" || strings.TrimSpace(ref.FilePath) == "" {
		return Reference{}, false
	}
	return ref, true
}

func ReferenceFromMetadataStrings(metadata map[string]string) (Reference, bool) {
	id := strings.TrimSpace(metadata["recordingId"])
	filePath := strings.TrimSpace(metadata["recordingPath"])
	startedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(metadata["recordingStartedAt"]))
	if id == "" || filePath == "" || err != nil {
		return Reference{}, false
	}

	return Reference{
		ID:        id,
		FilePath:  filePath,
		StartedAt: startedAt.UTC(),
		Width:     parseIntDefault(metadata["recordingWidth"], defaultCastCols),
		Height:    parseIntDefault(metadata["recordingHeight"], defaultCastRows),
		Format:    defaultString(strings.TrimSpace(metadata["recordingFormat"]), "asciicast"),
		Protocol:  strings.ToUpper(strings.TrimSpace(metadata["recordingProtocol"])),
	}, true
}

func AppendAsciicastOutput(filePath string, startedAt time.Time, output string) error {
	return AppendAsciicastOutputAt(filePath, startedAt, time.Now().UTC(), output)
}

func AppendAsciicastOutputAt(filePath string, startedAt, eventAt time.Time, output string) error {
	if strings.TrimSpace(filePath) == "" || output == "" {
		return nil
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open recording file for append: %w", err)
	}
	defer file.Close()

	elapsed := eventAt.UTC().Sub(startedAt.UTC()).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	event, err := json.Marshal([]any{elapsed, "o", output})
	if err != nil {
		return fmt.Errorf("marshal asciicast event: %w", err)
	}
	if _, err := file.Write(append(event, '\n')); err != nil {
		return fmt.Errorf("append asciicast event: %w", err)
	}
	return nil
}

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

func buildRecordingPlan(recordingRoot, userID, connectionID, protocol, ext, gatewayDir string, now time.Time) (recordingPlan, error) {
	recordingRoot = strings.TrimSpace(recordingRoot)
	if recordingRoot == "" {
		return recordingPlan{}, fmt.Errorf("recording path is not configured")
	}

	subdir := strings.TrimSpace(gatewayDir)
	if subdir == "" {
		subdir = defaultGatewayDir
	}

	components := []struct {
		label string
		value string
	}{
		{label: "userId", value: userID},
		{label: "connectionId", value: connectionID},
		{label: "protocol", value: protocol},
		{label: "ext", value: ext},
		{label: "gatewayDir", value: subdir},
	}
	for _, component := range components {
		if !isSafePathComponent(component.value) {
			return recordingPlan{}, fmt.Errorf("invalid recording path component (%s)", component.label)
		}
	}

	recordingRoot = filepath.Clean(recordingRoot)
	hostPath := filepath.Join(recordingRoot, subdir, userID, fmt.Sprintf("%s-%s-%d.%s", connectionID, strings.ToLower(protocol), now.UTC().UnixMilli(), ext))
	hostPath = filepath.Clean(hostPath)
	relative, err := filepath.Rel(recordingRoot, hostPath)
	if err != nil {
		return recordingPlan{}, fmt.Errorf("compute recording path: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return recordingPlan{}, fmt.Errorf("recording path escapes allowed directory")
	}

	guacdPath := path.Join("/recordings", filepath.ToSlash(relative))
	return recordingPlan{
		HostPath:   hostPath,
		HostDir:    filepath.Dir(hostPath),
		GuacdPath:  guacdPath,
		GuacdDir:   path.Dir(guacdPath),
		GuacdName:  path.Base(guacdPath),
		RecordedAt: now.UTC(),
	}, nil
}

func writeAsciicastHeader(filePath string, header map[string]any) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("create recording file: %w", err)
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(header); err != nil {
		return fmt.Errorf("write recording header: %w", err)
	}
	if err := os.Chmod(filePath, 0o666); err != nil {
		return fmt.Errorf("set recording file permissions: %w", err)
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

func isSafePathComponent(value string) bool {
	if value == "." || value == ".." {
		return false
	}
	return safePathComponentPattern.MatchString(value)
}

func parseTimeValue(value any) (time.Time, bool) {
	text := stringify(value)
	if text == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func intValue(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed)
		}
	case string:
		return parseIntDefault(typed, fallback)
	}
	return fallback
}

func parseIntDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func stringify(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
