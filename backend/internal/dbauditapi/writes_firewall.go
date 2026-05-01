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

func (s Service) CreateFirewallRule(ctx context.Context, claims authn.Claims, name, pattern, action string, scope, description *string, enabled *bool, priority *int, ip *string) (firewallRule, error) {
	if s.DB == nil {
		return firewallRule{}, errors.New("database is unavailable")
	}
	name, err := validateName(name)
	if err != nil {
		return firewallRule{}, err
	}
	pattern, err = validateSafeRegex(pattern, "firewall rule")
	if err != nil {
		return firewallRule{}, err
	}
	action, err = normalizeFirewallAction(action)
	if err != nil {
		return firewallRule{}, err
	}
	ruleID := uuid.NewString()
	now := time.Now().UTC()
	if _, err := s.DB.Exec(ctx, `
INSERT INTO "DbFirewallRule" (id, "tenantId", name, pattern, action, scope, description, enabled, priority, "createdAt", "updatedAt")
VALUES ($1, $2, $3, $4, $5::"FirewallAction", $6, $7, $8, $9, $10, $11)
`, ruleID, claims.TenantID, name, pattern, action, normalizeOptionalString(scope), normalizeOptionalString(description), defaultBool(enabled, true), defaultInt(priority, 0), now, now); err != nil {
		return firewallRule{}, fmt.Errorf("create firewall rule: %w", err)
	}
	item, err := s.GetFirewallRule(ctx, claims.TenantID, ruleID)
	if err != nil {
		return firewallRule{}, fmt.Errorf("reload firewall rule: %w", err)
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "DB_FIREWALL_RULE_CREATE", "DbFirewallRule", item.ID, map[string]any{
		"name": item.Name, "pattern": item.Pattern, "firewallAction": item.Action,
	}, ip)
	return item, nil
}

func (s Service) UpdateFirewallRule(ctx context.Context, claims authn.Claims, ruleID string, payload map[string]json.RawMessage, ip *string) (firewallRule, error) {
	if s.DB == nil {
		return firewallRule{}, errors.New("database is unavailable")
	}
	setClauses := []string{}
	args := []any{ruleID, claims.TenantID}
	add := func(clause string, value any) {
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf(clause, len(args)))
	}
	if raw, ok := payload["name"]; ok {
		value, err := decodeString(raw)
		if err != nil {
			return firewallRule{}, &requestError{status: http.StatusBadRequest, message: "name must be a string"}
		}
		value, err = validateName(value)
		if err != nil {
			return firewallRule{}, err
		}
		add(`name = $%d`, value)
	}
	if raw, ok := payload["pattern"]; ok {
		value, err := decodeString(raw)
		if err != nil {
			return firewallRule{}, &requestError{status: http.StatusBadRequest, message: "pattern must be a string"}
		}
		value, err = validateSafeRegex(value, "firewall rule")
		if err != nil {
			return firewallRule{}, err
		}
		add(`pattern = $%d`, value)
	}
	if raw, ok := payload["action"]; ok {
		value, err := decodeString(raw)
		if err != nil {
			return firewallRule{}, &requestError{status: http.StatusBadRequest, message: "action must be a string"}
		}
		value, err = normalizeFirewallAction(value)
		if err != nil {
			return firewallRule{}, err
		}
		add(`action = $%d::"FirewallAction"`, value)
	}
	if raw, ok := payload["scope"]; ok {
		value, err := decodeOptionalString(raw)
		if err != nil {
			return firewallRule{}, &requestError{status: http.StatusBadRequest, message: "scope must be a string or null"}
		}
		add(`scope = $%d`, value)
	}
	if raw, ok := payload["description"]; ok {
		value, err := decodeOptionalString(raw)
		if err != nil {
			return firewallRule{}, &requestError{status: http.StatusBadRequest, message: "description must be a string or null"}
		}
		add(`description = $%d`, value)
	}
	if raw, ok := payload["enabled"]; ok {
		value, err := decodeBool(raw)
		if err != nil {
			return firewallRule{}, &requestError{status: http.StatusBadRequest, message: "enabled must be a boolean"}
		}
		add(`enabled = $%d`, value)
	}
	if raw, ok := payload["priority"]; ok {
		value, err := decodeInt(raw)
		if err != nil {
			return firewallRule{}, &requestError{status: http.StatusBadRequest, message: "priority must be an integer"}
		}
		add(`priority = $%d`, value)
	}
	if len(setClauses) == 0 {
		return s.GetFirewallRule(ctx, claims.TenantID, ruleID)
	}
	setClauses = append(setClauses, `"updatedAt" = NOW()`)
	query := fmt.Sprintf(`
UPDATE "DbFirewallRule"
SET %s
WHERE id = $1 AND "tenantId" = $2
`, strings.Join(setClauses, ", "))
	cmd, err := s.DB.Exec(ctx, query, args...)
	if err != nil {
		return firewallRule{}, fmt.Errorf("update firewall rule: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return firewallRule{}, &requestError{status: http.StatusNotFound, message: "Firewall rule not found"}
	}
	item, err := s.GetFirewallRule(ctx, claims.TenantID, ruleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return firewallRule{}, &requestError{status: http.StatusNotFound, message: "Firewall rule not found"}
		}
		return firewallRule{}, fmt.Errorf("reload firewall rule: %w", err)
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "DB_FIREWALL_RULE_UPDATE", "DbFirewallRule", item.ID, map[string]any{"name": item.Name}, ip)
	return item, nil
}

func (s Service) DeleteFirewallRule(ctx context.Context, claims authn.Claims, ruleID string, ip *string) error {
	if s.DB == nil {
		return errors.New("database is unavailable")
	}
	cmd, err := s.DB.Exec(ctx, `DELETE FROM "DbFirewallRule" WHERE id = $1 AND "tenantId" = $2`, ruleID, claims.TenantID)
	if err != nil {
		return fmt.Errorf("delete firewall rule: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return &requestError{status: http.StatusNotFound, message: "Firewall rule not found"}
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "DB_FIREWALL_RULE_DELETE", "DbFirewallRule", ruleID, nil, ip)
	return nil
}
