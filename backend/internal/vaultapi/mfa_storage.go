package vaultapi

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/webauthnflow"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func (s Service) loadVaultRecovery(ctx context.Context, userID string) ([]byte, error) {
	if s.Redis == nil || len(s.ServerKey) != 32 {
		return nil, nil
	}
	raw, err := s.Redis.Get(ctx, "vault:recovery:"+userID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("load vault recovery: %w", err)
	}

	var field encryptedField
	if err := json.Unmarshal(raw, &field); err != nil {
		return nil, fmt.Errorf("decode vault recovery: %w", err)
	}
	hexKey, err := decryptEncryptedField(s.ServerKey, field)
	if err != nil {
		return nil, fmt.Errorf("decrypt vault recovery: %w", err)
	}
	masterKey, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode vault recovery key: %w", err)
	}
	return masterKey, nil
}

func (s Service) loadTOTPUnlockUser(ctx context.Context, userID string) (totpUnlockUser, error) {
	if s.DB == nil {
		return totpUnlockUser{}, fmt.Errorf("database is unavailable")
	}

	var user totpUnlockUser
	if err := s.DB.QueryRow(
		ctx,
		`SELECT COALESCE("totpEnabled", false),
		        "encryptedTotpSecret",
		        "totpSecretIV",
		        "totpSecretTag",
		        "totpSecret"
		   FROM "User"
		  WHERE id = $1`,
		userID,
	).Scan(
		&user.TOTPEnabled,
		&user.EncryptedTOTPSecret,
		&user.TOTPSecretIV,
		&user.TOTPSecretTag,
		&user.TOTPSecret,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return totpUnlockUser{}, &requestError{status: 404, message: "User not found"}
		}
		return totpUnlockUser{}, fmt.Errorf("load TOTP settings: %w", err)
	}
	return user, nil
}

type smsUnlockUser struct {
	SMSMFAEnabled bool
	PhoneNumber   string
}

func (s Service) loadSMSUnlockUser(ctx context.Context, userID string) (smsUnlockUser, error) {
	if s.DB == nil {
		return smsUnlockUser{}, fmt.Errorf("database is unavailable")
	}

	var user smsUnlockUser
	if err := s.DB.QueryRow(
		ctx,
		`SELECT COALESCE("smsMfaEnabled", false), COALESCE("phoneNumber", '')
		   FROM "User"
		  WHERE id = $1`,
		userID,
	).Scan(&user.SMSMFAEnabled, &user.PhoneNumber); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return smsUnlockUser{}, &requestError{status: 404, message: "User not found"}
		}
		return smsUnlockUser{}, fmt.Errorf("load sms unlock user: %w", err)
	}
	return user, nil
}

func (s Service) storeOTP(ctx context.Context, userID, code string) error {
	command, err := s.DB.Exec(
		ctx,
		`UPDATE "User"
		    SET "smsOtpHash" = $2,
		        "smsOtpExpiresAt" = $3,
		        "updatedAt" = NOW()
		  WHERE id = $1`,
		userID,
		hashSMSCode(code),
		time.Now().Add(smsOTPTTL),
	)
	if err != nil {
		return fmt.Errorf("store sms otp: %w", err)
	}
	if command.RowsAffected() == 0 {
		return &requestError{status: 404, message: "User not found"}
	}
	return nil
}

func (s Service) verifyOTP(ctx context.Context, userID, code string) (bool, error) {
	if s.DB == nil {
		return false, fmt.Errorf("database is unavailable")
	}

	var (
		storedHash *string
		expiresAt  *time.Time
	)
	if err := s.DB.QueryRow(
		ctx,
		`SELECT "smsOtpHash", "smsOtpExpiresAt"
		   FROM "User"
		  WHERE id = $1`,
		userID,
	).Scan(&storedHash, &expiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, &requestError{status: 404, message: "User not found"}
		}
		return false, fmt.Errorf("load sms otp: %w", err)
	}

	if storedHash == nil || expiresAt == nil {
		return false, nil
	}
	if expiresAt.Before(time.Now()) {
		_, _ = s.DB.Exec(
			ctx,
			`UPDATE "User"
			    SET "smsOtpHash" = NULL,
			        "smsOtpExpiresAt" = NULL,
			        "updatedAt" = NOW()
			  WHERE id = $1`,
			userID,
		)
		return false, nil
	}
	if *storedHash != hashSMSCode(code) {
		return false, nil
	}

	_, err := s.DB.Exec(
		ctx,
		`UPDATE "User"
		    SET "smsOtpHash" = NULL,
		        "smsOtpExpiresAt" = NULL,
		        "updatedAt" = NOW()
		  WHERE id = $1`,
		userID,
	)
	if err != nil {
		return false, fmt.Errorf("clear sms otp: %w", err)
	}
	return true, nil
}

func (s Service) loadWebAuthnDescriptors(ctx context.Context, userID string) ([]webauthnflow.CredentialDescriptor, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}

	rows, err := s.DB.Query(
		ctx,
		`SELECT "credentialId", transports
		   FROM "WebAuthnCredential"
		  WHERE "userId" = $1
		  ORDER BY "createdAt" DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("load webauthn credentials: %w", err)
	}
	defer rows.Close()

	result := make([]webauthnflow.CredentialDescriptor, 0)
	for rows.Next() {
		var (
			credentialID string
			transports   []string
		)
		if err := rows.Scan(&credentialID, &transports); err != nil {
			return nil, fmt.Errorf("scan webauthn credential: %w", err)
		}
		result = append(result, webauthnflow.CredentialDescriptor{
			ID:         strings.TrimSpace(credentialID),
			Type:       "public-key",
			Transports: transports,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webauthn credentials: %w", err)
	}
	return result, nil
}

func (s Service) loadStoredWebAuthnCredentials(ctx context.Context, userID string) ([]webauthnflow.StoredCredential, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}

	rows, err := s.DB.Query(
		ctx,
		`SELECT "credentialId", "publicKey", counter
		   FROM "WebAuthnCredential"
		  WHERE "userId" = $1
		  ORDER BY "createdAt" DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("load webauthn stored credentials: %w", err)
	}
	defer rows.Close()

	result := make([]webauthnflow.StoredCredential, 0)
	for rows.Next() {
		var (
			credentialID string
			publicKey    string
			counter      sql.NullInt64
		)
		if err := rows.Scan(&credentialID, &publicKey, &counter); err != nil {
			return nil, fmt.Errorf("scan webauthn stored credential: %w", err)
		}
		value := int64(0)
		if counter.Valid && counter.Int64 > 0 {
			value = counter.Int64
		}
		result = append(result, webauthnflow.StoredCredential{
			CredentialID: strings.TrimSpace(credentialID),
			PublicKey:    strings.TrimSpace(publicKey),
			Counter:      uint32(value),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webauthn stored credentials: %w", err)
	}
	return result, nil
}
