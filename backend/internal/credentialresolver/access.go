package credentialresolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (r Resolver) Resolve(ctx context.Context, userID, secretID, connectionType, tenantID string) (SecretCredentials, error) {
	record, accessType, err := r.requireViewAccess(ctx, userID, secretID, tenantID)
	if err != nil {
		return SecretCredentials{}, err
	}

	decryptedJSON, err := r.decryptSecretPayload(ctx, userID, record, accessType)
	if err != nil {
		return SecretCredentials{}, err
	}

	var payload secretPayload
	if err := json.Unmarshal([]byte(decryptedJSON), &payload); err != nil {
		return SecretCredentials{}, fmt.Errorf("decode secret payload: %w", err)
	}

	secretType := strings.ToUpper(strings.TrimSpace(payload.Type))
	switch secretType {
	case "LOGIN":
		return SecretCredentials{
			Type:         secretType,
			Username:     payload.Username,
			Password:     payload.Password,
			Domain:       payload.Domain,
			SecretAccess: accessType,
		}, nil
	case "SSH_KEY":
		if strings.EqualFold(strings.TrimSpace(connectionType), "RDP") {
			return SecretCredentials{}, &RequestError{Status: http.StatusBadRequest, Message: "SSH_KEY secrets cannot be used with RDP connections"}
		}
		return SecretCredentials{
			Type:         secretType,
			Username:     payload.Username,
			Password:     "",
			Domain:       payload.Domain,
			PrivateKey:   payload.PrivateKey,
			Passphrase:   payload.Passphrase,
			SecretAccess: accessType,
		}, nil
	default:
		return SecretCredentials{}, &RequestError{
			Status:  http.StatusBadRequest,
			Message: fmt.Sprintf(`Secret type "%s" is not compatible with connection credentials. Use LOGIN or SSH_KEY.`, strings.TrimSpace(payload.Type)),
		}
	}
}

func (r Resolver) ValidateConnectionSecretReference(ctx context.Context, userID, secretID, connectionType, tenantID string) error {
	record, _, err := r.requireViewAccess(ctx, userID, secretID, tenantID)
	if err != nil {
		return err
	}
	return validateConnectionSecretReferenceType(record.Type, connectionType)
}

func validateConnectionSecretReferenceType(secretType, connectionType string) error {
	switch strings.ToUpper(strings.TrimSpace(secretType)) {
	case "LOGIN":
		return nil
	case "SSH_KEY":
		switch strings.ToUpper(strings.TrimSpace(connectionType)) {
		case "RDP", "VNC":
			return &RequestError{Status: http.StatusBadRequest, Message: "SSH_KEY secrets cannot be used with RDP/VNC connections"}
		default:
			return nil
		}
	default:
		return &RequestError{Status: http.StatusBadRequest, Message: "Credential secret must be of type LOGIN or SSH_KEY"}
	}
}

func (r Resolver) requireViewAccess(ctx context.Context, userID, secretID, tenantID string) (secretRecord, string, error) {
	record, err := r.loadSecret(ctx, secretID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return secretRecord{}, "", &RequestError{Status: http.StatusNotFound, Message: "Credential secret not found or inaccessible"}
		}
		return secretRecord{}, "", err
	}

	switch record.Scope {
	case "PERSONAL":
		if strings.TrimSpace(record.UserID) == userID {
			return record, "owner", nil
		}
	case "TEAM":
		if record.TeamID != nil && *record.TeamID != "" {
			if tenantID != "" && (record.TeamTenantID == nil || *record.TeamTenantID != tenantID) {
				break
			}
			ok, err := r.hasTeamMembership(ctx, *record.TeamID, userID)
			if err != nil {
				return secretRecord{}, "", err
			}
			if ok {
				return record, "team", nil
			}
		}
	case "TENANT":
		if record.TenantID != nil && *record.TenantID != "" {
			if tenantID != "" && *record.TenantID != tenantID {
				break
			}
			ok, err := r.hasTenantVaultMembership(ctx, *record.TenantID, userID)
			if err != nil {
				return secretRecord{}, "", err
			}
			if ok {
				return record, "tenant", nil
			}
		}
	}

	shared, err := r.hasSharedSecret(ctx, secretID, userID)
	if err != nil {
		return secretRecord{}, "", err
	}
	if shared {
		return record, "shared", nil
	}

	return secretRecord{}, "", &RequestError{Status: http.StatusNotFound, Message: "Credential secret not found or inaccessible"}
}

