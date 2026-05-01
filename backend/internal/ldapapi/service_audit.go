package ldapapi

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s Service) insertAudit(ctx context.Context, action string, userID *string, targetID *string, details map[string]any) error {
	if s.DB == nil {
		return nil
	}
	payload, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details)
VALUES ($1, $2, $3, 'ldap', $4, $5::jsonb)
`, uuid.NewString(), userID, action, targetID, string(payload))
	return err
}
