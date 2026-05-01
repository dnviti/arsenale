package syncprofiles

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var roleHierarchy = map[string]int{
	"GUEST":      1,
	"AUDITOR":    2,
	"CONSULTANT": 3,
	"MEMBER":     4,
	"OPERATOR":   5,
	"ADMIN":      6,
	"OWNER":      7,
}

type Service struct {
	DB                  *pgxpool.Pool
	ServerEncryptionKey []byte
	Scheduler           *schedulerState
}

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string {
	return e.message
}

type encryptedField struct {
	Ciphertext string
	IV         string
	Tag        string
}

type syncProfileConfig struct {
	URL              string            `json:"url"`
	Filters          map[string]string `json:"filters"`
	PlatformMapping  map[string]string `json:"platformMapping"`
	DefaultProtocol  string            `json:"defaultProtocol"`
	DefaultPort      map[string]int    `json:"defaultPort"`
	ConflictStrategy string            `json:"conflictStrategy"`
}

type syncProfileResponse struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	TenantID        string            `json:"tenantId"`
	Provider        string            `json:"provider"`
	Config          syncProfileConfig `json:"config"`
	CronExpression  *string           `json:"cronExpression"`
	Enabled         bool              `json:"enabled"`
	TeamID          *string           `json:"teamId"`
	LastSyncAt      *time.Time        `json:"lastSyncAt"`
	LastSyncStatus  *string           `json:"lastSyncStatus"`
	LastSyncDetails json.RawMessage   `json:"lastSyncDetails"`
	CreatedByID     string            `json:"createdById"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	HasAPIToken     bool              `json:"hasApiToken"`
}

type syncLogEntry struct {
	ID            string          `json:"id"`
	SyncProfileID string          `json:"syncProfileId"`
	Status        string          `json:"status"`
	StartedAt     time.Time       `json:"startedAt"`
	CompletedAt   *time.Time      `json:"completedAt"`
	Details       json.RawMessage `json:"details"`
	TriggeredBy   string          `json:"triggeredBy"`
}

type syncLogsResponse struct {
	Logs  []syncLogEntry `json:"logs"`
	Total int            `json:"total"`
	Page  int            `json:"page"`
	Limit int            `json:"limit"`
}

type createPayload struct {
	Name             string            `json:"name"`
	Provider         string            `json:"provider"`
	URL              string            `json:"url"`
	APIToken         string            `json:"apiToken"`
	Filters          map[string]string `json:"filters"`
	PlatformMapping  map[string]string `json:"platformMapping"`
	DefaultProtocol  *string           `json:"defaultProtocol"`
	DefaultPort      map[string]int    `json:"defaultPort"`
	ConflictStrategy *string           `json:"conflictStrategy"`
	CronExpression   *string           `json:"cronExpression"`
	TeamID           *string           `json:"teamId"`
}

type optionalNullableString struct {
	Present bool
	Value   *string
}

// UnmarshalJSON preserves the difference between an omitted PATCH field and an explicit null.
func (o *optionalNullableString) UnmarshalJSON(data []byte) error {
	o.Present = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

type updatePayload struct {
	Name             *string                `json:"name"`
	URL              *string                `json:"url"`
	APIToken         *string                `json:"apiToken"`
	Filters          *map[string]string     `json:"filters"`
	PlatformMapping  *map[string]string     `json:"platformMapping"`
	DefaultProtocol  *string                `json:"defaultProtocol"`
	DefaultPort      *map[string]int        `json:"defaultPort"`
	ConflictStrategy *string                `json:"conflictStrategy"`
	CronExpression   optionalNullableString `json:"cronExpression"`
	Enabled          *bool                  `json:"enabled"`
	TeamID           optionalNullableString `json:"teamId"`
}
