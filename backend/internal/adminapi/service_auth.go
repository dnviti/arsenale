package adminapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) requireTenantAdmin(ctx context.Context, claims authn.Claims) error {
	if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.TenantID) == "" {
		return &requestError{status: http.StatusForbidden, message: "Tenant membership required"}
	}

	membership, err := s.TenantAuth.ResolveMembership(ctx, claims.UserID, claims.TenantID)
	if err != nil {
		return fmt.Errorf("resolve tenant membership: %w", err)
	}
	if membership == nil {
		return &requestError{status: http.StatusForbidden, message: "Tenant membership required"}
	}
	if !roleAtLeast(membership.Role, "ADMIN") {
		return &requestError{status: http.StatusForbidden, message: "Insufficient tenant role"}
	}
	return nil
}

func roleAtLeast(actual, required string) bool {
	rank := map[string]int{
		"GUEST":      1,
		"AUDITOR":    2,
		"CONSULTANT": 3,
		"MEMBER":     4,
		"OPERATOR":   5,
		"ADMIN":      6,
		"OWNER":      7,
	}
	return rank[strings.ToUpper(actual)] >= rank[strings.ToUpper(required)]
}
