package oauthapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func (s Service) GenerateLinkCode(ctx context.Context, userID string) (string, error) {
	code, err := randomCode()
	if err != nil {
		return "", err
	}
	entry := linkCodeEntry{
		UserID:    userID,
		ExpiresAt: time.Now().Add(linkCodeTTL).UnixMilli(),
	}
	if s.Redis != nil {
		payload, err := json.Marshal(entry)
		if err != nil {
			return "", fmt.Errorf("marshal link code entry: %w", err)
		}
		if err := s.Redis.Set(ctx, "link:code:"+code, payload, linkCodeTTL).Err(); err != nil {
			return "", fmt.Errorf("store link code: %w", err)
		}
		return code, nil
	}

	linkCodeMu.Lock()
	defer linkCodeMu.Unlock()
	cleanupExpiredLinkCodesLocked(time.Now().UnixMilli())
	linkCodeStore[code] = entry
	return code, nil
}

func (s Service) ConsumeLinkCode(ctx context.Context, code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", &requestError{status: http.StatusUnauthorized, message: "Missing authentication"}
	}

	if s.Redis != nil {
		payload, err := s.Redis.GetDel(ctx, "link:code:"+code).Bytes()
		if err == nil {
			var entry linkCodeEntry
			if err := json.Unmarshal(payload, &entry); err != nil {
				return "", fmt.Errorf("decode link code payload: %w", err)
			}
			if entry.ExpiresAt <= time.Now().UnixMilli() {
				return "", &requestError{status: http.StatusUnauthorized, message: "Invalid or expired link code"}
			}
			return entry.UserID, nil
		}
		if !errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("load link code: %w", err)
		}
	}

	linkCodeMu.Lock()
	defer linkCodeMu.Unlock()
	cleanupExpiredLinkCodesLocked(time.Now().UnixMilli())
	entry, ok := linkCodeStore[code]
	if !ok {
		return "", &requestError{status: http.StatusUnauthorized, message: "Invalid or expired link code"}
	}
	delete(linkCodeStore, code)
	if entry.ExpiresAt <= time.Now().UnixMilli() {
		return "", &requestError{status: http.StatusUnauthorized, message: "Invalid or expired link code"}
	}
	return entry.UserID, nil
}

func (s Service) GenerateRelayCode(ctx context.Context, userID string) (string, error) {
	code, err := randomCode()
	if err != nil {
		return "", err
	}
	entry := linkCodeEntry{
		UserID:    userID,
		ExpiresAt: time.Now().Add(relayCodeTTL).UnixMilli(),
	}
	if s.Redis != nil {
		payload, err := json.Marshal(entry)
		if err != nil {
			return "", fmt.Errorf("marshal relay code entry: %w", err)
		}
		if err := s.Redis.Set(ctx, "relay:code:"+code, payload, relayCodeTTL).Err(); err != nil {
			return "", fmt.Errorf("store relay code: %w", err)
		}
		return code, nil
	}

	relayCodeMu.Lock()
	defer relayCodeMu.Unlock()
	cleanupExpiredRelayCodesLocked(time.Now().UnixMilli())
	relayCodeStore[code] = entry
	return code, nil
}

func (s Service) ConsumeRelayCode(ctx context.Context, code string) (string, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return "", nil
	}

	if s.Redis != nil {
		payload, err := s.Redis.GetDel(ctx, "relay:code:"+code).Bytes()
		if err == nil {
			var entry linkCodeEntry
			if err := json.Unmarshal(payload, &entry); err != nil {
				return "", fmt.Errorf("decode relay code payload: %w", err)
			}
			if entry.ExpiresAt <= time.Now().UnixMilli() {
				return "", nil
			}
			return entry.UserID, nil
		}
		if !errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("load relay code: %w", err)
		}
	}

	relayCodeMu.Lock()
	defer relayCodeMu.Unlock()
	cleanupExpiredRelayCodesLocked(time.Now().UnixMilli())
	entry, ok := relayCodeStore[code]
	if !ok {
		return "", nil
	}
	delete(relayCodeStore, code)
	if entry.ExpiresAt <= time.Now().UnixMilli() {
		return "", nil
	}
	return entry.UserID, nil
}

