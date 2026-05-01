package credentialresolver

import (
	"context"
	"encoding/json"
	"fmt"
)

func (r Resolver) EncryptPayloadForScope(ctx context.Context, userID, scope string, teamID, tenantID *string, payload json.RawMessage) (string, string, string, error) {
	if !json.Valid(payload) {
		return "", "", "", fmt.Errorf("invalid secret payload json")
	}

	key, err := r.resolveSecretKey(ctx, userID, secretRecord{
		Scope:    scope,
		TeamID:   teamID,
		TenantID: tenantID,
	})
	if err != nil {
		return "", "", "", err
	}
	defer zeroBytes(key)

	field, err := encryptValue(key, string(payload))
	if err != nil {
		return "", "", "", fmt.Errorf("encrypt secret payload: %w", err)
	}
	return field.Ciphertext, field.IV, field.Tag, nil
}

func (r Resolver) EncryptWithKey(key []byte, plaintext string) (string, string, string, error) {
	field, err := encryptValue(key, plaintext)
	if err != nil {
		return "", "", "", fmt.Errorf("encrypt secret payload: %w", err)
	}
	return field.Ciphertext, field.IV, field.Tag, nil
}

func (r Resolver) LoadUserMasterKey(ctx context.Context, userID string) ([]byte, error) {
	key, _, err := r.getUserMasterKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (r Resolver) LoadSecretSummary(ctx context.Context, secretID string) (SecretSummary, error) {
	return r.loadSecretSummary(ctx, secretID)
}

func (r Resolver) LoadSecretPayload(ctx context.Context, userID, secretID, tenantID string) (json.RawMessage, error) {
	record, accessType, err := r.requireViewAccess(ctx, userID, secretID, tenantID)
	if err != nil {
		return nil, err
	}

	decryptedJSON, err := r.decryptSecretPayload(ctx, userID, record, accessType)
	if err != nil {
		return nil, err
	}
	if !json.Valid([]byte(decryptedJSON)) {
		return nil, fmt.Errorf("decoded secret payload is not valid json")
	}
	return json.RawMessage(decryptedJSON), nil
}
