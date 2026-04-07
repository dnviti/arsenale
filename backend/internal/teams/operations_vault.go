package teams

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dnviti/arsenale/backend/internal/rediscompat"
	"github.com/redis/go-redis/v9"
)

func (s Service) getVaultMasterKey(ctx context.Context, userID string) ([]byte, time.Duration, error) {
	if s.Redis == nil || len(s.ServerEncryptionKey) == 0 {
		return nil, 0, nil
	}

	userKey := "vault:user:" + userID
	recoveryKey := "vault:recovery:" + userID
	keys := []string{userKey, recoveryKey}

	for _, key := range keys {
		payload, err := s.Redis.Get(ctx, key).Bytes()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			return nil, 0, fmt.Errorf("load vault session: %w", err)
		}

		var field encryptedField
		normalized, err := rediscompat.DecodeJSONPayload(payload, &field)
		if err != nil {
			return nil, 0, fmt.Errorf("decode vault session payload format: %w", err)
		}

		hexKey, err := decryptEncryptedField(s.ServerEncryptionKey, field)
		if err != nil {
			return nil, 0, fmt.Errorf("decrypt vault session: %w", err)
		}
		masterKey, err := hex.DecodeString(hexKey)
		if err != nil {
			return nil, 0, fmt.Errorf("decode vault master key: %w", err)
		}

		var ttl time.Duration
		if pttl, ttlErr := s.Redis.PTTL(ctx, key).Result(); ttlErr == nil && pttl > 0 {
			ttl = pttl
		}

		if key == userKey {
			if ttl > 0 {
				_ = s.Redis.Set(ctx, userKey, normalized, ttl).Err()
			}
			return masterKey, ttl, nil
		}

		if ttl <= 0 {
			ttl = s.VaultTTL
		}
		if ttl <= 0 {
			ttl = 30 * time.Minute
		}

		_ = s.Redis.Set(ctx, userKey, normalized, ttl).Err()
		return masterKey, ttl, nil
	}

	return nil, 0, nil
}

func (s Service) getCachedTeamKey(ctx context.Context, teamID, userID string) ([]byte, error) {
	if s.Redis == nil || len(s.ServerEncryptionKey) == 0 {
		return nil, nil
	}

	key := fmt.Sprintf("vault:team:%s:%s", teamID, userID)
	payload, err := s.Redis.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("load team vault session: %w", err)
	}

	var field encryptedField
	normalized, err := rediscompat.DecodeJSONPayload(payload, &field)
	if err != nil {
		return nil, fmt.Errorf("decode team vault session payload: %w", err)
	}

	hexKey, err := decryptEncryptedField(s.ServerEncryptionKey, field)
	if err != nil {
		return nil, fmt.Errorf("decrypt team vault session: %w", err)
	}
	teamKey, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("decode team vault key: %w", err)
	}

	if pttl, ttlErr := s.Redis.PTTL(ctx, key).Result(); ttlErr == nil && pttl > 0 {
		_ = s.Redis.Set(ctx, key, normalized, pttl).Err()
	}
	return teamKey, nil
}

func (s Service) storeTeamVaultSession(ctx context.Context, teamID, userID string, teamKey []byte, ttl time.Duration) error {
	if s.Redis == nil || len(s.ServerEncryptionKey) == 0 {
		return nil
	}
	if ttl <= 0 {
		ttl = s.VaultTTL
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}

	field, err := encryptHexPayload(s.ServerEncryptionKey, hex.EncodeToString(teamKey))
	if err != nil {
		return fmt.Errorf("encrypt team vault session: %w", err)
	}
	raw, err := json.Marshal(field)
	if err != nil {
		return fmt.Errorf("marshal team vault session: %w", err)
	}
	if err := s.Redis.Set(ctx, fmt.Sprintf("vault:team:%s:%s", teamID, userID), raw, ttl).Err(); err != nil {
		return fmt.Errorf("store team vault session: %w", err)
	}
	return nil
}
