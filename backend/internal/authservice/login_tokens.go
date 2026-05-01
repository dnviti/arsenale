package authservice

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

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
	now := time.Now()
	active := user.ActiveTenant
	refreshTTL := s.RefreshCookieTTL
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}
	if active != nil && active.JWTRefreshExpiresSeconds != nil && *active.JWTRefreshExpiresSeconds > 0 {
		refreshTTL = time.Duration(*active.JWTRefreshExpiresSeconds) * time.Second
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
		now.Add(refreshTTL),
		ipUaHash,
	); err != nil {
		return issuedLogin{}, fmt.Errorf("insert refresh token: %w", err)
	}

	return issuedLogin{
		accessToken:       accessToken,
		refreshToken:      refreshToken,
		refreshExpires:    refreshTTL,
		user:              buildLoginUserResponse(user),
		tenantMemberships: buildTenantMemberships(user),
	}, nil
}
