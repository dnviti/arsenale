package authservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s Service) Logout(ctx context.Context, refreshToken, ipAddress string) (string, error) {
	if s.DB == nil || refreshToken == "" {
		return "", nil
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin logout: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		userID      *string
		tokenFamily *string
	)
	err = tx.QueryRow(ctx, `SELECT "userId", "tokenFamily" FROM "RefreshToken" WHERE token = $1`, refreshToken).Scan(&userID, &tokenFamily)
	if err != nil && err != pgx.ErrNoRows {
		return "", fmt.Errorf("load refresh token: %w", err)
	}
	if tokenFamily != nil && *tokenFamily != "" {
		if _, err := tx.Exec(ctx, `DELETE FROM "RefreshToken" WHERE "tokenFamily" = $1`, *tokenFamily); err != nil {
			return "", fmt.Errorf("delete refresh token family: %w", err)
		}
	} else if _, err := tx.Exec(ctx, `DELETE FROM "RefreshToken" WHERE token = $1`, refreshToken); err != nil {
		return "", fmt.Errorf("delete refresh token: %w", err)
	}

	if userID != nil && *userID != "" {
		if err := insertAuditLog(ctx, tx, userID, "LOGOUT", map[string]any{}, ipAddress); err != nil {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit logout: %w", err)
	}

	if userID == nil {
		return "", nil
	}
	return *userID, nil
}

func (s Service) Refresh(ctx context.Context, refreshToken, ipAddress, userAgent string) (issuedLogin, error) {
	if s.DB == nil {
		return issuedLogin{}, fmt.Errorf("postgres is not configured")
	}
	if refreshToken == "" {
		return issuedLogin{}, &requestError{status: 401, message: "Missing refresh token"}
	}

	type storedRefreshToken struct {
		ID              string
		UserID          string
		TokenFamily     string
		FamilyCreatedAt time.Time
		IPUAHash        *string
		RevokedAt       *time.Time
		ExpiresAt       time.Time
		UserEnabled     bool
	}

	var stored storedRefreshToken
	err := s.DB.QueryRow(
		ctx,
		`SELECT rt.id, rt."userId", rt."tokenFamily", rt."familyCreatedAt", rt."ipUaHash", rt."revokedAt", rt."expiresAt", u.enabled
		   FROM "RefreshToken" rt
		   JOIN "User" u ON u.id = rt."userId"
		  WHERE rt.token = $1`,
		refreshToken,
	).Scan(
		&stored.ID,
		&stored.UserID,
		&stored.TokenFamily,
		&stored.FamilyCreatedAt,
		&stored.IPUAHash,
		&stored.RevokedAt,
		&stored.ExpiresAt,
		&stored.UserEnabled,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return issuedLogin{}, &requestError{status: 401, message: "Invalid or expired refresh token"}
		}
		return issuedLogin{}, fmt.Errorf("load refresh token: %w", err)
	}

	if s.TokenBinding && stored.IPUAHash != nil && *stored.IPUAHash != "" {
		if computeBindingHash(ipAddress, userAgent) != *stored.IPUAHash {
			_, _ = s.DB.Exec(ctx, `DELETE FROM "RefreshToken" WHERE "tokenFamily" = $1`, stored.TokenFamily)
			_ = s.insertStandaloneAuditLog(ctx, &stored.UserID, "TOKEN_HIJACK_ATTEMPT", map[string]any{
				"tokenFamily": stored.TokenFamily,
				"reason":      "Refresh token presented from different IP/User-Agent",
			}, ipAddress)
			return issuedLogin{}, &requestError{status: 401, message: "Invalid or expired refresh token"}
		}
	}

	if stored.RevokedAt != nil {
		_, _ = s.DB.Exec(ctx, `DELETE FROM "RefreshToken" WHERE "tokenFamily" = $1`, stored.TokenFamily)
		_ = s.insertStandaloneAuditLog(ctx, &stored.UserID, "REFRESH_TOKEN_REUSE", map[string]any{
			"tokenFamily": stored.TokenFamily,
			"reason":      "Rotated refresh token reused — all family tokens revoked",
		}, ipAddress)
		return issuedLogin{}, &requestError{status: 401, message: "Invalid or expired refresh token"}
	}

	if stored.ExpiresAt.Before(time.Now()) {
		_, _ = s.DB.Exec(ctx, `DELETE FROM "RefreshToken" WHERE id = $1`, stored.ID)
		return issuedLogin{}, &requestError{status: 401, message: "Invalid or expired refresh token"}
	}

	if !stored.UserEnabled {
		_, _ = s.DB.Exec(ctx, `DELETE FROM "RefreshToken" WHERE "tokenFamily" = $1`, stored.TokenFamily)
		return issuedLogin{}, &requestError{status: 401, message: "Invalid or expired refresh token"}
	}

	user, err := s.loadLoginUserByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return issuedLogin{}, &requestError{status: 401, message: "Invalid or expired refresh token"}
		}
		return issuedLogin{}, err
	}

	if _, err := s.DB.Exec(ctx, `UPDATE "RefreshToken" SET "revokedAt" = NOW() WHERE id = $1`, stored.ID); err != nil {
		return issuedLogin{}, fmt.Errorf("revoke refresh token: %w", err)
	}

	return s.issueTokensForFamily(ctx, user, ipAddress, userAgent, stored.TokenFamily, stored.FamilyCreatedAt)
}

