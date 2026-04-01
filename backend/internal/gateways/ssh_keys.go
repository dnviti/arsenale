package gateways

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/ssh"
)

type sshKeyPairResponse struct {
	ID                   string     `json:"id"`
	PublicKey            string     `json:"publicKey"`
	Fingerprint          string     `json:"fingerprint"`
	Algorithm            string     `json:"algorithm"`
	ExpiresAt            *time.Time `json:"expiresAt"`
	AutoRotateEnabled    bool       `json:"autoRotateEnabled"`
	RotationIntervalDays int        `json:"rotationIntervalDays"`
	LastAutoRotatedAt    *time.Time `json:"lastAutoRotatedAt"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

type sshKeyRotationStatus struct {
	AutoRotateEnabled    bool       `json:"autoRotateEnabled"`
	RotationIntervalDays int        `json:"rotationIntervalDays"`
	ExpiresAt            *time.Time `json:"expiresAt"`
	LastAutoRotatedAt    *time.Time `json:"lastAutoRotatedAt"`
	NextRotationDate     *time.Time `json:"nextRotationDate"`
	DaysUntilRotation    *int       `json:"daysUntilRotation"`
	KeyExists            bool       `json:"keyExists"`
}

type sshKeyPairRecord struct {
	ID                   string
	TenantID             string
	EncryptedPrivateKey  string
	PrivateKeyIV         string
	PrivateKeyTag        string
	PublicKey            string
	Fingerprint          string
	Algorithm            string
	ExpiresAt            *time.Time
	AutoRotateEnabled    bool
	RotationIntervalDays int
	LastAutoRotatedAt    *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type rotationPolicyPayload struct {
	AutoRotateEnabled    *bool        `json:"autoRotateEnabled"`
	RotationIntervalDays *int         `json:"rotationIntervalDays"`
	ExpiresAt            optionalTime `json:"expiresAt"`
}

type optionalTime struct {
	Present bool
	Value   *time.Time
}

func (o *optionalTime) UnmarshalJSON(data []byte) error {
	o.Present = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	parsed = parsed.UTC()
	o.Value = &parsed
	return nil
}

func (s Service) GenerateSSHKeyPair(ctx context.Context, userID, tenantID, ipAddress string) (sshKeyPairResponse, error) {
	if s.DB == nil {
		return sshKeyPairResponse{}, fmt.Errorf("database is unavailable")
	}
	if _, err := s.loadSSHKeyPair(ctx, tenantID); err == nil {
		return sshKeyPairResponse{}, &requestError{status: http.StatusConflict, message: "SSH key pair already exists for this tenant. Use rotate to replace it."}
	} else if !errorsIsNotFound(err) {
		return sshKeyPairResponse{}, err
	}

	record, err := s.insertSSHKeyPair(ctx, tenantID, nil, false)
	if err != nil {
		return sshKeyPairResponse{}, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return sshKeyPairResponse{}, fmt.Errorf("begin ssh key audit transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.insertSSHKeyAuditLogTx(ctx, tx, userID, "SSH_KEY_GENERATE", record.ID, nil, ipAddress); err != nil {
		return sshKeyPairResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sshKeyPairResponse{}, fmt.Errorf("commit ssh key audit transaction: %w", err)
	}
	return sshKeyPairRecordToResponse(record), nil
}

func (s Service) GetSSHKeyPair(ctx context.Context, tenantID string) (sshKeyPairResponse, error) {
	record, err := s.loadSSHKeyPair(ctx, tenantID)
	if err != nil {
		return sshKeyPairResponse{}, err
	}
	return sshKeyPairRecordToResponse(record), nil
}

func (s Service) GetSSHPrivateKey(ctx context.Context, tenantID string) (string, error) {
	record, err := s.loadSSHKeyPair(ctx, tenantID)
	if err != nil {
		return "", err
	}
	if len(s.ServerEncryptionKey) != 32 {
		return "", fmt.Errorf("server encryption key is unavailable")
	}
	privateKey, err := decryptEncryptedField(s.ServerEncryptionKey, encryptedField{
		Ciphertext: record.EncryptedPrivateKey,
		IV:         record.PrivateKeyIV,
		Tag:        record.PrivateKeyTag,
	})
	if err != nil {
		return "", fmt.Errorf("decrypt ssh private key: %w", err)
	}
	return privateKey, nil
}

func (s Service) RotateSSHKeyPair(ctx context.Context, userID, tenantID, ipAddress string) (sshKeyPairResponse, error) {
	if s.DB == nil {
		return sshKeyPairResponse{}, fmt.Errorf("database is unavailable")
	}

	var existing *sshKeyPairRecord
	record, err := s.loadSSHKeyPair(ctx, tenantID)
	if err == nil {
		existing = &record
	} else if !errorsIsNotFound(err) {
		return sshKeyPairResponse{}, err
	}

	rotated, err := s.insertSSHKeyPair(ctx, tenantID, existing, true)
	if err != nil {
		return sshKeyPairResponse{}, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return sshKeyPairResponse{}, fmt.Errorf("begin ssh key audit transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := s.insertSSHKeyAuditLogTx(ctx, tx, userID, "SSH_KEY_ROTATE", rotated.ID, nil, ipAddress); err != nil {
		return sshKeyPairResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sshKeyPairResponse{}, fmt.Errorf("commit ssh key audit transaction: %w", err)
	}
	return sshKeyPairRecordToResponse(rotated), nil
}

func (s Service) UpdateSSHKeyRotationPolicy(ctx context.Context, userID, tenantID, ipAddress string, input rotationPolicyPayload) (sshKeyPairResponse, error) {
	if s.DB == nil {
		return sshKeyPairResponse{}, fmt.Errorf("database is unavailable")
	}
	if err := validateRotationPolicyPayload(input); err != nil {
		return sshKeyPairResponse{}, err
	}

	record, err := s.loadSSHKeyPair(ctx, tenantID)
	if err != nil {
		return sshKeyPairResponse{}, err
	}

	autoRotateEnabled := record.AutoRotateEnabled
	if input.AutoRotateEnabled != nil {
		autoRotateEnabled = *input.AutoRotateEnabled
	}
	rotationIntervalDays := record.RotationIntervalDays
	if input.RotationIntervalDays != nil {
		rotationIntervalDays = *input.RotationIntervalDays
	}
	expiresAt := record.ExpiresAt
	if input.ExpiresAt.Present {
		expiresAt = input.ExpiresAt.Value
	}
	if autoRotateEnabled && expiresAt == nil {
		expiry := time.Now().UTC().Add(time.Duration(rotationIntervalDays) * 24 * time.Hour)
		expiresAt = &expiry
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return sshKeyPairResponse{}, fmt.Errorf("begin ssh key policy transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var updated sshKeyPairRecord
	var expiresAtValue sql.NullTime
	var lastAutoRotatedAt sql.NullTime
	if err := tx.QueryRow(ctx, `
UPDATE "SshKeyPair"
   SET "autoRotateEnabled" = $2,
       "rotationIntervalDays" = $3,
       "expiresAt" = $4,
       "updatedAt" = NOW()
 WHERE "tenantId" = $1
 RETURNING id,
           "tenantId",
           "encryptedPrivateKey",
           "privateKeyIV",
           "privateKeyTag",
           "publicKey",
           fingerprint,
           algorithm,
           "expiresAt",
           "autoRotateEnabled",
           "rotationIntervalDays",
           "lastAutoRotatedAt",
           "createdAt",
           "updatedAt"
`, tenantID, autoRotateEnabled, rotationIntervalDays, expiresAt).Scan(
		&updated.ID,
		&updated.TenantID,
		&updated.EncryptedPrivateKey,
		&updated.PrivateKeyIV,
		&updated.PrivateKeyTag,
		&updated.PublicKey,
		&updated.Fingerprint,
		&updated.Algorithm,
		&expiresAtValue,
		&updated.AutoRotateEnabled,
		&updated.RotationIntervalDays,
		&lastAutoRotatedAt,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	); err != nil {
		return sshKeyPairResponse{}, fmt.Errorf("update ssh key rotation policy: %w", err)
	}
	updated.ExpiresAt = nullTimePtr(expiresAtValue)
	updated.LastAutoRotatedAt = nullTimePtr(lastAutoRotatedAt)

	details := map[string]any{}
	if input.AutoRotateEnabled != nil {
		details["autoRotateEnabled"] = *input.AutoRotateEnabled
	}
	if input.RotationIntervalDays != nil {
		details["rotationIntervalDays"] = *input.RotationIntervalDays
	}
	if input.ExpiresAt.Present {
		if input.ExpiresAt.Value == nil {
			details["expiresAt"] = nil
		} else {
			details["expiresAt"] = input.ExpiresAt.Value.UTC().Format(time.RFC3339)
		}
	}
	if err := s.insertSSHKeyAuditLogTx(ctx, tx, userID, "SSH_KEY_ROTATE", updated.ID, map[string]any{"policyUpdate": details}, ipAddress); err != nil {
		return sshKeyPairResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sshKeyPairResponse{}, fmt.Errorf("commit ssh key policy transaction: %w", err)
	}
	return sshKeyPairRecordToResponse(updated), nil
}

func (s Service) GetSSHKeyRotationStatus(ctx context.Context, tenantID string) (sshKeyRotationStatus, error) {
	record, err := s.loadSSHKeyPair(ctx, tenantID)
	if err != nil {
		if errorsIsNotFound(err) {
			return sshKeyRotationStatus{
				AutoRotateEnabled:    false,
				RotationIntervalDays: 90,
				KeyExists:            false,
			}, nil
		}
		return sshKeyRotationStatus{}, err
	}
	return computeSSHKeyRotationStatus(record.AutoRotateEnabled, record.RotationIntervalDays, record.ExpiresAt, record.LastAutoRotatedAt, time.Now().UTC(), sshKeyRotationAdvanceDays()), nil
}

func (s Service) insertSSHKeyPair(ctx context.Context, tenantID string, existing *sshKeyPairRecord, replace bool) (sshKeyPairRecord, error) {
	if len(s.ServerEncryptionKey) != 32 {
		return sshKeyPairRecord{}, fmt.Errorf("server encryption key is unavailable")
	}
	privatePEM, publicKey, fingerprint, err := generateSSHKeyMaterial()
	if err != nil {
		return sshKeyPairRecord{}, err
	}
	encrypted, err := encryptValue(s.ServerEncryptionKey, privatePEM)
	if err != nil {
		return sshKeyPairRecord{}, fmt.Errorf("encrypt ssh private key: %w", err)
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return sshKeyPairRecord{}, fmt.Errorf("begin ssh key transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if replace {
		if _, err := tx.Exec(ctx, `DELETE FROM "SshKeyPair" WHERE "tenantId" = $1`, tenantID); err != nil {
			return sshKeyPairRecord{}, fmt.Errorf("delete existing ssh key pair: %w", err)
		}
	}

	id := uuid.NewString()
	autoRotateEnabled := false
	rotationIntervalDays := 90
	var expiresAt *time.Time
	var lastAutoRotatedAt *time.Time
	if existing != nil {
		autoRotateEnabled = existing.AutoRotateEnabled
		rotationIntervalDays = existing.RotationIntervalDays
		expiresAt = existing.ExpiresAt
		lastAutoRotatedAt = existing.LastAutoRotatedAt
	}

	var created sshKeyPairRecord
	var expiresAtValue sql.NullTime
	var lastAutoRotatedAtValue sql.NullTime
	if err := tx.QueryRow(ctx, `
INSERT INTO "SshKeyPair" (
	id,
	"tenantId",
	"encryptedPrivateKey",
	"privateKeyIV",
	"privateKeyTag",
	"publicKey",
	fingerprint,
	algorithm,
	"updatedAt",
	"autoRotateEnabled",
	"rotationIntervalDays",
	"expiresAt",
	"lastAutoRotatedAt"
)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'ed25519', NOW(), $8, $9, $10, $11)
RETURNING id,
          "tenantId",
          "encryptedPrivateKey",
          "privateKeyIV",
          "privateKeyTag",
          "publicKey",
          fingerprint,
          algorithm,
          "expiresAt",
          "autoRotateEnabled",
          "rotationIntervalDays",
          "lastAutoRotatedAt",
          "createdAt",
          "updatedAt"
