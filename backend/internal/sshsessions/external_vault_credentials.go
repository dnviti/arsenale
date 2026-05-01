package sshsessions

import (
	"context"
	"errors"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/externalvaultapi"
)

func (s Service) resolveCredentialsFromExternalVault(ctx context.Context, tenantID string, access connectionAccess) (resolvedCredentials, error) {
	if access.Connection.ExternalVaultProviderID == nil || strings.TrimSpace(*access.Connection.ExternalVaultProviderID) == "" ||
		access.Connection.ExternalVaultPath == nil || strings.TrimSpace(*access.Connection.ExternalVaultPath) == "" {
		return resolvedCredentials{}, &requestError{status: 404, message: "Connection not found or credentials unavailable"}
	}
	if strings.TrimSpace(tenantID) == "" {
		return resolvedCredentials{}, &requestError{status: 400, message: "Tenant context is required for external vault credential resolution"}
	}

	resolver := externalvaultapi.Service{
		DB:                  s.DB,
		ServerEncryptionKey: s.ServerEncryptionKey,
	}
	creds, err := resolver.ResolveCredentials(ctx, tenantID, *access.Connection.ExternalVaultProviderID, *access.Connection.ExternalVaultPath)
	if err != nil {
		var resolveErr *externalvaultapi.ResolveError
		if errors.As(err, &resolveErr) {
			return resolvedCredentials{}, &requestError{status: resolveErr.Status, message: resolveErr.Message}
		}
		return resolvedCredentials{}, err
	}

	return resolvedCredentials{
		Username:         creds.Username,
		Password:         creds.Password,
		Domain:           creds.Domain,
		PrivateKey:       creds.PrivateKey,
		Passphrase:       creds.Passphrase,
		CredentialSource: "external-vault",
	}, nil
}
