package tenantvaultapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) InitTenantVault(ctx context.Context, tenantID, initiatorUserID, ipAddress string) (initResponse, error) {
	return s.provisionTenantVault(ctx, tenantID, initiatorUserID, ipAddress, true, false)
}

func (s Service) EnsureTenantVaultProvisioned(ctx context.Context, tenantID, initiatorUserID, ipAddress string) error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("FEATURE_KEYCHAIN_ENABLED")), "false") {
		return nil
	}
	_, err := s.provisionTenantVault(ctx, tenantID, initiatorUserID, ipAddress, false, true)
	return err
}

func (s Service) provisionTenantVault(ctx context.Context, tenantID, initiatorUserID, ipAddress string, requireInitiatorAccess, allowExisting bool) (initResponse, error) {
	if err := s.ensureAvailable(); err != nil {
		return initResponse{}, err
	}
	if strings.TrimSpace(tenantID) == "" {
		return initResponse{}, &requestError{status: http.StatusBadRequest, message: "Tenant context required"}
	}
	if err := ensureKeychainEnabled(); err != nil {
		return initResponse{}, err
	}

	var (
		initiatorMasterKey []byte
		err                error
	)
	if requireInitiatorAccess {
		initiatorMasterKey, err = s.requireUserMasterKey(ctx, initiatorUserID)
		if err != nil {
			return initResponse{}, err
		}
	} else {
		initiatorMasterKey, err = s.loadUserMasterKey(ctx, initiatorUserID)
		if err != nil {
			return initResponse{}, err
		}
	}
	if len(initiatorMasterKey) > 0 {
		defer zeroBytes(initiatorMasterKey)
	}
	initiatorHasAccess := len(initiatorMasterKey) == 32

	tenantKey, err := generateTenantMasterKey()
	if err != nil {
		return initResponse{}, err
	}
	defer zeroBytes(tenantKey)

	var encKeyForInitiator encryptedField
	if initiatorHasAccess {
		encKeyForInitiator, err = encryptTenantKey(tenantKey, initiatorMasterKey)
		if err != nil {
			return initResponse{}, fmt.Errorf("encrypt tenant key for initiator: %w", err)
		}
	}

	var distributedCount, pendingCount int
	if err := s.withTx(ctx, func(tx pgx.Tx) error {
		hasKey, err := s.loadTenantInitializationState(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if hasKey {
			if allowExisting {
				return nil
			}
			return &requestError{status: http.StatusBadRequest, message: "Tenant vault is already initialized"}
		}

		if _, err := tx.Exec(ctx, `
UPDATE "Tenant"
   SET "hasTenantVaultKey" = true,
       "updatedAt" = NOW()
 WHERE id = $1
`, tenantID); err != nil {
			return fmt.Errorf("initialize tenant vault: %w", err)
		}

		if initiatorHasAccess {
			if _, err := tx.Exec(ctx, `
INSERT INTO "TenantVaultMember" (
	id, "tenantId", "userId", "encryptedTenantVaultKey", "tenantVaultKeyIV", "tenantVaultKeyTag", "createdAt"
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
`, uuid.NewString(), tenantID, initiatorUserID, encKeyForInitiator.Ciphertext, encKeyForInitiator.IV, encKeyForInitiator.Tag); err != nil {
				return fmt.Errorf("create initiator tenant vault membership: %w", err)
			}
		} else {
			escrowKey := deriveEscrowKey(s.ServerKey, tenantID)
			encKey, encErr := encryptTenantKey(tenantKey, escrowKey)
			zeroBytes(escrowKey)
			if encErr != nil {
				return fmt.Errorf("encrypt tenant key with escrow for initiator: %w", encErr)
			}
			if err := s.upsertPendingDistribution(ctx, tx, tenantID, initiatorUserID, initiatorUserID, encKey); err != nil {
				return err
			}
			pendingCount++
		}

		userIDs, err := s.listAcceptedTenantUsers(ctx, tx, tenantID, initiatorUserID)
		if err != nil {
			return err
		}
		for _, userID := range userIDs {
			userMasterKey, err := s.loadUserMasterKey(ctx, userID)
			if err != nil {
				return err
			}
			if len(userMasterKey) == 32 {
				func() {
					defer zeroBytes(userMasterKey)
					encKey, encErr := encryptTenantKey(tenantKey, userMasterKey)
					if encErr != nil {
						err = fmt.Errorf("encrypt tenant key for user %s: %w", userID, encErr)
						return
					}
					_, err = tx.Exec(ctx, `
INSERT INTO "TenantVaultMember" (
	id, "tenantId", "userId", "encryptedTenantVaultKey", "tenantVaultKeyIV", "tenantVaultKeyTag", "createdAt"
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
`, uuid.NewString(), tenantID, userID, encKey.Ciphertext, encKey.IV, encKey.Tag)
				}()
				if err != nil {
					return err
				}
				distributedCount++
				continue
			}

			escrowKey := deriveEscrowKey(s.ServerKey, tenantID)
			encKey, encErr := encryptTenantKey(tenantKey, escrowKey)
			zeroBytes(escrowKey)
			if encErr != nil {
				return fmt.Errorf("encrypt tenant key with escrow for user %s: %w", userID, encErr)
			}
			if err := s.upsertPendingDistribution(ctx, tx, tenantID, userID, initiatorUserID, encKey); err != nil {
				return err
			}
			pendingCount++
		}

		return s.insertAuditLogTx(ctx, tx, initiatorUserID, "TENANT_VAULT_INIT", "Tenant", tenantID, map[string]any{
			"distributedMembers": distributedCount,
			"pendingMembers":     pendingCount,
			"initiatorHasAccess": initiatorHasAccess,
		}, ipAddress)
	}); err != nil {
		return initResponse{}, err
	}

	if initiatorHasAccess && requireInitiatorAccess {
		if err := s.storeTenantVaultSession(ctx, tenantID, initiatorUserID, tenantKey); err != nil {
			return initResponse{}, err
		}
	}
	return initResponse{Initialized: true}, nil
}

func (s Service) ProcessPendingDistributionsForUser(ctx context.Context, userID string) error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("FEATURE_KEYCHAIN_ENABLED")), "false") {
		return nil
	}
	if strings.TrimSpace(userID) == "" {
		return nil
	}
	if err := s.ensureAvailable(); err != nil {
		return err
	}

	masterKey, err := s.loadUserMasterKey(ctx, userID)
	if err != nil {
		return err
	}
	if len(masterKey) != 32 {
		return nil
	}
	defer zeroBytes(masterKey)

	type pendingDistributionRecord struct {
		TenantID string
		Field    encryptedField
	}

	var pendingDistributions []pendingDistributionRecord
	if err := s.withTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT "tenantId", "encryptedTenantVaultKey", "tenantVaultKeyIV", "tenantVaultKeyTag"
  FROM "PendingVaultKeyDistribution"
 WHERE "targetUserId" = $1
 FOR UPDATE
