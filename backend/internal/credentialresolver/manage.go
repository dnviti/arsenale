package credentialresolver

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (r Resolver) RequireManageSecret(ctx context.Context, userID, secretID, tenantID string) (SecretManageAccess, error) {
	if r.DB == nil {
		return SecretManageAccess{}, errors.New("database is unavailable")
	}

	record, err := r.loadSecret(ctx, secretID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SecretManageAccess{}, &RequestError{Status: 404, Message: "Secret not found"}
		}
		return SecretManageAccess{}, err
	}

	access := SecretManageAccess{
		ID:       record.ID,
		Scope:    record.Scope,
		TeamID:   record.TeamID,
		TenantID: record.TenantID,
	}

	switch record.Scope {
	case "PERSONAL":
		if record.UserID == userID {
			return access, nil
		}
	case "TEAM":
		if record.TeamID == nil || *record.TeamID == "" {
			break
		}
		if tenantID != "" && record.TeamTenantID != nil && *record.TeamTenantID != "" && *record.TeamTenantID != tenantID {
			break
		}
		role, err := r.loadTeamRole(ctx, *record.TeamID, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				break
			}
			return SecretManageAccess{}, err
		}
		if role == "TEAM_ADMIN" || role == "TEAM_EDITOR" {
			access.TeamRole = role
			return access, nil
		}
	case "TENANT":
		if record.TenantID == nil || *record.TenantID == "" {
			break
		}
		if tenantID != "" && *record.TenantID != tenantID {
			break
		}
		ok, err := r.hasTenantManageAccess(ctx, *record.TenantID, userID)
		if err != nil {
			return SecretManageAccess{}, err
		}
		if ok {
			return access, nil
		}
	}

	return SecretManageAccess{}, &RequestError{Status: 404, Message: "Secret not found"}
}

func (r Resolver) loadTeamRole(ctx context.Context, teamID, userID string) (string, error) {
	var role string
	if err := r.DB.QueryRow(
		ctx,
		`SELECT role::text
		   FROM "TeamMember"
		  WHERE "teamId" = $1
		    AND "userId" = $2`,
		teamID,
		userID,
	).Scan(&role); err != nil {
		return "", fmt.Errorf("load team membership: %w", err)
	}
	return role, nil
}

func (r Resolver) hasTenantManageAccess(ctx context.Context, tenantID, userID string) (bool, error) {
	var exists bool
	if err := r.DB.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			  FROM "TenantMember"
			 WHERE "tenantId" = $1
			   AND "userId" = $2
			   AND status = 'ACCEPTED'
			   AND role IN ('OWNER', 'ADMIN')
			   AND ("expiresAt" IS NULL OR "expiresAt" > NOW())
		)`,
		tenantID,
		userID,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check tenant manage access: %w", err)
	}
	return exists, nil
}
