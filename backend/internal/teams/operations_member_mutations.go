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
)

func (s Service) AddMember(ctx context.Context, teamID, targetUserID, role string, expiresAt *time.Time, actingUserID, tenantID, ipAddress string) (teamMemberResponse, error) {
	if s.DB == nil {
		return teamMemberResponse{}, errors.New("database is unavailable")
	}
	if !isValidTeamRole(role) {
		return teamMemberResponse{}, &requestError{status: 400, message: "role must be one of TEAM_ADMIN, TEAM_EDITOR, TEAM_VIEWER"}
	}
	membership, err := s.requireMembership(ctx, teamID, actingUserID, tenantID)
	if err != nil {
		return teamMemberResponse{}, err
	}
	if membership.Role != "TEAM_ADMIN" {
		return teamMemberResponse{}, &requestError{status: 403, message: "Insufficient team role"}
	}

	actingMasterKey, actingTTL, err := s.getVaultMasterKey(ctx, actingUserID)
	if err != nil {
		return teamMemberResponse{}, err
	}
	if len(actingMasterKey) == 0 {
		return teamMemberResponse{}, &requestError{status: 403, message: "Your vault is locked. Please unlock it first."}
	}
	defer zeroBytes(actingMasterKey)

	targetMasterKey, _, err := s.getVaultMasterKey(ctx, targetUserID)
	if err != nil {
		return teamMemberResponse{}, err
	}
	if len(targetMasterKey) == 0 {
		return teamMemberResponse{}, &requestError{status: 403, message: "Target user's vault is locked. They must unlock their vault first."}
	}
	defer zeroBytes(targetMasterKey)

	teamKey, err := s.getCachedTeamKey(ctx, teamID, actingUserID)
	if err != nil {
		return teamMemberResponse{}, err
	}
	if len(teamKey) == 0 {
		var encField encryptedField
		row := s.DB.QueryRow(ctx, `
SELECT "encryptedTeamVaultKey", "teamVaultKeyIV", "teamVaultKeyTag"
FROM "TeamMember"
WHERE "teamId" = $1 AND "userId" = $2
`, teamID, actingUserID)
		if err := row.Scan(&encField.Ciphertext, &encField.IV, &encField.Tag); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return teamMemberResponse{}, &requestError{status: 404, message: "Member not found"}
			}
			return teamMemberResponse{}, fmt.Errorf("load acting member team key: %w", err)
		}
		if strings.TrimSpace(encField.Ciphertext) == "" || strings.TrimSpace(encField.IV) == "" || strings.TrimSpace(encField.Tag) == "" {
			return teamMemberResponse{}, &requestError{status: 500, message: "Unable to access team vault key"}
		}
		hexKey, err := decryptEncryptedField(actingMasterKey, encField)
		if err != nil {
			return teamMemberResponse{}, &requestError{status: 500, message: "Unable to access team vault key"}
		}
		teamKey, err = hex.DecodeString(hexKey)
		if err != nil {
			return teamMemberResponse{}, fmt.Errorf("decode team key: %w", err)
		}
		defer zeroBytes(teamKey)
		if err := s.storeTeamVaultSession(ctx, teamID, actingUserID, teamKey, actingTTL); err != nil {
			return teamMemberResponse{}, err
		}
	}
	defer zeroBytes(teamKey)

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return teamMemberResponse{}, fmt.Errorf("begin team member create: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var teamTenantID string
	if err := tx.QueryRow(ctx, `SELECT "tenantId" FROM "Team" WHERE id = $1`, teamID).Scan(&teamTenantID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return teamMemberResponse{}, pgx.ErrNoRows
		}
		return teamMemberResponse{}, fmt.Errorf("load team: %w", err)
	}
	if teamTenantID != tenantID {
		return teamMemberResponse{}, &requestError{status: 403, message: "Access denied"}
	}

	var accepted bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS(
	SELECT 1
	FROM "TenantMember"
	WHERE "tenantId" = $1
	  AND "userId" = $2
	  AND status = 'ACCEPTED'
	  AND ("expiresAt" IS NULL OR "expiresAt" > NOW())
)
`, tenantID, targetUserID).Scan(&accepted); err != nil {
		return teamMemberResponse{}, fmt.Errorf("check tenant membership: %w", err)
	}
	if !accepted {
		return teamMemberResponse{}, &requestError{status: 400, message: "User is not a member of this organization"}
	}

	var exists bool
	if err := tx.QueryRow(ctx, `
