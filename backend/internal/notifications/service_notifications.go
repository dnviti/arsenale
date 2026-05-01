package notifications

import (
	"context"
	"database/sql"
	"fmt"
)

func (s Service) ListNotifications(ctx context.Context, userID string, limit, offset int) (notificationsResponse, error) {
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	filterClause := ""
	if !s.Features.RecordingsEnabled {
		filterClause = ` AND type <> 'RECORDING_READY'::"NotificationType"`
	}

	rows, err := s.DB.Query(ctx, `
SELECT id, type::text, message, read, "relatedId", "createdAt"
FROM "Notification"
WHERE "userId" = $1`+filterClause+`
ORDER BY "createdAt" DESC
OFFSET $2 LIMIT $3
`, userID, offset, limit)
	if err != nil {
		return notificationsResponse{}, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	result := make([]notificationEntry, 0)
	for rows.Next() {
		var item notificationEntry
		var relatedID sql.NullString
		if err := rows.Scan(&item.ID, &item.Type, &item.Message, &item.Read, &relatedID, &item.CreatedAt); err != nil {
			return notificationsResponse{}, fmt.Errorf("scan notification: %w", err)
		}
		if relatedID.Valid {
			item.RelatedID = &relatedID.String
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return notificationsResponse{}, fmt.Errorf("iterate notifications: %w", err)
	}

	var total int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM "Notification" WHERE "userId" = $1`+filterClause, userID).Scan(&total); err != nil {
		return notificationsResponse{}, fmt.Errorf("count notifications: %w", err)
	}
	var unreadCount int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*) FROM "Notification" WHERE "userId" = $1 AND read = false`+filterClause, userID).Scan(&unreadCount); err != nil {
		return notificationsResponse{}, fmt.Errorf("count unread notifications: %w", err)
	}

	return notificationsResponse{Data: result, Total: total, UnreadCount: unreadCount}, nil
}

func (s Service) MarkRead(ctx context.Context, userID, notificationID string) error {
	_, err := s.DB.Exec(ctx, `UPDATE "Notification" SET read = true WHERE id = $1 AND "userId" = $2`, notificationID, userID)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}

func (s Service) MarkAllRead(ctx context.Context, userID string) error {
	_, err := s.DB.Exec(ctx, `UPDATE "Notification" SET read = true WHERE "userId" = $1 AND read = false`, userID)
	if err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}

func (s Service) DeleteNotification(ctx context.Context, userID, notificationID string) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM "Notification" WHERE id = $1 AND "userId" = $2`, notificationID, userID)
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}
	return nil
}
