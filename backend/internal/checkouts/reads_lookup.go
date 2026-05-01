package checkouts

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s Service) Get(ctx context.Context, checkoutID, userID string) (checkoutEntry, error) {
	if s.DB == nil {
		return checkoutEntry{}, errors.New("database is unavailable")
	}

	entry, err := s.loadByID(ctx, checkoutID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return checkoutEntry{}, &requestError{status: 404, message: "Checkout request not found"}
		}
		return checkoutEntry{}, err
	}

	if entry.RequesterID != userID && (entry.ApproverID == nil || *entry.ApproverID != userID) {
		allowed, err := s.userCanApproveResource(ctx, userID, entry.SecretID, entry.ConnectionID)
		if err != nil {
			return checkoutEntry{}, err
		}
		if !allowed {
			return checkoutEntry{}, &requestError{status: 403, message: "You are not authorized to view this checkout request"}
		}
	}
	return entry, nil
}

func (s Service) loadByID(ctx context.Context, checkoutID string) (checkoutEntry, error) {
	row := s.DB.QueryRow(ctx, `
SELECT
	cr.id,
	cr."secretId",
	cr."connectionId",
	cr."requesterId",
	cr."approverId",
	cr.status::text,
	cr."durationMinutes",
	cr.reason,
	cr."expiresAt",
	cr."createdAt",
	cr."updatedAt",
	requester.email,
	requester.username,
	approver.email,
	approver.username
FROM "SecretCheckoutRequest" cr
JOIN "User" requester ON requester.id = cr."requesterId"
LEFT JOIN "User" approver ON approver.id = cr."approverId"
WHERE cr.id = $1
`, checkoutID)
	entry, err := scanCheckout(row)
	if err != nil {
		return checkoutEntry{}, err
	}
	items := []checkoutEntry{entry}
	if err := s.attachResourceNames(ctx, items); err != nil {
		return checkoutEntry{}, err
	}
	return items[0], nil
}

func (s Service) attachResourceNames(ctx context.Context, items []checkoutEntry) error {
	secretIDs := make([]string, 0)
	connectionIDs := make([]string, 0)
	for _, item := range items {
		if item.SecretID != nil {
			secretIDs = append(secretIDs, *item.SecretID)
		}
		if item.ConnectionID != nil {
			connectionIDs = append(connectionIDs, *item.ConnectionID)
		}
	}

	secretNames, err := s.loadNameMap(ctx, `SELECT id, name FROM "VaultSecret" WHERE id = ANY($1)`, uniqueStrings(secretIDs))
	if err != nil {
		return fmt.Errorf("load checkout secret names: %w", err)
	}
	connectionNames, err := s.loadNameMap(ctx, `SELECT id, name FROM "Connection" WHERE id = ANY($1)`, uniqueStrings(connectionIDs))
	if err != nil {
		return fmt.Errorf("load checkout connection names: %w", err)
	}

	for i := range items {
		if items[i].SecretID != nil {
			if name, ok := secretNames[*items[i].SecretID]; ok {
				items[i].SecretName = stringPtr(name)
			}
		}
		if items[i].ConnectionID != nil {
			if name, ok := connectionNames[*items[i].ConnectionID]; ok {
				items[i].ConnectionName = stringPtr(name)
			}
		}
	}
	return nil
}

func (s Service) listIDs(ctx context.Context, query string, arg any) ([]string, error) {
	rows, err := s.DB.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		items = append(items, id)
	}
	return items, rows.Err()
}

func (s Service) loadNameMap(ctx context.Context, query string, ids []string) (map[string]string, error) {
	if len(ids) == 0 {
		return map[string]string{}, nil
	}
	rows, err := s.DB.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string, len(ids))
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result[id] = name
	}
	return result, rows.Err()
}
