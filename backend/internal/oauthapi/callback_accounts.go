package oauthapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) findOrCreateOAuthUser(ctx context.Context, profile oauthProfile, tokens oauthProviderTokens, samlAttributes map[string]any, ipAddress string) (oauthLoginResult, error) {
	if s.DB == nil {
		return oauthLoginResult{}, errors.New("database is unavailable")
	}

	samlAttrsRaw, err := marshalSAMLAttributes(samlAttributes)
	if err != nil {
		return oauthLoginResult{}, err
	}

	var (
		userID            string
		username          *string
		avatarData        *string
		enabled           bool
		needsVaultSetup   bool
		allowlistDecision ipAllowlistDecision
	)

	row := s.DB.QueryRow(ctx, `
SELECT oa.id, u.id, u.username, u."avatarData", u.enabled, NOT COALESCE(u."vaultSetupComplete", false)
FROM "OAuthAccount" oa
JOIN "User" u ON u.id = oa."userId"
WHERE oa.provider = $1
  AND oa."providerUserId" = $2
`, profile.Provider, profile.ProviderUserID)
	var accountID string
	err = row.Scan(&accountID, &userID, &username, &avatarData, &enabled, &needsVaultSetup)
	switch {
	case err == nil:
		if !enabled {
			return oauthLoginResult{}, &requestError{status: http.StatusForbidden, message: "Your account has been disabled. Contact your administrator."}
		}
		if _, err := s.DB.Exec(ctx, `
UPDATE "OAuthAccount"
   SET "accessToken" = $2,
       "refreshToken" = NULLIF($3, ''),
       "providerEmail" = $4,
       "samlAttributes" = COALESCE($5::jsonb, "samlAttributes")
 WHERE id = $1
`, accountID, tokens.AccessToken, tokens.RefreshToken, profile.Email, samlAttrsRaw); err != nil {
			return oauthLoginResult{}, fmt.Errorf("update oauth account: %w", err)
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return oauthLoginResult{}, fmt.Errorf("load oauth account: %w", err)
	default:
		err = s.DB.QueryRow(ctx, `
SELECT id, username, "avatarData", enabled, NOT COALESCE("vaultSetupComplete", false)
FROM "User"
WHERE LOWER(email) = LOWER($1)
`, profile.Email).Scan(&userID, &username, &avatarData, &enabled, &needsVaultSetup)
		switch {
		case err == nil:
			if !enabled {
				return oauthLoginResult{}, &requestError{status: http.StatusForbidden, message: "Your account has been disabled. Contact your administrator."}
			}
			if _, err := s.DB.Exec(ctx, `
INSERT INTO "OAuthAccount" (id, "userId", provider, "providerUserId", "providerEmail", "accessToken", "refreshToken", "samlAttributes")
VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8::jsonb)
`, uuid.NewString(), userID, profile.Provider, profile.ProviderUserID, profile.Email, tokens.AccessToken, tokens.RefreshToken, samlAttrsRaw); err != nil {
				return oauthLoginResult{}, fmt.Errorf("create linked oauth account: %w", err)
			}
		case !errors.Is(err, pgx.ErrNoRows):
			return oauthLoginResult{}, fmt.Errorf("load oauth user by email: %w", err)
		default:
			enabledSignup, err := s.getSelfSignupEnabled(ctx)
			if err != nil {
				return oauthLoginResult{}, err
			}
			if !enabledSignup {
				return oauthLoginResult{}, &requestError{status: http.StatusForbidden, message: "Registration is currently disabled. Contact your administrator."}
			}
			tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return oauthLoginResult{}, fmt.Errorf("begin oauth signup: %w", err)
			}
			defer func() { _ = tx.Rollback(ctx) }()

			userID = uuid.NewString()
			displayName := nullableString(profile.DisplayName)
			if err := tx.QueryRow(ctx, `
INSERT INTO "User" (id, email, username, "passwordHash", "vaultSalt", "encryptedVaultKey", "vaultKeyIV", "vaultKeyTag", "vaultSetupComplete", "emailVerified")
VALUES ($1, $2, $3, NULL, NULL, NULL, NULL, NULL, false, true)
RETURNING username, "avatarData", enabled, NOT COALESCE("vaultSetupComplete", false)
`, userID, profile.Email, displayName).Scan(&username, &avatarData, &enabled, &needsVaultSetup); err != nil {
				return oauthLoginResult{}, fmt.Errorf("create oauth user: %w", err)
			}
			if _, err := tx.Exec(ctx, `
INSERT INTO "OAuthAccount" (id, "userId", provider, "providerUserId", "providerEmail", "accessToken", "refreshToken", "samlAttributes")
VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8::jsonb)
`, uuid.NewString(), userID, profile.Provider, profile.ProviderUserID, profile.Email, tokens.AccessToken, tokens.RefreshToken, samlAttrsRaw); err != nil {
				return oauthLoginResult{}, fmt.Errorf("create oauth account: %w", err)
			}
			if err := tx.Commit(ctx); err != nil {
				return oauthLoginResult{}, fmt.Errorf("commit oauth signup: %w", err)
			}
		}
	}

	_ = s.inferDomainFromSAML(ctx, userID, samlAttributes, profile.Email)

	allowlist, err := s.loadTenantAllowlist(ctx, userID)
	if err != nil {
		return oauthLoginResult{}, err
	}
	allowlistDecision = evaluateIPAllowlist(allowlist, ipAddress)

	return oauthLoginResult{
		UserID:            userID,
		Email:             profile.Email,
		Username:          username,
		AvatarData:        avatarData,
		NeedsVaultSetup:   needsVaultSetup,
		AllowlistDecision: allowlistDecision,
	}, nil
}

