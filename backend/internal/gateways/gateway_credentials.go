package gateways

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type credentialFields struct {
	username *encryptedField
	password *encryptedField
	sshKey   *encryptedField
}

func (s Service) prepareCredentialFields(ctx context.Context, userID, gatewayType string, username, password, sshPrivateKey *string) (credentialFields, error) {
	gatewayType = strings.ToUpper(strings.TrimSpace(gatewayType))
	if gatewayType != "SSH_BASTION" {
		if hasText(username) || hasText(password) || hasText(sshPrivateKey) {
			switch gatewayType {
			case "MANAGED_SSH":
				return credentialFields{}, &requestError{status: http.StatusBadRequest, message: "MANAGED_SSH gateways use the server-managed key pair. Do not supply credentials."}
			case "DB_PROXY":
				return credentialFields{}, &requestError{status: http.StatusBadRequest, message: "DB_PROXY gateways do not use direct credentials. Credentials are injected per-session from the vault."}
			default:
				return credentialFields{}, &requestError{status: http.StatusBadRequest, message: "Credentials can only be set for SSH_BASTION gateways"}
			}
		}
		return credentialFields{}, nil
	}
	if !hasText(username) && !hasText(password) && !hasText(sshPrivateKey) {
		return credentialFields{}, nil
	}
	masterKey, err := s.getVaultKey(ctx, userID)
	if err != nil {
		return credentialFields{}, err
	}
	if len(masterKey) == 0 {
		return credentialFields{}, &requestError{status: http.StatusForbidden, message: "Vault is locked. Please unlock it first."}
	}
	defer zeroBytes(masterKey)

	var result credentialFields
	if hasText(username) {
		field, err := encryptValue(masterKey, strings.TrimSpace(*username))
		if err != nil {
			return credentialFields{}, err
		}
		result.username = &field
	}
	if hasText(password) {
		field, err := encryptValue(masterKey, *password)
		if err != nil {
			return credentialFields{}, err
		}
		result.password = &field
	}
	if hasText(sshPrivateKey) {
		field, err := encryptValue(masterKey, *sshPrivateKey)
		if err != nil {
			return credentialFields{}, err
		}
		result.sshKey = &field
	}
	return result, nil
}

func (s Service) mergeCredentialFields(ctx context.Context, userID string, record gatewayRecord, input updatePayload) (credentialFields, error) {
	result := credentialFields{
		username: encryptedFieldFromRecord(record.EncryptedUsername, record.UsernameIV, record.UsernameTag),
		password: encryptedFieldFromRecord(record.EncryptedPassword, record.PasswordIV, record.PasswordTag),
		sshKey:   encryptedFieldFromRecord(record.EncryptedSSHKey, record.SSHKeyIV, record.SSHKeyTag),
	}
	if record.Type != "SSH_BASTION" {
		if input.Username.Present || input.Password.Present || input.SSHPrivateKey.Present {
			return credentialFields{}, &requestError{status: http.StatusBadRequest, message: "Credentials can only be set for SSH_BASTION gateways"}
		}
		return result, nil
	}
	if !input.Username.Present && !input.Password.Present && !input.SSHPrivateKey.Present {
		return result, nil
	}
	masterKey, err := s.getVaultKey(ctx, userID)
	if err != nil {
		return credentialFields{}, err
	}
	if len(masterKey) == 0 {
		return credentialFields{}, &requestError{status: http.StatusForbidden, message: "Vault is locked. Please unlock it first."}
	}
	defer zeroBytes(masterKey)

	if input.Username.Present {
		if input.Username.Value == nil {
			result.username = nil
		} else {
			field, err := encryptValue(masterKey, strings.TrimSpace(*input.Username.Value))
			if err != nil {
				return credentialFields{}, err
			}
			result.username = &field
		}
	}
	if input.Password.Present {
		if input.Password.Value == nil {
			result.password = nil
		} else {
			field, err := encryptValue(masterKey, *input.Password.Value)
			if err != nil {
				return credentialFields{}, err
			}
			result.password = &field
		}
	}
	if input.SSHPrivateKey.Present {
		if input.SSHPrivateKey.Value == nil {
			result.sshKey = nil
		} else {
			field, err := encryptValue(masterKey, *input.SSHPrivateKey.Value)
			if err != nil {
				return credentialFields{}, err
			}
			result.sshKey = &field
		}
	}
	return result, nil
}

func (s Service) insertAuditLogTx(ctx context.Context, tx pgx.Tx, userID, action, targetID string, details map[string]any, ipAddress string) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal gateway audit details: %w", err)
	}
	_, err = tx.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details, "ipAddress", "createdAt")
VALUES ($1, $2, $3::"AuditAction", 'Gateway', $4, $5::jsonb, NULLIF($6, ''), NOW())
`, uuid.NewString(), userID, action, targetID, string(payload), ipAddress)
	if err != nil {
		return fmt.Errorf("insert gateway audit log: %w", err)
	}
	return nil
}

func encryptedFieldFromRecord(ciphertext, iv, tag *string) *encryptedField {
	if ciphertext == nil || iv == nil || tag == nil {
		return nil
	}
	return &encryptedField{
		Ciphertext: *ciphertext,
		IV:         *iv,
		Tag:        *tag,
	}
}

func encryptedFieldParts(field *encryptedField) (ciphertext, iv, tag *string) {
	if field == nil {
		return nil, nil, nil
	}
	return &field.Ciphertext, &field.IV, &field.Tag
}