func (s Service) SwitchTenant(ctx context.Context, userID, targetTenantID, ipAddress, userAgent string) (issuedLogin, error) {
	if s.DB == nil {
		return issuedLogin{}, fmt.Errorf("postgres is not configured")
	}
	if targetTenantID == "" {
		return issuedLogin{}, &requestError{status: 400, message: "tenantId is required"}
	}

	tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return issuedLogin{}, fmt.Errorf("begin tenant switch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var expiresAt sql.NullTime
	if err := tx.QueryRow(ctx, `
SELECT "expiresAt"
FROM "TenantMember"
WHERE "tenantId" = $1 AND "userId" = $2
`, targetTenantID, userID).Scan(&expiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return issuedLogin{}, &requestError{status: 403, message: "You are not a member of this organization"}
		}
		return issuedLogin{}, fmt.Errorf("load tenant membership: %w", err)
	}
	if expiresAt.Valid && !expiresAt.Time.After(time.Now()) {
		return issuedLogin{}, &requestError{status: 403, message: "Your membership in this organization has expired"}
	}

	if _, err := tx.Exec(ctx, `
UPDATE "TenantMember"
SET "isActive" = false
WHERE "userId" = $1 AND "isActive" = true
`, userID); err != nil {
		return issuedLogin{}, fmt.Errorf("deactivate active memberships: %w", err)
	}

	if _, err := tx.Exec(ctx, `
UPDATE "TenantMember"
SET status = 'ACCEPTED', "isActive" = true
WHERE "tenantId" = $1 AND "userId" = $2
`, targetTenantID, userID); err != nil {
		return issuedLogin{}, fmt.Errorf("activate target membership: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM "RefreshToken" WHERE "userId" = $1`, userID); err != nil {
		return issuedLogin{}, fmt.Errorf("delete refresh tokens: %w", err)
	}

	if err := insertAuditLog(ctx, tx, &userID, "TENANT_SWITCH", map[string]any{
		"targetTenantId": targetTenantID,
	}, ipAddress); err != nil {
		return issuedLogin{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return issuedLogin{}, fmt.Errorf("commit tenant switch: %w", err)
	}

	user, err := s.loadLoginUserByID(ctx, userID)
	if err != nil {
		return issuedLogin{}, err
	}
	return s.issueTokens(ctx, user, ipAddress, userAgent)
}

func (s Service) IssueDeviceAuthTokens(ctx context.Context, userID, ipAddress, userAgent string) (map[string]any, time.Duration, error) {
	user, err := s.loadLoginUserByID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	result, err := s.issueTokens(ctx, user, ipAddress, userAgent)
	if err != nil {
		return nil, 0, err
	}

	return map[string]any{
		"access_token":  result.accessToken,
		"refresh_token": result.refreshToken,
		"token_type":    "Bearer",
		"user":          result.user,
	}, result.refreshExpires, nil
}
