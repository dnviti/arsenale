package accesspolicies

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	DB *pgxpool.Pool
}

type policyResponse struct {
	ID                   string    `json:"id"`
	TargetType           string    `json:"targetType"`
	TargetID             string    `json:"targetId"`
	AllowedTimeWindows   *string   `json:"allowedTimeWindows"`
	RequireTrustedDevice bool      `json:"requireTrustedDevice"`
	RequireMFAStepUp     bool      `json:"requireMfaStepUp"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string {
	return e.message
}
