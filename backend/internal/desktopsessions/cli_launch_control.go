package desktopsessions

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s Service) issueViewerControlToken(ctx context.Context, grant desktopLaunchGrantRecord, sessionID string) (string, time.Time, error) {
	controlID, controlSecret, controlValue, err := newOpaqueGrant()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate viewer control token: %w", err)
	}
	expiresIn := parseEnvInt("CLI_DESKTOP_CONTROL_TOKEN_TTL_SECONDS", defaultViewerControlTTLSeconds)
	if expiresIn <= 0 {
		expiresIn = defaultViewerControlTTLSeconds
	}
	expiresAt := time.Now().UTC().Add(time.Duration(expiresIn) * time.Second)

	if _, err := s.DB.Exec(ctx, `
INSERT INTO "DesktopViewerControlToken" (
  id, "secretHash", "tenantId", "userId", "sessionId", protocol, "expiresAt"
) VALUES (
  $1, $2, NULLIF($3, ''), $4, $5, $6::"SessionProtocol", $7
)`,
		controlID,
		hashOpaqueSecret(controlSecret),
		grant.TenantID,
		grant.UserID,
		sessionID,
		grant.Protocol,
		expiresAt,
	); err != nil {
		return "", time.Time{}, fmt.Errorf("store viewer control token: %w", err)
	}
	return controlValue, expiresAt, nil
}

func (s Service) AuthorizeViewerControl(ctx context.Context, sessionID, controlToken string) (desktopViewerControlRecord, error) {
	if s.DB == nil {
		return desktopViewerControlRecord{}, fmt.Errorf("database is unavailable")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return desktopViewerControlRecord{}, &requestError{status: http.StatusBadRequest, message: "sessionId is required"}
	}
	controlID, controlSecret, err := splitOpaqueGrant(controlToken)
	if err != nil {
		return desktopViewerControlRecord{}, &requestError{status: http.StatusBadRequest, message: "controlToken is required"}
	}

	var (
		record    desktopViewerControlRecord
		secret    string
		tenantID  sql.NullString
		expiresAt time.Time
		revokedAt sql.NullTime
	)
	err = s.DB.QueryRow(ctx, `
SELECT id, "secretHash", COALESCE("tenantId", ''), "userId", "sessionId", protocol::text, "expiresAt", "revokedAt"
FROM "DesktopViewerControlToken"
WHERE id = $1`, controlID).Scan(
		&record.ID,
		&secret,
		&tenantID,
		&record.UserID,
		&record.SessionID,
		&record.Protocol,
		&expiresAt,
		&revokedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return desktopViewerControlRecord{}, &requestError{status: http.StatusUnauthorized, message: "invalid viewer control token"}
		}
		return desktopViewerControlRecord{}, fmt.Errorf("load viewer control token: %w", err)
	}
	record.TenantID = tenantID.String
	if !opaqueSecretMatches(controlSecret, secret) {
		return desktopViewerControlRecord{}, &requestError{status: http.StatusUnauthorized, message: "invalid viewer control token"}
	}
	if record.SessionID != sessionID {
		return desktopViewerControlRecord{}, &requestError{status: http.StatusForbidden, message: "viewer control token is not valid for this session"}
	}
	if revokedAt.Valid {
		return desktopViewerControlRecord{}, &requestError{status: http.StatusGone, message: "viewer control token has been revoked"}
	}
	if !time.Now().UTC().Before(expiresAt.UTC()) {
		return desktopViewerControlRecord{}, &requestError{status: http.StatusGone, message: "viewer control token has expired"}
	}
	return record, nil
}

func (s Service) revokeViewerControlToken(ctx context.Context, tokenID string) error {
	if s.DB == nil || strings.TrimSpace(tokenID) == "" {
		return nil
	}
	_, err := s.DB.Exec(ctx, `UPDATE "DesktopViewerControlToken" SET "revokedAt" = COALESCE("revokedAt", NOW()) WHERE id = $1`, tokenID)
	return err
}
