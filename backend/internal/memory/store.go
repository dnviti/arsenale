package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	memorydb "github.com/dnviti/arsenale/backend/internal/memory/dbgen"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db      *pgxpool.Pool
	queries *memorydb.Queries
}

func NewStore(db *pgxpool.Pool) *Store {
	store := &Store{db: db}
	if db != nil {
		store.queries = memorydb.New(db)
	}
	return store
}

func (s *Store) Enabled() bool {
	return s != nil && s.db != nil
}

func (s *Store) UpsertNamespace(ctx context.Context, ns contracts.MemoryNamespace) (contracts.MemoryNamespaceRecord, error) {
	if !s.Enabled() {
		return contracts.MemoryNamespaceRecord{}, errors.New("memory store is not configured")
	}

	key := NamespaceKey(ns)

	row, err := s.queries.UpsertNamespace(ctx, memorydb.UpsertNamespaceParams{
		ID:           uuid.NewString(),
		NamespaceKey: key,
		TenantID:     ns.TenantID,
		Scope:        string(ns.Scope),
		PrincipalID:  ns.PrincipalID,
		AgentID:      ns.AgentID,
		RunID:        ns.RunID,
		WorkflowID:   ns.WorkflowID,
		MemoryType:   string(ns.Type),
		Name:         ns.Name,
	})
	if err != nil {
		return contracts.MemoryNamespaceRecord{}, fmt.Errorf("upsert memory namespace %q: %w", key, err)
	}

	return mapNamespaceRecord(row), nil
}

func (s *Store) ListNamespaces(ctx context.Context, tenantID string) ([]contracts.MemoryNamespaceRecord, error) {
	if !s.Enabled() {
		return nil, errors.New("memory store is not configured")
	}
	if strings.TrimSpace(tenantID) == "" {
		return nil, errors.New("tenantId is required")
	}

	rows, err := s.queries.ListNamespaces(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list memory namespaces: %w", err)
	}

	records := make([]contracts.MemoryNamespaceRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, mapNamespaceRecord(row))
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

	row, err := s.queries.CreateItem(ctx, memorydb.CreateItemParams{
		ID:           uuid.NewString(),
		NamespaceKey: record.Key,
		Content:      req.Content,
		Summary:      req.Summary,
		Metadata:     metadataJSON,
	})
	if err != nil {
		return contracts.MemoryItem{}, fmt.Errorf("append memory item: %w", err)
	}

	return mapMemoryItem(row)
}

func (s *Store) ListItems(ctx context.Context, namespaceKey string) ([]contracts.MemoryItem, error) {
	if !s.Enabled() {
		return nil, errors.New("memory store is not configured")
	}
	if strings.TrimSpace(namespaceKey) == "" {
		return nil, errors.New("namespaceKey is required")
	}

	rows, err := s.queries.ListItems(ctx, namespaceKey)
	if err != nil {
		return nil, fmt.Errorf("list memory items: %w", err)
	}

	items := make([]contracts.MemoryItem, 0, len(rows))
	for _, row := range rows {
		item, err := mapMemoryItem(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

func mapNamespaceRecord(row memorydb.MemoryNamespace) contracts.MemoryNamespaceRecord {
	return contracts.MemoryNamespaceRecord{
		ID:  row.ID,
		Key: row.NamespaceKey,
		Namespace: contracts.MemoryNamespace{
			TenantID:    row.TenantID,
			Scope:       contracts.MemoryScope(row.Scope),
			PrincipalID: row.PrincipalID,
			AgentID:     row.AgentID,
			RunID:       row.RunID,
			WorkflowID:  row.WorkflowID,
			Type:        contracts.MemoryType(row.MemoryType),
			Name:        row.Name,
		},
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapMemoryItem(row memorydb.MemoryItem) (contracts.MemoryItem, error) {
	item := contracts.MemoryItem{
		ID:           row.ID,
		NamespaceKey: row.NamespaceKey,
		Content:      row.Content,
		Summary:      row.Summary,
		CreatedAt:    row.CreatedAt,
	}

	if len(row.Metadata) > 0 {
		if err := json.Unmarshal(row.Metadata, &item.Metadata); err != nil {
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
