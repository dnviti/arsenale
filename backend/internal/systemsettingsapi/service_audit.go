package systemsettingsapi

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func insertAuditLog(ctx context.Context, tx pgx.Tx, userID, key string, value any) error {
	payload, err := json.Marshal(map[string]any{
		"key":   key,
		"value": value,
	})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details)
VALUES ($1, $2, 'APP_CONFIG_UPDATE', 'system_setting', $3, $4::jsonb)
`, uuid.NewString(), userID, key, payload)
	return err
}
