package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) writeError(w http.ResponseWriter, err error) {
	var reqErr *requestError
	if errors.As(err, &reqErr) {
		app.ErrorJSON(w, reqErr.status, reqErr.message)
		return
	}
	app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
}

func insertAuditLog(ctx context.Context, tx pgx.Tx, userID, action string, details map[string]any) error {
	payload, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
INSERT INTO "AuditLog" (id, "userId", action, details)
VALUES ($1, $2, $3, $4::jsonb)
`, uuid.NewString(), userID, action, payload)
	return err
}

func (s Service) insertStandaloneAuditLog(ctx context.Context, userID, action string, details map[string]any) error {
	if s.DB == nil {
		return nil
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin audit log insert: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := insertAuditLog(ctx, tx, userID, action, details); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit audit log: %w", err)
	}
	return nil
}
