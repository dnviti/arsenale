package connections

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s Service) lookupActorName(ctx context.Context, userID string) (string, error) {
	var actorName string
	if err := s.DB.QueryRow(ctx, `SELECT COALESCE(NULLIF(username, ''), email, 'Someone') FROM "User" WHERE id = $1`, userID).Scan(&actorName); err != nil {
		return "", fmt.Errorf("load actor identity: %w", err)
	}
	return actorName, nil
}

func (s Service) insertNotification(ctx context.Context, userID, notificationType, message, relatedID string) error {
	_, err := s.DB.Exec(ctx, `
INSERT INTO "Notification" (id, "userId", type, message, read, "relatedId", "createdAt")
VALUES ($1, $2, $3::"NotificationType", $4, false, NULLIF($5, ''), NOW())
`, uuid.NewString(), userID, notificationType, message, relatedID)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}