func (s Service) ConsumeAuthCode(ctx context.Context, code string) (map[string]any, error) {
	entry, err := s.consumeAuthCodeEntry(ctx, strings.TrimSpace(code))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"accessToken":     entry.AccessToken,
		"csrfToken":       entry.CSRFToken,
		"needsVaultSetup": entry.NeedsVaultSetup,
		"userId":          entry.UserID,
		"email":           entry.Email,
		"username":        entry.Username,
		"avatarData":      entry.AvatarData,
		"tenantId":        entry.TenantID,
		"tenantRole":      entry.TenantRole,
	}, nil
}

func (s Service) consumeAuthCodeEntry(ctx context.Context, code string) (authCodeEntry, error) {
	if code == "" {
		return authCodeEntry{}, &requestError{status: http.StatusBadRequest, message: "Missing authorization code"}
	}

	if s.Redis != nil {
		payload, err := s.Redis.GetDel(ctx, "auth:code:"+code).Bytes()
		if err == nil {
			var entry authCodeEntry
			if err := json.Unmarshal(payload, &entry); err != nil {
				return authCodeEntry{}, fmt.Errorf("decode auth code payload: %w", err)
			}
			if entry.ExpiresAt <= time.Now().UnixMilli() {
				return authCodeEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired authorization code"}
			}
			return entry, nil
		}
		if !errors.Is(err, redis.Nil) {
			return authCodeEntry{}, fmt.Errorf("load auth code: %w", err)
		}
	}

	authCodeMu.Lock()
	defer authCodeMu.Unlock()
	cleanupExpiredAuthCodesLocked(time.Now().UnixMilli())
	entry, ok := authCodeStore[code]
	if !ok {
		return authCodeEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired authorization code"}
	}
	delete(authCodeStore, code)
	if entry.ExpiresAt <= time.Now().UnixMilli() {
		return authCodeEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired authorization code"}
	}
	return entry, nil
}

func (s Service) storeAuthCodeEntry(ctx context.Context, entry authCodeEntry) (string, error) {
	code, err := randomCode()
	if err != nil {
		return "", err
	}
	entry.ExpiresAt = time.Now().Add(authCodeTTL).UnixMilli()

	if s.Redis != nil {
		payload, err := json.Marshal(entry)
		if err != nil {
			return "", fmt.Errorf("marshal auth code entry: %w", err)
		}
		if err := s.Redis.Set(ctx, "auth:code:"+code, payload, authCodeTTL).Err(); err != nil {
			return "", fmt.Errorf("store auth code: %w", err)
		}
		return code, nil
	}

	authCodeMu.Lock()
	defer authCodeMu.Unlock()
	cleanupExpiredAuthCodesLocked(time.Now().UnixMilli())
	authCodeStore[code] = entry
	return code, nil
}

func cleanupExpiredLinkCodesLocked(now int64) {
	for code, entry := range linkCodeStore {
		if entry.ExpiresAt <= now {
			delete(linkCodeStore, code)
		}
	}
}

func cleanupExpiredRelayCodesLocked(now int64) {
	for code, entry := range relayCodeStore {
		if entry.ExpiresAt <= now {
			delete(relayCodeStore, code)
		}
	}
}

func cleanupExpiredOIDCPKCELocked(now int64) {
	for code, entry := range oidcPKCEStore {
		if entry.ExpiresAt <= now {
			delete(oidcPKCEStore, code)
		}
	}
}

func cleanupExpiredAuthCodesLocked(now int64) {
	for code, entry := range authCodeStore {
		if entry.ExpiresAt <= now {
			delete(authCodeStore, code)
		}
	}
}
