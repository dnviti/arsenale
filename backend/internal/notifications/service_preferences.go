package notifications

import (
	"context"
	"fmt"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/runtimefeatures"
	"github.com/google/uuid"
)

func (s Service) GetPreferences(ctx context.Context, userID string) ([]notificationPreference, error) {
	rows, err := s.DB.Query(ctx, `
SELECT type::text, "inApp", email
FROM "NotificationPreference"
WHERE "userId" = $1
`, userID)
	if err != nil {
		return nil, fmt.Errorf("list notification preferences: %w", err)
	}
	defer rows.Close()

	availableTypes := availableNotificationTypes(s.Features)
	stored := make(map[string]notificationPreference, len(availableTypes))
	for rows.Next() {
		var pref notificationPreference
		if err := rows.Scan(&pref.Type, &pref.InApp, &pref.Email); err != nil {
			return nil, fmt.Errorf("scan notification preference: %w", err)
		}
		stored[pref.Type] = pref
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification preferences: %w", err)
	}

	result := make([]notificationPreference, 0, len(availableTypes))
	for _, t := range availableTypes {
		if pref, ok := stored[t]; ok {
			result = append(result, pref)
			continue
		}
		result = append(result, defaultPreference(t))
	}
	return result, nil
}

func (s Service) UpsertPreference(ctx context.Context, userID, prefType string, inApp, email *bool) (notificationPreference, error) {
	normalized := strings.ToUpper(strings.TrimSpace(prefType))
	if !notificationTypeEnabled(s.Features, normalized) {
		return notificationPreference{}, fmt.Errorf("notification type is unavailable on this platform")
	}

	defaults := defaultPreference(normalized)
	inAppValue := defaults.InApp
	emailValue := defaults.Email
	if inApp != nil {
		inAppValue = *inApp
	}
	if email != nil {
		emailValue = *email
	}

	var result notificationPreference
	if err := s.DB.QueryRow(ctx, `
INSERT INTO "NotificationPreference" (id, "userId", type, "inApp", email, "createdAt", "updatedAt")
VALUES ($1, $2, $3::"NotificationType", $4, $5, NOW(), NOW())
ON CONFLICT ("userId", type)
DO UPDATE SET
  "inApp" = EXCLUDED."inApp",
  email = EXCLUDED.email,
  "updatedAt" = NOW()
RETURNING type::text, "inApp", email
`, uuid.NewString(), userID, normalized, inAppValue, emailValue).Scan(&result.Type, &result.InApp, &result.Email); err != nil {
		return notificationPreference{}, fmt.Errorf("upsert notification preference: %w", err)
	}
	return result, nil
}

func (s Service) BulkUpsertPreferences(ctx context.Context, userID string, prefs []preferenceUpdatePayload) ([]notificationPreference, error) {
	if len(prefs) == 0 {
		return []notificationPreference{}, nil
	}
	result := make([]notificationPreference, 0, len(prefs))
	for _, pref := range prefs {
		item, err := s.UpsertPreference(ctx, userID, pref.Type, pref.InApp, pref.Email)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func defaultPreference(prefType string) notificationPreference {
	_, emailEnabled := emailDefaultTrue[prefType]
	return notificationPreference{
		Type:  prefType,
		InApp: true,
		Email: emailEnabled,
	}
}

func availableNotificationTypes(features runtimefeatures.Manifest) []string {
	types := make([]string, 0, len(allTypes))
	for _, prefType := range allTypes {
		if notificationTypeEnabled(features, prefType) {
			types = append(types, prefType)
		}
	}
	return types
}

func notificationTypeEnabled(features runtimefeatures.Manifest, prefType string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(prefType))
	if _, ok := validTypeSet[normalized]; !ok {
		return false
	}
	if normalized == "RECORDING_READY" && !features.RecordingsEnabled {
		return false
	}
	return true
}
