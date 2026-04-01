package secretsmeta

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/credentialresolver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/hkdf"
)

type externalShareCreateResponse struct {
	ID             string `json:"id"`
	ShareURL       string `json:"shareUrl"`
	ExpiresAt      string `json:"expiresAt"`
	MaxAccessCount *int   `json:"maxAccessCount"`
	HasPin         bool   `json:"hasPin"`
}

func (s Service) HandleCreateExternalShare(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	body, err := readBodyBytes(r)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	payload, err := parseCreateExternalShareInput(body)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.CreateExternalShare(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"), payload, requestIP(r))
	if err != nil {
		s.handleSecretsError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleRevokeExternalShare(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.RevokeExternalShare(r.Context(), claims.UserID, claims.TenantID, r.PathValue("shareId"), requestIP(r))
	if err != nil {
		s.handleSecretsError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) CreateExternalShare(ctx context.Context, userID, tenantID, secretID string, input createExternalShareInput, ipAddress string) (externalShareCreateResponse, error) {
	_, err := s.resolver().RequireManageSecret(ctx, userID, secretID, tenantID)
	if err != nil {
		var reqErr *credentialresolver.RequestError
		if errorsAsResolver(err, &reqErr) {
			return externalShareCreateResponse{}, &secretsRequestError{status: http.StatusForbidden, message: "You do not have permission to share this secret"}
		}
		return externalShareCreateResponse{}, err
	}

	detail, err := s.resolver().ResolveSecret(ctx, userID, secretID, tenantID)
	if err != nil {
		return externalShareCreateResponse{}, err
	}

	token, err := randomBase64URL(32)
	if err != nil {
		return externalShareCreateResponse{}, fmt.Errorf("generate external share token: %w", err)
	}
	shareID := uuid.NewString()

	var (
		key             []byte
		pinSalt         *string
		tokenSalt       *string
		maxAccessCount  any
		expiresAt       = time.Now().UTC().Add(time.Duration(input.ExpiresInMinutes) * time.Minute)
		hasPin          = input.Pin != nil && strings.TrimSpace(*input.Pin) != ""
		clientURLPrefix = strings.TrimRight(strings.TrimSpace(s.ClientURL), "/")
	)
	if input.MaxAccessCount != nil {
		maxAccessCount = *input.MaxAccessCount
	}

	if hasPin {
		salt, err := randomHex(32)
		if err != nil {
			return externalShareCreateResponse{}, fmt.Errorf("generate share pin salt: %w", err)
		}
		pinSalt = &salt
		key, err = deriveKeyFromTokenAndPin(token, *input.Pin, salt)
		if err != nil {
			return externalShareCreateResponse{}, fmt.Errorf("derive external share pin key: %w", err)
		}
	} else {
		salt, err := randomBase64Std(32)
		if err != nil {
			return externalShareCreateResponse{}, fmt.Errorf("generate share token salt: %w", err)
		}
		tokenSalt = &salt
		key, err = deriveKeyFromToken(token, shareID, salt)
		if err != nil {
			return externalShareCreateResponse{}, fmt.Errorf("derive external share key: %w", err)
		}
	}
	defer zeroBytes(key)

	encrypted, err := encryptExternalSharePayload(string(detail.Data), key)
	if err != nil {
		return externalShareCreateResponse{}, err
	}

	if s.DB == nil {
		return externalShareCreateResponse{}, fmt.Errorf("database is unavailable")
	}
	if _, err := s.DB.Exec(ctx, `
INSERT INTO "ExternalSecretShare" (
	id,
	"secretId",
	"createdByUserId",
	"tokenHash",
	"encryptedData",
	"dataIV",
	"dataTag",
	"hasPin",
	"pinSalt",
	"tokenSalt",
	"expiresAt",
	"maxAccessCount",
	"secretType",
	"secretName",
	"isRevoked",
	"createdAt"
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::"SecretType", $14, false, NOW())
`, shareID, secretID, userID, hashToken(token), encrypted.Ciphertext, encrypted.IV, encrypted.Tag, hasPin, nullableString(pinSalt), nullableString(tokenSalt), expiresAt, maxAccessCount, detail.Type, detail.Name); err != nil {
		return externalShareCreateResponse{}, fmt.Errorf("insert external secret share: %w", err)
	}

	_ = s.insertTypedAuditLog(ctx, userID, "SECRET_EXTERNAL_SHARE", "VaultSecret", secretID, map[string]any{
		"shareId":        shareID,
		"hasPin":         hasPin,
		"expiresAt":      expiresAt.Format(time.RFC3339),
		"maxAccessCount": nullableAuditInt(input.MaxAccessCount),
	}, ipAddress)

	if clientURLPrefix == "" {
		clientURLPrefix = "https://localhost:3000"
	}

	return externalShareCreateResponse{
		ID:             shareID,
		ShareURL:       clientURLPrefix + "/share/" + token,
		ExpiresAt:      expiresAt.Format(time.RFC3339),
		MaxAccessCount: input.MaxAccessCount,
		HasPin:         hasPin,
	}, nil
}

func (s Service) RevokeExternalShare(ctx context.Context, userID, tenantID, shareID, ipAddress string) (map[string]bool, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}

	var secretID string
	if err := s.DB.QueryRow(ctx, `
SELECT "secretId"
FROM "ExternalSecretShare"
WHERE id = $1
`, shareID).Scan(&secretID); err != nil {
		if errorsIsNoRows(err) {
			return nil, &secretsRequestError{status: http.StatusNotFound, message: "Share not found"}
		}
		return nil, fmt.Errorf("load external secret share: %w", err)
	}

	if _, err := s.resolver().RequireManageSecret(ctx, userID, secretID, tenantID); err != nil {
		var reqErr *credentialresolver.RequestError
		if errorsAsResolver(err, &reqErr) {
			return nil, &secretsRequestError{status: http.StatusForbidden, message: "You do not have permission to revoke this share"}
		}
		return nil, err
	}

	if _, err := s.DB.Exec(ctx, `
UPDATE "ExternalSecretShare"
SET "isRevoked" = true
WHERE id = $1
`, shareID); err != nil {
		return nil, fmt.Errorf("revoke external secret share: %w", err)
	}

	_ = s.insertTypedAuditLog(ctx, userID, "SECRET_EXTERNAL_REVOKE", "ExternalSecretShare", shareID, map[string]any{
		"secretId": secretID,
	}, ipAddress)

	return map[string]bool{"revoked": true}, nil
}

type encryptedSharePayload struct {
	Ciphertext string
	IV         string
	Tag        string
}

func encryptExternalSharePayload(plaintext string, key []byte) (encryptedSharePayload, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return encryptedSharePayload{}, fmt.Errorf("create share cipher: %w", err)
	}

	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return encryptedSharePayload{}, fmt.Errorf("generate share iv: %w", err)
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		return encryptedSharePayload{}, fmt.Errorf("create share gcm: %w", err)
	}

	sealed := gcm.Seal(nil, iv, []byte(plaintext), nil)
	tagOffset := len(sealed) - gcm.Overhead()
	return encryptedSharePayload{
		Ciphertext: hex.EncodeToString(sealed[:tagOffset]),
		IV:         hex.EncodeToString(iv),
		Tag:        hex.EncodeToString(sealed[tagOffset:]),
	}, nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func deriveKeyFromToken(token, shareID, saltBase64 string) ([]byte, error) {
	ikm, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}

	var salt []byte
	if strings.TrimSpace(saltBase64) != "" {
		salt, err = base64.StdEncoding.DecodeString(saltBase64)
		if err != nil {
			return nil, fmt.Errorf("decode token salt: %w", err)
		}
	}

	key := make([]byte, 32)
	reader := hkdf.New(sha256.New, ikm, salt, []byte(shareID))
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, fmt.Errorf("derive hkdf key: %w", err)
	}
	return key, nil
}

func deriveKeyFromTokenAndPin(token, pin, saltHex string) ([]byte, error) {
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil, fmt.Errorf("decode pin salt: %w", err)
	}
	return argon2.IDKey([]byte(token+pin), salt, 3, 64*1024, 1, 32), nil
}

func randomBase64URL(length int) (string, error) {
	value := make([]byte, length)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func randomBase64Std(length int) (string, error) {
	value := make([]byte, length)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(value), nil
}

func randomHex(length int) (string, error) {
	value := make([]byte, length)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func nullableAuditInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func zeroBytes(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

func errorsIsNoRows(err error) bool {
	return err != nil && (err == sql.ErrNoRows || err == pgx.ErrNoRows)
}

func errorsAsResolver(err error, target **credentialresolver.RequestError) bool {
	if err == nil {
		return false
	}
	var reqErr *credentialresolver.RequestError
	if errors.As(err, &reqErr) {
		*target = reqErr
		return true
	}
	return false
}