func (r Resolver) loadSecret(ctx context.Context, secretID string) (secretRecord, error) {
	if r.DB == nil {
		return secretRecord{}, fmt.Errorf("database is unavailable")
	}

	var record secretRecord
	if err := r.DB.QueryRow(
		ctx,
		`SELECT
			s.id,
			s.type::text,
			s.scope::text,
			s."userId",
			s."teamId",
			s."tenantId",
			t."tenantId",
			s."encryptedData",
			s."dataIV",
			s."dataTag"
		   FROM "VaultSecret" s
		   LEFT JOIN "Team" t ON t.id = s."teamId"
		  WHERE s.id = $1`,
		secretID,
	).Scan(
		&record.ID,
		&record.Type,
		&record.Scope,
		&record.UserID,
		&record.TeamID,
		&record.TenantID,
		&record.TeamTenantID,
		&record.EncryptedData,
		&record.DataIV,
		&record.DataTag,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return secretRecord{}, pgx.ErrNoRows
		}
		return secretRecord{}, fmt.Errorf("load secret: %w", err)
	}
	return record, nil
}

func (r Resolver) hasTeamMembership(ctx context.Context, teamID, userID string) (bool, error) {
	var exists bool
	if err := r.DB.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			  FROM "TeamMember"
			 WHERE "teamId" = $1
			   AND "userId" = $2
		)`,
		teamID,
		userID,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check team membership: %w", err)
	}
	return exists, nil
}

func (r Resolver) hasTenantVaultMembership(ctx context.Context, tenantID, userID string) (bool, error) {
	var exists bool
	if err := r.DB.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			  FROM "TenantVaultMember"
			 WHERE "tenantId" = $1
			   AND "userId" = $2
		)`,
		tenantID,
		userID,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check tenant vault membership: %w", err)
	}
	return exists, nil
}

func (r Resolver) hasSharedSecret(ctx context.Context, secretID, userID string) (bool, error) {
	var exists bool
	if err := r.DB.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			  FROM "SharedSecret"
			 WHERE "secretId" = $1
			   AND "sharedWithUserId" = $2
		)`,
		secretID,
		userID,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check shared secret access: %w", err)
	}
	return exists, nil
}

func (r Resolver) loadSharedSecret(ctx context.Context, secretID, userID string) (sharedSecretRecord, error) {
	if r.DB == nil {
		return sharedSecretRecord{}, fmt.Errorf("database is unavailable")
	}

	var record sharedSecretRecord
	if err := r.DB.QueryRow(
		ctx,
		`SELECT "encryptedData", "dataIV", "dataTag", permission::text
		   FROM "SharedSecret"
		  WHERE "secretId" = $1
		    AND "sharedWithUserId" = $2`,
		secretID,
		userID,
	).Scan(&record.EncryptedData, &record.DataIV, &record.DataTag, &record.Permission); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return sharedSecretRecord{}, &RequestError{Status: http.StatusNotFound, Message: "Credential secret not found or inaccessible"}
		}
		return sharedSecretRecord{}, fmt.Errorf("load shared secret: %w", err)
	}
	return record, nil
}

func (r Resolver) decryptSecretPayload(ctx context.Context, userID string, record secretRecord, accessType string) (string, error) {
	if accessType == "shared" {
		sharedRecord, err := r.loadSharedSecret(ctx, record.ID, userID)
		if err != nil {
			return "", err
		}
		key, _, err := r.requireUserMasterKey(ctx, userID)
		if err != nil {
			return "", err
		}
		defer zeroBytes(key)
		return decryptEncryptedField(key, encryptedField{
			Ciphertext: sharedRecord.EncryptedData,
			IV:         sharedRecord.DataIV,
			Tag:        sharedRecord.DataTag,
		})
	}

	key, err := r.resolveSecretKey(ctx, userID, record)
	if err != nil {
		return "", err
	}
	defer zeroBytes(key)
	return decryptEncryptedField(key, encryptedField{
		Ciphertext: record.EncryptedData,
		IV:         record.DataIV,
		Tag:        record.DataTag,
	})
}
