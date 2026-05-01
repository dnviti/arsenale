package publicshareapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s Service) GetInfo(ctx context.Context, token string) (shareInfoResponse, error) {
	share, err := s.loadShareByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fakeInfo(), nil
		}
		return shareInfoResponse{}, err
	}

	now := time.Now()
	return shareInfoResponse{
		ID:          share.ID,
		SecretName:  share.SecretName,
		SecretType:  share.SecretType,
		HasPin:      share.HasPin,
		ExpiresAt:   share.ExpiresAt.UTC().Format(time.RFC3339),
		IsExpired:   share.ExpiresAt.Before(now),
		IsExhausted: share.MaxAccessCount.Valid && share.AccessCount >= int(share.MaxAccessCount.Int32),
		IsRevoked:   share.IsRevoked,
	}, nil
}

func (s Service) Access(ctx context.Context, token, pin, ipAddress string) (shareAccessResponse, error) {
	share, err := s.loadShareByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return shareAccessResponse{}, &requestError{status: http.StatusGone, message: "Share is no longer available"}
		}
		return shareAccessResponse{}, err
	}

	if share.IsRevoked || share.ExpiresAt.Before(time.Now()) || (share.MaxAccessCount.Valid && share.AccessCount >= int(share.MaxAccessCount.Int32)) {
		return shareAccessResponse{}, &requestError{status: http.StatusGone, message: "Share is no longer available"}
	}
	if share.HasPin && pin == "" {
		return shareAccessResponse{}, &requestError{status: http.StatusBadRequest, message: "PIN is required"}
	}

	var key []byte
	if share.HasPin {
		if !share.PinSalt.Valid {
			return shareAccessResponse{}, fmt.Errorf("share %s missing pin salt", share.ID)
		}
		key, err = deriveKeyFromTokenAndPin(token, pin, share.PinSalt.String)
	} else {
		key, err = deriveKeyFromToken(token, share.ID, share.TokenSalt.String)
	}
	if err != nil {
		return shareAccessResponse{}, fmt.Errorf("derive share key: %w", err)
	}
	defer zero(key)

	plaintext, err := decryptPayload(share.EncryptedData, share.DataIV, share.DataTag, key)
	if err != nil {
		return shareAccessResponse{}, &requestError{status: http.StatusForbidden, message: "Invalid PIN or corrupted data"}
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(plaintext), &data); err != nil {
		return shareAccessResponse{}, fmt.Errorf("decode share payload: %w", err)
	}

	if _, err := s.DB.Exec(ctx, `UPDATE "ExternalSecretShare" SET "accessCount" = "accessCount" + 1 WHERE id = $1`, share.ID); err != nil {
		return shareAccessResponse{}, fmt.Errorf("increment share access count: %w", err)
	}
	if err := s.insertAuditLog(ctx, nil, "SECRET_EXTERNAL_ACCESS", "ExternalSecretShare", share.ID, map[string]any{
		"secretId":   share.SecretID,
		"secretName": share.SecretName,
	}, ipAddress); err != nil {
		return shareAccessResponse{}, fmt.Errorf("insert access audit log: %w", err)
	}

	return shareAccessResponse{
		SecretName: share.SecretName,
		SecretType: share.SecretType,
		Data:       data,
	}, nil
}

func (s Service) loadShareByToken(ctx context.Context, token string) (shareRecord, error) {
	tokenHash := hashToken(token)
	row := s.DB.QueryRow(ctx, `
SELECT
	id,
	"secretId",
	"secretName",
	"secretType"::text,
	"encryptedData",
	"dataIV",
	"dataTag",
	"hasPin",
	"pinSalt",
	"tokenSalt",
	"expiresAt",
	"maxAccessCount",
	"accessCount",
	"isRevoked"
FROM "ExternalSecretShare"
WHERE "tokenHash" = $1
`, tokenHash)

	var share shareRecord
	if err := row.Scan(
		&share.ID,
		&share.SecretID,
		&share.SecretName,
		&share.SecretType,
		&share.EncryptedData,
		&share.DataIV,
		&share.DataTag,
		&share.HasPin,
		&share.PinSalt,
		&share.TokenSalt,
		&share.ExpiresAt,
		&share.MaxAccessCount,
		&share.AccessCount,
		&share.IsRevoked,
	); err != nil {
		if strings.Contains(err.Error(), "no rows") {
			return shareRecord{}, sql.ErrNoRows
		}
		return shareRecord{}, fmt.Errorf("load external share: %w", err)
	}
	return share, nil
}

func (s Service) insertAuditLog(ctx context.Context, userID *string, action, targetType, targetID string, details map[string]any, ipAddress string) error {
	var rawDetails []byte
	if details != nil {
		encoded, err := json.Marshal(details)
		if err != nil {
			return fmt.Errorf("marshal audit details: %w", err)
		}
		rawDetails = encoded
	}

	_, err := s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (
	id,
	"userId",
	action,
	"targetType",
	"targetId",
	details,
	"ipAddress",
	"createdAt"
) VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), NOW())
`, uuid.NewString(), nullableString(userID), action, nullableStringValue(targetType), nullableStringValue(targetID), rawDetails, strings.TrimSpace(ipAddress))
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func fakeInfo() shareInfoResponse {
	return shareInfoResponse{
		ID:          uuid.NewString(),
		SecretName:  "Shared Secret",
		SecretType:  "LOGIN",
		HasPin:      true,
		ExpiresAt:   time.Unix(0, 0).UTC().Format(time.RFC3339),
		IsExpired:   true,
		IsExhausted: false,
		IsRevoked:   false,
	}
}
