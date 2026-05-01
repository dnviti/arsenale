package sessionrecording

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TenantRecordingEnabled(ctx context.Context, db *pgxpool.Pool, tenantID string) (bool, error) {
	if db == nil || strings.TrimSpace(tenantID) == "" {
		return true, nil
	}

	var enabled bool
	if err := db.QueryRow(ctx, `SELECT "recordingEnabled" FROM "Tenant" WHERE id = $1`, tenantID).Scan(&enabled); err != nil {
		return false, fmt.Errorf("load tenant recording policy: %w", err)
	}
	return enabled, nil
}
