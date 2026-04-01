package credentialresolver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r Resolver) ResolveSecret(ctx context.Context, userID, secretID, tenantID string) (SecretDetail, error) {
	record, accessType, err := r.requireViewAccess(ctx, userID, secretID, tenantID)
	if err != nil {
		return SecretDetail{}, err
	}

	summary, err := r.loadSecretSummary(ctx, record.ID)
	if err != nil {
		return SecretDetail{}, err
	}

	decryptedJSON, err := r.decryptSecretPayload(ctx, userID, record, accessType)
	if err != nil {
		return SecretDetail{}, err
	}
	if !json.Valid([]byte(decryptedJSON)) {
		return SecretDetail{}, fmt.Errorf("decode secret payload: invalid json")
	}

	detail := SecretDetail{
		SecretSummary: summary,
		Data:          json.RawMessage(decryptedJSON),
	}

	if accessType == "shared" {
		sharedRecord, err := r.loadSharedSecret(ctx, record.ID, userID)
		if err != nil {
			return SecretDetail{}, err
		}
		detail.Shared = true
		detail.Permission = sharedRecord.Permission
	}

	return detail, nil
}

func (r Resolver) ListSecretVersions(ctx context.Context, userID, secretID, tenantID string) ([]SecretVersion, error) {
	if _, _, err := r.requireViewAccess(ctx, userID, secretID, tenantID); err != nil {
		return nil, err
	}
	if r.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}

	rows, err := r.DB.Query(ctx, `
SELECT
	v.id,
	v.version,
	v."changedBy",
	v."changeNote",
	v."createdAt",
	u.email,
	u.username
FROM "VaultSecretVersion" v
JOIN "User" u ON u.id = v."changedBy"
WHERE v."secretId" = $1
ORDER BY v.version DESC
`, secretID)
	if err != nil {
		return nil, fmt.Errorf("list secret versions: %w", err)
	}
	defer rows.Close()

	versions := make([]SecretVersion, 0)
	for rows.Next() {
		var (
			item       SecretVersion
			changeNote sql.NullString
			username   sql.NullString
			email      string
		)
		if err := rows.Scan(
			&item.ID,
			&item.Version,
			&item.ChangedBy,
			&changeNote,
			&item.CreatedAt,
			&email,
			&username,
		); err != nil {
			return nil, fmt.Errorf("scan secret version: %w", err)
		}
		if changeNote.Valid {
			item.ChangeNote = &changeNote.String
		}

		changer := &SecretVersionUser{Email: email}
		if username.Valid {
			changer.Username = &username.String
		}
		item.Changer = changer
		versions = append(versions, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate secret versions: %w", err)
	}

	return versions, nil
}

func (r Resolver) ResolveSecretVersionData(ctx context.Context, userID, secretID, tenantID string, version int) (json.RawMessage, error) {
	record, accessType, err := r.requireViewAccess(ctx, userID, secretID, tenantID)
	if err != nil {
		return nil, err
	}
	if accessType == "shared" {
		return nil, &RequestError{Status: 403, Message: "Version data is not available for shared secrets"}
	}
	if r.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}

	var field encryptedField
	if err := r.DB.QueryRow(
		ctx,
		`SELECT "encryptedData", "dataIV", "dataTag"
		   FROM "VaultSecretVersion"
		  WHERE "secretId" = $1
		    AND version = $2`,
		secretID,
		version,
	).Scan(&field.Ciphertext, &field.IV, &field.Tag); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &RequestError{Status: 404, Message: "Version not found"}
		}
		return nil, fmt.Errorf("load secret version: %w", err)
	}

	key, err := r.resolveSecretKey(ctx, userID, record)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(key)

	decryptedJSON, err := decryptEncryptedField(key, field)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret version: %w", err)
	}
	if !json.Valid([]byte(decryptedJSON)) {
		return nil, fmt.Errorf("decode secret version: invalid json")
	}

	return json.RawMessage(decryptedJSON), nil
}

func (r Resolver) loadSecretSummary(ctx context.Context, secretID string) (SecretSummary, error) {
	if r.DB == nil {
		return SecretSummary{}, fmt.Errorf("database is unavailable")
	}

	var (
		item      SecretSummary
		metadata  []byte
		expiresAt sql.NullTime
	)
	if err := r.DB.QueryRow(
		ctx,
		`SELECT
			id,
			name,
			description,
			type::text,
			scope::text,
			"teamId",
			"tenantId",
			"folderId",
			metadata,
			tags,
			"isFavorite",
			COALESCE("pwnedCount", 0),
			"expiresAt",
			"currentVersion",
			"createdAt",
			"updatedAt"
		   FROM "VaultSecret"
		  WHERE id = $1`,
		secretID,
	).Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.Type,
		&item.Scope,
		&item.TeamID,
		&item.TenantID,
		&item.FolderID,
		&metadata,
		&item.Tags,
		&item.IsFavorite,
		&item.PwnedCount,
		&expiresAt,
		&item.CurrentVersion,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SecretSummary{}, &RequestError{Status: 404, Message: "Secret not found"}
		}
		return SecretSummary{}, fmt.Errorf("load secret summary: %w", err)
	}

	if len(metadata) > 0 && string(metadata) != "null" {
		if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
			return SecretSummary{}, fmt.Errorf("decode secret metadata: %w", err)
		}
	}
	if expiresAt.Valid {
		value := expiresAt.Time.UTC()
		item.ExpiresAt = &value
	}

	return item, nil
}
