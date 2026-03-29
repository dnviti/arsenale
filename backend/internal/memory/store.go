package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
CREATE TABLE IF NOT EXISTS memory_namespaces (
  id TEXT PRIMARY KEY,
  namespace_key TEXT NOT NULL UNIQUE,
  tenant_id TEXT NOT NULL,
  scope TEXT NOT NULL,
  principal_id TEXT NOT NULL DEFAULT '',
  agent_id TEXT NOT NULL DEFAULT '',
  run_id TEXT NOT NULL DEFAULT '',
  workflow_id TEXT NOT NULL DEFAULT '',
  memory_type TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS memory_items (
  id TEXT PRIMARY KEY,
  namespace_key TEXT NOT NULL REFERENCES memory_namespaces(namespace_key) ON DELETE CASCADE,
  content TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_memory_namespaces_tenant_scope
  ON memory_namespaces (tenant_id, scope, memory_type);

CREATE INDEX IF NOT EXISTS idx_memory_items_namespace_created_at
  ON memory_items (namespace_key, created_at DESC);
`)
	if err != nil {
		return fmt.Errorf("ensure memory schema: %w", err)
	}

	return nil
}

func (s *Store) UpsertNamespace(ctx context.Context, ns contracts.MemoryNamespace) (contracts.MemoryNamespaceRecord, error) {
	if !s.Enabled() {
		return contracts.MemoryNamespaceRecord{}, errors.New("memory store is not configured")
	}

	key := NamespaceKey(ns)
	id := uuid.NewString()

	row := s.db.QueryRow(ctx, `
INSERT INTO memory_namespaces (
  id, namespace_key, tenant_id, scope, principal_id, agent_id, run_id, workflow_id, memory_type, name
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (namespace_key) DO UPDATE SET
  updated_at = now()
RETURNING id, namespace_key, tenant_id, scope, principal_id, agent_id, run_id, workflow_id, memory_type, name, created_at, updated_at
`, id, key, ns.TenantID, ns.Scope, ns.PrincipalID, ns.AgentID, ns.RunID, ns.WorkflowID, ns.Type, ns.Name)

	record, err := scanNamespaceRecord(row)
	if err != nil {
		return contracts.MemoryNamespaceRecord{}, fmt.Errorf("upsert memory namespace %q: %w", key, err)
	}

	return record, nil
}

func (s *Store) ListNamespaces(ctx context.Context, tenantID string) ([]contracts.MemoryNamespaceRecord, error) {
	if !s.Enabled() {
		return nil, errors.New("memory store is not configured")
	}
	if strings.TrimSpace(tenantID) == "" {
		return nil, errors.New("tenantId is required")
	}

	rows, err := s.db.Query(ctx, `
SELECT id, namespace_key, tenant_id, scope, principal_id, agent_id, run_id, workflow_id, memory_type, name, created_at, updated_at
FROM memory_namespaces
WHERE tenant_id = $1
ORDER BY scope ASC, memory_type ASC, name ASC
`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list memory namespaces: %w", err)
	}
	defer rows.Close()

	var records []contracts.MemoryNamespaceRecord
	for rows.Next() {
		record, err := scanNamespaceRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory namespaces: %w", err)
	}

	return records, nil
}

func (s *Store) AppendItem(ctx context.Context, req contracts.MemoryWriteRequest) (contracts.MemoryItem, error) {
	if !s.Enabled() {
		return contracts.MemoryItem{}, errors.New("memory store is not configured")
	}
	if strings.TrimSpace(req.Content) == "" {
		return contracts.MemoryItem{}, errors.New("content is required")
	}

	record, err := s.UpsertNamespace(ctx, req.Namespace)
	if err != nil {
		return contracts.MemoryItem{}, err
	}

	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		return contracts.MemoryItem{}, fmt.Errorf("marshal memory metadata: %w", err)
	}

	row := s.db.QueryRow(ctx, `
INSERT INTO memory_items (id, namespace_key, content, summary, metadata)
VALUES ($1, $2, $3, $4, $5::jsonb)
RETURNING id, namespace_key, content, summary, metadata, created_at
`, uuid.NewString(), record.Key, req.Content, req.Summary, string(metadataJSON))

	item, err := scanMemoryItem(row)
	if err != nil {
		return contracts.MemoryItem{}, fmt.Errorf("append memory item: %w", err)
	}

	return item, nil
}

func (s *Store) ListItems(ctx context.Context, namespaceKey string) ([]contracts.MemoryItem, error) {
	if !s.Enabled() {
		return nil, errors.New("memory store is not configured")
	}
	if strings.TrimSpace(namespaceKey) == "" {
		return nil, errors.New("namespaceKey is required")
	}

	rows, err := s.db.Query(ctx, `
SELECT id, namespace_key, content, summary, metadata, created_at
FROM memory_items
WHERE namespace_key = $1
ORDER BY created_at ASC
`, namespaceKey)
	if err != nil {
		return nil, fmt.Errorf("list memory items: %w", err)
	}
	defer rows.Close()

	var items []contracts.MemoryItem
	for rows.Next() {
		item, err := scanMemoryItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory items: %w", err)
	}

	return items, nil
}

type namespaceScanner interface {
	Scan(dest ...any) error
}

func scanNamespaceRecord(scanner namespaceScanner) (contracts.MemoryNamespaceRecord, error) {
	var record contracts.MemoryNamespaceRecord
	if err := scanner.Scan(
		&record.ID,
		&record.Key,
		&record.Namespace.TenantID,
		&record.Namespace.Scope,
		&record.Namespace.PrincipalID,
		&record.Namespace.AgentID,
		&record.Namespace.RunID,
		&record.Namespace.WorkflowID,
		&record.Namespace.Type,
		&record.Namespace.Name,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return contracts.MemoryNamespaceRecord{}, err
	}

	return record, nil
}

type itemScanner interface {
	Scan(dest ...any) error
}

func scanMemoryItem(scanner itemScanner) (contracts.MemoryItem, error) {
	var (
		item         contracts.MemoryItem
		metadataJSON []byte
	)

	if err := scanner.Scan(
		&item.ID,
		&item.NamespaceKey,
		&item.Content,
		&item.Summary,
		&metadataJSON,
		&item.CreatedAt,
	); err != nil {
		return contracts.MemoryItem{}, err
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &item.Metadata); err != nil {
			return contracts.MemoryItem{}, fmt.Errorf("decode memory metadata: %w", err)
		}
	}
	if item.Metadata == nil {
		item.Metadata = map[string]string{}
	}

	return item, nil
}

func MustGetNamespaceKey(record contracts.MemoryNamespaceRecord) string {
	return record.Key
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
