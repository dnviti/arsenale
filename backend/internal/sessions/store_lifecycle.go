package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CloseStaleSessionsForConnection(ctx context.Context, userID, connectionID, protocol string) (int, error) {
	if s.db == nil {
		return 0, nil
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin close stale sessions: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(
		ctx,
		`SELECT s.id,
		        s."userId",
		        s."connectionId",
		        s.protocol::text,
		        s."gatewayId",
		        s."instanceId",
		        s."ipAddress",
		        s."startedAt",
		        s.status::text
		   FROM "ActiveSession" s
		  WHERE s."userId" = $1
		    AND s."connectionId" = $2
		    AND s.protocol = $3::"SessionProtocol"
		    AND s.status <> 'CLOSED'::"SessionStatus"
		  ORDER BY s."startedAt" DESC
		  FOR UPDATE`,
		userID,
		connectionID,
		protocol,
	)
	if err != nil {
		return 0, fmt.Errorf("query stale sessions: %w", err)
	}
	defer rows.Close()

	var stale []sessionRecord
	for rows.Next() {
		var record sessionRecord
		if err := rows.Scan(
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
			return 0, fmt.Errorf("scan stale session: %w", err)
		}
		stale = append(stale, record)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate stale sessions: %w", err)
	}

	if len(stale) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit empty stale session close: %w", err)
		}
		return 0, nil
	}

	closedAt := time.Now().UTC()
	recordingIDs := make([]string, 0, len(stale))
	for _, record := range stale {
		if _, err := tx.Exec(
			ctx,
			`UPDATE "ActiveSession"
			    SET status = 'CLOSED'::"SessionStatus",
			        "endedAt" = $2
			  WHERE id = $1`,
			record.ID,
			closedAt,
		); err != nil {
			return 0, fmt.Errorf("close stale session %s: %w", record.ID, err)
		}

		recordingID := ""
		if shouldAutoCompleteRecording(record.Protocol) {
			recordingID, err = lookupRecordingID(ctx, tx, record.ID)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return 0, fmt.Errorf("lookup recording id for stale session %s: %w", record.ID, err)
			}
			if recordingID != "" {
				recordingIDs = append(recordingIDs, recordingID)
			}
		}

		var gatewayName any
		if record.GatewayID != nil && *record.GatewayID != "" {
			name, nameErr := loadGatewayName(ctx, tx, *record.GatewayID)
			if nameErr == nil && name != "" {
				gatewayName = name
			}
		}

		details, err := json.Marshal(map[string]any{
			"sessionId":         record.ID,
			"protocol":          record.Protocol,
			"reason":            "superseded_by_new_session",
			"durationMs":        closedAt.Sub(record.StartedAt).Milliseconds(),
			"durationFormatted": formatDuration(closedAt.Sub(record.StartedAt).Milliseconds()),
			"gatewayName":       gatewayName,
			"instanceId":        stringPtrValue(record.InstanceID),
			"recordingId":       emptyToNil(recordingID),
		})
		if err != nil {
			return 0, fmt.Errorf("marshal stale session audit details: %w", err)
		}

		if err := insertAuditLog(ctx, tx, auditLogParams{
			UserID:     record.UserID,
			Action:     "SESSION_END",
			TargetType: "Connection",
			TargetID:   record.ConnectionID,
			Details:    details,
			IPAddress:  record.IPAddress,
			GatewayID:  record.GatewayID,
		}); err != nil {
			return 0, fmt.Errorf("insert stale session audit: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit stale session close: %w", err)
	}
	if err := completeSessionRecordings(ctx, s.db, recordingIDs); err != nil {
		return 0, err
	}

	return len(stale), nil
}

