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
