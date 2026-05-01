package rdgatewayapi

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

func (s Service) insertAuditLog(ctx context.Context, userID, action, targetType, targetID string, details map[string]any, ip *string) error {
	if s.DB == nil || strings.TrimSpace(userID) == "" {
		return nil
	}
	var payload any
	if details != nil {
		payload = details
	}
	_, err := s.DB.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, "targetType", "targetId", details, "ipAddress")
VALUES ($1, $2, $3::"AuditAction", $4::"AuditTargetType", $5, $6, $7)
`, uuid.NewString(), userID, action, targetType, targetID, payload, ip)
	return err
}
