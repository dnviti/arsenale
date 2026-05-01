package vaultapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/credentialresolver"
)

func (s Service) RevealPassword(ctx context.Context, userID, tenantID, connectionID, password string) (map[string]any, error) {
	userMasterKey, err := s.resolveRevealMasterKey(ctx, userID, password)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(userMasterKey)

	record, err := s.loadRevealCredential(ctx, connectionID, userID, tenantID)
	if err != nil {
		return nil, err
	}

	if record.CredentialSecretID != nil && *record.CredentialSecretID != "" {
		resolver := credentialresolver.Resolver{
			DB:        s.DB,
			Redis:     s.Redis,
			ServerKey: s.ServerKey,
			VaultTTL:  s.VaultTTL,
		}
		secretCreds, err := resolver.Resolve(ctx, userID, *record.CredentialSecretID, record.ConnectionType, tenantID)
		if err != nil {
			return nil, s.mapRevealCredentialError(err)
		}
		return map[string]any{"password": secretCreds.Password}, nil
	}

	key := userMasterKey
	if record.AccessType == "team" {
		key, err = s.resolveTeamRevealKey(ctx, record, userMasterKey)
		if err != nil {
			return nil, err
		}
		defer zeroBytes(key)
	}

	return s.revealInlineCredential(record, key)
}

func (s Service) resolveRevealMasterKey(ctx context.Context, userID, password string) ([]byte, error) {
	masterKey, err := s.loadVaultSession(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(masterKey) > 0 {
		return masterKey, nil
	}

	creds, err := s.loadVaultCredentials(ctx, userID)
	if err != nil {
		return nil, err
	}
	if creds.VaultSalt == nil || creds.EncryptedVaultKey == nil || creds.VaultKeyIV == nil || creds.VaultKeyTag == nil ||
		*creds.VaultSalt == "" || *creds.EncryptedVaultKey == "" || *creds.VaultKeyIV == "" || *creds.VaultKeyTag == "" {
		return nil, &requestError{status: http.StatusBadRequest, message: "Vault not set up. Please set a vault password first."}
	}

	derived := deriveKeyFromPassword(password, *creds.VaultSalt)
	if len(derived) == 0 {
		return nil, &requestError{status: http.StatusUnauthorized, message: "Invalid password"}
	}
	defer zeroBytes(derived)

	masterKey, err = decryptMasterKey(encryptedField{
		Ciphertext: *creds.EncryptedVaultKey,
		IV:         *creds.VaultKeyIV,
		Tag:        *creds.VaultKeyTag,
	}, derived)
	if err != nil {
		return nil, &requestError{status: http.StatusUnauthorized, message: "Invalid password"}
	}
	if err := s.storeVaultSession(ctx, userID, masterKey); err != nil {
		zeroBytes(masterKey)
		return nil, err
	}
	return masterKey, nil
}

func (s Service) resolveTeamRevealKey(ctx context.Context, record revealCredentialRecord, userMasterKey []byte) ([]byte, error) {
	if record.TeamID == nil || *record.TeamID == "" {
		return nil, &requestError{status: http.StatusForbidden, message: "Connection not found or insufficient permissions"}
	}

	field, err := s.loadTeamRevealKey(ctx, *record.TeamID, record.UserID)
	if err != nil {
		return nil, err
	}
	hexKey, err := decryptEncryptedField(userMasterKey, field)
	if err != nil {
		return nil, &requestError{status: http.StatusForbidden, message: "Connection not found or insufficient permissions"}
	}
	teamKey, err := decodeHexKey(hexKey)
	if err != nil {
		return nil, err
	}
	return teamKey, nil
}

func (s Service) revealInlineCredential(record revealCredentialRecord, key []byte) (map[string]any, error) {
	if record.Password == nil || record.Password.IV == "" || record.Password.Tag == "" || record.Password.Ciphertext == "" {
		return nil, &requestError{status: http.StatusForbidden, message: "Connection not found or insufficient permissions"}
	}
	plaintext, err := decryptEncryptedField(key, *record.Password)
	if err != nil {
		return nil, &requestError{status: http.StatusForbidden, message: "Connection not found or insufficient permissions"}
	}
	return map[string]any{"password": plaintext}, nil
}

func (s Service) mapRevealCredentialError(err error) error {
	var reqErr *credentialresolver.RequestError
	if errors.As(err, &reqErr) {
		return &requestError{status: reqErr.Status, message: reqErr.Message}
	}
	return err
}
