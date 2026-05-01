package authservice

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

func (s Service) loadLoginUser(ctx context.Context, email string) (loginUser, error) {
	return s.loadLoginUserByIDOrEmail(ctx, "email", email)
}

func (s Service) loadLoginUserByID(ctx context.Context, userID string) (loginUser, error) {
	return s.loadLoginUserByIDOrEmail(ctx, "id", userID)
}

func (s Service) loadLoginUserByIDOrEmail(ctx context.Context, field, value string) (loginUser, error) {
	var user loginUser
	query := `SELECT id, email, username, "avatarData", "passwordHash", "vaultSalt", "encryptedVaultKey", "vaultKeyIV", "vaultKeyTag",
		        enabled, "emailVerified", "totpEnabled", "smsMfaEnabled", "webauthnEnabled", COALESCE("phoneNumber", ''), COALESCE("phoneVerified", false), "failedLoginAttempts", "lockedUntil"
		   FROM "User"
		  WHERE email = $1`
	if field == "id" {
		query = `SELECT id, email, username, "avatarData", "passwordHash", "vaultSalt", "encryptedVaultKey", "vaultKeyIV", "vaultKeyTag",
		        enabled, "emailVerified", "totpEnabled", "smsMfaEnabled", "webauthnEnabled", COALESCE("phoneNumber", ''), COALESCE("phoneVerified", false), "failedLoginAttempts", "lockedUntil"
		   FROM "User"
		  WHERE id = $1`
	}
	err := s.DB.QueryRow(ctx, query, value).Scan(
		&user.ID, &user.Email, &user.Username, &user.AvatarData, &user.PasswordHash, &user.VaultSalt,
		&user.EncryptedVaultKey, &user.VaultKeyIV, &user.VaultKeyTag, &user.Enabled, &user.EmailVerified,
		&user.TOTPEnabled, &user.SMSMFAEnabled, &user.WebAuthnEnabled, &user.PhoneNumber, &user.PhoneVerified,
		&user.FailedLoginAttempts, &user.LockedUntil,
	)
	if err != nil {
		return loginUser{}, err
	}

	rows, err := s.DB.Query(
		ctx,
		`SELECT tm."tenantId", t.name, t.slug, tm.role::text, tm.status::text, tm."isActive", tm."joinedAt",
		        t."mfaRequired", t."ipAllowlistEnabled", t."ipAllowlistMode", t."ipAllowlistEntries",
		        t."jwtExpiresInSeconds", t."jwtRefreshExpiresInSeconds",
		        t."accountLockoutThreshold", t."accountLockoutDurationMs"
		   FROM "TenantMember" tm
		   JOIN "Tenant" t ON t.id = tm."tenantId"
		  WHERE tm."userId" = $1
		    AND (tm."expiresAt" IS NULL OR tm."expiresAt" > NOW())
		  ORDER BY tm."joinedAt" ASC`,
		user.ID,
	)
	if err != nil {
		return loginUser{}, fmt.Errorf("query memberships: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			membership       loginMembership
			allowlistMode    sql.NullString
			allowlistEntries []string
		)
		if err := rows.Scan(
			&membership.TenantID, &membership.Name, &membership.Slug, &membership.Role, &membership.Status, &membership.IsActive,
			&membership.JoinedAt, &membership.MFARequired, &membership.IPAllowlistEnabled, &allowlistMode, &allowlistEntries,
			&membership.JWTExpiresInSeconds,
			&membership.JWTRefreshExpiresSeconds, &membership.AccountLockoutThreshold, &membership.AccountLockoutDurationMs,
		); err != nil {
			return loginUser{}, fmt.Errorf("scan membership: %w", err)
		}
		membership.IPAllowlistMode = "flag"
		if allowlistMode.Valid && strings.TrimSpace(allowlistMode.String) != "" {
			membership.IPAllowlistMode = strings.TrimSpace(allowlistMode.String)
		}
		membership.IPAllowlistEntries = allowlistEntries
		user.Memberships = append(user.Memberships, membership)
	}
	if err := rows.Err(); err != nil {
		return loginUser{}, fmt.Errorf("iterate memberships: %w", err)
	}

	accepted := make([]loginMembership, 0)
	for _, membership := range user.Memberships {
		if membership.Status == "ACCEPTED" {
			accepted = append(accepted, membership)
		}
		if membership.IsActive && membership.Status == "ACCEPTED" {
			copyMembership := membership
			user.ActiveTenant = &copyMembership
		}
	}

	if user.ActiveTenant == nil && len(accepted) == 1 {
		if _, err := s.DB.Exec(ctx, `UPDATE "TenantMember" SET "isActive" = true WHERE "tenantId" = $1 AND "userId" = $2`, accepted[0].TenantID, user.ID); err != nil {
			return loginUser{}, fmt.Errorf("activate tenant membership: %w", err)
		}
		accepted[0].IsActive = true
		user.ActiveTenant = &accepted[0]
		for i := range user.Memberships {
			if user.Memberships[i].TenantID == accepted[0].TenantID {
				user.Memberships[i].IsActive = true
			}
		}
	}

	user, err = s.normalizeLoginUserForRuntime(ctx, user)
	if err != nil {
		return loginUser{}, err
	}

	user.HasLegacyOrAdvancedAuth = user.WebAuthnEnabled
	return user, nil
}

func buildLoginUserResponse(user loginUser) loginUserResponse {
	resultUser := loginUserResponse{
		ID:         user.ID,
		Email:      user.Email,
		Username:   user.Username,
		AvatarData: user.AvatarData,
	}
	if user.ActiveTenant != nil {
		resultUser.TenantID = user.ActiveTenant.TenantID
		resultUser.TenantRole = user.ActiveTenant.Role
	}
	return resultUser
}

func buildTenantMemberships(user loginUser) []tenantMembership {
	memberships := make([]tenantMembership, 0, len(user.Memberships))
	for _, membership := range user.Memberships {
		memberships = append(memberships, tenantMembership{
			TenantID: membership.TenantID,
			Name:     membership.Name,
			Slug:     membership.Slug,
			Role:     membership.Role,
			Status:   membership.Status,
			Pending:  membership.Status == "PENDING",
			IsActive: membership.IsActive,
		})
	}
	sort.Slice(memberships, func(i, j int) bool {
		rank := func(item tenantMembership) int {
			if item.IsActive {
				return 0
			}
			if item.Pending {
				return 2
			}
			return 1
		}
		ri, rj := rank(memberships[i]), rank(memberships[j])
		if ri != rj {
			return ri < rj
		}
		return memberships[i].Name < memberships[j].Name
	})
	return memberships
}
