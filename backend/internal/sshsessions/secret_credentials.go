package sshsessions

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/credentialresolver"
)

func (s Service) resolveCredentialsFromSecret(ctx context.Context, userID, tenantID string, access connectionAccess) (resolvedCredentials, error) {
	if access.Connection.CredentialSecretID == nil || *access.Connection.CredentialSecretID == "" {
		return resolvedCredentials{}, &requestError{status: http.StatusNotFound, message: "Connection not found or credentials unavailable"}
	}

	resolver := credentialresolver.Resolver{
		DB:        s.DB,
		Redis:     s.Redis,
		ServerKey: s.ServerEncryptionKey,
	}
	secretCreds, err := resolver.Resolve(ctx, userID, *access.Connection.CredentialSecretID, access.Connection.Type, tenantID)
	if err != nil {
		return resolvedCredentials{}, mapCredentialResolverError(err)
	}

	result := resolvedCredentials{
		Username:         secretCreds.Username,
		Password:         secretCreds.Password,
		Domain:           secretCreds.Domain,
		PrivateKey:       secretCreds.PrivateKey,
		Passphrase:       secretCreds.Passphrase,
		CredentialSource: "saved",
	}

	if result.Username == "" {
		if username, err := s.loadFallbackUsername(ctx, userID, access); err != nil {
			return resolvedCredentials{}, err
		} else if username != "" {
			result.Username = username
		}
	}

	return result, nil
}

func mapCredentialResolverError(err error) error {
	var reqErr *credentialresolver.RequestError
	if errors.As(err, &reqErr) {
		return &requestError{status: reqErr.Status, message: reqErr.Message}
	}
	return err
}

func (s Service) loadFallbackUsername(ctx context.Context, userID string, access connectionAccess) (string, error) {
	if access.Connection.EncryptedUsername == nil || access.Connection.UsernameIV == nil || access.Connection.UsernameTag == nil {
		return "", nil
	}

	key, err := s.loadFallbackKey(ctx, userID, access)
	if err != nil {
		return "", err
	}
	if len(key) == 0 {
		return "", nil
	}
	defer zeroBytes(key)

	username, err := decryptEncryptedField(key, encryptedField{
		Ciphertext: *access.Connection.EncryptedUsername,
		IV:         *access.Connection.UsernameIV,
		Tag:        *access.Connection.UsernameTag,
	})
	if err != nil {
		return "", fmt.Errorf("decrypt fallback username: %w", err)
	}
	return username, nil
}

func (s Service) loadFallbackKey(ctx context.Context, userID string, access connectionAccess) ([]byte, error) {
	if access.AccessType == "team" && access.Connection.TeamID != nil && *access.Connection.TeamID != "" {
		key, err := s.getTeamVaultKey(ctx, *access.Connection.TeamID, userID)
		if err != nil {
			return nil, err
		}
		if len(key) == 0 {
			return nil, &requestError{status: 403, message: "Vault is locked. Please unlock it first."}
		}
		return key, nil
	}

	key, _, err := s.getUserMasterKey(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(key) == 0 {
		return nil, &requestError{status: 403, message: "Vault is locked. Please unlock it first."}
	}
	return key, nil
}
