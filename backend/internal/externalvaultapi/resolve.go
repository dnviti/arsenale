package externalvaultapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type ResolveError struct {
	Status  int
	Message string
}

func (e *ResolveError) Error() string {
	return e.Message
}

type ResolvedCredentials struct {
	Username   string
	Password   string
	Domain     string
	PrivateKey string
	Passphrase string
}

func (s Service) ResolveCredentials(ctx context.Context, tenantID, providerID, secretPath string) (ResolvedCredentials, error) {
	record, err := s.getProvider(ctx, strings.TrimSpace(tenantID), strings.TrimSpace(providerID))
	if err != nil {
		var reqErr *requestError
		if errors.As(err, &reqErr) {
			return ResolvedCredentials{}, &ResolveError{Status: reqErr.status, Message: reqErr.message}
		}
		return ResolvedCredentials{}, err
	}
	if !record.Enabled {
		return ResolvedCredentials{}, &ResolveError{Status: 400, Message: "External vault provider is disabled"}
	}

	data, err := s.readSecret(ctx, record, strings.TrimSpace(secretPath))
	if err != nil {
		return ResolvedCredentials{}, err
	}
	return mapSecretToCredentials(data, strings.TrimSpace(secretPath))
}

func mapSecretToCredentials(data map[string]string, secretPath string) (ResolvedCredentials, error) {
	username := firstSecretValue(data, "username", "user")
	password := firstSecretValue(data, "password", "pass")
	if username == "" && password == "" {
		return ResolvedCredentials{}, &ResolveError{
			Status:  502,
			Message: fmt.Sprintf("Secret at %q does not contain username/password fields", secretPath),
		}
	}

	return ResolvedCredentials{
		Username:   username,
		Password:   password,
		Domain:     firstSecretValue(data, "domain"),
		PrivateKey: firstSecretValue(data, "private_key", "privateKey"),
		Passphrase: firstSecretValue(data, "passphrase"),
	}, nil
}

func firstSecretValue(data map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(data[key]); value != "" {
			return value
		}
	}
	return ""
}
