package keystrokepolicies

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (s Service) insertAuditLog(ctx context.Context, userID, action, targetID string, details map[string]any) error {
	if s.DB == nil || userID == "" {
		return nil
	}
	payload := "{}"
	if details != nil {
		encoded, err := json.Marshal(details)
		if err != nil {
			return err
		}
		payload = string(encoded)
	}
	_, err := s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)
`, uuid.NewString(), userID, action, "KeystrokePolicy", targetID, payload)
	return err
}
