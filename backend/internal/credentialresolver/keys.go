package credentialresolver

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dnviti/arsenale/backend/internal/rediscompat"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func (r Resolver) resolveSecretKey(ctx context.Context, userID string, record secretRecord) ([]byte, error) {
	switch record.Scope {
	case "PERSONAL":
		key, _, err := r.requireUserMasterKey(ctx, userID)
		return key, err
	case "TEAM":
		if record.TeamID == nil || *record.TeamID == "" {
			return nil, fmt.Errorf("team secret missing team id")
		}
		return r.getTeamVaultKey(ctx, *record.TeamID, userID)
	case "TENANT":
		if record.TenantID == nil || *record.TenantID == "" {
			return nil, fmt.Errorf("tenant secret missing tenant id")
		}
		return r.getTenantVaultKey(ctx, *record.TenantID, userID)
	default:
		return nil, fmt.Errorf("unsupported secret scope: %s", record.Scope)
	}
}

func (r Resolver) requireUserMasterKey(ctx context.Context, userID string) ([]byte, time.Duration, error) {
	key, ttl, err := r.getUserMasterKey(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	if len(key) == 0 {
		return nil, 0, &RequestError{Status: 403, Message: "Vault is locked. Please unlock it first."}
	}
	return key, ttl, nil
}

func (r Resolver) getUserMasterKey(ctx context.Context, userID string) ([]byte, time.Duration, error) {
	if r.Redis == nil || len(r.ServerKey) == 0 {
		return nil, 0, nil
	}

	userKey := "vault:user:" + userID
	recoveryKey := "vault:recovery:" + userID
	for _, key := range []string{userKey, recoveryKey} {
		payload, err := r.Redis.Get(ctx, key).Bytes()
		switch {
		case err == nil:
			var field encryptedField
			normalized, err := rediscompat.DecodeJSONPayload(payload, &field)
			if err != nil {
				return nil, 0, fmt.Errorf("decode vault session payload: %w", err)
			}

			hexKey, err := decryptEncryptedField(r.ServerKey, field)
			if err != nil {
				return nil, 0, fmt.Errorf("decrypt vault session: %w", err)
			}
			masterKey, err := hex.DecodeString(hexKey)
			if err != nil {
				return nil, 0, fmt.Errorf("decode vault master key: %w", err)
			}

			ttl := r.defaultVaultTTL()
			if pttl, ttlErr := r.Redis.PTTL(ctx, key).Result(); ttlErr == nil && pttl > 0 {
				ttl = pttl
			}
			if ttl > 0 {
				_ = r.Redis.Set(ctx, userKey, normalized, ttl).Err()
			}
			return masterKey, ttl, nil
		case errors.Is(err, redis.Nil):
			continue
		default:
			return nil, 0, fmt.Errorf("load vault session: %w", err)
		}
	}

	return nil, 0, nil
}

func (r Resolver) getTeamVaultKey(ctx context.Context, teamID, userID string) ([]byte, error) {
	if cached, err := r.getCachedTeamKey(ctx, teamID, userID); err == nil && len(cached) > 0 {
		return cached, nil
	} else if err != nil {
		return nil, err
	}

	userKey, ttl, err := r.requireUserMasterKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(userKey)

	var (
		ciphertext sql.NullString
		iv         sql.NullString
		tag        sql.NullString
	)
	if err := r.DB.QueryRow(
		ctx,
		`SELECT "encryptedTeamVaultKey", "teamVaultKeyIV", "teamVaultKeyTag"
		   FROM "TeamMember"
		  WHERE "teamId" = $1
		    AND "userId" = $2`,
		teamID,
		userID,
	).Scan(&ciphertext, &iv, &tag); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &RequestError{Status: 404, Message: "Credential secret not found or inaccessible"}
		}
		return nil, fmt.Errorf("load team vault key: %w", err)
	}
	if !ciphertext.Valid || !iv.Valid || !tag.Valid || ciphertext.String == "" || iv.String == "" || tag.String == "" {
		return nil, fmt.Errorf("team vault key is unavailable")
	}

	hexKey, err := decryptEncryptedField(userKey, encryptedField{
		Ciphertext: ciphertext.String,
		IV:         iv.String,
		Tag:        tag.String,
	})
	if err != nil {
		return nil, fmt.Errorf("decrypt team vault key: %w", err)
	}
	teamKey, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode team vault key: %w", err)
	}
	if err := r.storeCachedKey(ctx, "vault:team:"+teamID+":"+userID, teamKey, ttl); err != nil {
		zeroBytes(teamKey)
		return nil, err
	}
	return teamKey, nil
}

