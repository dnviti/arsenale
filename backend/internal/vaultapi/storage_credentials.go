package vaultapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s Service) loadVaultCredentials(ctx context.Context, userID string) (vaultCredentials, error) {
	if s.DB == nil {
		return vaultCredentials{}, fmt.Errorf("database is unavailable")
	}

	var creds vaultCredentials
	if err := s.DB.QueryRow(
		ctx,
		`SELECT "passwordHash",
		        "vaultSalt",
		        "encryptedVaultKey",
		        "vaultKeyIV",
		        "vaultKeyTag",
		        COALESCE("vaultNeedsRecovery", false),
		        "encryptedVaultRecoveryKey",
		        "vaultRecoveryKeyIV",
		        "vaultRecoveryKeyTag",
		        "vaultRecoveryKeySalt"
		   FROM "User"
		  WHERE id = $1`,
		userID,
	).Scan(
		&creds.PasswordHash,
		&creds.VaultSalt,
		&creds.EncryptedVaultKey,
		&creds.VaultKeyIV,
		&creds.VaultKeyTag,
		&creds.VaultNeedsRecovery,
		&creds.EncryptedVaultRecoveryKey,
		&creds.VaultRecoveryKeyIV,
		&creds.VaultRecoveryKeyTag,
		&creds.VaultRecoveryKeySalt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return vaultCredentials{}, &requestError{status: 404, message: "User not found"}
		}
		return vaultCredentials{}, fmt.Errorf("load vault credentials: %w", err)
	}
	return creds, nil
}
