package tenants

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func (s Service) GetTenant(ctx context.Context, tenantID string) (tenantResponse, error) {
	row := s.DB.QueryRow(ctx, `
SELECT
	t.id,
	t.name,
	t.slug,
	t."mfaRequired",
	t."vaultAutoLockMaxMinutes",
	(
		SELECT COUNT(*)::int
		FROM "TenantMember" tm
		WHERE tm."tenantId" = t.id
		  AND tm.status = 'ACCEPTED'
	) AS "userCount",
	t."defaultSessionTimeoutSeconds",
	t."maxConcurrentSessions",
	t."absoluteSessionTimeoutSeconds",
	t."dlpDisableCopy",
	t."dlpDisablePaste",
	t."dlpDisableDownload",
	t."dlpDisableUpload",
	t."enforcedConnectionSettings",
	t."tunnelDefaultEnabled",
	t."tunnelAutoTokenRotation",
	t."tunnelTokenRotationDays",
	t."tunnelRequireForRemote",
	t."tunnelTokenMaxLifetimeDays",
	t."tunnelAgentAllowedCidrs",
	t."loginRateLimitWindowMs",
	t."loginRateLimitMaxAttempts",
	t."accountLockoutThreshold",
	t."accountLockoutDurationMs",
	t."impossibleTravelSpeedKmh",
	t."jwtExpiresInSeconds",
	t."jwtRefreshExpiresInSeconds",
	t."vaultDefaultTtlMinutes",
	t."recordingEnabled",
	t."recordingRetentionDays",
	t."fileUploadMaxSizeBytes",
	t."userDriveQuotaBytes",
	(
		SELECT COUNT(*)::int
		FROM "Team" team
		WHERE team."tenantId" = t.id
	) AS "teamCount",
	t."createdAt",
	t."updatedAt"
FROM "Tenant" t
WHERE t.id = $1
`, tenantID)

	var (
		result                     tenantResponse
		vaultAutoLock              sql.NullInt32
		tunnelTokenMaxLifetime     sql.NullInt32
		loginRateLimitWindow       sql.NullInt32
		loginRateLimitMaxAttempts  sql.NullInt32
		accountLockoutThreshold    sql.NullInt32
		accountLockoutDuration     sql.NullInt32
		impossibleTravelSpeed      sql.NullInt32
		jwtExpires                 sql.NullInt32
		jwtRefreshExpires          sql.NullInt32
		vaultDefaultTTL            sql.NullInt32
		recordingRetentionDays     sql.NullInt32
		fileUploadMaxSizeBytes     sql.NullInt32
		userDriveQuotaBytes        sql.NullInt32
		enforcedConnectionSettings []byte
		tunnelAgentAllowedCIDRs    []string
	)

	if err := row.Scan(
		&result.ID,
		&result.Name,
		&result.Slug,
		&result.MFARequired,
		&vaultAutoLock,
		&result.UserCount,
		&result.DefaultSessionTimeoutSeconds,
		&result.MaxConcurrentSessions,
		&result.AbsoluteSessionTimeoutSeconds,
		&result.DLPDisableCopy,
		&result.DLPDisablePaste,
		&result.DLPDisableDownload,
		&result.DLPDisableUpload,
		&enforcedConnectionSettings,
		&result.TunnelDefaultEnabled,
		&result.TunnelAutoTokenRotation,
		&result.TunnelTokenRotationDays,
		&result.TunnelRequireForRemote,
		&tunnelTokenMaxLifetime,
		&tunnelAgentAllowedCIDRs,
		&loginRateLimitWindow,
		&loginRateLimitMaxAttempts,
		&accountLockoutThreshold,
		&accountLockoutDuration,
		&impossibleTravelSpeed,
		&jwtExpires,
		&jwtRefreshExpires,
		&vaultDefaultTTL,
		&result.RecordingEnabled,
		&recordingRetentionDays,
		&fileUploadMaxSizeBytes,
		&userDriveQuotaBytes,
		&result.TeamCount,
		&result.CreatedAt,
		&result.UpdatedAt,
	); err != nil {
		return tenantResponse{}, fmt.Errorf("get tenant: %w", err)
	}

	result.VaultAutoLockMaxMinutes = nullInt(vaultAutoLock)
	result.TunnelTokenMaxLifetimeDays = nullInt(tunnelTokenMaxLifetime)
	result.LoginRateLimitWindowMs = nullInt(loginRateLimitWindow)
	result.LoginRateLimitMaxAttempts = nullInt(loginRateLimitMaxAttempts)
	result.AccountLockoutThreshold = nullInt(accountLockoutThreshold)
	result.AccountLockoutDurationMs = nullInt(accountLockoutDuration)
	result.ImpossibleTravelSpeedKmh = nullInt(impossibleTravelSpeed)
	result.JWTExpiresInSeconds = nullInt(jwtExpires)
	result.JWTRefreshExpiresInSeconds = nullInt(jwtRefreshExpires)
	result.VaultDefaultTTLMinutes = nullInt(vaultDefaultTTL)
	result.RecordingRetentionDays = nullInt(recordingRetentionDays)
	result.FileUploadMaxSizeBytes = nullInt(fileUploadMaxSizeBytes)
	result.UserDriveQuotaBytes = nullInt(userDriveQuotaBytes)
	result.TunnelAgentAllowedCIDRs = tunnelAgentAllowedCIDRs
	if len(enforcedConnectionSettings) > 0 {
		result.EnforcedConnectionSettings = json.RawMessage(enforcedConnectionSettings)
	}

	return normalizeTenantForRuntime(result, s.Features.RecordingsEnabled), nil
}

