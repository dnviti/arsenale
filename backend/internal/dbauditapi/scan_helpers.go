package dbauditapi

import (
	"database/sql"
	"fmt"
)

func scanFirewallRules(rows pgxRows) ([]firewallRule, error) {
	items := make([]firewallRule, 0)
	for rows.Next() {
		var (
			item        firewallRule
			scope       sql.NullString
			description sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.Name,
			&item.Pattern,
			&item.Action,
			&scope,
			&description,
			&item.Enabled,
			&item.Priority,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan firewall rule: %w", err)
		}
		if scope.Valid {
			item.Scope = &scope.String
		}
		if description.Valid {
			item.Description = &description.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate firewall rules: %w", err)
	}
	return items, nil
}

func scanMaskingPolicies(rows pgxRows) ([]maskingPolicy, error) {
	items := make([]maskingPolicy, 0)
	for rows.Next() {
		var (
			item        maskingPolicy
			scope       sql.NullString
			description sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.Name,
			&item.ColumnPattern,
			&item.Strategy,
			&item.ExemptRoles,
			&scope,
			&description,
			&item.Enabled,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan masking policy: %w", err)
		}
		if scope.Valid {
			item.Scope = &scope.String
		}
		if description.Valid {
			item.Description = &description.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate masking policies: %w", err)
	}
	return items, nil
}

func scanRateLimitPolicies(rows pgxRows) ([]rateLimitPolicy, error) {
	items := make([]rateLimitPolicy, 0)
	for rows.Next() {
		var (
			item      rateLimitPolicy
			queryType sql.NullString
			scope     sql.NullString
		)
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.Name,
			&queryType,
			&item.WindowMS,
			&item.MaxQueries,
			&item.BurstMax,
			&item.ExemptRoles,
			&scope,
			&item.Action,
			&item.Enabled,
			&item.Priority,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan rate limit policy: %w", err)
		}
		if queryType.Valid {
			item.QueryType = &queryType.String
		}
		if scope.Valid {
			item.Scope = &scope.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rate limit policies: %w", err)
	}
	return items, nil
}

type pgxRows interface {
	Next() bool
	Scan(...any) error
	Err() error
}