SELECT EXISTS(
	SELECT 1
	FROM "TeamMember"
	WHERE "teamId" = $1 AND "userId" = $2
)
`, teamID, targetUserID).Scan(&exists); err != nil {
		return teamMemberResponse{}, fmt.Errorf("check team membership: %w", err)
	}
	if exists {
		return teamMemberResponse{}, &requestError{status: 400, message: "User is already a team member"}
	}

	encKey, err := encryptHexPayload(targetMasterKey, hex.EncodeToString(teamKey))
	if err != nil {
		return teamMemberResponse{}, fmt.Errorf("encrypt team key for target member: %w", err)
	}

	var result teamMemberResponse
	var (
		username     sql.NullString
		avatar       sql.NullString
		storedExpiry sql.NullTime
	)
	if err := tx.QueryRow(ctx, `
WITH inserted AS (
	INSERT INTO "TeamMember" (
		id, "teamId", "userId", role,
		"encryptedTeamVaultKey", "teamVaultKeyIV", "teamVaultKeyTag", "expiresAt"
	)
	VALUES ($1, $2, $3, $4::"TeamRole", $5, $6, $7, $8)
	RETURNING "userId", role::text, "joinedAt", "expiresAt"
)
SELECT u.id, u.email, u.username, u."avatarData", i.role, i."joinedAt", i."expiresAt"
FROM inserted i
JOIN "User" u ON u.id = i."userId"
`, uuid.NewString(), teamID, targetUserID, role, encKey.Ciphertext, encKey.IV, encKey.Tag, expiresAt).Scan(
		&result.UserID,
		&result.Email,
		&username,
		&avatar,
		&result.Role,
		&result.JoinedAt,
		&storedExpiry,
	); err != nil {
		return teamMemberResponse{}, fmt.Errorf("create team member: %w", err)
	}
	if username.Valid {
		result.Username = &username.String
	}
	if avatar.Valid {
		result.AvatarData = &avatar.String
	}
	if storedExpiry.Valid {
		value := storedExpiry.Time
		result.ExpiresAt = &value
		result.Expired = !value.After(time.Now())
	}

	var expiryValue any
	if result.ExpiresAt != nil {
		expiryValue = result.ExpiresAt.Format(time.RFC3339)
	}
	if err := insertAuditLog(ctx, tx, actingUserID, "TEAM_ADD_MEMBER", "TeamMember", targetUserID, map[string]any{
		"teamId":    teamID,
		"role":      role,
		"expiresAt": expiryValue,
	}, ipAddress); err != nil {
		return teamMemberResponse{}, fmt.Errorf("insert team member create audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return teamMemberResponse{}, fmt.Errorf("commit team member create: %w", err)
	}
	return result, nil
}

func (s Service) UpdateMemberRole(ctx context.Context, teamID, targetUserID, newRole, actingUserID, tenantID, ipAddress string) (map[string]any, error) {
	if s.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	if !isValidTeamRole(newRole) {
		return nil, &requestError{status: 400, message: "role must be one of TEAM_ADMIN, TEAM_EDITOR, TEAM_VIEWER"}
	}
	membership, err := s.requireMembership(ctx, teamID, actingUserID, tenantID)
	if err != nil {
		return nil, err
	}
	if membership.Role != "TEAM_ADMIN" {
		return nil, &requestError{status: 403, message: "Insufficient team role"}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin team member role update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentRole string
	if err := tx.QueryRow(ctx, `
SELECT role::text
FROM "TeamMember"
WHERE "teamId" = $1 AND "userId" = $2
`, teamID, targetUserID).Scan(&currentRole); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &requestError{status: 404, message: "Member not found"}
		}
		return nil, fmt.Errorf("load team member: %w", err)
	}

	if currentRole == "TEAM_ADMIN" && newRole != "TEAM_ADMIN" {
		var adminCount int
		if err := tx.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM "TeamMember"
WHERE "teamId" = $1 AND role = 'TEAM_ADMIN'
`, teamID).Scan(&adminCount); err != nil {
			return nil, fmt.Errorf("count team admins: %w", err)
		}
		if adminCount <= 1 {
			return nil, &requestError{status: 400, message: "Cannot demote the last team admin"}
		}
	}

	if _, err := tx.Exec(ctx, `
UPDATE "TeamMember"
SET role = $3::"TeamRole"
WHERE "teamId" = $1 AND "userId" = $2
`, teamID, targetUserID, newRole); err != nil {
		return nil, fmt.Errorf("update team member role: %w", err)
	}
	if err := insertAuditLog(ctx, tx, actingUserID, "TEAM_UPDATE_MEMBER_ROLE", "TeamMember", targetUserID, map[string]any{
		"teamId":  teamID,
		"newRole": newRole,
	}, ipAddress); err != nil {
		return nil, fmt.Errorf("insert team member role audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit team member role update: %w", err)
	}
	return map[string]any{"userId": targetUserID, "role": newRole}, nil
}