func (s *Store) StartSession(ctx context.Context, params StartSessionParams) (string, error) {
	if s.db == nil {
		return "", errors.New("postgres is not configured")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin start session: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	sessionID := uuid.NewString()
	metadataJSON, err := json.Marshal(params.Metadata)
	if err != nil {
		return "", fmt.Errorf("marshal session metadata: %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO "ActiveSession" (
			 id, "userId", "connectionId", "gatewayId", "instanceId", protocol, status, "socketId", "guacTokenHash", "ipAddress", metadata
		 ) VALUES (
			 $1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6::"SessionProtocol", 'ACTIVE'::"SessionStatus", NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), $10::jsonb
		 )`,
		sessionID,
		params.UserID,
		params.ConnectionID,
		params.GatewayID,
		params.InstanceID,
		params.Protocol,
		params.SocketID,
		params.GuacTokenHash,
		params.IPAddress,
		string(metadataJSON),
	); err != nil {
		return "", fmt.Errorf("insert active session: %w", err)
	}

	if params.RecordingID != "" {
		if _, err := tx.Exec(
			ctx,
			`UPDATE "SessionRecording"
			    SET "sessionId" = $2
			  WHERE id = $1`,
			params.RecordingID,
			sessionID,
		); err != nil {
			return "", fmt.Errorf("link session recording: %w", err)
		}
	}

	detailsMap := make(map[string]any, len(params.Metadata)+8)
	for key, value := range params.Metadata {
		detailsMap[key] = value
	}
	detailsMap["sessionId"] = sessionID
	detailsMap["protocol"] = params.Protocol
	if params.RecordingID != "" {
		detailsMap["recordingId"] = params.RecordingID
	}
	if params.RoutingDecision != nil {
		if params.RoutingDecision.Strategy != "" {
			detailsMap["lbStrategy"] = params.RoutingDecision.Strategy
		}
		if params.RoutingDecision.CandidateCount > 0 {
			detailsMap["lbCandidates"] = params.RoutingDecision.CandidateCount
		}
		if params.RoutingDecision.SelectedSessionCount > 0 {
			detailsMap["lbSelectedSessions"] = params.RoutingDecision.SelectedSessionCount
		}
	}

	gatewayID := stringToPtr(params.GatewayID)
	if gatewayID != nil {
		gatewayName, err := loadGatewayName(ctx, tx, *gatewayID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("load gateway name: %w", err)
		}
		if gatewayName != "" {
			detailsMap["gatewayName"] = gatewayName
		}
		if params.InstanceID != "" {
			detailsMap["instanceId"] = params.InstanceID
		}
	}

	detailsJSON, err := json.Marshal(detailsMap)
	if err != nil {
		return "", fmt.Errorf("marshal session start audit details: %w", err)
	}

	if err := insertAuditLog(ctx, tx, auditLogParams{
		UserID:     params.UserID,
		Action:     "SESSION_START",
		TargetType: "Connection",
		TargetID:   params.ConnectionID,
		Details:    detailsJSON,
		IPAddress:  stringToPtr(params.IPAddress),
		GatewayID:  gatewayID,
	}); err != nil {
		return "", fmt.Errorf("insert session start audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit start session: %w", err)
	}

	return sessionID, nil
}

func (s *Store) EndOwnedSession(ctx context.Context, sessionID, userID, reason string) error {
	if s.db == nil {
		return errors.New("postgres is not configured")
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin end session: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	record, err := loadOwnedSessionForUpdate(ctx, tx, sessionID, userID)
	if err != nil {
		return err
	}

	if record.Status == "CLOSED" {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit closed session end noop: %w", err)
		}
		return nil
	}

	closedAt := time.Now().UTC()
	if _, err := tx.Exec(
		ctx,
		`UPDATE "ActiveSession"
		    SET status = 'CLOSED'::"SessionStatus",
		        "endedAt" = $2
		  WHERE id = $1`,
		record.ID,
		closedAt,
	); err != nil {
		return fmt.Errorf("close session: %w", err)
	}

	recordingID, err := lookupRecordingID(ctx, tx, record.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lookup recording id: %w", err)
	}

	if record.GatewayID != nil && *record.GatewayID != "" {
		gatewayName, nameErr := loadGatewayName(ctx, tx, *record.GatewayID)
		if nameErr == nil && gatewayName != "" {
			record.GatewayName = &gatewayName
		}
	}

	details := map[string]any{
		"sessionId":         record.ID,
		"protocol":          record.Protocol,
		"durationMs":        closedAt.Sub(record.StartedAt).Milliseconds(),
		"durationFormatted": formatDuration(closedAt.Sub(record.StartedAt).Milliseconds()),
		"gatewayName":       stringPtrValue(record.GatewayName),
		"instanceId":        stringPtrValue(record.InstanceID),
	}
	if reason != "" {
		details["reason"] = reason
	}
	if recordingID != "" {
		details["recordingId"] = recordingID
	}

	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal session end audit details: %w", err)
	}

	if err := insertAuditLog(ctx, tx, auditLogParams{
		UserID:     record.UserID,
		Action:     "SESSION_END",
		TargetType: "Connection",
		TargetID:   record.ConnectionID,
		Details:    detailsJSON,
		IPAddress:  record.IPAddress,
		GatewayID:  record.GatewayID,
	}); err != nil {
		return fmt.Errorf("insert session end audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit end session: %w", err)
	}
	if shouldAutoCompleteRecording(record.Protocol) {
		if err := completeSessionRecordings(ctx, s.db, []string{recordingID}); err != nil {
			return err
		}
	}

	return nil
}
