package importexportapi

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/dnviti/arsenale/backend/internal/rediscompat"
	"github.com/redis/go-redis/v9"
)

func nullableEncryptedField(ciphertext, iv, tag sql.NullString) *encryptedField {
	if !ciphertext.Valid || !iv.Valid || !tag.Valid {
		return nil
	}
	return &encryptedField{Ciphertext: ciphertext.String, IV: iv.String, Tag: tag.String}
}

func decryptNullableField(key []byte, field *encryptedField) *string {
	if field == nil {
		return nil
	}
	value, err := decryptEncryptedField(key, *field)
	if err != nil {
		return nil
	}
	return &value
}

func decryptEncryptedField(key []byte, field encryptedField) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, 16)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	nonce, err := hex.DecodeString(field.IV)
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}
	ciphertext, err := hex.DecodeString(field.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	tag, err := hex.DecodeString(field.Tag)
	if err != nil {
		return "", fmt.Errorf("decode tag: %w", err)
	}
	plaintext, err := gcm.Open(nil, nonce, append(ciphertext, tag...), nil)
	if err != nil {
		return "", fmt.Errorf("decrypt value: %w", err)
	}
	return string(plaintext), nil
}

func (s Service) getVaultKey(ctx context.Context, userID string) ([]byte, error) {
	if s.Redis == nil || len(s.ServerEncryptionKey) == 0 {
		return nil, nil
	}
	key := "vault:user:" + userID
	payload, err := s.Redis.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("load vault session: %w", err)
	}

	var field encryptedField
	raw, err := rediscompat.DecodeJSONPayload(payload, &field)
	if err != nil {
		return nil, fmt.Errorf("decode vault session payload: %w", err)
	}

	hexKey, err := decryptEncryptedField(s.ServerEncryptionKey, field)
	if err != nil {
		return nil, fmt.Errorf("decrypt vault session: %w", err)
	}
	masterKey, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode vault master key: %w", err)
	}
	if ttl, ttlErr := s.Redis.PTTL(ctx, key).Result(); ttlErr == nil && ttl > 0 {
		_ = s.Redis.Set(ctx, key, raw, ttl).Err()
	}
	return masterKey, nil
}

func zeroBytes(value []byte) {
	for idx := range value {
		value[idx] = 0
	}
}
