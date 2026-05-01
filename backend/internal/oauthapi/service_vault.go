package oauthapi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
)

func (s Service) SetupVaultForOAuthUser(ctx context.Context, userID, vaultPassword string) error {
	if s.DB == nil {
		return fmt.Errorf("database is unavailable")
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin oauth vault setup transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var vaultSetupComplete bool
	if err := tx.QueryRow(ctx, `SELECT COALESCE("vaultSetupComplete", false) FROM "User" WHERE id = $1`, userID).Scan(&vaultSetupComplete); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &requestError{status: http.StatusNotFound, message: "User not found"}
		}
		return fmt.Errorf("load oauth vault user: %w", err)
	}
	if vaultSetupComplete {
		return &requestError{status: http.StatusBadRequest, message: "Vault is already set up."}
	}

	vaultSalt := generateSalt()
	masterKey, err := generateMasterKey()
	if err != nil {
		return fmt.Errorf("generate master key: %w", err)
	}
	defer zeroBytes(masterKey)

	derivedKey := deriveKeyFromPassword(vaultPassword, vaultSalt)
	if len(derivedKey) == 0 {
		return fmt.Errorf("derive vault key: invalid salt")
	}
	defer zeroBytes(derivedKey)

	encryptedVault, err := encryptMasterKey(masterKey, derivedKey)
	if err != nil {
		return fmt.Errorf("encrypt master key: %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`UPDATE "User"
		    SET "vaultSalt" = $2,
		        "encryptedVaultKey" = $3,
		        "vaultKeyIV" = $4,
		        "vaultKeyTag" = $5,
		        "vaultSetupComplete" = true
		  WHERE id = $1`,
		userID,
		vaultSalt,
		encryptedVault.Ciphertext,
		encryptedVault.IV,
		encryptedVault.Tag,
	); err != nil {
		return fmt.Errorf("update oauth vault setup: %w", err)
	}
	if err := insertAuditLog(ctx, tx, userID, "VAULT_SETUP", map[string]any{}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit oauth vault setup transaction: %w", err)
	}

	if err := s.storeVaultSession(ctx, userID, masterKey); err != nil {
		return err
	}
	return nil
}

func validatePassword(password string) error {
	switch {
	case len(password) < 10:
		return &requestError{status: http.StatusBadRequest, message: "Password must be at least 10 characters"}
	case !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz"):
		return &requestError{status: http.StatusBadRequest, message: "Password must contain a lowercase letter"}
	case !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"):
		return &requestError{status: http.StatusBadRequest, message: "Password must contain an uppercase letter"}
	case !strings.ContainsAny(password, "0123456789"):
		return &requestError{status: http.StatusBadRequest, message: "Password must contain a digit"}
	default:
		return nil
	}
}

func generateSalt() string {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		panic(fmt.Errorf("generate salt: %w", err))
	}
	return hex.EncodeToString(buf)
}

func generateMasterKey() ([]byte, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func deriveKeyFromPassword(password, saltHex string) []byte {
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil
	}
	return argon2.IDKey([]byte(password), salt, 3, 65536, 1, 32)
}

func encryptMasterKey(masterKey, derivedKey []byte) (encryptedField, error) {
	if len(derivedKey) != 32 {
		return encryptedField{}, fmt.Errorf("derived key must be 32 bytes")
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return encryptedField{}, fmt.Errorf("create cipher: %w", err)
	}
	aead, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return encryptedField{}, fmt.Errorf("create gcm: %w", err)
	}

	iv := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return encryptedField{}, fmt.Errorf("generate iv: %w", err)
	}

	ciphertextWithTag := aead.Seal(nil, iv, []byte(hex.EncodeToString(masterKey)), nil)
	tagSize := aead.Overhead()
	if len(ciphertextWithTag) < tagSize {
		return encryptedField{}, fmt.Errorf("encrypted payload too short")
	}
	return encryptedField{
		Ciphertext: hex.EncodeToString(ciphertextWithTag[:len(ciphertextWithTag)-tagSize]),
		IV:         hex.EncodeToString(iv),
		Tag:        hex.EncodeToString(ciphertextWithTag[len(ciphertextWithTag)-tagSize:]),
	}, nil
}

func encryptValue(key []byte, plaintext string) (encryptedField, error) {
	if len(key) != 32 {
		return encryptedField{}, fmt.Errorf("invalid key length")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return encryptedField{}, fmt.Errorf("create cipher: %w", err)
	}
	iv := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return encryptedField{}, fmt.Errorf("generate iv: %w", err)
	}
	aead, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return encryptedField{}, fmt.Errorf("create gcm: %w", err)
	}
	sealed := aead.Seal(nil, iv, []byte(plaintext), nil)
	tagOffset := len(sealed) - aead.Overhead()
	return encryptedField{
		Ciphertext: hex.EncodeToString(sealed[:tagOffset]),
		IV:         hex.EncodeToString(iv),
		Tag:        hex.EncodeToString(sealed[tagOffset:]),
	}, nil
}

func (s Service) storeVaultSession(ctx context.Context, userID string, masterKey []byte) error {
	if s.Redis == nil || len(s.ServerKey) == 0 {
		return nil
	}
	encrypted, err := encryptValue(s.ServerKey, hex.EncodeToString(masterKey))
	if err != nil {
		return fmt.Errorf("encrypt vault session: %w", err)
	}
	raw, err := json.Marshal(encrypted)
	if err != nil {
		return fmt.Errorf("marshal vault session: %w", err)
	}
	ttl := s.VaultTTL
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	if err := s.Redis.Set(ctx, "vault:user:"+userID, raw, ttl).Err(); err != nil {
		return fmt.Errorf("store vault session: %w", err)
	}
	recoveryTTL := 7 * 24 * time.Hour
	if err := s.Redis.Set(ctx, "vault:recovery:"+userID, raw, recoveryTTL).Err(); err != nil {
		return fmt.Errorf("store vault recovery: %w", err)
	}
	if s.TenantVaultService != nil {
		if err := s.TenantVaultService.ProcessPendingDistributionsForUser(ctx, userID); err != nil {
			return fmt.Errorf("process pending tenant vault distributions: %w", err)
		}
	}
	return nil
}

func zeroBytes(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

func insertAuditLog(ctx context.Context, tx pgx.Tx, userID, action string, details map[string]any) error {
	rawDetails, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}
	if _, err := tx.Exec(
		ctx,
		`INSERT INTO "AuditLog" (
			id, "userId", action, details, "createdAt"
		) VALUES (
			$1, $2, $3::"AuditAction", $4::jsonb, $5
		)`,
		uuid.NewString(),
		userID,
		action,
		string(rawDetails),
		time.Now(),
	); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}
