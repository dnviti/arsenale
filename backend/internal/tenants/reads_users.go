package tenants

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s Service) ListTenantUsers(ctx context.Context, tenantID string) ([]tenantUserResponse, error) {
	rows, err := s.DB.Query(ctx, `
SELECT
	u.id,
	u.email,
	u.username,
	u."avatarData",
	tm.role::text,
	tm.status::text,
	u."totpEnabled",
	u."smsMfaEnabled",
	u.enabled,
	u."createdAt",
	tm."expiresAt"
FROM "TenantMember" tm
JOIN "User" u ON u.id = tm."userId"
WHERE tm."tenantId" = $1
`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list tenant users: %w", err)
	}
	defer rows.Close()

	result := make([]tenantUserResponse, 0)
	now := time.Now()
	for rows.Next() {
		var (
			item      tenantUserResponse
			username  sql.NullString
			avatar    sql.NullString
			expiresAt sql.NullTime
		)
		if err := rows.Scan(
			&item.ID,
			&item.Email,
			&username,
			&avatar,
			&item.Role,
			&item.Status,
			&item.TOTPEnabled,
			&item.SMSMFAEnabled,
			&item.Enabled,
			&item.CreatedAt,
			&expiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan tenant user: %w", err)
		}
		if username.Valid {
			item.Username = &username.String
		}
		if avatar.Valid {
			item.AvatarData = &avatar.String
		}
		item.Pending = item.Status == "PENDING"
		if expiresAt.Valid {
			value := expiresAt.Time
			item.ExpiresAt = &value
			item.Expired = !value.After(now)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenant users: %w", err)
	}

	roleOrder := map[string]int{
		"OWNER":      0,
		"ADMIN":      1,
		"OPERATOR":   2,
		"MEMBER":     3,
		"CONSULTANT": 4,
		"AUDITOR":    5,
		"GUEST":      6,
	}
	sort.Slice(result, func(i, j int) bool {
		leftPending := 0
		rightPending := 0
		if result[i].Pending {
			leftPending = 1
		}
		if result[j].Pending {
			rightPending = 1
		}
		if leftPending != rightPending {
			return leftPending < rightPending
		}
		leftOrder := roleOrder[result[i].Role]
		rightOrder := roleOrder[result[j].Role]
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		return strings.ToLower(result[i].Email) < strings.ToLower(result[j].Email)
	})

	return result, nil
}

func (s Service) GetUserProfile(ctx context.Context, tenantID, targetUserID, viewerRole string) (tenantUserProfileResponse, error) {
	row := s.DB.QueryRow(ctx, `
SELECT
	tm.role::text,
	tm."joinedAt",
	u.id,
	u.username,
	u."avatarData",
	u.email,
	u."totpEnabled",
	u."smsMfaEnabled",
	u."webauthnEnabled",
	u."updatedAt"
FROM "TenantMember" tm
JOIN "User" u ON u.id = tm."userId"
WHERE tm."tenantId" = $1
  AND tm."userId" = $2
`, tenantID, targetUserID)

	var (
		result          tenantUserProfileResponse
		username        sql.NullString
		avatarData      sql.NullString
		email           sql.NullString
		totpEnabled     bool
		smsMFAEnabled   bool
		webAuthnEnabled bool
		updatedAt       time.Time
	)

	if err := row.Scan(
		&result.Role,
		&result.JoinedAt,
		&result.ID,
		&username,
		&avatarData,
		&email,
		&totpEnabled,
		&smsMFAEnabled,
		&webAuthnEnabled,
		&updatedAt,
	); err != nil {
		return tenantUserProfileResponse{}, fmt.Errorf("get tenant user profile: %w", err)
	}
	if username.Valid {
		result.Username = &username.String
	}
	if avatarData.Valid {
		result.AvatarData = &avatarData.String
	}

	teamRows, err := s.DB.Query(ctx, `
SELECT t.id, t.name, tm.role::text
FROM "TeamMember" tm
JOIN "Team" t ON t.id = tm."teamId"
WHERE tm."userId" = $1
  AND t."tenantId" = $2
ORDER BY t.name ASC
`, targetUserID, tenantID)
	if err != nil {
		return tenantUserProfileResponse{}, fmt.Errorf("list tenant user teams: %w", err)
	}
	defer teamRows.Close()

	result.Teams = make([]tenantUserProfileTeam, 0)
	for teamRows.Next() {
		var team tenantUserProfileTeam
		if err := teamRows.Scan(&team.ID, &team.Name, &team.Role); err != nil {
			return tenantUserProfileResponse{}, fmt.Errorf("scan tenant user team: %w", err)
		}
		result.Teams = append(result.Teams, team)
	}
	if err := teamRows.Err(); err != nil {
		return tenantUserProfileResponse{}, fmt.Errorf("iterate tenant user teams: %w", err)
	}

	if claimsCanAdminTenant(viewerRole) {
		if email.Valid {
			result.Email = &email.String
		}
		result.TOTPEnabled = &totpEnabled
		result.SMSMFAEnabled = &smsMFAEnabled
		result.WebAuthnEnabled = &webAuthnEnabled
		result.UpdatedAt = &updatedAt

		var lastActivity sql.NullTime
		if err := s.DB.QueryRow(ctx, `
SELECT "createdAt"
FROM "AuditLog"
WHERE "userId" = $1
ORDER BY "createdAt" DESC
LIMIT 1
`, targetUserID).Scan(&lastActivity); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return tenantUserProfileResponse{}, fmt.Errorf("get tenant user last activity: %w", err)
		}
		if lastActivity.Valid {
			value := lastActivity.Time
			result.LastActivity = &value
		}
	}

	return result, nil
}
