package authservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const cliRefreshTokenFamilyPrefix = "cli:"

// RefreshToken.expiresAt is non-null, so CLI sessions use a far-future expiry.
var persistentRefreshTokenExpiresAt = time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)

type refreshTokenLifetime struct {
	expiresAt time.Time
	cookieTTL time.Duration
}

func (s Service) issueTokens(ctx context.Context, user loginUser, ipAddress, userAgent string) (issuedLogin, error) {
	return s.issueTokensForFamily(ctx, user, ipAddress, userAgent, uuid.NewString(), time.Now())
}

func (s Service) issueAccessToken(user loginUser, ipAddress, userAgent string) (string, error) {
	now := time.Now()
	active := user.ActiveTenant
	accessTTL := s.AccessTokenTTL
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	if active != nil && active.JWTExpiresInSeconds != nil && *active.JWTExpiresInSeconds > 0 {
		accessTTL = time.Duration(*active.JWTExpiresInSeconds) * time.Second
	}

	var ipUaHash *string
	if s.TokenBinding {
		hash := computeBindingHash(ipAddress, userAgent)
		ipUaHash = &hash
	}

	claims := jwt.MapClaims{
		"userId": user.ID,
		"email":  user.Email,
		"type":   "access",
		"iat":    now.Unix(),
		"exp":    now.Add(accessTTL).Unix(),
	}
	if ipUaHash != nil {
		claims["ipUaHash"] = *ipUaHash
	}
	if active != nil {
		claims["tenantId"] = active.TenantID
		claims["tenantRole"] = active.Role
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.JWTSecret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return accessToken, nil
}

func (s Service) issueTokensForFamily(ctx context.Context, user loginUser, ipAddress, userAgent, tokenFamily string, familyCreatedAt time.Time) (issuedLogin, error) {
	return s.issueTokensForFamilyWithLifetime(ctx, user, ipAddress, userAgent, tokenFamily, familyCreatedAt, s.browserRefreshTokenLifetime(user, time.Now()))
}

func (s Service) issueCLITokens(ctx context.Context, user loginUser, ipAddress, userAgent string) (issuedLogin, error) {
	return s.issueTokensForFamilyWithLifetime(ctx, user, ipAddress, userAgent, newCLIRefreshTokenFamily(), time.Now(), cliRefreshTokenLifetime())
}

func (s Service) browserRefreshTokenLifetime(user loginUser, now time.Time) refreshTokenLifetime {
	ttl := s.refreshTokenTTL(user)
	return refreshTokenLifetime{
		expiresAt: now.Add(ttl),
		cookieTTL: ttl,
	}
}

func (s Service) refreshTokenTTL(user loginUser) time.Duration {
	active := user.ActiveTenant
	refreshTTL := s.RefreshCookieTTL
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}
	if active != nil && active.JWTRefreshExpiresSeconds != nil && *active.JWTRefreshExpiresSeconds > 0 {
		refreshTTL = time.Duration(*active.JWTRefreshExpiresSeconds) * time.Second
	}
	return refreshTTL
}

func cliRefreshTokenLifetime() refreshTokenLifetime {
	return refreshTokenLifetime{expiresAt: persistentRefreshTokenExpiresAt}
}

func newCLIRefreshTokenFamily() string {
	return cliRefreshTokenFamilyPrefix + uuid.NewString()
}

func isCLIRefreshTokenFamily(tokenFamily string) bool {
	return strings.HasPrefix(tokenFamily, cliRefreshTokenFamilyPrefix)
}

func (s Service) issueTokensForFamilyWithLifetime(
	ctx context.Context,
	user loginUser,
	ipAddress string,
	userAgent string,
	tokenFamily string,
	familyCreatedAt time.Time,
	lifetime refreshTokenLifetime,
) (issuedLogin, error) {
	if lifetime.expiresAt.IsZero() {
		lifetime = s.browserRefreshTokenLifetime(user, time.Now())
	}

	refreshToken := uuid.NewString()
	var ipUaHash *string
	if s.TokenBinding {
		hash := computeBindingHash(ipAddress, userAgent)
		ipUaHash = &hash
	}

	accessToken, err := s.issueAccessToken(user, ipAddress, userAgent)
	if err != nil {
		return issuedLogin{}, err
	}

	if _, err := s.DB.Exec(
		ctx,
		`INSERT INTO "RefreshToken" (id, token, "userId", "tokenFamily", "familyCreatedAt", "expiresAt", "ipUaHash")
		  VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.NewString(),
		refreshToken,
		user.ID,
		tokenFamily,
		familyCreatedAt,
		lifetime.expiresAt,
		ipUaHash,
	); err != nil {
		return issuedLogin{}, fmt.Errorf("insert refresh token: %w", err)
	}

	return issuedLogin{
		accessToken:       accessToken,
		refreshToken:      refreshToken,
		refreshExpires:    lifetime.cookieTTL,
		user:              buildLoginUserResponse(user),
		tenantMemberships: buildTenantMemberships(user),
	}, nil
}
