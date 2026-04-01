package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
CREATE TABLE IF NOT EXISTS orchestrator_connections (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  kind TEXT NOT NULL,
  scope TEXT NOT NULL,
  endpoint TEXT NOT NULL,
  namespace TEXT NOT NULL DEFAULT '',
  labels JSONB NOT NULL DEFAULT '{}'::jsonb,
  capabilities JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
)
`)
	if err != nil {
		return fmt.Errorf("ensure orchestrator_connections schema: %w", err)
	}

	return nil
}

func (s *Store) ListConnections(ctx context.Context) ([]contracts.OrchestratorConnection, error) {
	if !s.Enabled() {
		return nil, errors.New("orchestrator store is not configured")
	}

	rows, err := s.db.Query(ctx, `
SELECT id, name, kind, scope, endpoint, namespace, labels, capabilities
FROM orchestrator_connections
ORDER BY name ASC
`)
	if err != nil {
		return nil, fmt.Errorf("list orchestrator connections: %w", err)
	}
	defer rows.Close()

	connections := make([]contracts.OrchestratorConnection, 0)
	for rows.Next() {
		connection, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		connections = append(connections, connection)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate orchestrator connections: %w", err)
	}

	return connections, nil
}

func (s *Store) GetConnection(ctx context.Context, name string) (contracts.OrchestratorConnection, error) {
	if !s.Enabled() {
		return contracts.OrchestratorConnection{}, errors.New("orchestrator store is not configured")
	}

	row := s.db.QueryRow(ctx, `
SELECT id, name, kind, scope, endpoint, namespace, labels, capabilities
FROM orchestrator_connections
WHERE name = $1
`, name)

	connection, err := scanConnection(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return contracts.OrchestratorConnection{}, err
		}
		return contracts.OrchestratorConnection{}, fmt.Errorf("get orchestrator connection %q: %w", name, err)
	}

	return connection, nil
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

	row := s.db.QueryRow(ctx, `
INSERT INTO orchestrator_connections (id, name, kind, scope, endpoint, namespace, labels, capabilities)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb)
ON CONFLICT (name) DO UPDATE SET
  kind = EXCLUDED.kind,
  scope = EXCLUDED.scope,
  endpoint = EXCLUDED.endpoint,
  namespace = EXCLUDED.namespace,
  labels = EXCLUDED.labels,
  capabilities = EXCLUDED.capabilities,
  updated_at = now()
RETURNING id, name, kind, scope, endpoint, namespace, labels, capabilities
`, conn.ID, conn.Name, conn.Kind, conn.Scope, conn.Endpoint, conn.Namespace, string(labelsJSON), string(capabilitiesJSON))

	connection, err := scanConnection(row)
	if err != nil {
		return contracts.OrchestratorConnection{}, fmt.Errorf("upsert orchestrator connection %q: %w", conn.Name, err)
	}

	return connection, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanConnection(scanner rowScanner) (contracts.OrchestratorConnection, error) {
	var (
		connection       contracts.OrchestratorConnection
		labelsJSON       []byte
		capabilitiesJSON []byte
	)

	err := scanner.Scan(
		&connection.ID,
		&connection.Name,
		&connection.Kind,
		&connection.Scope,
		&connection.Endpoint,
		&connection.Namespace,
		&labelsJSON,
		&capabilitiesJSON,
	)
	if err != nil {
		return contracts.OrchestratorConnection{}, err
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
