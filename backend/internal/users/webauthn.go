package users

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/webauthnflow"
)

func (s Service) prepareIdentityWebAuthnOptions(ctx context.Context, userID string) (map[string]interface{}, error) {
	descriptors, err := s.loadWebAuthnDescriptors(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(descriptors) == 0 {
		return nil, &requestError{status: 400, message: "WebAuthn is not configured properly."}
	}

	flow := webauthnflow.New(s.Redis)
	options, err := flow.BuildAuthenticationOptions(descriptors)
	if err != nil {
		return nil, err
	}
	return webauthnflow.OptionsMetadata(options)
}

func (s Service) verifyIdentityWebAuthn(ctx context.Context, userID string, options map[string]interface{}, rawCredential json.RawMessage) (bool, error) {
	if len(rawCredential) == 0 {
		return false, &requestError{status: 400, message: "WebAuthn credential is required."}
	}

	challenge, _ := options["challenge"].(string)
	if strings.TrimSpace(challenge) == "" {
		return false, &requestError{status: 400, message: "Verification session is missing WebAuthn options."}
	}

	credentials, err := s.loadStoredWebAuthnCredentials(ctx, userID)
	if err != nil {
		return false, nil
	}
	if len(credentials) == 0 {
		return false, nil
	}

	if _, err := webauthnflow.VerifyAuthentication(rawCredential, challenge, credentials); err != nil {
		return false, nil
	}
	return true, nil
}

func (s Service) loadWebAuthnDescriptors(ctx context.Context, userID string) ([]webauthnflow.CredentialDescriptor, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("postgres is not configured")
	}

	rows, err := s.DB.Query(
		ctx,
		`SELECT "credentialId", transports
		   FROM "WebAuthnCredential"
		  WHERE "userId" = $1
		  ORDER BY "createdAt" DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("load webauthn credentials: %w", err)
	}
	defer rows.Close()

	result := make([]webauthnflow.CredentialDescriptor, 0)
	for rows.Next() {
		var (
			credentialID string
			transports   []string
		)
		if err := rows.Scan(&credentialID, &transports); err != nil {
			return nil, fmt.Errorf("scan webauthn credential: %w", err)
		}
		result = append(result, webauthnflow.CredentialDescriptor{
			ID:         strings.TrimSpace(credentialID),
			Type:       "public-key",
			Transports: transports,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webauthn credentials: %w", err)
	}
	return result, nil
}

func (s Service) loadStoredWebAuthnCredentials(ctx context.Context, userID string) ([]webauthnflow.StoredCredential, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("postgres is not configured")
	}

	rows, err := s.DB.Query(
		ctx,
		`SELECT "credentialId", "publicKey", counter
		   FROM "WebAuthnCredential"
		  WHERE "userId" = $1
		  ORDER BY "createdAt" DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("load webauthn stored credentials: %w", err)
	}
	defer rows.Close()

	result := make([]webauthnflow.StoredCredential, 0)
	for rows.Next() {
		var (
			credentialID string
			publicKey    string
			counter      int64
		)
		if err := rows.Scan(&credentialID, &publicKey, &counter); err != nil {
			return nil, fmt.Errorf("scan webauthn stored credential: %w", err)
		}
		if counter < 0 {
			counter = 0
		}
		result = append(result, webauthnflow.StoredCredential{
			CredentialID: strings.TrimSpace(credentialID),
			PublicKey:    strings.TrimSpace(publicKey),
			Counter:      uint32(counter),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate webauthn stored credentials: %w", err)
	}
	return result, nil
}
