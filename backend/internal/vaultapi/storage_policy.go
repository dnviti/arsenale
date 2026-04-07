package vaultapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s Service) loadUserSettings(ctx context.Context, userID string) (userVaultSettings, error) {
	if s.DB == nil {
		return userVaultSettings{}, fmt.Errorf("database is unavailable")
	}

	var result userVaultSettings
	if err := s.DB.QueryRow(
		ctx,
		`SELECT COALESCE("vaultNeedsRecovery", false),
		        COALESCE("webauthnEnabled", false),
		        COALESCE("totpEnabled", false),
		        COALESCE("smsMfaEnabled", false),
		        "vaultAutoLockMinutes"
		   FROM "User"
		  WHERE id = $1`,
		userID,
	).Scan(
		&result.VaultNeedsRecovery,
		&result.WebAuthnEnabled,
		&result.TOTPEnabled,
		&result.SMSMFAEnabled,
		&result.AutoLockMinutes,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return userVaultSettings{}, &requestError{status: 404, message: "User not found"}
		}
		return userVaultSettings{}, fmt.Errorf("load user vault settings: %w", err)
	}
	return result, nil
}

func (s Service) loadTenantPolicy(ctx context.Context, userID, tenantID string) (tenantVaultPolicy, error) {
	if s.DB == nil {
		return tenantVaultPolicy{}, fmt.Errorf("database is unavailable")
	}

	if strings.TrimSpace(tenantID) != "" {
		policy, found, err := s.queryTenantPolicy(ctx, userID, tenantID)
		if err != nil {
			return tenantVaultPolicy{}, err
		}
		if found {
			return policy, nil
		}
	}

	policy, _, err := s.queryAnyTenantPolicy(ctx, userID)
	if err != nil {
		return tenantVaultPolicy{}, err
	}
	return policy, nil
}

func (s Service) queryTenantPolicy(ctx context.Context, userID, tenantID string) (tenantVaultPolicy, bool, error) {
	var policy tenantVaultPolicy
	err := s.DB.QueryRow(
		ctx,
		`SELECT t."vaultAutoLockMaxMinutes", t."vaultDefaultTtlMinutes"
		   FROM "TenantMember" tm
		   JOIN "Tenant" t ON t.id = tm."tenantId"
		  WHERE tm."userId" = $1
		    AND tm."tenantId" = $2
		    AND tm."isActive" = true
		    AND tm.status::text = 'ACCEPTED'
		    AND (tm."expiresAt" IS NULL OR tm."expiresAt" > NOW())
		  LIMIT 1`,
		userID,
		tenantID,
	).Scan(&policy.MaxMinutes, &policy.DefaultMinutes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return tenantVaultPolicy{}, false, nil
		}
		return tenantVaultPolicy{}, false, fmt.Errorf("load tenant vault policy: %w", err)
	}
	return policy, true, nil
}

func (s Service) queryAnyTenantPolicy(ctx context.Context, userID string) (tenantVaultPolicy, bool, error) {
	var policy tenantVaultPolicy
	err := s.DB.QueryRow(
		ctx,
		`SELECT t."vaultAutoLockMaxMinutes", t."vaultDefaultTtlMinutes"
		   FROM "TenantMember" tm
		   JOIN "Tenant" t ON t.id = tm."tenantId"
		  WHERE tm."userId" = $1
		    AND tm."isActive" = true
		    AND tm.status::text = 'ACCEPTED'
		    AND (tm."expiresAt" IS NULL OR tm."expiresAt" > NOW())
		  ORDER BY tm."joinedAt" ASC, tm."tenantId" ASC
		  LIMIT 1`,
		userID,
	).Scan(&policy.MaxMinutes, &policy.DefaultMinutes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return tenantVaultPolicy{}, false, nil
		}
		return tenantVaultPolicy{}, false, fmt.Errorf("load tenant vault policy: %w", err)
	}
	return policy, true, nil
}

func resolveEffectiveMinutes(userPref, tenantDefault, tenantMax *int) int {
	effective := envDefaultVaultMinutes()
	if tenantDefault != nil {
		effective = *tenantDefault
	}
	if userPref != nil {
		effective = *userPref
	}

	if tenantMax != nil && *tenantMax > 0 {
		if effective == 0 {
			effective = *tenantMax
		} else if effective > *tenantMax {
			effective = *tenantMax
		}
	}

	return effective
}

func envDefaultVaultMinutes() int {
	raw := strings.TrimSpace(os.Getenv("VAULT_TTL_MINUTES"))
	if raw == "" {
		return 30
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 30
	}
	return value
}
