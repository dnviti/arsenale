package sessions

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionClosed   = errors.New("session already closed")
)

type RoutingDecision struct {
	Strategy             string `json:"strategy,omitempty"`
	CandidateCount       int    `json:"candidateCount,omitempty"`
	SelectedSessionCount int    `json:"selectedSessionCount,omitempty"`
}

type StartSessionParams struct {
	UserID          string
	ConnectionID    string
	GatewayID       string
	InstanceID      string
	Protocol        string
	SocketID        string
	GuacTokenHash   string
	IPAddress       string
	Metadata        map[string]any
	RoutingDecision *RoutingDecision
	RecordingID     string
}

type sessionRecord struct {
	ID           string
	UserID       string
	ConnectionID string
	Protocol     string
	GatewayID    *string
	GatewayName  *string
	InstanceID   *string
	IPAddress    *string
	StartedAt    time.Time
	Status       string
}

type SessionState struct {
	Record   sessionRecord
	Metadata map[string]any
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

type auditLogParams struct {
	UserID     string
	Action     string
	TargetType string
	TargetID   string
	Details    []byte
	IPAddress  *string
	GatewayID  *string
}