func (s Service) RemoveMember(ctx context.Context, teamID, targetUserID, actingUserID, tenantID, ipAddress string) (map[string]any, error) {
	if s.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	membership, err := s.requireMembership(ctx, teamID, actingUserID, tenantID)
	if err != nil {
		return nil, err
	}
	if membership.Role != "TEAM_ADMIN" {
		return nil, &requestError{status: 403, message: "Insufficient team role"}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin team member delete: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var targetRole string
	if err := tx.QueryRow(ctx, `
SELECT role::text
FROM "TeamMember"
WHERE "teamId" = $1 AND "userId" = $2
`, teamID, targetUserID).Scan(&targetRole); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &requestError{status: 404, message: "Member not found"}
		}
		return nil, fmt.Errorf("load team member: %w", err)
	}

	if targetRole == "TEAM_ADMIN" {
		var adminCount int
		if err := tx.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM "TeamMember"
WHERE "teamId" = $1 AND role = 'TEAM_ADMIN'
`, teamID).Scan(&adminCount); err != nil {
			return nil, fmt.Errorf("count team admins: %w", err)
		}
		if adminCount <= 1 {
			return nil, &requestError{status: 400, message: "Cannot remove the last team admin"}
		}
	}

	if _, err := tx.Exec(ctx, `
DELETE FROM "TeamMember"
WHERE "teamId" = $1 AND "userId" = $2
`, teamID, targetUserID); err != nil {
		return nil, fmt.Errorf("remove team member: %w", err)
	}
	if err := insertAuditLog(ctx, tx, actingUserID, "TEAM_REMOVE_MEMBER", "TeamMember", targetUserID, map[string]any{
		"teamId": teamID,
	}, ipAddress); err != nil {
		return nil, fmt.Errorf("insert team member remove audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit team member delete: %w", err)
	}
	return map[string]any{"removed": true}, nil
}

func (s Service) UpdateMemberExpiry(ctx context.Context, teamID, targetUserID string, expiresAt *time.Time, actingUserID, tenantID, ipAddress string) (teamMemberResponse, error) {
	if s.DB == nil {
		return teamMemberResponse{}, errors.New("database is unavailable")
	}
	membership, err := s.requireMembership(ctx, teamID, actingUserID, tenantID)
	if err != nil {
		return teamMemberResponse{}, err
	}
	if membership.Role != "TEAM_ADMIN" {
		return teamMemberResponse{}, &requestError{status: 403, message: "Insufficient team role"}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return teamMemberResponse{}, fmt.Errorf("begin team member expiry update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var result teamMemberResponse
	var (
		username     sql.NullString
		avatar       sql.NullString
		storedExpiry sql.NullTime
	)
	if err := tx.QueryRow(ctx, `
UPDATE "TeamMember" tm
SET "expiresAt" = $3
FROM "User" u
WHERE tm."teamId" = $1
  AND tm."userId" = $2
  AND u.id = tm."userId"
RETURNING u.id, u.email, u.username, u."avatarData", tm.role::text, tm."joinedAt", tm."expiresAt"
`, teamID, targetUserID, expiresAt).Scan(
		&result.UserID,
		&result.Email,
		&username,
		&avatar,
		&result.Role,
		&result.JoinedAt,
		&storedExpiry,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return teamMemberResponse{}, &requestError{status: 404, message: "Member not found"}
		}
		return teamMemberResponse{}, fmt.Errorf("update team member expiry: %w", err)
	}
	if username.Valid {
		result.Username = &username.String
	}
	if avatar.Valid {
		result.AvatarData = &avatar.String
	}
	if storedExpiry.Valid {
		value := storedExpiry.Time
		result.ExpiresAt = &value
		result.Expired = !value.After(time.Now())
	}

	var expiryValue any
	if result.ExpiresAt != nil {
		expiryValue = result.ExpiresAt.Format(time.RFC3339)
	}
	if err := insertAuditLog(ctx, tx, actingUserID, "TEAM_MEMBERSHIP_EXPIRY_UPDATE", "TeamMember", targetUserID, map[string]any{
		"teamId":    teamID,
		"expiresAt": expiryValue,
	}, ipAddress); err != nil {
		return teamMemberResponse{}, fmt.Errorf("insert team member expiry audit: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return teamMemberResponse{}, fmt.Errorf("commit team member expiry update: %w", err)
	}
	return result, nil
}