`, id, tenantID, encrypted.Ciphertext, encrypted.IV, encrypted.Tag, publicKey, fingerprint, autoRotateEnabled, rotationIntervalDays, expiresAt, lastAutoRotatedAt).Scan(
		&created.ID,
		&created.TenantID,
		&created.EncryptedPrivateKey,
		&created.PrivateKeyIV,
		&created.PrivateKeyTag,
		&created.PublicKey,
		&created.Fingerprint,
		&created.Algorithm,
		&expiresAtValue,
		&created.AutoRotateEnabled,
		&created.RotationIntervalDays,
		&lastAutoRotatedAtValue,
		&created.CreatedAt,
		&created.UpdatedAt,
	); err != nil {
		return sshKeyPairRecord{}, fmt.Errorf("insert ssh key pair: %w", err)
	}
	created.ExpiresAt = nullTimePtr(expiresAtValue)
	created.LastAutoRotatedAt = nullTimePtr(lastAutoRotatedAtValue)

	if err := tx.Commit(ctx); err != nil {
		return sshKeyPairRecord{}, fmt.Errorf("commit ssh key transaction: %w", err)
	}
	return created, nil
}

func (s Service) loadSSHKeyPair(ctx context.Context, tenantID string) (sshKeyPairRecord, error) {
	if s.DB == nil {
		return sshKeyPairRecord{}, fmt.Errorf("database is unavailable")
	}
	var (
		record            sshKeyPairRecord
		expiresAt         sql.NullTime
		lastAutoRotatedAt sql.NullTime
	)
	err := s.DB.QueryRow(ctx, `
