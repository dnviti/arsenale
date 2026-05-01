package tenants

import (
	"context"
	"database/sql"
	"fmt"
)

func (s Service) GetIPAllowlist(ctx context.Context, tenantID string) (ipAllowlistResponse, error) {
	row := s.DB.QueryRow(ctx, `
SELECT "ipAllowlistEnabled", "ipAllowlistMode", "ipAllowlistEntries"
FROM "Tenant"
WHERE id = $1
`, tenantID)

	var (
		result  ipAllowlistResponse
		mode    sql.NullString
		entries []string
	)
	if err := row.Scan(&result.Enabled, &mode, &entries); err != nil {
		return ipAllowlistResponse{}, fmt.Errorf("get tenant ip allowlist: %w", err)
	}
	if mode.Valid && mode.String != "" {
		result.Mode = mode.String
	} else {
		result.Mode = "flag"
	}
	result.Entries = entries
	return result, nil
}

func (s Service) UpdateIPAllowlist(ctx context.Context, tenantID string, payload ipAllowlistResponse) (ipAllowlistResponse, error) {
	row := s.DB.QueryRow(ctx, `
UPDATE "Tenant"
SET
	"ipAllowlistEnabled" = $2,
	"ipAllowlistMode" = $3,
	"ipAllowlistEntries" = $4
WHERE id = $1
RETURNING "ipAllowlistEnabled", "ipAllowlistMode", "ipAllowlistEntries"
`, tenantID, payload.Enabled, payload.Mode, payload.Entries)

	var (
		result  ipAllowlistResponse
		mode    sql.NullString
		entries []string
	)
	if err := row.Scan(&result.Enabled, &mode, &entries); err != nil {
		return ipAllowlistResponse{}, fmt.Errorf("update tenant ip allowlist: %w", err)
	}
	if mode.Valid && mode.String != "" {
		result.Mode = mode.String
	} else {
		result.Mode = "flag"
	}
	result.Entries = entries
	return result, nil
}
