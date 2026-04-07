package gateways

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

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
	if err := scanSSHKeyPair(tx.QueryRow(ctx, `
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
`, tenantID, autoRotateEnabled, rotationIntervalDays, expiresAt), &updated); err != nil {
		return sshKeyPairResponse{}, fmt.Errorf("update ssh key rotation policy: %w", err)
	}

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