`, userID)
		if err != nil {
			return fmt.Errorf("list pending tenant vault distributions: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var record pendingDistributionRecord
			if err := rows.Scan(&record.TenantID, &record.Field.Ciphertext, &record.Field.IV, &record.Field.Tag); err != nil {
				return fmt.Errorf("scan pending tenant vault distribution: %w", err)
			}
			pendingDistributions = append(pendingDistributions, record)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate pending tenant vault distributions: %w", err)
		}

		for _, record := range pendingDistributions {
			ok, err := s.isAcceptedTenantMember(ctx, tx, record.TenantID, userID)
			if err != nil {
				return err
			}
			if !ok {
				if _, err := tx.Exec(ctx, `
DELETE FROM "PendingVaultKeyDistribution"
 WHERE "tenantId" = $1 AND "targetUserId" = $2
`, record.TenantID, userID); err != nil {
					return fmt.Errorf("clear stale tenant vault distribution: %w", err)
				}
				continue
			}

			escrowKey := deriveEscrowKey(s.ServerKey, record.TenantID)
			tenantKey, err := decryptTenantKey(record.Field, escrowKey)
			zeroBytes(escrowKey)
			if err != nil {
				return fmt.Errorf("decrypt pending tenant vault key: %w", err)
			}

			encKey, err := encryptTenantKey(tenantKey, masterKey)
			zeroBytes(tenantKey)
			if err != nil {
				return fmt.Errorf("encrypt tenant key for pending distribution: %w", err)
			}

			if _, err := tx.Exec(ctx, `
