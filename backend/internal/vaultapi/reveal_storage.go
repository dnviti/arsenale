package vaultapi

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type revealCredentialRecord struct {
	UserID             string
	AccessType         string
	ConnectionType     string
	TeamID             *string
	CredentialSecretID *string
	Password           *encryptedField
}

type notFoundError string

func (e notFoundError) Error() string {
	return string(e)
}

func decodeHexKey(value string) ([]byte, error) {
	key, err := hex.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("decode vault key: %w", err)
	}
	return key, nil
}

func (s Service) loadVaultSession(ctx context.Context, userID string) ([]byte, error) {
	if s.Redis == nil || len(s.ServerKey) != 32 {
		return nil, nil
	}
	raw, err := s.Redis.Get(ctx, "vault:user:"+userID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("load vault session: %w", err)
	}

	var field encryptedField
	if err := json.Unmarshal(raw, &field); err != nil {
		return nil, fmt.Errorf("decode vault session: %w", err)
	}
	hexKey, err := decryptEncryptedField(s.ServerKey, field)
	if err != nil {
		return nil, fmt.Errorf("decrypt vault session: %w", err)
	}
	masterKey, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode vault session key: %w", err)
	}
	if ttl, ttlErr := s.Redis.PTTL(ctx, "vault:user:"+userID).Result(); ttlErr == nil && ttl > 0 {
		_ = s.Redis.Set(ctx, "vault:user:"+userID, raw, ttl).Err()
	}
	return masterKey, nil
}

func (s Service) loadRevealCredential(ctx context.Context, connectionID, userID, tenantID string) (revealCredentialRecord, error) {
	if record, err := s.loadOwnedRevealCredential(ctx, connectionID, userID); err == nil {
		return record, nil
	} else if !errors.Is(err, notFoundError("owned reveal credential")) {
		return revealCredentialRecord{}, err
	}

	if record, err := s.loadTeamRevealCredential(ctx, connectionID, userID, tenantID); err == nil {
		return record, nil
	} else if !errors.Is(err, notFoundError("team reveal credential")) {
		return revealCredentialRecord{}, err
	}

	if record, err := s.loadSharedRevealCredential(ctx, connectionID, userID, tenantID); err == nil {
		return record, nil
	} else if !errors.Is(err, notFoundError("shared reveal credential")) {
		return revealCredentialRecord{}, err
	}

	return revealCredentialRecord{}, &requestError{status: 403, message: "Connection not found or insufficient permissions"}
}

func (s Service) loadOwnedRevealCredential(ctx context.Context, connectionID, userID string) (revealCredentialRecord, error) {
	var (
		connectionType     string
		credentialSecretID sql.NullString
		encryptedPassword  sql.NullString
		passwordIV         sql.NullString
		passwordTag        sql.NullString
	)

	err := s.DB.QueryRow(ctx, `
SELECT
	c.type::text,
	c."credentialSecretId",
	c."encryptedPassword",
	c."passwordIV",
	c."passwordTag"
FROM "Connection" c
WHERE c.id = $1
  AND c."userId" = $2
  AND c."teamId" IS NULL
`, connectionID, userID).Scan(&connectionType, &credentialSecretID, &encryptedPassword, &passwordIV, &passwordTag)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return revealCredentialRecord{}, notFoundError("owned reveal credential")
		}
		return revealCredentialRecord{}, fmt.Errorf("load owned reveal credential: %w", err)
	}
	return buildRevealCredentialRecord(userID, "owner", connectionType, nil, credentialSecretID, encryptedPassword, passwordIV, passwordTag), nil
}

