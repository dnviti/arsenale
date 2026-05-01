package systemsettingsapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/tenantauth"
)

func (s Service) requireReader(ctx context.Context, claims authn.Claims) (*tenantauth.Membership, error) {
	return s.requireRole(ctx, claims, map[string]bool{
		"AUDITOR": true,
		"ADMIN":   true,
		"OWNER":   true,
	})
}

func (s Service) requireWriter(ctx context.Context, claims authn.Claims) (*tenantauth.Membership, error) {
	return s.requireRole(ctx, claims, map[string]bool{
		"ADMIN": true,
		"OWNER": true,
	})
}

func (s Service) requireRole(ctx context.Context, claims authn.Claims, allowed map[string]bool) (*tenantauth.Membership, error) {
	if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.TenantID) == "" {
		return nil, &requestError{status: 403, message: "Tenant membership required"}
	}

	membership, err := s.TenantAuth.ResolveMembership(ctx, claims.UserID, claims.TenantID)
	if err != nil {
		return nil, fmt.Errorf("resolve tenant membership: %w", err)
	}
	if membership == nil {
		return nil, &requestError{status: 403, message: "Tenant membership required"}
	}
	if !allowed[strings.ToUpper(strings.TrimSpace(membership.Role))] {
		return nil, &requestError{status: 403, message: "Insufficient tenant role"}
	}
	return membership, nil
}

func roleAtLeast(actual, required string) bool {
	ranks := map[string]int{
		"GUEST":      1,
		"AUDITOR":    2,
		"CONSULTANT": 3,
		"MEMBER":     4,
		"OPERATOR":   5,
		"ADMIN":      6,
		"OWNER":      7,
	}
	return ranks[strings.ToUpper(strings.TrimSpace(actual))] >= ranks[strings.ToUpper(strings.TrimSpace(required))]
}