SELECT id,
       "tenantId",
       "encryptedPrivateKey",
       "privateKeyIV",
       "privateKeyTag",
       "publicKey",
       fingerprint,
       algorithm,
       "expiresAt",
       "autoRotateEnabled",
       "rotationIntervalDays",
       "lastAutoRotatedAt",
       "createdAt",
       "updatedAt"
  FROM "SshKeyPair"
 WHERE "tenantId" = $1
`, tenantID).Scan(
		&record.ID,
		&record.TenantID,
		&record.EncryptedPrivateKey,
		&record.PrivateKeyIV,
		&record.PrivateKeyTag,
		&record.PublicKey,
		&record.Fingerprint,
		&record.Algorithm,
		&expiresAt,
		&record.AutoRotateEnabled,
		&record.RotationIntervalDays,
		&lastAutoRotatedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return sshKeyPairRecord{}, &requestError{status: http.StatusNotFound, message: "No SSH key pair found for this tenant"}
		}
		return sshKeyPairRecord{}, fmt.Errorf("load ssh key pair: %w", err)
	}
	record.ExpiresAt = nullTimePtr(expiresAt)
	record.LastAutoRotatedAt = nullTimePtr(lastAutoRotatedAt)
	return record, nil
}

func generateSSHKeyMaterial() (privatePEM string, publicKey string, fingerprint string, err error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", "", fmt.Errorf("generate ed25519 key pair: %w", err)
	}

	pkcs8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", "", fmt.Errorf("marshal private key: %w", err)
	}
	privateBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})

	sshPublicKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		return "", "", "", fmt.Errorf("marshal public key: %w", err)
	}

	return string(privateBytes), strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPublicKey))), ssh.FingerprintSHA256(sshPublicKey), nil
}

func computeSSHKeyRotationStatus(autoRotateEnabled bool, rotationIntervalDays int, expiresAt, lastAutoRotatedAt *time.Time, now time.Time, advanceDays int) sshKeyRotationStatus {
	status := sshKeyRotationStatus{
		AutoRotateEnabled:    autoRotateEnabled,
		RotationIntervalDays: rotationIntervalDays,
		ExpiresAt:            expiresAt,
		LastAutoRotatedAt:    lastAutoRotatedAt,
		KeyExists:            true,
	}
	if !autoRotateEnabled || expiresAt == nil {
		return status
	}

	nextRotationDate := expiresAt.UTC().Add(-time.Duration(advanceDays) * 24 * time.Hour)
	daysUntilRotation := int(nextRotationDate.Sub(now.UTC()).Hours() / 24)
	if nextRotationDate.After(now.UTC()) && nextRotationDate.Sub(now.UTC())%(24*time.Hour) != 0 {
		daysUntilRotation++
	}
	if daysUntilRotation < 0 {
		daysUntilRotation = 0
	}
	status.NextRotationDate = &nextRotationDate
	status.DaysUntilRotation = &daysUntilRotation
	return status
}

func sshKeyPairRecordToResponse(record sshKeyPairRecord) sshKeyPairResponse {
	return sshKeyPairResponse{
		ID:                   record.ID,
		PublicKey:            record.PublicKey,
		Fingerprint:          record.Fingerprint,
		Algorithm:            record.Algorithm,
		ExpiresAt:            record.ExpiresAt,
		AutoRotateEnabled:    record.AutoRotateEnabled,
		RotationIntervalDays: record.RotationIntervalDays,
		LastAutoRotatedAt:    record.LastAutoRotatedAt,
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
	}
}

func (s Service) insertSSHKeyAuditLogTx(ctx context.Context, tx pgx.Tx, userID, action, targetID string, details map[string]any, ipAddress string) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal ssh key audit details: %w", err)
	}
	_, err = tx.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details, "ipAddress", "createdAt")
VALUES ($1, $2, $3::"AuditAction", 'SshKeyPair', $4, $5::jsonb, NULLIF($6, ''), NOW())
`, uuid.NewString(), userID, action, targetID, string(payload), ipAddress)
	if err != nil {
		return fmt.Errorf("insert ssh key audit log: %w", err)
	}
	return nil
}

func validateRotationPolicyPayload(input rotationPolicyPayload) error {
	if input.RotationIntervalDays != nil && (*input.RotationIntervalDays < 1 || *input.RotationIntervalDays > 365) {
		return &requestError{status: http.StatusBadRequest, message: "rotationIntervalDays must be between 1 and 365"}
	}
	return nil
}

func sshKeyRotationAdvanceDays() int {
	value := strings.TrimSpace(os.Getenv("KEY_ROTATION_ADVANCE_DAYS"))
	if value == "" {
		return 7
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 7
	}
	return parsed
}

func errorsIsNotFound(err error) bool {
	var reqErr *requestError
	return errors.As(err, &reqErr) && reqErr.status == http.StatusNotFound
}
