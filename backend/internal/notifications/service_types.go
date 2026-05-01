package notifications

import (
	"time"

	"github.com/dnviti/arsenale/backend/internal/runtimefeatures"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	DB       *pgxpool.Pool
	Features runtimefeatures.Manifest
}

type notificationEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	Read      bool      `json:"read"`
	RelatedID *string   `json:"relatedId"`
	CreatedAt time.Time `json:"createdAt"`
}

type notificationsResponse struct {
	Data        []notificationEntry `json:"data"`
	Total       int                 `json:"total"`
	UnreadCount int                 `json:"unreadCount"`
}

type notificationPreference struct {
	Type  string `json:"type"`
	InApp bool   `json:"inApp"`
	Email bool   `json:"email"`
}

type bulkPreferencesPayload struct {
	Preferences []preferenceUpdatePayload `json:"preferences"`
}

type preferenceUpdatePayload struct {
	Type  string `json:"type"`
	InApp *bool  `json:"inApp"`
	Email *bool  `json:"email"`
}

var allTypes = []string{
	"CONNECTION_SHARED",
	"SHARE_PERMISSION_UPDATED",
	"SHARE_REVOKED",
	"SECRET_SHARED",
	"SECRET_SHARE_REVOKED",
	"SECRET_EXPIRING",
	"SECRET_EXPIRED",
	"TENANT_INVITATION",
	"RECORDING_READY",
	"IMPOSSIBLE_TRAVEL_DETECTED",
	"SECRET_CHECKOUT_REQUESTED",
	"SECRET_CHECKOUT_APPROVED",
	"SECRET_CHECKOUT_DENIED",
	"SECRET_CHECKOUT_EXPIRED",
	"LATERAL_MOVEMENT_ALERT",
	"SESSION_TERMINATED_POLICY_VIOLATION",
	"TENANT_VAULT_KEY_RECEIVED",
}

var validTypeSet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(allTypes))
	for _, t := range allTypes {
		m[t] = struct{}{}
	}
	return m
}()

var emailDefaultTrue = map[string]struct{}{
	"IMPOSSIBLE_TRAVEL_DETECTED":          {},
	"LATERAL_MOVEMENT_ALERT":              {},
	"SECRET_EXPIRING":                     {},
	"SESSION_TERMINATED_POLICY_VIOLATION": {},
}