func (r Resolver) getTenantVaultKey(ctx context.Context, tenantID, userID string) ([]byte, error) {
	if cached, err := r.getCachedTenantKey(ctx, tenantID, userID); err == nil && len(cached) > 0 {
		return cached, nil
	} else if err != nil {
		return nil, err
	}

	userKey, ttl, err := r.requireUserMasterKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(userKey)

	var field encryptedField
	if err := r.DB.QueryRow(
		ctx,
		`SELECT "encryptedTenantVaultKey", "tenantVaultKeyIV", "tenantVaultKeyTag"
		   FROM "TenantVaultMember"
		  WHERE "tenantId" = $1
		    AND "userId" = $2`,
		tenantID,
		userID,
	).Scan(&field.Ciphertext, &field.IV, &field.Tag); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &RequestError{Status: 404, Message: "Tenant vault key not found. An admin must distribute the key to you."}
		}
		return nil, fmt.Errorf("load tenant vault key: %w", err)
	}

	hexKey, err := decryptEncryptedField(userKey, field)
	if err != nil {
		return nil, fmt.Errorf("decrypt tenant vault key: %w", err)
	}
	tenantKey, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode tenant vault key: %w", err)
	}
	if err := r.storeCachedKey(ctx, "vault:tenant:"+tenantID+":"+userID, tenantKey, ttl); err != nil {
		zeroBytes(tenantKey)
		return nil, err
	}
	return tenantKey, nil
}

func (r Resolver) getCachedTeamKey(ctx context.Context, teamID, userID string) ([]byte, error) {
	return r.loadCachedKey(ctx, "vault:team:"+teamID+":"+userID)
}

func (r Resolver) getCachedTenantKey(ctx context.Context, tenantID, userID string) ([]byte, error) {
	return r.loadCachedKey(ctx, "vault:tenant:"+tenantID+":"+userID)
}

func (r Resolver) loadCachedKey(ctx context.Context, redisKey string) ([]byte, error) {
	if r.Redis == nil || len(r.ServerKey) == 0 {
		return nil, nil
	}

	payload, err := r.Redis.Get(ctx, redisKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("load cached vault key: %w", err)
	}

	var field encryptedField
	normalized, err := rediscompat.DecodeJSONPayload(payload, &field)
	if err != nil {
		return nil, fmt.Errorf("decode cached vault key: %w", err)
	}
	hexKey, err := decryptEncryptedField(r.ServerKey, field)
	if err != nil {
		return nil, fmt.Errorf("decrypt cached vault key: %w", err)
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode cached vault key bytes: %w", err)
	}
	if pttl, ttlErr := r.Redis.PTTL(ctx, redisKey).Result(); ttlErr == nil && pttl > 0 {
		_ = r.Redis.Set(ctx, redisKey, normalized, pttl).Err()
	}
	return key, nil
}

func (r Resolver) storeCachedKey(ctx context.Context, redisKey string, key []byte, ttl time.Duration) error {
	if r.Redis == nil || len(r.ServerKey) == 0 {
		return nil
	}

	field, err := encryptCachedKey(r.ServerKey, hex.EncodeToString(key))
	if err != nil {
		return fmt.Errorf("encrypt cached vault key: %w", err)
	}
	raw, err := json.Marshal(field)
	if err != nil {
		return fmt.Errorf("marshal cached vault key: %w", err)
	}
	if ttl <= 0 {
		ttl = r.defaultVaultTTL()
	}
	if err := r.Redis.Set(ctx, redisKey, raw, ttl).Err(); err != nil {
		return fmt.Errorf("store cached vault key: %w", err)
	}
	return nil
}

func (r Resolver) defaultVaultTTL() time.Duration {
	if r.VaultTTL > 0 {
		return r.VaultTTL
	}
	return 30 * time.Minute
}

func encryptCachedKey(key []byte, plaintext string) (encryptedField, error) {
	if len(key) != 32 {
		return encryptedField{}, fmt.Errorf("invalid key length")
	}

	encrypted, err := encryptValue(key, plaintext)
	if err != nil {
		return encryptedField{}, err
	}
	return encrypted, nil
}

func encryptValue(key []byte, plaintext string) (encryptedField, error) {
	if len(key) != 32 {
		return encryptedField{}, fmt.Errorf("invalid key length")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return encryptedField{}, fmt.Errorf("create cipher: %w", err)
	}
	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return encryptedField{}, fmt.Errorf("generate iv: %w", err)
	}
	aead, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return encryptedField{}, fmt.Errorf("create gcm: %w", err)
	}
	sealed := aead.Seal(nil, iv, []byte(plaintext), nil)
	tagOffset := len(sealed) - aead.Overhead()
	return encryptedField{
		Ciphertext: hex.EncodeToString(sealed[:tagOffset]),
		IV:         hex.EncodeToString(iv),
		Tag:        hex.EncodeToString(sealed[tagOffset:]),
	}, nil
}
