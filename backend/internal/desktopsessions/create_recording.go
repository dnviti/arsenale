package desktopsessions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s Service) startRecording(ctx context.Context, userID, connectionID, protocol, gatewayDir string, width, height *int) (string, *recordingSettings, error) {
	plan, err := buildRecordingPlan(s.recordingRoot(), userID, connectionID, protocol, "guac", gatewayDir, time.Now().UTC())
	if err != nil {
		return "", nil, err
	}
	if err := os.MkdirAll(plan.HostDir, 0o777); err != nil {
		return "", nil, err
	}
	_ = os.Chmod(plan.HostDir, 0o777)
	parentDir := filepath.Dir(plan.HostDir)
	if parentDir != "" && parentDir != "." {
		_ = os.Chmod(parentDir, 0o777)
	}

	recordingID := uuid.NewString()
	if _, err := s.DB.Exec(ctx, `
INSERT INTO "SessionRecording" (
	id, "userId", "connectionId", protocol, "filePath", width, height, format, status
) VALUES (
	$1, $2, $3, $4::"SessionProtocol", $5, $6, $7, 'guac', 'RECORDING'::"RecordingStatus"
)
`, recordingID, userID, connectionID, protocol, plan.HostPath, width, height); err != nil {
		return "", nil, fmt.Errorf("insert session recording: %w", err)
	}

	s.insertRecordingAudit(ctx, recordingID, userID, connectionID, protocol)

	return recordingID, &recordingSettings{
		RecordingPath: plan.GuacdDir,
		RecordingName: plan.GuacdName,
	}, nil
}

func (s Service) insertRecordingAudit(ctx context.Context, recordingID, userID, connectionID, protocol string) {
	details, err := json.Marshal(map[string]any{
		"recordingId":  recordingID,
		"protocol":     protocol,
		"connectionId": connectionID,
	})
	if err != nil {
		return
	}

	_, _ = s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details)
VALUES ($1, $2, 'RECORDING_START'::"AuditAction", 'Recording', $3, $4::jsonb)
`, uuid.NewString(), strings.TrimSpace(userID), recordingID, string(details))
}

func (s Service) recordSessionError(ctx context.Context, userID, protocol string, details sessionErrorContext, ipAddress string, err error) {
	if s.DB == nil || strings.TrimSpace(details.ConnectionID) == "" {
		return
	}

	rawDetails, marshalErr := json.Marshal(map[string]any{
		"protocol": protocol,
		"error":    err.Error(),
		"host":     emptyToNil(details.Host),
		"port":     zeroToNil(details.Port),
	})
	if marshalErr != nil {
		return
	}

	_, _ = s.DB.Exec(
		ctx,
		`INSERT INTO "AuditLog" (
			id, "userId", action, "targetType", "targetId", details, "ipAddress", "gatewayId", "geoCoords", flags
		) VALUES (
			$1, NULLIF($2, ''), 'SESSION_ERROR'::"AuditAction", 'Connection', NULLIF($3, ''), $4::jsonb, NULLIF($5, ''), NULLIF($6, ''), ARRAY[]::double precision[], ARRAY[]::text[]
		)`,
		uuid.NewString(),
		strings.TrimSpace(userID),
		details.ConnectionID,
		string(rawDetails),
		strings.TrimSpace(ipAddress),
		strings.TrimSpace(details.GatewayID),
	)
}

func (s Service) recordingRoot() string {
	root := strings.TrimSpace(s.RecordingPath)
	if root == "" {
		return guacdRecordRoot
	}
	return root
}

func (s Service) driveBasePath() string {
	basePath := strings.TrimSpace(s.DriveBasePath)
	if basePath == "" {
		return "/guacd-drive"
	}
	return basePath
}
