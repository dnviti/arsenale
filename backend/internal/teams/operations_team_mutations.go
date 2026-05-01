package teams

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s Service) CreateTeam(ctx context.Context, tenantID, creatorUserID string, payload createTeamPayload, ipAddress string) (teamResponse, error) {
	if s.DB == nil {
		return teamResponse{}, errors.New("database is unavailable")
	}

	name := strings.TrimSpace(payload.Name)
	if len(name) < 2 || len(name) > 100 {
		return teamResponse{}, &requestError{status: 400, message: "name must be between 2 and 100 characters"}
	}
	var description *string
	if payload.Description != nil {
		value := strings.TrimSpace(*payload.Description)
		if len(value) > 500 {
			return teamResponse{}, &requestError{status: 400, message: "description must be 500 characters or fewer"}
		}
		description = &value
	}

	userMasterKey, ttl, err := s.getVaultMasterKey(ctx, creatorUserID)
	if err != nil {
		return teamResponse{}, err
	}
	if len(userMasterKey) == 0 {
		return teamResponse{}, &requestError{status: 403, message: "Vault is locked. Please unlock it first."}
	}
	defer zeroBytes(userMasterKey)

	teamKey, err := generateRandomKey()
	if err != nil {
		return teamResponse{}, fmt.Errorf("generate team key: %w", err)
	}
	defer zeroBytes(teamKey)

	encKey, err := encryptHexPayload(userMasterKey, hex.EncodeToString(teamKey))
	if err != nil {
		return teamResponse{}, fmt.Errorf("encrypt team key: %w", err)
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return teamResponse{}, fmt.Errorf("begin team create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()
	var result teamResponse
	var desc sql.NullString
	var updatedAt time.Time
	if err := tx.QueryRow(ctx, `
INSERT INTO "Team" (id, name, description, "tenantId", "createdAt", "updatedAt")
VALUES ($1, $2, $3, $4, $5, $5)
RETURNING id, name, description, "createdAt", "updatedAt"
`, uuid.NewString(), name, description, tenantID, now).Scan(&result.ID, &result.Name, &desc, &result.CreatedAt, &updatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return teamResponse{}, &requestError{status: 409, message: "A team with this name already exists"}
		}
		return teamResponse{}, fmt.Errorf("create team: %w", err)
	}
	if desc.Valid {
		result.Description = &desc.String
	}
	result.MemberCount = 1
	result.MyRole = "TEAM_ADMIN"
	result.UpdatedAt = &updatedAt

	if _, err := tx.Exec(ctx, `
INSERT INTO "TeamMember" (
	id, "teamId", "userId", role,
	"encryptedTeamVaultKey", "teamVaultKeyIV", "teamVaultKeyTag"
)
VALUES ($1, $2, $3, 'TEAM_ADMIN', $4, $5, $6)
`, uuid.NewString(), result.ID, creatorUserID, encKey.Ciphertext, encKey.IV, encKey.Tag); err != nil {
		return teamResponse{}, fmt.Errorf("create team membership: %w", err)
	}

	if err := insertAuditLog(ctx, tx, creatorUserID, "TEAM_CREATE", "Team", result.ID, map[string]any{
		"name": name,
	}, ipAddress); err != nil {
		return teamResponse{}, fmt.Errorf("insert team create audit: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return teamResponse{}, fmt.Errorf("commit team create: %w", err)
	}
	if err := s.storeTeamVaultSession(ctx, result.ID, creatorUserID, teamKey, ttl); err != nil {
		return teamResponse{}, err
	}
	return result, nil
}

func (s Service) UpdateTeam(ctx context.Context, teamID, userID, tenantID string, payload updateTeamPayload, ipAddress string) (teamResponse, error) {
	if s.DB == nil {
		return teamResponse{}, errors.New("database is unavailable")
	}
	membership, err := s.requireMembership(ctx, teamID, userID, tenantID)
	if err != nil {
		return teamResponse{}, err
	}
	if membership.Role != "TEAM_ADMIN" {
		return teamResponse{}, &requestError{status: 403, message: "Insufficient team role"}
	}

	setClauses := make([]string, 0, 3)
	args := []any{teamID}
	addClause := func(clause string, value any) {
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf(clause, len(args)))
	}

	if payload.Name.Present {
		if payload.Name.Value == nil || strings.TrimSpace(*payload.Name.Value) == "" {
			return teamResponse{}, &requestError{status: 400, message: "name cannot be empty"}
		}
		name := strings.TrimSpace(*payload.Name.Value)
		if len(name) < 2 || len(name) > 100 {
			return teamResponse{}, &requestError{status: 400, message: "name must be between 2 and 100 characters"}
		}
		addClause(`name = $%d`, name)
	}
	if payload.Description.Present {
		if payload.Description.Value == nil {
			addClause(`description = $%d`, nil)
		} else {
			description := strings.TrimSpace(*payload.Description.Value)
			if len(description) > 500 {
				return teamResponse{}, &requestError{status: 400, message: "description must be 500 characters or fewer"}
			}
			addClause(`description = $%d`, description)
		}
	}
	if len(setClauses) == 0 {
		return teamResponse{}, &requestError{status: 400, message: "No fields to update"}
	}

	args = append(args, time.Now().UTC())
	setClauses = append(setClauses, fmt.Sprintf(`"updatedAt" = $%d`, len(args)))
	query := fmt.Sprintf(`
UPDATE "Team"
SET %s
WHERE id = $1
RETURNING id, name, description, "createdAt", "updatedAt"
`, strings.Join(setClauses, ", "))

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return teamResponse{}, fmt.Errorf("begin team update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		result      teamResponse
		description sql.NullString
		updatedAt   time.Time
	)
	if err := tx.QueryRow(ctx, query, args...).Scan(&result.ID, &result.Name, &description, &result.CreatedAt, &updatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return teamResponse{}, &requestError{status: 409, message: "A team with this name already exists"}
		}
		return teamResponse{}, fmt.Errorf("update team: %w", err)
	}
	if description.Valid {
		result.Description = &description.String
	}
	result.MyRole = membership.Role
	result.UpdatedAt = &updatedAt

	if err := insertAuditLog(ctx, tx, userID, "TEAM_UPDATE", "Team", teamID, map[string]any{
		"fields": changedTeamFields(payload),
	}, ipAddress); err != nil {
		return teamResponse{}, fmt.Errorf("insert team update audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return teamResponse{}, fmt.Errorf("commit team update: %w", err)
	}
	return result, nil
}

func (s Service) DeleteTeam(ctx context.Context, teamID, userID, tenantID, ipAddress string) (map[string]any, error) {
	if s.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	membership, err := s.requireMembership(ctx, teamID, userID, tenantID)
	if err != nil {
		return nil, err
	}
	if membership.Role != "TEAM_ADMIN" {
		return nil, &requestError{status: 403, message: "Insufficient team role"}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin team delete: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `UPDATE "Connection" SET "teamId" = NULL WHERE "teamId" = $1`, teamID); err != nil {
		return nil, fmt.Errorf("clear team connections: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE "Folder" SET "teamId" = NULL WHERE "teamId" = $1`, teamID); err != nil {
		return nil, fmt.Errorf("clear team folders: %w", err)
	}
	commandTag, err := tx.Exec(ctx, `DELETE FROM "Team" WHERE id = $1`, teamID)
	if err != nil {
		return nil, fmt.Errorf("delete team: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return nil, pgx.ErrNoRows
	}
	if err := insertAuditLog(ctx, tx, userID, "TEAM_DELETE", "Team", teamID, nil, ipAddress); err != nil {
		return nil, fmt.Errorf("insert team delete audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit team delete: %w", err)
	}
	return map[string]any{"deleted": true}, nil
}
