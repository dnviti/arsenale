package connections

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
)

type shareTargetUser struct {
	ID    string
	Email string
}

func (s Service) resolveShareableConnection(ctx context.Context, userID, tenantID, connectionID string) (accessResult, error) {
	access, err := s.resolveAccess(ctx, userID, tenantID, connectionID)
	if err != nil {
		return accessResult{}, err
	}
	if access.AccessType == "shared" {
		return accessResult{}, pgx.ErrNoRows
	}
	if access.AccessType == "team" && (access.Connection.TeamRole == nil || *access.Connection.TeamRole != "TEAM_ADMIN") {
		return accessResult{}, &requestError{status: 403, message: "Only team admins can manage team connection shares"}
	}
	return access, nil
}

func (s Service) resolveShareTargetUser(ctx context.Context, target shareTarget) (shareTargetUser, error) {
	target.Email = normalizeOptionalStringPtrValue(target.Email)
	target.UserID = normalizeOptionalStringPtrValue(target.UserID)
	if err := validateShareTarget(target); err != nil {
		return shareTargetUser{}, err
	}

	var user shareTargetUser
	if target.UserID != nil {
		if err := s.DB.QueryRow(ctx, `SELECT id, email FROM "User" WHERE id = $1`, *target.UserID).Scan(&user.ID, &user.Email); err != nil {
			if err == pgx.ErrNoRows {
				return shareTargetUser{}, &requestError{status: 404, message: "User not found"}
			}
			return shareTargetUser{}, fmt.Errorf("load share target: %w", err)
		}
		return user, nil
	}
	if err := s.DB.QueryRow(ctx, `SELECT id, email FROM "User" WHERE LOWER(email) = LOWER($1)`, *target.Email).Scan(&user.ID, &user.Email); err != nil {
		if err == pgx.ErrNoRows {
			return shareTargetUser{}, &requestError{status: 404, message: "User not found"}
		}
		return shareTargetUser{}, fmt.Errorf("load share target: %w", err)
	}
	return user, nil
}

func (s Service) assertShareableTenantBoundary(ctx context.Context, actingUserID, targetUserID string) error {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOW_EXTERNAL_SHARING")), "true") {
		return nil
	}
	actingTenantIDs, err := s.loadAcceptedTenantIDs(ctx, actingUserID)
	if err != nil {
		return err
	}
	targetTenantIDs, err := s.loadAcceptedTenantIDs(ctx, targetUserID)
	if err != nil {
		return err
	}
	if len(actingTenantIDs) == 0 && len(targetTenantIDs) == 0 {
		return nil
	}
	for _, actingTenantID := range actingTenantIDs {
		for _, targetTenantID := range targetTenantIDs {
			if actingTenantID == targetTenantID {
				return nil
			}
		}
	}
	return &requestError{status: 403, message: "Cannot share connections with users outside your tenant"}
}

func (s Service) loadAcceptedTenantIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.DB.Query(ctx, `
SELECT "tenantId"
FROM "TenantMember"
WHERE "userId" = $1
  AND status = 'ACCEPTED'
  AND ("expiresAt" IS NULL OR "expiresAt" > NOW())
`, userID)
	if err != nil {
		return nil, fmt.Errorf("list tenant memberships: %w", err)
	}
	defer rows.Close()

	result := make([]string, 0)
	for rows.Next() {
		var tenantID string
		if err := rows.Scan(&tenantID); err != nil {
			return nil, fmt.Errorf("scan tenant membership: %w", err)
		}
		result = append(result, tenantID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenant memberships: %w", err)
	}
	return result, nil
}