INSERT INTO "TenantVaultMember" (
	id, "tenantId", "userId", "encryptedTenantVaultKey", "tenantVaultKeyIV", "tenantVaultKeyTag", "createdAt"
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT ("tenantId", "userId")
DO UPDATE
      SET "encryptedTenantVaultKey" = EXCLUDED."encryptedTenantVaultKey",
          "tenantVaultKeyIV" = EXCLUDED."tenantVaultKeyIV",
          "tenantVaultKeyTag" = EXCLUDED."tenantVaultKeyTag"
`, uuid.NewString(), record.TenantID, userID, encKey.Ciphertext, encKey.IV, encKey.Tag); err != nil {
				return fmt.Errorf("upsert tenant vault membership from pending distribution: %w", err)
			}

			if _, err := tx.Exec(ctx, `
DELETE FROM "PendingVaultKeyDistribution"
 WHERE "tenantId" = $1 AND "targetUserId" = $2
`, record.TenantID, userID); err != nil {
				return fmt.Errorf("clear fulfilled tenant vault distribution: %w", err)
			}

			if err := s.insertAuditLogTx(ctx, tx, userID, "TENANT_VAULT_KEY_DISTRIBUTE", "User", userID, map[string]any{
				"tenantId":  record.TenantID,
				"pending":   false,
				"automatic": true,
			}, ""); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s Service) DistributeTenantKeyToUser(ctx context.Context, tenantID, targetUserID, distributorUserID, ipAddress string) (distributeResponse, error) {
	if err := s.ensureAvailable(); err != nil {
		return distributeResponse{}, err
	}
	if strings.TrimSpace(tenantID) == "" {
		return distributeResponse{}, &requestError{status: http.StatusBadRequest, message: "Tenant context required"}
	}
	if strings.TrimSpace(targetUserID) == "" {
		return distributeResponse{}, &requestError{status: http.StatusBadRequest, message: "targetUserId is required"}
	}
	if err := ensureKeychainEnabled(); err != nil {
		return distributeResponse{}, err
	}

	tenantKey, err := s.requireTenantKey(ctx, tenantID, distributorUserID)
	if err != nil {
		return distributeResponse{}, err
	}
	defer zeroBytes(tenantKey)

	result := distributeResponse{}
	if err := s.withTx(ctx, func(tx pgx.Tx) error {
		hasKey, err := s.loadTenantInitializationState(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		if !hasKey {
			return &requestError{status: http.StatusBadRequest, message: "Tenant vault is not initialized"}
		}
		hasAccess, err := s.hasTenantVaultAccess(ctx, tx, tenantID, targetUserID)
		if err != nil {
			return err
		}
		if hasAccess {
			return &requestError{status: http.StatusBadRequest, message: "User already has the tenant vault key"}
		}
		ok, err := s.isAcceptedTenantMember(ctx, tx, tenantID, targetUserID)
		if err != nil {
			return err
		}
		if !ok {
			return &requestError{status: http.StatusBadRequest, message: "User is not a member of this tenant"}
		}

		targetMasterKey, err := s.loadUserMasterKey(ctx, targetUserID)
		if err != nil {
			return err
		}
		if len(targetMasterKey) == 32 {
			defer zeroBytes(targetMasterKey)
			encKey, encErr := encryptTenantKey(tenantKey, targetMasterKey)
			if encErr != nil {
				return fmt.Errorf("encrypt tenant key for target user: %w", encErr)
			}
			if _, err := tx.Exec(ctx, `
INSERT INTO "TenantVaultMember" (
	id, "tenantId", "userId", "encryptedTenantVaultKey", "tenantVaultKeyIV", "tenantVaultKeyTag", "createdAt"
) VALUES ($1, $2, $3, $4, $5, $6, NOW())
`, uuid.NewString(), tenantID, targetUserID, encKey.Ciphertext, encKey.IV, encKey.Tag); err != nil {
				return fmt.Errorf("create tenant vault membership: %w", err)
			}
			if _, err := tx.Exec(ctx, `
DELETE FROM "PendingVaultKeyDistribution"
 WHERE "tenantId" = $1 AND "targetUserId" = $2
`, tenantID, targetUserID); err != nil {
				return fmt.Errorf("clear pending tenant key distribution: %w", err)
			}
			result.Distributed = true
			result.Pending = false
		} else {
			escrowKey := deriveEscrowKey(s.ServerKey, tenantID)
			encKey, encErr := encryptTenantKey(tenantKey, escrowKey)
			zeroBytes(escrowKey)
			if encErr != nil {
				return fmt.Errorf("encrypt tenant key with escrow: %w", encErr)
			}
			if err := s.upsertPendingDistribution(ctx, tx, tenantID, targetUserID, distributorUserID, encKey); err != nil {
				return err
			}
			result.Distributed = false
			result.Pending = true
		}

		return s.insertAuditLogTx(ctx, tx, distributorUserID, "TENANT_VAULT_KEY_DISTRIBUTE", "User", targetUserID, map[string]any{
			"tenantId": tenantID,
			"pending":  result.Pending,
		}, ipAddress)
	}); err != nil {
		return distributeResponse{}, err
	}

	return result, nil
}

func (s Service) requireTenantKey(ctx context.Context, tenantID, userID string) ([]byte, error) {
	cached, err := s.loadCachedTenantKey(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	if len(cached) == 32 {
		return cached, nil
	}

	userMasterKey, err := s.requireUserMasterKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(userMasterKey)

	var field encryptedField
	if err := s.DB.QueryRow(ctx, `
SELECT "encryptedTenantVaultKey", "tenantVaultKeyIV", "tenantVaultKeyTag"
  FROM "TenantVaultMember"
 WHERE "tenantId" = $1 AND "userId" = $2
`, tenantID, userID).Scan(&field.Ciphertext, &field.IV, &field.Tag); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &requestError{status: http.StatusNotFound, message: "Tenant vault key not found. An admin must distribute the key to you."}
		}
		return nil, fmt.Errorf("load tenant vault membership: %w", err)
	}
	tenantKey, err := decryptTenantKey(field, userMasterKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt tenant vault key: %w", err)
	}
	if err := s.storeTenantVaultSession(ctx, tenantID, userID, tenantKey); err != nil {
		zeroBytes(tenantKey)
		return nil, err
	}
	return tenantKey, nil
}

func (s Service) requireUserMasterKey(ctx context.Context, userID string) ([]byte, error) {
	masterKey, err := s.loadUserMasterKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(masterKey) != 32 {
		return nil, &requestError{status: http.StatusForbidden, message: "Vault is locked. Please unlock it first."}
	}
	return masterKey, nil
}

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
		return &requestError{status: http.StatusForbidden, message: "The Keychain feature is currently disabled."}
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
			return false, &requestError{status: http.StatusNotFound, message: "Tenant not found"}
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
