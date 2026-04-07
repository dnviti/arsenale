package gateways

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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
	if err := scanSSHKeyPair(tx.QueryRow(ctx, `
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
`, id, tenantID, encrypted.Ciphertext, encrypted.IV, encrypted.Tag, publicKey, fingerprint, autoRotateEnabled, rotationIntervalDays, expiresAt, lastAutoRotatedAt), &created); err != nil {
		return sshKeyPairRecord{}, fmt.Errorf("insert ssh key pair: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return sshKeyPairRecord{}, fmt.Errorf("commit ssh key transaction: %w", err)
	}
	return created, nil
}

func (s Service) loadSSHKeyPair(ctx context.Context, tenantID string) (sshKeyPairRecord, error) {
	if s.DB == nil {
		return sshKeyPairRecord{}, fmt.Errorf("database is unavailable")
	}

	var record sshKeyPairRecord
	err := scanSSHKeyPair(s.DB.QueryRow(ctx, `
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
`, tenantID), &record)
	if err != nil {
		if err == pgx.ErrNoRows {
			return sshKeyPairRecord{}, &requestError{status: http.StatusNotFound, message: "No SSH key pair found for this tenant"}
		}
		return sshKeyPairRecord{}, fmt.Errorf("load ssh key pair: %w", err)
	}
	return record, nil
}

func scanSSHKeyPair(row rowScanner, record *sshKeyPairRecord) error {
	var expiresAt sql.NullTime
	var lastAutoRotatedAt sql.NullTime
	if err := row.Scan(
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
	); err != nil {
		return err
	}
	record.ExpiresAt = nullTimePtr(expiresAt)
	record.LastAutoRotatedAt = nullTimePtr(lastAutoRotatedAt)
	return nil
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
