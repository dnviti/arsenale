package keystrokepolicies

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxPatternLength     = 500
	maxPatternsPerPolicy = 50
)

type Service struct {
	DB *pgxpool.Pool
}

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string {
	return e.message
}

type policyResponse struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenantId"`
	Name          string    `json:"name"`
	Description   *string   `json:"description"`
	Action        string    `json:"action"`
	RegexPatterns []string  `json:"regexPatterns"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type createPayload struct {
	Name          string   `json:"name"`
	Description   *string  `json:"description"`
	Action        string   `json:"action"`
	RegexPatterns []string `json:"regexPatterns"`
	Enabled       *bool    `json:"enabled"`
}

type updatePayload struct {
	Name          *string   `json:"name"`
	Description   **string  `json:"description"`
	Action        *string   `json:"action"`
	RegexPatterns *[]string `json:"regexPatterns"`
	Enabled       *bool     `json:"enabled"`
}

type rowScanner interface {
	Scan(dest ...any) error
}
