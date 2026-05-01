package users

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func (s Service) InitiatePasswordChange(ctx context.Context, userID string) (passwordChangeInitResult, error) {
	result, err := s.initiateVerification(ctx, userID, "password-change")
	if err != nil {
		return passwordChangeInitResult{}, err
	}
	if result.Method == "password" {
		return passwordChangeInitResult{SkipVerification: true}, nil
	}
	return passwordChangeInitResult{
		SkipVerification: false,
		VerificationID:   result.VerificationID,
		Method:           result.Method,
		Metadata:         result.Metadata,
	}, nil
}

func (s Service) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string, verificationID *string, ipAddress string) (passwordChangeResult, error) {
	if s.DB == nil {
		return passwordChangeResult{}, fmt.Errorf("postgres is not configured")
	}
	if err := validatePassword(newPassword); err != nil {
		return passwordChangeResult{}, err
	}

	var (
		passwordHash      *string
		vaultSalt         *string
		encryptedVaultKey *string
		vaultKeyIV        *string
		vaultKeyTag       *string
	)
	if err := s.DB.QueryRow(
		ctx,
		`SELECT "passwordHash", "vaultSalt", "encryptedVaultKey", "vaultKeyIV", "vaultKeyTag"
		   FROM "User"
		  WHERE id = $1`,
		userID,
	).Scan(&passwordHash, &vaultSalt, &encryptedVaultKey, &vaultKeyIV, &vaultKeyTag); err != nil {
		return passwordChangeResult{}, err
	}

	if passwordHash == nil || *passwordHash == "" {
		return passwordChangeResult{}, &requestError{status: http.StatusBadRequest, message: "Cannot change password for OAuth-only accounts."}
	}
	if vaultSalt == nil || encryptedVaultKey == nil || vaultKeyIV == nil || vaultKeyTag == nil ||
		*vaultSalt == "" || *encryptedVaultKey == "" || *vaultKeyIV == "" || *vaultKeyTag == "" {
		return passwordChangeResult{}, &requestError{status: http.StatusBadRequest, message: "Vault is not set up."}
	}

	var masterKey []byte
	if verificationID != nil && *verificationID != "" {
		if err := s.consumeVerificationSession(ctx, *verificationID, userID, "password-change"); err != nil {
			return passwordChangeResult{}, err
		}
		sessionKey, err := s.getVaultMasterKey(ctx, userID)
		if err != nil {
			return passwordChangeResult{}, err
		}
		if len(sessionKey) == 0 {
			return passwordChangeResult{}, &requestError{status: http.StatusForbidden, message: "Vault is locked. Please unlock it first."}
		}
		masterKey = sessionKey
	} else {
		if err := bcrypt.CompareHashAndPassword([]byte(*passwordHash), []byte(oldPassword)); err != nil {
			return passwordChangeResult{}, &requestError{status: http.StatusUnauthorized, message: "Current password is incorrect"}
		}
		oldDerivedKey := deriveKeyFromPassword(oldPassword, *vaultSalt)
		defer zeroBytes(oldDerivedKey)

		decrypted, err := decryptMasterKey(encryptedField{
			Ciphertext: *encryptedVaultKey,
			IV:         *vaultKeyIV,
			Tag:        *vaultKeyTag,
		}, oldDerivedKey)
		if err != nil {
			return passwordChangeResult{}, fmt.Errorf("decrypt master key: %w", err)
		}
		masterKey = decrypted
	}
	defer zeroBytes(masterKey)

	if err := assertPasswordNotBreached(ctx, newPassword); err != nil {
		return passwordChangeResult{}, err
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcryptRounds)
	if err != nil {
		return passwordChangeResult{}, fmt.Errorf("hash password: %w", err)
	}
	newVaultSalt := generateSalt()
	newDerivedKey := deriveKeyFromPassword(newPassword, newVaultSalt)
	defer zeroBytes(newDerivedKey)
	newEncryptedVault, err := encryptMasterKey(masterKey, newDerivedKey)
	if err != nil {
		return passwordChangeResult{}, fmt.Errorf("encrypt master key: %w", err)
	}

	newRecoveryKey, err := generateRecoveryKey()
	if err != nil {
		return passwordChangeResult{}, fmt.Errorf("generate recovery key: %w", err)
	}
	newRecoverySalt := generateSalt()
	recoveryDerivedKey := deriveKeyFromPassword(newRecoveryKey, newRecoverySalt)
	defer zeroBytes(recoveryDerivedKey)
	recoveryEncrypted, err := encryptMasterKey(masterKey, recoveryDerivedKey)
	if err != nil {
		return passwordChangeResult{}, fmt.Errorf("encrypt recovery key: %w", err)
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return passwordChangeResult{}, fmt.Errorf("begin change password: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(
		ctx,
		`UPDATE "User"
		    SET "passwordHash" = $2,
		        "vaultSalt" = $3,
		        "encryptedVaultKey" = $4,
		        "vaultKeyIV" = $5,
		        "vaultKeyTag" = $6,
		        "encryptedVaultRecoveryKey" = $7,
		        "vaultRecoveryKeyIV" = $8,
		        "vaultRecoveryKeyTag" = $9,
		        "vaultRecoveryKeySalt" = $10,
		        "updatedAt" = NOW()
		  WHERE id = $1`,
		userID,
		string(newPasswordHash),
		newVaultSalt,
		newEncryptedVault.Ciphertext,
		newEncryptedVault.IV,
		newEncryptedVault.Tag,
		recoveryEncrypted.Ciphertext,
		recoveryEncrypted.IV,
		recoveryEncrypted.Tag,
		newRecoverySalt,
	); err != nil {
		return passwordChangeResult{}, fmt.Errorf("update password: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM "RefreshToken" WHERE "userId" = $1`, userID); err != nil {
		return passwordChangeResult{}, fmt.Errorf("delete refresh tokens: %w", err)
	}

	if err := insertAuditLog(ctx, tx, userID, "PASSWORD_CHANGE", map[string]any{}, ipAddress); err != nil {
		return passwordChangeResult{}, err
	}
	if err := insertAuditLog(ctx, tx, userID, "VAULT_RECOVERY_KEY_GENERATED", map[string]any{}, ipAddress); err != nil {
		return passwordChangeResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return passwordChangeResult{}, fmt.Errorf("commit change password: %w", err)
	}

	if s.Redis != nil {
		_ = s.Redis.Del(ctx, "vault:user:"+userID).Err()
	}

	return passwordChangeResult{Success: true, RecoveryKey: newRecoveryKey}, nil
}
