package oauthapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s Service) ListAccounts(ctx context.Context, userID string) ([]linkedAccount, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}

	rows, err := s.DB.Query(
		ctx,
		`SELECT id, provider::text, "providerEmail", "createdAt"
		   FROM "OAuthAccount"
		  WHERE "userId" = $1
		  ORDER BY "createdAt" ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list oauth accounts: %w", err)
	}
	defer rows.Close()

	items := make([]linkedAccount, 0)
	for rows.Next() {
		var item linkedAccount
		if err := rows.Scan(&item.ID, &item.Provider, &item.ProviderEmail, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan oauth account: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate oauth accounts: %w", err)
	}
	return items, nil
}

func (s Service) UnlinkAccount(ctx context.Context, userID, provider string) error {
	if s.DB == nil {
		return fmt.Errorf("database is unavailable")
	}
	normalized, err := normalizeProvider(provider)
	if err != nil {
		return err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin oauth unlink transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var passwordHash *string
	if err := tx.QueryRow(ctx, `SELECT "passwordHash" FROM "User" WHERE id = $1`, userID).Scan(&passwordHash); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &requestError{status: http.StatusNotFound, message: "User not found"}
		}
		return fmt.Errorf("load unlink user: %w", err)
	}

	rows, err := tx.Query(ctx, `SELECT id, provider::text FROM "OAuthAccount" WHERE "userId" = $1`, userID)
	if err != nil {
		return fmt.Errorf("list unlink oauth accounts: %w", err)
	}
	defer rows.Close()

	var (
		targetID    string
		totalCount  int
		seenAccount bool
	)
	for rows.Next() {
		var accountID string
		var accountProvider string
		if err := rows.Scan(&accountID, &accountProvider); err != nil {
			return fmt.Errorf("scan unlink oauth account: %w", err)
		}
		totalCount++
		if accountProvider == normalized {
			targetID = accountID
			seenAccount = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate unlink oauth accounts: %w", err)
	}
	if !seenAccount {
		return &requestError{status: http.StatusNotFound, message: "OAuth account not found"}
	}
	if strings.TrimSpace(deref(passwordHash)) == "" && totalCount <= 1 {
		return &requestError{
			status:  http.StatusBadRequest,
			message: "Cannot unlink your only sign-in method. Set a password first or link another OAuth provider.",
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM "OAuthAccount" WHERE id = $1`, targetID); err != nil {
		return fmt.Errorf("delete oauth account: %w", err)
	}
	if err := insertAuditLog(ctx, tx, userID, "OAUTH_UNLINK", map[string]any{"provider": strings.ToLower(normalized)}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit oauth unlink transaction: %w", err)
	}
	return nil
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
