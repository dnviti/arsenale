package oauthapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (s Service) storeSAMLRequest(ctx context.Context, requestID, linkUserID string) (string, error) {
	state, err := randomCode()
	if err != nil {
		return "", err
	}
	entry := samlRequestEntry{
		RequestID:  strings.TrimSpace(requestID),
		LinkUserID: strings.TrimSpace(linkUserID),
		ExpiresAt:  time.Now().Add(samlRequestTTL).UnixMilli(),
	}

	if s.Redis != nil {
		payload, err := json.Marshal(entry)
		if err != nil {
			return "", fmt.Errorf("marshal saml request entry: %w", err)
		}
		if err := s.Redis.Set(ctx, "saml:request:"+state, payload, samlRequestTTL).Err(); err != nil {
			return "", fmt.Errorf("store saml request: %w", err)
		}
		return state, nil
	}

	samlRequestMu.Lock()
	defer samlRequestMu.Unlock()
	cleanupExpiredSAMLRequestsLocked(time.Now().UnixMilli())
	samlRequestStore[state] = entry
	return state, nil
}

func (s Service) consumeSAMLRequest(ctx context.Context, state string) (samlRequestEntry, error) {
	state = strings.TrimSpace(state)
	if state == "" {
		return samlRequestEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired SAML request"}
	}

	if s.Redis != nil {
		payload, err := s.Redis.GetDel(ctx, "saml:request:"+state).Bytes()
		if err == nil {
			var entry samlRequestEntry
			if err := json.Unmarshal(payload, &entry); err != nil {
				return samlRequestEntry{}, fmt.Errorf("decode saml request entry: %w", err)
			}
			if entry.ExpiresAt <= time.Now().UnixMilli() || strings.TrimSpace(entry.RequestID) == "" {
				return samlRequestEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired SAML request"}
			}
			return entry, nil
		}
		return samlRequestEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired SAML request"}
	}

	samlRequestMu.Lock()
	defer samlRequestMu.Unlock()
	cleanupExpiredSAMLRequestsLocked(time.Now().UnixMilli())
	entry, ok := samlRequestStore[state]
	if !ok {
		return samlRequestEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired SAML request"}
	}
	delete(samlRequestStore, state)
	if entry.ExpiresAt <= time.Now().UnixMilli() || strings.TrimSpace(entry.RequestID) == "" {
		return samlRequestEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired SAML request"}
	}
	return entry, nil
}

func cleanupExpiredSAMLRequestsLocked(now int64) {
	for state, entry := range samlRequestStore {
		if entry.ExpiresAt <= now {
			delete(samlRequestStore, state)
		}
	}
}
