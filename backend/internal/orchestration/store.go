package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	orchestrationdb "github.com/dnviti/arsenale/backend/internal/orchestration/dbgen"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db      *pgxpool.Pool
	queries *orchestrationdb.Queries
}

func NewStore(db *pgxpool.Pool) *Store {
	store := &Store{db: db}
	if db != nil {
		store.queries = orchestrationdb.New(db)
	}
	return store
}

func (s *Store) Enabled() bool {
	return s != nil && s.db != nil
}

func (s *Store) ListConnections(ctx context.Context) ([]contracts.OrchestratorConnection, error) {
	if !s.Enabled() {
		return nil, errors.New("orchestrator store is not configured")
	}

	rows, err := s.queries.ListConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("list orchestrator connections: %w", err)
	}

	connections := make([]contracts.OrchestratorConnection, 0, len(rows))
	for _, row := range rows {
		connection, err := decodeConnection(row.ID, row.Name, row.Kind, row.Scope, row.Endpoint, row.Namespace, row.Labels, row.Capabilities)
		if err != nil {
			return nil, err
		}
		connections = append(connections, connection)
	}

	return connections, nil
}

func (s *Store) GetConnection(ctx context.Context, name string) (contracts.OrchestratorConnection, error) {
	if !s.Enabled() {
		return contracts.OrchestratorConnection{}, errors.New("orchestrator store is not configured")
	}

	row, err := s.queries.GetConnection(ctx, name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return contracts.OrchestratorConnection{}, err
		}
		return contracts.OrchestratorConnection{}, fmt.Errorf("get orchestrator connection %q: %w", name, err)
	}

	return decodeConnection(row.ID, row.Name, row.Kind, row.Scope, row.Endpoint, row.Namespace, row.Labels, row.Capabilities)
}

func (s *Store) UpsertConnection(ctx context.Context, conn contracts.OrchestratorConnection) (contracts.OrchestratorConnection, error) {
	if !s.Enabled() {
		return contracts.OrchestratorConnection{}, errors.New("orchestrator store is not configured")
	}
	if conn.ID == "" {
		conn.ID = uuid.NewString()
	}

	labelsJSON, err := json.Marshal(conn.Labels)
	if err != nil {
		return contracts.OrchestratorConnection{}, fmt.Errorf("marshal connection labels: %w", err)
	}
	capabilitiesJSON, err := json.Marshal(conn.Capabilities)
	if err != nil {
		return contracts.OrchestratorConnection{}, fmt.Errorf("marshal connection capabilities: %w", err)
	}

	row, err := s.queries.UpsertConnection(ctx, orchestrationdb.UpsertConnectionParams{
		ID:           conn.ID,
		Name:         conn.Name,
		Kind:         string(conn.Kind),
		Scope:        string(conn.Scope),
		Endpoint:     conn.Endpoint,
		Namespace:    conn.Namespace,
		Labels:       labelsJSON,
		Capabilities: capabilitiesJSON,
	})
	if err != nil {
		return contracts.OrchestratorConnection{}, fmt.Errorf("upsert orchestrator connection %q: %w", conn.Name, err)
	}

	return decodeConnection(row.ID, row.Name, row.Kind, row.Scope, row.Endpoint, row.Namespace, row.Labels, row.Capabilities)
}

func decodeConnection(id, name, kind, scope, endpoint, namespace string, labelsJSON, capabilitiesJSON []byte) (contracts.OrchestratorConnection, error) {
	connection := contracts.OrchestratorConnection{
		ID:        id,
		Name:      name,
		Kind:      contracts.OrchestratorConnectionKind(kind),
		Scope:     contracts.OrchestratorScope(scope),
		Endpoint:  endpoint,
		Namespace: namespace,
	}

	if len(labelsJSON) > 0 {
		if err := json.Unmarshal(labelsJSON, &connection.Labels); err != nil {
			return contracts.OrchestratorConnection{}, fmt.Errorf("decode labels: %w", err)
		}
	}
	if len(capabilitiesJSON) > 0 {
		if err := json.Unmarshal(capabilitiesJSON, &connection.Capabilities); err != nil {
			return contracts.OrchestratorConnection{}, fmt.Errorf("decode capabilities: %w", err)
		}
	}
	if connection.Labels == nil {
		connection.Labels = map[string]string{}
	}
	if connection.Capabilities == nil {
		connection.Capabilities = []string{}
	}

	return connection, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
