package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	agentsdb "github.com/dnviti/arsenale/backend/internal/agents/dbgen"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db      *pgxpool.Pool
	queries *agentsdb.Queries
}

func NewStore(db *pgxpool.Pool) *Store {
	store := &Store{db: db}
	if db != nil {
		store.queries = agentsdb.New(db)
	}
	return store
}

func (s *Store) Enabled() bool {
	return s != nil && s.db != nil
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

	row, err := s.queries.CreateRun(ctx, agentsdb.CreateRunParams{
		ID:                    uuid.NewString(),
		TenantID:              req.TenantID,
		DefinitionID:          req.DefinitionID,
		Trigger:               req.Trigger,
		Goals:                 goalsJSON,
		RequestedCapabilities: capabilitiesJSON,
		Status:                string(contracts.AgentRunQueued),
		RequiresApproval:      requiresApproval,
	})
	if err != nil {
		return contracts.AgentRun{}, fmt.Errorf("create agent run: %w", err)
	}

	return mapRun(row)
}

func (s *Store) GetRun(ctx context.Context, id string) (contracts.AgentRun, error) {
	if !s.Enabled() {
		return contracts.AgentRun{}, errors.New("agent store is not configured")
	}

	row, err := s.queries.GetRun(ctx, id)
	if err != nil {
		return contracts.AgentRun{}, err
	}
	return mapRun(row)
}

func (s *Store) ListRuns(ctx context.Context, tenantID string) ([]contracts.AgentRun, error) {
	if !s.Enabled() {
		return nil, errors.New("agent store is not configured")
	}
	if strings.TrimSpace(tenantID) == "" {
		return nil, errors.New("tenantId is required")
	}

	rows, err := s.queries.ListRuns(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list agent runs: %w", err)
	}

	runs := make([]contracts.AgentRun, 0, len(rows))
	for _, row := range rows {
		run, err := mapRun(row)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}

	return runs, nil
}

func mapRun(row agentsdb.AgentRun) (contracts.AgentRun, error) {
	run := contracts.AgentRun{
		ID:               row.ID,
		TenantID:         row.TenantID,
		DefinitionID:     row.DefinitionID,
		Trigger:          row.Trigger,
		Status:           contracts.AgentRunStatus(row.Status),
		RequiresApproval: row.RequiresApproval,
		RequestedAt:      row.RequestedAt,
		LastTransitionAt: row.LastTransitionAt,
	}

	if len(row.Goals) > 0 {
		if err := json.Unmarshal(row.Goals, &run.Goals); err != nil {
			return contracts.AgentRun{}, fmt.Errorf("decode goals: %w", err)
		}
	}
	if len(row.RequestedCapabilities) > 0 {
		if err := json.Unmarshal(row.RequestedCapabilities, &run.RequestedCaps); err != nil {
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
