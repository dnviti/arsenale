package sshproxyapi

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/desktopbroker"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type proxyGrantRecord struct {
	ID           string
	TenantID     string
	UserID       string
	ConnectionID string
	ExpiresAt    time.Time
}

func (s Service) redeemProxyGrant(ctx context.Context, rawGrant, ipAddress string) (proxyGrantRecord, error) {
	if s.DB == nil {
		return proxyGrantRecord{}, errors.New("database is unavailable")
	}
	grantID, grantSecret, err := splitProxyGrant(rawGrant)
	if err != nil {
		return proxyGrantRecord{}, err
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return proxyGrantRecord{}, fmt.Errorf("begin SSH proxy grant redemption: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		record     proxyGrantRecord
		secretHash string
		tenantID   sql.NullString
		consumedAt sql.NullTime
	)
	if err := tx.QueryRow(ctx, `
SELECT id, "secretHash", "tenantId", "userId", "connectionId", "expiresAt", "consumedAt"
FROM "SSHProxyGrant"
WHERE id = $1
FOR UPDATE
`, grantID).Scan(
		&record.ID,
		&secretHash,
		&tenantID,
		&record.UserID,
		&record.ConnectionID,
		&record.ExpiresAt,
		&consumedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return proxyGrantRecord{}, errors.New("SSH proxy grant is invalid")
		}
		return proxyGrantRecord{}, fmt.Errorf("load SSH proxy grant: %w", err)
	}
	if tenantID.Valid {
		record.TenantID = tenantID.String
	}

	if !proxyGrantSecretMatches(grantSecret, secretHash) {
		_ = s.insertAuditLog(ctx, record.UserID, "SSH_PROXY_AUTH_FAILURE", record.ConnectionID, map[string]any{
			"reason": "secret_mismatch",
		}, stringToPtr(ipAddress))
		return proxyGrantRecord{}, errors.New("SSH proxy grant is invalid")
	}
	if consumedAt.Valid {
		_ = s.insertAuditLog(ctx, record.UserID, "SSH_PROXY_AUTH_FAILURE", record.ConnectionID, map[string]any{
			"reason": "grant_reused",
		}, stringToPtr(ipAddress))
		return proxyGrantRecord{}, errors.New("SSH proxy grant has already been used")
	}
	if time.Now().UTC().After(record.ExpiresAt.UTC()) {
		_ = s.insertAuditLog(ctx, record.UserID, "SSH_PROXY_AUTH_FAILURE", record.ConnectionID, map[string]any{
			"reason": "grant_expired",
		}, stringToPtr(ipAddress))
		return proxyGrantRecord{}, errors.New("SSH proxy grant has expired")
	}

	if _, err := tx.Exec(ctx, `
UPDATE "SSHProxyGrant"
SET "consumedAt" = NOW(),
    "consumedIpAddress" = NULLIF($2, '')
WHERE id = $1
`, record.ID, strings.TrimSpace(ipAddress)); err != nil {
		return proxyGrantRecord{}, fmt.Errorf("consume SSH proxy grant: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return proxyGrantRecord{}, fmt.Errorf("commit SSH proxy grant redemption: %w", err)
	}
	return record, nil
}

func (s Service) attachProxyGrantSession(ctx context.Context, grantID, sessionID string) error {
	if s.DB == nil {
		return errors.New("database is unavailable")
	}
	_, err := s.DB.Exec(ctx, `
UPDATE "SSHProxyGrant"
SET "consumedSessionId" = NULLIF($2, '')
WHERE id = $1
`, grantID, sessionID)
	if err != nil {
		return fmt.Errorf("attach SSH proxy grant session: %w", err)
	}
	return nil
}

func (s Service) checkLateralMovement(ctx context.Context, userID, connectionID, ipAddress string) (bool, error) {
	if !parseEnvBool("LATERAL_MOVEMENT_DETECTION_ENABLED", true) {
		return true, nil
	}
	windowMinutes := parsePositiveInt(getenv("LATERAL_MOVEMENT_WINDOW_MINUTES", "5"), 5)
	threshold := parsePositiveInt(getenv("LATERAL_MOVEMENT_MAX_DISTINCT_TARGETS", "10"), 10)
	lockoutMinutes := parsePositiveInt(getenv("LATERAL_MOVEMENT_LOCKOUT_MINUTES", "30"), 30)
	since := time.Now().UTC().Add(-time.Duration(windowMinutes) * time.Minute)

	rows, err := s.DB.Query(ctx, `
SELECT DISTINCT "targetId"
FROM "AuditLog"
WHERE "userId" = $1
  AND action = 'SESSION_START'::"AuditAction"
  AND "createdAt" >= $2
  AND "targetId" IS NOT NULL
`, userID, since)
	if err != nil {
		return false, fmt.Errorf("check SSH proxy lateral movement: %w", err)
	}
	defer rows.Close()

	targets := map[string]struct{}{connectionID: {}}
	for rows.Next() {
		var targetID string
		if err := rows.Scan(&targetID); err != nil {
			return false, fmt.Errorf("scan SSH proxy lateral movement target: %w", err)
		}
		if strings.TrimSpace(targetID) != "" {
			targets[targetID] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate SSH proxy lateral movement targets: %w", err)
	}
	if len(targets) <= threshold {
		return true, nil
	}

	details, _ := json.Marshal(map[string]any{
		"distinctTargets":     len(targets),
		"threshold":           threshold,
		"windowMinutes":       windowMinutes,
		"recentConnectionIds": mapKeys(targets),
		"deniedConnectionId":  connectionID,
	})
	_, _ = s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details, "ipAddress", "createdAt")
VALUES ($1, $2, 'ANOMALOUS_LATERAL_MOVEMENT'::"AuditAction", 'User', $3, $4::jsonb, NULLIF($5, ''), NOW())
`, uuid.NewString(), userID, userID, string(details), ipAddress)
	_, _ = s.DB.Exec(ctx, `UPDATE "User" SET "lockedUntil" = $2 WHERE id = $1`, userID, time.Now().UTC().Add(time.Duration(lockoutMinutes)*time.Minute))
	return false, nil
}

func proxyGrantSecretMatches(secret, expectedHash string) bool {
	got := desktopbroker.HashToken(strings.TrimSpace(secret))
	return subtle.ConstantTimeCompare([]byte(got), []byte(strings.TrimSpace(expectedHash))) == 1
}

func stringToPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func parseEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func mapKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	return result
}
