package connections

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/rediscompat"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type sourceCredentials struct {
	Username encryptedField
	Password encryptedField
	Domain   *encryptedField
}

func (s Service) loadSharingSourceKey(ctx context.Context, userID string, connection connectionResponse) ([]byte, error) {
	if connection.TeamID == nil {
		key, err := s.getVaultKey(ctx, userID)
		if err != nil {
			return nil, err
		}
		if len(key) == 0 {
			return nil, &requestError{status: http.StatusForbidden, message: "Your vault is locked. Please unlock it first."}
		}
		return key, nil
	}
	return s.getTeamVaultKey(ctx, *connection.TeamID, userID)
}

func (s Service) getTeamVaultKey(ctx context.Context, teamID, userID string) ([]byte, error) {
	if s.Redis != nil && len(s.ServerEncryptionKey) > 0 {
		cacheKey := fmt.Sprintf("vault:team:%s:%s", teamID, userID)
		payload, err := s.Redis.Get(ctx, cacheKey).Bytes()
		if err == nil {
			var field encryptedField
			if normalized, decodeErr := rediscompat.DecodeJSONPayload(payload, &field); decodeErr == nil {
				hexKey, err := decryptEncryptedField(s.ServerEncryptionKey, field)
				if err == nil {
					teamKey, err := hex.DecodeString(hexKey)
					if err == nil {
						if pttl, ttlErr := s.Redis.PTTL(ctx, cacheKey).Result(); ttlErr == nil && pttl > 0 {
							_ = s.Redis.Set(ctx, cacheKey, normalized, pttl).Err()
						}
						return teamKey, nil
					}
				}
			}
		} else if !errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("load team vault session: %w", err)
		}
	}

	actingMasterKey, err := s.getVaultKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(actingMasterKey) == 0 {
		return nil, &requestError{status: http.StatusForbidden, message: "Your vault is locked. Please unlock it first."}
	}
	defer zeroBytes(actingMasterKey)

	var field encryptedField
	if err := s.DB.QueryRow(ctx, `
SELECT "encryptedTeamVaultKey", "teamVaultKeyIV", "teamVaultKeyTag"
FROM "TeamMember"
WHERE "teamId" = $1 AND "userId" = $2
`, teamID, userID).Scan(&field.Ciphertext, &field.IV, &field.Tag); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, fmt.Errorf("load acting member team key: %w", err)
	}
	if strings.TrimSpace(field.Ciphertext) == "" || strings.TrimSpace(field.IV) == "" || strings.TrimSpace(field.Tag) == "" {
		return nil, &requestError{status: http.StatusInternalServerError, message: "Unable to access team vault key"}
	}

	hexKey, err := decryptEncryptedField(actingMasterKey, field)
	if err != nil {
		return nil, &requestError{status: http.StatusInternalServerError, message: "Unable to access team vault key"}
	}
	teamKey, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode team key: %w", err)
	}
	return teamKey, nil
}

func (s Service) loadSharableCredentials(ctx context.Context, connectionID string) (sourceCredentials, error) {
	var fields struct {
		UsernameCiphertext sql.NullString
		UsernameIV         sql.NullString
		UsernameTag        sql.NullString
		PasswordCiphertext sql.NullString
		PasswordIV         sql.NullString
		PasswordTag        sql.NullString
		DomainCiphertext   sql.NullString
		DomainIV           sql.NullString
		DomainTag          sql.NullString
	}
	if err := s.DB.QueryRow(ctx, `
SELECT
	"encryptedUsername",
	"usernameIV",
	"usernameTag",
	"encryptedPassword",
	"passwordIV",
	"passwordTag",
	"encryptedDomain",
	"domainIV",
	"domainTag"
FROM "Connection"
WHERE id = $1
`, connectionID).Scan(
		&fields.UsernameCiphertext,
		&fields.UsernameIV,
		&fields.UsernameTag,
		&fields.PasswordCiphertext,
		&fields.PasswordIV,
		&fields.PasswordTag,
		&fields.DomainCiphertext,
		&fields.DomainIV,
		&fields.DomainTag,
	); err != nil {
		return sourceCredentials{}, fmt.Errorf("load connection credentials: %w", err)
	}
	if !fields.UsernameCiphertext.Valid || !fields.UsernameIV.Valid || !fields.UsernameTag.Valid ||
		!fields.PasswordCiphertext.Valid || !fields.PasswordIV.Valid || !fields.PasswordTag.Valid {
		return sourceCredentials{}, &requestError{status: http.StatusBadRequest, message: "Connection has no credentials to share"}
	}

	result := sourceCredentials{
		Username: encryptedField{
			Ciphertext: fields.UsernameCiphertext.String,
			IV:         fields.UsernameIV.String,
			Tag:        fields.UsernameTag.String,
		},
		Password: encryptedField{
			Ciphertext: fields.PasswordCiphertext.String,
			IV:         fields.PasswordIV.String,
			Tag:        fields.PasswordTag.String,
		},
	}
	if fields.DomainCiphertext.Valid && fields.DomainIV.Valid && fields.DomainTag.Valid {
		result.Domain = &encryptedField{
			Ciphertext: fields.DomainCiphertext.String,
			IV:         fields.DomainIV.String,
			Tag:        fields.DomainTag.String,
		}
	}
	return result, nil
}

func reencryptSharedCredentials(sourceKey, targetKey []byte, creds sourceCredentials) (encryptedField, encryptedField, *encryptedField, error) {
	username, err := decryptEncryptedField(sourceKey, creds.Username)
	if err != nil {
		return encryptedField{}, encryptedField{}, nil, &requestError{status: http.StatusBadRequest, message: "Connection has no credentials to share"}
	}
	password, err := decryptEncryptedField(sourceKey, creds.Password)
	if err != nil {
		return encryptedField{}, encryptedField{}, nil, &requestError{status: http.StatusBadRequest, message: "Connection has no credentials to share"}
	}

	encUsername, err := encryptValue(targetKey, username)
	if err != nil {
		return encryptedField{}, encryptedField{}, nil, err
	}
	encPassword, err := encryptValue(targetKey, password)
	if err != nil {
		return encryptedField{}, encryptedField{}, nil, err
	}

	var encDomain *encryptedField
	if creds.Domain != nil {
		domain, err := decryptEncryptedField(sourceKey, *creds.Domain)
		if err == nil && strings.TrimSpace(domain) != "" {
			field, err := encryptValue(targetKey, domain)
			if err != nil {
				return encryptedField{}, encryptedField{}, nil, err
			}
			encDomain = &field
		}
	}
	return encUsername, encPassword, encDomain, nil
}
