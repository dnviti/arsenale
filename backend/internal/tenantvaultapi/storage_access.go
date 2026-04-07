package tenantvaultapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

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
			return nil, &requestError{status: 404, message: "Tenant vault key not found. An admin must distribute the key to you."}
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
		return nil, &requestError{status: 403, message: "Vault is locked. Please unlock it first."}
	}
	return masterKey, nil
}