func (s Service) loadTeamRevealCredential(ctx context.Context, connectionID, userID, tenantID string) (revealCredentialRecord, error) {
	var credentialSecretID sql.NullString
	var connectionType string
	var teamID sql.NullString
	var encryptedPassword sql.NullString
	var passwordIV sql.NullString
	var passwordTag sql.NullString

	err := s.DB.QueryRow(ctx, `
SELECT
	c.type::text,
	c."teamId",
	c."credentialSecretId",
	c."encryptedPassword",
	c."passwordIV",
	c."passwordTag"
FROM "Connection" c
JOIN "TeamMember" tm ON tm."teamId" = c."teamId" AND tm."userId" = $2
JOIN "Team" t ON t.id = c."teamId"
WHERE c.id = $1
  AND c."teamId" IS NOT NULL
  AND ($3 = '' OR t."tenantId" = $3)
`, connectionID, userID, tenantID).Scan(&connectionType, &teamID, &credentialSecretID, &encryptedPassword, &passwordIV, &passwordTag)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return revealCredentialRecord{}, notFoundError("team reveal credential")
		}
		return revealCredentialRecord{}, fmt.Errorf("load team reveal credential: %w", err)
	}
	return buildRevealCredentialRecord(userID, "team", connectionType, nullStringPtr(teamID), credentialSecretID, encryptedPassword, passwordIV, passwordTag), nil
}

func (s Service) loadSharedRevealCredential(ctx context.Context, connectionID, userID, tenantID string) (revealCredentialRecord, error) {
	var credentialSecretID sql.NullString
	var connectionType string
	var teamID sql.NullString
	var encryptedPassword sql.NullString
	var passwordIV sql.NullString
	var passwordTag sql.NullString

	err := s.DB.QueryRow(ctx, `
SELECT
	c.type::text,
	c."teamId",
	c."credentialSecretId",
	sc."encryptedPassword",
	sc."passwordIV",
	sc."passwordTag"
FROM "SharedConnection" sc
JOIN "Connection" c ON c.id = sc."connectionId"
LEFT JOIN "Team" t ON t.id = c."teamId"
WHERE sc."connectionId" = $1
  AND sc."sharedWithUserId" = $2
  AND sc.permission::text = 'FULL_ACCESS'
  AND ($3 = '' OR c."teamId" IS NULL OR t."tenantId" = $3)
`, connectionID, userID, tenantID).Scan(&connectionType, &teamID, &credentialSecretID, &encryptedPassword, &passwordIV, &passwordTag)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return revealCredentialRecord{}, notFoundError("shared reveal credential")
		}
		return revealCredentialRecord{}, fmt.Errorf("load shared reveal credential: %w", err)
	}
	return buildRevealCredentialRecord(userID, "shared", connectionType, nullStringPtr(teamID), credentialSecretID, encryptedPassword, passwordIV, passwordTag), nil
}

func (s Service) loadTeamRevealKey(ctx context.Context, teamID, userID string) (encryptedField, error) {
	var (
		ciphertext sql.NullString
		iv         sql.NullString
		tag        sql.NullString
	)
	if err := s.DB.QueryRow(
		ctx,
		`SELECT "encryptedTeamVaultKey", "teamVaultKeyIV", "teamVaultKeyTag"
		   FROM "TeamMember"
		  WHERE "teamId" = $1
		    AND "userId" = $2`,
		teamID,
		userID,
	).Scan(&ciphertext, &iv, &tag); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return encryptedField{}, &requestError{status: 403, message: "Connection not found or insufficient permissions"}
		}
		return encryptedField{}, fmt.Errorf("load team reveal key: %w", err)
	}
	if !ciphertext.Valid || !iv.Valid || !tag.Valid || ciphertext.String == "" || iv.String == "" || tag.String == "" {
		return encryptedField{}, fmt.Errorf("team vault key is unavailable")
	}
	return encryptedField{
		Ciphertext: ciphertext.String,
		IV:         iv.String,
		Tag:        tag.String,
	}, nil
}

func buildRevealCredentialRecord(userID, accessType, connectionType string, teamID *string, credentialSecretID, encryptedPassword, passwordIV, passwordTag sql.NullString) revealCredentialRecord {
	record := revealCredentialRecord{
		UserID:         userID,
		AccessType:     accessType,
		ConnectionType: strings.TrimSpace(connectionType),
		TeamID:         teamID,
	}
	if credentialSecretID.Valid && credentialSecretID.String != "" {
		record.CredentialSecretID = &credentialSecretID.String
	}
	if encryptedPassword.Valid && passwordIV.Valid && passwordTag.Valid {
		record.Password = &encryptedField{
			Ciphertext: encryptedPassword.String,
			IV:         passwordIV.String,
			Tag:        passwordTag.String,
		}
	}
	return record
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	return &value.String
}
