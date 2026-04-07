package dbauditapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
)

func (s Service) CreateMaskingPolicy(ctx context.Context, claims authn.Claims, name, columnPattern, strategy string, exemptRoles []string, scope, description *string, enabled *bool, ip *string) (maskingPolicy, error) {
	if s.DB == nil {
		return maskingPolicy{}, errors.New("database is unavailable")
	}
	name, err := validateName(name)
	if err != nil {
		return maskingPolicy{}, err
	}
	columnPattern, err = validateSafeRegex(columnPattern, "masking policy")
	if err != nil {
		return maskingPolicy{}, err
	}
	strategy, err = normalizeMaskingStrategy(strategy)
	if err != nil {
		return maskingPolicy{}, err
	}
	policyID := uuid.NewString()
	now := time.Now().UTC()
	if _, err := s.DB.Exec(ctx, `
INSERT INTO "DbMaskingPolicy" (id, "tenantId", name, "columnPattern", strategy, "exemptRoles", scope, description, enabled, "createdAt", "updatedAt")
VALUES ($1, $2, $3, $4, $5::"MaskingStrategy", $6, $7, $8, $9, $10, $11)
`, policyID, claims.TenantID, name, columnPattern, strategy, defaultStringSlice(exemptRoles), normalizeOptionalString(scope), normalizeOptionalString(description), defaultBool(enabled, true), now, now); err != nil {
		return maskingPolicy{}, fmt.Errorf("create masking policy: %w", err)
	}
	item, err := s.GetMaskingPolicy(ctx, claims.TenantID, policyID)
	if err != nil {
		return maskingPolicy{}, fmt.Errorf("reload masking policy: %w", err)
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "DB_MASKING_POLICY_CREATE", "DbMaskingPolicy", item.ID, map[string]any{"name": item.Name, "strategy": item.Strategy}, ip)
	return item, nil
}

func (s Service) UpdateMaskingPolicy(ctx context.Context, claims authn.Claims, policyID string, payload map[string]json.RawMessage, ip *string) (maskingPolicy, error) {
	if s.DB == nil {
		return maskingPolicy{}, errors.New("database is unavailable")
	}
	setClauses := []string{}
	args := []any{policyID, claims.TenantID}
	add := func(clause string, value any) {
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf(clause, len(args)))
	}
	if raw, ok := payload["name"]; ok {
		value, err := decodeString(raw)
		if err != nil {
			return maskingPolicy{}, &requestError{status: http.StatusBadRequest, message: "name must be a string"}
		}
		value, err = validateName(value)
		if err != nil {
			return maskingPolicy{}, err
		}
		add(`name = $%d`, value)
	}
	if raw, ok := payload["columnPattern"]; ok {
		value, err := decodeString(raw)
		if err != nil {
			return maskingPolicy{}, &requestError{status: http.StatusBadRequest, message: "columnPattern must be a string"}
		}
		value, err = validateSafeRegex(value, "masking policy")
		if err != nil {
			return maskingPolicy{}, err
		}
		add(`"columnPattern" = $%d`, value)
	}
	if raw, ok := payload["strategy"]; ok {
		value, err := decodeString(raw)
		if err != nil {
			return maskingPolicy{}, &requestError{status: http.StatusBadRequest, message: "strategy must be a string"}
		}
		value, err = normalizeMaskingStrategy(value)
		if err != nil {
			return maskingPolicy{}, err
		}
		add(`strategy = $%d::"MaskingStrategy"`, value)
	}
	if raw, ok := payload["exemptRoles"]; ok {
		values, err := decodeStringSlice(raw)
		if err != nil {
			return maskingPolicy{}, &requestError{status: http.StatusBadRequest, message: "exemptRoles must be an array of strings"}
		}
		add(`"exemptRoles" = $%d`, values)
	}
	if raw, ok := payload["scope"]; ok {
		value, err := decodeOptionalString(raw)
		if err != nil {
			return maskingPolicy{}, &requestError{status: http.StatusBadRequest, message: "scope must be a string or null"}
		}
		add(`scope = $%d`, value)
	}
	if raw, ok := payload["description"]; ok {
		value, err := decodeOptionalString(raw)
		if err != nil {
			return maskingPolicy{}, &requestError{status: http.StatusBadRequest, message: "description must be a string or null"}
		}
		add(`description = $%d`, value)
	}
	if raw, ok := payload["enabled"]; ok {
		value, err := decodeBool(raw)
		if err != nil {
			return maskingPolicy{}, &requestError{status: http.StatusBadRequest, message: "enabled must be a boolean"}
		}
		add(`enabled = $%d`, value)
	}
	if len(setClauses) == 0 {
		return s.GetMaskingPolicy(ctx, claims.TenantID, policyID)
	}
	setClauses = append(setClauses, `"updatedAt" = NOW()`)
	query := fmt.Sprintf(`
UPDATE "DbMaskingPolicy"
SET %s
WHERE id = $1 AND "tenantId" = $2
`, strings.Join(setClauses, ", "))
	cmd, err := s.DB.Exec(ctx, query, args...)
	if err != nil {
		return maskingPolicy{}, fmt.Errorf("update masking policy: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return maskingPolicy{}, &requestError{status: http.StatusNotFound, message: "Masking policy not found"}
	}
	item, err := s.GetMaskingPolicy(ctx, claims.TenantID, policyID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return maskingPolicy{}, &requestError{status: http.StatusNotFound, message: "Masking policy not found"}
		}
		return maskingPolicy{}, fmt.Errorf("reload masking policy: %w", err)
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "DB_MASKING_POLICY_UPDATE", "DbMaskingPolicy", item.ID, map[string]any{"name": item.Name}, ip)
	return item, nil
}

func (s Service) DeleteMaskingPolicy(ctx context.Context, claims authn.Claims, policyID string, ip *string) error {
	if s.DB == nil {
		return errors.New("database is unavailable")
	}
	cmd, err := s.DB.Exec(ctx, `DELETE FROM "DbMaskingPolicy" WHERE id = $1 AND "tenantId" = $2`, policyID, claims.TenantID)
	if err != nil {
		return fmt.Errorf("delete masking policy: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return &requestError{status: http.StatusNotFound, message: "Masking policy not found"}
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "DB_MASKING_POLICY_DELETE", "DbMaskingPolicy", policyID, nil, ip)
	return nil
}