func (s Service) linkOAuthAccount(ctx context.Context, userID string, profile oauthProfile, tokens oauthProviderTokens, samlAttributes map[string]any) error {
	if s.DB == nil {
		return errors.New("database is unavailable")
	}

	samlAttrsRaw, err := marshalSAMLAttributes(samlAttributes)
	if err != nil {
		return err
	}

	var existingUserID string
	err = s.DB.QueryRow(ctx, `
SELECT "userId"
FROM "OAuthAccount"
WHERE provider = $1
  AND "providerUserId" = $2
`, profile.Provider, profile.ProviderUserID).Scan(&existingUserID)
	switch {
	case err == nil:
		if existingUserID == userID {
			return nil
		}
		return &requestError{status: http.StatusConflict, message: "This OAuth account is already linked to a different user."}
	case !errors.Is(err, pgx.ErrNoRows):
		return fmt.Errorf("load linked oauth account: %w", err)
	}

	var existingAccountID string
	err = s.DB.QueryRow(ctx, `
SELECT id
FROM "OAuthAccount"
WHERE "userId" = $1
  AND provider = $2
`, userID, profile.Provider).Scan(&existingAccountID)
	switch {
	case err == nil:
		return &requestError{status: http.StatusConflict, message: fmt.Sprintf("You already have a %s account linked.", strings.ToLower(profile.Provider))}
	case !errors.Is(err, pgx.ErrNoRows):
		return fmt.Errorf("load user oauth account: %w", err)
	}

	if _, err := s.DB.Exec(ctx, `
INSERT INTO "OAuthAccount" (id, "userId", provider, "providerUserId", "providerEmail", "accessToken", "refreshToken", "samlAttributes")
VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8::jsonb)
`, uuid.NewString(), userID, profile.Provider, profile.ProviderUserID, profile.Email, tokens.AccessToken, tokens.RefreshToken, samlAttrsRaw); err != nil {
		return fmt.Errorf("create oauth link: %w", err)
	}
	_ = s.inferDomainFromSAML(ctx, userID, samlAttributes, profile.Email)
	return nil
}

func (s Service) getSelfSignupEnabled(ctx context.Context) (bool, error) {
	if strings.TrimSpace(oauthEnv("SELF_SIGNUP_ENABLED", "")) != "true" {
		return false, nil
	}
	if s.DB == nil {
		return false, errors.New("database is unavailable")
	}

	var value string
	err := s.DB.QueryRow(ctx, `SELECT value FROM "AppConfig" WHERE key = 'selfSignupEnabled'`).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("query self-signup flag: %w", err)
	}
	return strings.TrimSpace(strings.ToLower(value)) == "true", nil
}
