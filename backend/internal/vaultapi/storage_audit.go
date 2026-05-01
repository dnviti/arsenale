package vaultapi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) insertAuditLog(ctx context.Context, userID, action string, details map[string]any, ipAddress string) error {
	if s.DB == nil {
		return nil
	}
	rawDetails, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}
	if _, err := s.DB.Exec(
		ctx,
		`INSERT INTO "AuditLog" (
			id, "userId", action, details, "ipAddress", "createdAt"
		) VALUES (
			$1, $2, $3::"AuditAction", $4::jsonb, NULLIF($5, ''), $6
		)`,
		uuid.NewString(),
		userID,
		action,
		string(rawDetails),
		ipAddress,
		time.Now(),
	); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (s Service) insertAuditLogTx(ctx context.Context, tx pgx.Tx, userID, action string, details map[string]any, ipAddress string) error {
	rawDetails, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}
	if _, err := tx.Exec(
		ctx,
		`INSERT INTO "AuditLog" (
			id, "userId", action, details, "ipAddress", "createdAt"
		) VALUES (
			$1, $2, $3::"AuditAction", $4::jsonb, NULLIF($5, ''), $6
		)`,
		uuid.NewString(),
		userID,
		action,
		string(rawDetails),
		ipAddress,
		time.Now(),
	); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (s Service) insertConnectionAuditLog(ctx context.Context, userID, action, connectionID, ipAddress string) error {
	if s.DB == nil {
		return nil
	}
	if _, err := s.DB.Exec(
		ctx,
		`INSERT INTO "AuditLog" (
			id, "userId", action, "targetType", "targetId", details, "ipAddress", "createdAt"
		) VALUES (
			$1, $2, $3::"AuditAction", 'Connection', $4, '{}'::jsonb, NULLIF($5, ''), $6
		)`,
		uuid.NewString(),
		userID,
		action,
		connectionID,
		ipAddress,
		time.Now(),
	); err != nil {
		return fmt.Errorf("insert connection audit log: %w", err)
	}
	return nil
}
