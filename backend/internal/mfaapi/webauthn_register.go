package mfaapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/webauthnflow"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) RegisterWebAuthnCredential(ctx context.Context, userID string, rawCredential json.RawMessage, friendlyName, fallbackChallenge, ipAddress string) (webauthnCredentialInfo, error) {
	if s.DB == nil {
		return webauthnCredentialInfo{}, fmt.Errorf("database is unavailable")
	}
	if len(rawCredential) == 0 {
		return webauthnCredentialInfo{}, requestErr(400, "credential is required")
	}

	friendlyName = strings.TrimSpace(friendlyName)
	if friendlyName == "" {
		friendlyName = "Security Key"
	}
	if len(friendlyName) > 64 {
		return webauthnCredentialInfo{}, requestErr(400, "friendlyName must be between 1 and 64 characters")
	}

	challenge, err := s.resolveRegistrationChallenge(ctx, userID, fallbackChallenge)
	if err != nil {
		return webauthnCredentialInfo{}, err
	}

	credential, err := webauthnflow.VerifyRegistration(rawCredential, challenge)
	if err != nil {
		switch {
		case errors.Is(err, webauthnflow.ErrChallengeNotFound):
			return webauthnCredentialInfo{}, requestErr(400, "Challenge expired or not found. Please try again.")
		default:
			return webauthnCredentialInfo{}, requestErr(400, "WebAuthn registration verification failed.")
		}
	}

	var existingID string
	err = s.DB.QueryRow(
		ctx,
		`SELECT id
		   FROM "WebAuthnCredential"
		  WHERE "credentialId" = $1`,
		credential.CredentialID,
	).Scan(&existingID)
	switch {
	case err == nil:
		return webauthnCredentialInfo{}, requestErr(409, "Credential already registered")
	case err != nil && !errors.Is(err, pgx.ErrNoRows):
		return webauthnCredentialInfo{}, fmt.Errorf("check existing webauthn credential: %w", err)
	}

	now := time.Now().UTC()
	credentialID := uuid.NewString()
	if _, err := s.DB.Exec(
		ctx,
		`INSERT INTO "WebAuthnCredential" (
			id, "userId", "credentialId", "publicKey", counter, transports, "deviceType", "backedUp", "friendlyName", aaguid, "createdAt"
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`,
		credentialID,
		userID,
		credential.CredentialID,
		credential.PublicKey,
		int64(credential.Counter),
		credential.Transports,
		credential.DeviceType,
		credential.BackedUp,
		friendlyName,
		credential.AAGUID,
		now,
	); err != nil {
		return webauthnCredentialInfo{}, fmt.Errorf("create webauthn credential: %w", err)
	}

	if _, err := s.DB.Exec(
		ctx,
		`UPDATE "User"
		    SET "webauthnEnabled" = true,
		        "updatedAt" = NOW()
		  WHERE id = $1`,
		userID,
	); err != nil {
		return webauthnCredentialInfo{}, fmt.Errorf("enable webauthn flag: %w", err)
	}

	if err := s.insertAuditLog(ctx, userID, "WEBAUTHN_REGISTER", ipAddress); err != nil {
		return webauthnCredentialInfo{}, err
	}

	return webauthnCredentialInfo{
		ID:           credentialID,
		CredentialID: credential.CredentialID,
		FriendlyName: friendlyName,
		DeviceType:   credential.DeviceType,
		BackedUp:     credential.BackedUp,
		LastUsedAt:   nil,
		CreatedAt:    now.Format(time.RFC3339Nano),
	}, nil
}

func (s Service) resolveRegistrationChallenge(ctx context.Context, userID, fallbackChallenge string) (string, error) {
	flow := webauthnflow.New(s.Redis)
	challenge, err := flow.TakeChallenge(ctx, userID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(challenge) == "" {
		challenge = strings.TrimSpace(fallbackChallenge)
	}
	if strings.TrimSpace(challenge) == "" {
		return "", requestErr(400, "Challenge expired or not found. Please try again.")
	}
	return challenge, nil
}
