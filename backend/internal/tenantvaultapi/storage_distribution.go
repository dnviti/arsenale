package tenantvaultapi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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
