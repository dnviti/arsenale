package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Enabled() bool {
	return s != nil && s.db != nil
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}

	_, err := s.db.Exec(ctx, `
CREATE TABLE IF NOT EXISTS agent_runs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  definition_id TEXT NOT NULL,
  trigger TEXT NOT NULL DEFAULT '',
  goals JSONB NOT NULL DEFAULT '[]'::jsonb,
  requested_capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
  status TEXT NOT NULL,
  requires_approval BOOLEAN NOT NULL DEFAULT false,
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_transition_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_tenant_requested_at
  ON agent_runs (tenant_id, requested_at DESC);
`)
	if err != nil {
		return fmt.Errorf("ensure agent run schema: %w", err)
	}

	return nil
}

func (s *Store) CreateRun(ctx context.Context, req contracts.AgentRunRequest) (contracts.AgentRun, error) {
	if !s.Enabled() {
		return contracts.AgentRun{}, errors.New("agent store is not configured")
	}
	if err := ValidateRunRequest(req); err != nil {
		return contracts.AgentRun{}, err
	}

	goalsJSON, err := json.Marshal(req.Goals)
	if err != nil {
		return contracts.AgentRun{}, fmt.Errorf("marshal goals: %w", err)
	}
	capabilitiesJSON, err := json.Marshal(req.RequestedCapabilities)
	if err != nil {
		return contracts.AgentRun{}, fmt.Errorf("marshal requested capabilities: %w", err)
	}

	requiresApproval := false
	for _, capabilityID := range req.RequestedCapabilities {
		capability, lookupErr := lookupCapability(capabilityID)
		if lookupErr != nil {
			return contracts.AgentRun{}, lookupErr
		}
		if capability.RequiresApproval {
			requiresApproval = true
		}
	}

	row := s.db.QueryRow(ctx, `
INSERT INTO agent_runs (
  id, tenant_id, definition_id, trigger, goals, requested_capabilities, status, requires_approval
) VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7, $8)
RETURNING id, tenant_id, definition_id, trigger, goals, requested_capabilities, status, requires_approval, requested_at, last_transition_at
`, uuid.NewString(), req.TenantID, req.DefinitionID, req.Trigger, string(goalsJSON), string(capabilitiesJSON), contracts.AgentRunQueued, requiresApproval)

	run, err := scanRun(row)
	if err != nil {
		return contracts.AgentRun{}, fmt.Errorf("create agent run: %w", err)
	}

	return run, nil
}

func (s *Store) GetRun(ctx context.Context, id string) (contracts.AgentRun, error) {
	if !s.Enabled() {
		return contracts.AgentRun{}, errors.New("agent store is not configured")
	}

	row := s.db.QueryRow(ctx, `
SELECT id, tenant_id, definition_id, trigger, goals, requested_capabilities, status, requires_approval, requested_at, last_transition_at
FROM agent_runs
WHERE id = $1
`, id)

	run, err := scanRun(row)
	if err != nil {
		return contracts.AgentRun{}, err
	}
	return run, nil
}

func (s *Store) ListRuns(ctx context.Context, tenantID string) ([]contracts.AgentRun, error) {
	if !s.Enabled() {
		return nil, errors.New("agent store is not configured")
	}
	if strings.TrimSpace(tenantID) == "" {
		return nil, errors.New("tenantId is required")
	}

	rows, err := s.db.Query(ctx, `
SELECT id, tenant_id, definition_id, trigger, goals, requested_capabilities, status, requires_approval, requested_at, last_transition_at
FROM agent_runs
WHERE tenant_id = $1
ORDER BY requested_at DESC
`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list agent runs: %w", err)
	}
	defer rows.Close()

	var runs []contracts.AgentRun
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent runs: %w", err)
	}

	return runs, nil
}

type runScanner interface {
	Scan(dest ...any) error
}

func scanRun(scanner runScanner) (contracts.AgentRun, error) {
	var (
		run              contracts.AgentRun
		goalsJSON        []byte
		capabilitiesJSON []byte
	)

	if err := scanner.Scan(
		&run.ID,
		&run.TenantID,
		&run.DefinitionID,
		&run.Trigger,
		&goalsJSON,
		&capabilitiesJSON,
		&run.Status,
		&run.RequiresApproval,
		&run.RequestedAt,
		&run.LastTransitionAt,
	); err != nil {
		return contracts.AgentRun{}, err
	}
	if len(goalsJSON) > 0 {
		if err := json.Unmarshal(goalsJSON, &run.Goals); err != nil {
			return contracts.AgentRun{}, fmt.Errorf("decode goals: %w", err)
		}
	}
	if len(capabilitiesJSON) > 0 {
		if err := json.Unmarshal(capabilitiesJSON, &run.RequestedCaps); err != nil {
			return contracts.AgentRun{}, fmt.Errorf("decode requested capabilities: %w", err)
		}
	}
	if run.Goals == nil {
		run.Goals = []string{}
	}
	if run.RequestedCaps == nil {
		run.RequestedCaps = []string{}
	}

	return run, nil
}

func lookupCapability(id string) (contracts.CapabilityDefinition, error) {
	for _, capability := range catalog.Capabilities() {
		if capability.ID == id {
			return capability, nil
		}
	}
	return contracts.CapabilityDefinition{}, fmt.Errorf("unknown requested capability %q", id)
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
