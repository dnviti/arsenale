package tenantvaultapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) ensureAvailable() error {
	if s.DB == nil {
		return fmt.Errorf("database is unavailable")
	}
	if s.Redis == nil {
		return fmt.Errorf("redis is unavailable")
	}
	if len(s.ServerKey) != 32 {
		return fmt.Errorf("server encryption key is invalid")
	}
	return nil
}

func ensureKeychainEnabled() error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("FEATURE_KEYCHAIN_ENABLED")), "false") {
		return &requestError{status: 403, message: "The Keychain feature is currently disabled."}
	}
	return nil
}

func (s Service) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (s Service) loadTenantInitializationState(ctx context.Context, tx pgx.Tx, tenantID string) (bool, error) {
	var hasKey bool
	if err := tx.QueryRow(ctx, `
SELECT "hasTenantVaultKey"
  FROM "Tenant"
 WHERE id = $1
 FOR UPDATE
`, tenantID).Scan(&hasKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, &requestError{status: 404, message: "Tenant not found"}
		}
		return false, fmt.Errorf("load tenant vault state: %w", err)
	}
	return hasKey, nil
}

func (s Service) listAcceptedTenantUsers(ctx context.Context, tx pgx.Tx, tenantID, excludeUserID string) ([]string, error) {
	rows, err := tx.Query(ctx, `
SELECT "userId"
  FROM "TenantMember"
 WHERE "tenantId" = $1
   AND status::text = 'ACCEPTED'
   AND "userId" <> $2
   AND ("expiresAt" IS NULL OR "expiresAt" > NOW())
`, tenantID, excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("list accepted tenant users: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan accepted tenant user: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accepted tenant users: %w", err)
	}
	return userIDs, nil
}

func (s Service) hasTenantVaultAccess(ctx context.Context, tx pgx.Tx, tenantID, userID string) (bool, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS (
	SELECT 1
	  FROM "TenantVaultMember"
	 WHERE "tenantId" = $1 AND "userId" = $2
)
`, tenantID, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check tenant vault membership: %w", err)
	}
	return exists, nil
}

func (s Service) isAcceptedTenantMember(ctx context.Context, tx pgx.Tx, tenantID, userID string) (bool, error) {
	var status string
	var expiresAt *time.Time
	if err := tx.QueryRow(ctx, `
SELECT status::text, "expiresAt"
  FROM "TenantMember"
 WHERE "tenantId" = $1 AND "userId" = $2
`, tenantID, userID).Scan(&status, &expiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("load tenant membership: %w", err)
	}
	if status != "ACCEPTED" {
		return false, nil
	}
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return false, nil
	}
	return true, nil
}

func (s Service) upsertPendingDistribution(ctx context.Context, tx pgx.Tx, tenantID, targetUserID, distributorUserID string, field encryptedField) error {
	if _, err := tx.Exec(ctx, `
INSERT INTO "PendingVaultKeyDistribution" (
	id, "tenantId", "targetUserId", "encryptedTenantVaultKey", "tenantVaultKeyIV", "tenantVaultKeyTag", "distributorUserId", "createdAt"
) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
ON CONFLICT ("tenantId", "targetUserId")
DO UPDATE
      SET "encryptedTenantVaultKey" = EXCLUDED."encryptedTenantVaultKey",
          "tenantVaultKeyIV" = EXCLUDED."tenantVaultKeyIV",
          "tenantVaultKeyTag" = EXCLUDED."tenantVaultKeyTag",
          "distributorUserId" = EXCLUDED."distributorUserId"
`, uuid.NewString(), tenantID, targetUserID, field.Ciphertext, field.IV, field.Tag, distributorUserID); err != nil {
		return fmt.Errorf("upsert pending tenant vault distribution: %w", err)
	}
	return nil
}

func (s Service) insertAuditLogTx(ctx context.Context, tx pgx.Tx, userID, action, targetType, targetID string, details map[string]any, ipAddress string) error {
	rawDetails, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO "AuditLog" (
	id, "userId", action, "targetType", "targetId", details, "ipAddress", "createdAt"
) VALUES (
	$1, $2, $3::"AuditAction", NULLIF($4, ''), NULLIF($5, ''), $6::jsonb, NULLIF($7, ''), NOW()
)
`, uuid.NewString(), userID, action, targetType, targetID, string(rawDetails), ipAddress); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