func (s Service) ListUserTenants(ctx context.Context, userID string) ([]tenantMembershipResponse, error) {
	rows, err := s.DB.Query(ctx, `
SELECT
	tm."tenantId",
	t.name,
	t.slug,
	tm.role::text,
	tm.status::text,
	tm."isActive",
	tm."joinedAt"
FROM "TenantMember" tm
JOIN "Tenant" t ON t.id = tm."tenantId"
WHERE tm."userId" = $1
  AND (tm."expiresAt" IS NULL OR tm."expiresAt" > NOW())
ORDER BY tm."joinedAt" ASC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user tenants: %w", err)
	}
	defer rows.Close()

	items := make([]tenantMembershipResponse, 0)
	for rows.Next() {
		var item tenantMembershipResponse
		if err := rows.Scan(
			&item.TenantID,
			&item.Name,
			&item.Slug,
			&item.Role,
			&item.Status,
			&item.IsActive,
			&item.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user tenant: %w", err)
		}
		item.Pending = item.Status == "PENDING"
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user tenants: %w", err)
	}

	sort.Slice(items, func(i, j int) bool {
		rank := func(item tenantMembershipResponse) int {
			if item.IsActive {
				return 0
			}
			if item.Pending {
				return 2
			}
			return 1
		}
		leftRank := rank(items[i])
		rightRank := rank(items[j])
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return filterTenantMembershipsForRuntime(items, s.Features.MultiTenancyEnabled), nil
}

func (s Service) GetTenantMFAStats(ctx context.Context, tenantID string) (map[string]int, error) {
	row := s.DB.QueryRow(ctx, `
SELECT
	COUNT(*)::int AS total,
	COUNT(*) FILTER (WHERE NOT u."totpEnabled" AND NOT u."smsMfaEnabled")::int AS "withoutMfa"
FROM "TenantMember" tm
JOIN "User" u ON u.id = tm."userId"
WHERE tm."tenantId" = $1
`, tenantID)

	var total, withoutMFA int
	if err := row.Scan(&total, &withoutMFA); err != nil {
		return nil, fmt.Errorf("get tenant mfa stats: %w", err)
	}
	return map[string]int{"total": total, "withoutMfa": withoutMFA}, nil
}
