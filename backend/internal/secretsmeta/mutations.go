package secretsmeta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/credentialresolver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) HandleCreate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	body, err := readBodyBytes(r)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	payload, err := parseCreateSecretInput(body)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	item, err := s.CreateSecret(r.Context(), claims.UserID, claims.TenantID, payload)
	if err != nil {
		s.handleResolverError(w, err)
		return
	}

	_ = s.insertAuditLog(r.Context(), claims.UserID, "SECRET_CREATE", item.ID, map[string]any{
		"name":  payload.Name,
		"type":  payload.Type,
		"scope": payload.Scope,
	}, requestIP(r))
	app.WriteJSON(w, http.StatusCreated, item)
}

func (s Service) HandleUpdate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	body, err := readBodyBytes(r)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	payload, err := parseUpdateSecretInput(body)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	item, err := s.UpdateSecret(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"), payload)
	if err != nil {
		s.handleResolverError(w, err)
		return
	}

	_ = s.insertAuditLog(r.Context(), claims.UserID, "SECRET_UPDATE", item.ID, map[string]any{
		"fields": payload.Fields,
	}, requestIP(r))
	app.WriteJSON(w, http.StatusOK, item)
}

func (s Service) CreateSecret(ctx context.Context, userID, tenantID string, input createSecretInput) (credentialresolver.SecretSummary, error) {
	if s.DB == nil {
		return credentialresolver.SecretSummary{}, fmt.Errorf("database is unavailable")
	}

	scopeTenantID, err := s.resolveCreateScope(ctx, userID, tenantID, input.Scope, input.TeamID)
	if err != nil {
		return credentialresolver.SecretSummary{}, err
	}

	ciphertext, iv, tag, err := s.resolver().EncryptPayloadForScope(ctx, userID, input.Scope, input.TeamID, scopeTenantID, input.Data)
	if err != nil {
		return credentialresolver.SecretSummary{}, err
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return credentialresolver.SecretSummary{}, fmt.Errorf("begin create secret: %w", err)
	}
	defer tx.Rollback(ctx)

	secretID := uuid.NewString()
	var metadata any
	if input.Metadata != nil {
		metadata = input.Metadata
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO "VaultSecret" (
	id,
	name,
	description,
	type,
	scope,
	"userId",
	"teamId",
	"tenantId",
	"folderId",
	"encryptedData",
	"dataIV",
	"dataTag",
	metadata,
	tags,
	"pwnedCount",
	"expiresAt",
	"currentVersion",
	"createdAt",
	"updatedAt"
) VALUES ($1, $2, $3, $4::"SecretType", $5::"SecretScope", $6, $7, $8, $9, $10, $11, $12, $13, COALESCE($14, ARRAY[]::text[]), 0, $15, 1, NOW(), NOW())
`, secretID, input.Name, nullableString(input.Description), input.Type, input.Scope, userID, nullableString(input.TeamID), nullableString(scopeTenantID), nullableString(input.FolderID), ciphertext, iv, tag, metadata, input.Tags, nullableTime(input.ExpiresAt)); err != nil {
		return credentialresolver.SecretSummary{}, fmt.Errorf("insert secret: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO "VaultSecretVersion" (
	id,
	"secretId",
	version,
	"encryptedData",
	"dataIV",
	"dataTag",
	"changedBy",
	"changeNote"
) VALUES ($1, $2, 1, $3, $4, $5, $6, 'Initial version')
`, uuid.NewString(), secretID, ciphertext, iv, tag, userID); err != nil {
		return credentialresolver.SecretSummary{}, fmt.Errorf("insert initial secret version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return credentialresolver.SecretSummary{}, fmt.Errorf("commit create secret: %w", err)
	}

	s.enqueuePwnedCountCheck(secretID, input.Data)
	return s.resolver().LoadSecretSummary(ctx, secretID)
}

func (s Service) UpdateSecret(ctx context.Context, userID, tenantID, secretID string, input updateSecretInput) (credentialresolver.SecretSummary, error) {
	access, err := s.resolver().RequireManageSecret(ctx, userID, secretID, tenantID)
	if err != nil {
		return credentialresolver.SecretSummary{}, err
	}
	if s.DB == nil {
		return credentialresolver.SecretSummary{}, fmt.Errorf("database is unavailable")
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return credentialresolver.SecretSummary{}, fmt.Errorf("begin update secret: %w", err)
	}
	defer tx.Rollback(ctx)

	var currentVersion int
	if err := tx.QueryRow(ctx, `
SELECT "currentVersion"
FROM "VaultSecret"
WHERE id = $1
FOR UPDATE
`, secretID).Scan(&currentVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return credentialresolver.SecretSummary{}, &credentialresolver.RequestError{Status: http.StatusNotFound, Message: "Secret not found"}
		}
		return credentialresolver.SecretSummary{}, fmt.Errorf("load secret for update: %w", err)
	}

	assignments := make([]string, 0, 10)
	args := make([]any, 0, 16)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if input.Name != nil {
		assignments = append(assignments, fmt.Sprintf(`name = %s`, addArg(*input.Name)))
	}
	if input.DescriptionSet {
		assignments = append(assignments, fmt.Sprintf(`description = %s`, addArg(nullableString(input.Description))))
	}
	if input.MetadataSet {
		assignments = append(assignments, fmt.Sprintf(`metadata = %s`, addArg(nullableMetadata(input.Metadata))))
	}
	if input.Tags != nil {
		assignments = append(assignments, fmt.Sprintf(`tags = %s`, addArg(*input.Tags)))
	}
	if input.FolderIDSet {
		assignments = append(assignments, fmt.Sprintf(`"folderId" = %s`, addArg(nullableString(input.FolderID))))
	}
	if input.IsFavorite != nil {
		assignments = append(assignments, fmt.Sprintf(`"isFavorite" = %s`, addArg(*input.IsFavorite)))
	}
	if input.ExpiresAtSet {
		assignments = append(assignments, fmt.Sprintf(`"expiresAt" = %s`, addArg(nullableTime(input.ExpiresAt))))
	}

	if input.DataSet {
		ciphertext, iv, tag, err := s.resolver().EncryptPayloadForScope(ctx, userID, access.Scope, access.TeamID, access.TenantID, input.Data)
		if err != nil {
			return credentialresolver.SecretSummary{}, err
		}
		assignments = append(assignments,
			fmt.Sprintf(`"encryptedData" = %s`, addArg(ciphertext)),
			fmt.Sprintf(`"dataIV" = %s`, addArg(iv)),
			fmt.Sprintf(`"dataTag" = %s`, addArg(tag)),
			`"pwnedCount" = 0`,
			fmt.Sprintf(`"currentVersion" = %s`, addArg(currentVersion+1)),
		)
	}

	if len(assignments) == 0 {
		return credentialresolver.SecretSummary{}, &credentialresolver.RequestError{Status: http.StatusBadRequest, Message: "No fields to update"}
	}

	assignments = append(assignments, `"updatedAt" = NOW()`)
	args = append(args, secretID)
	updateSQL := fmt.Sprintf(`UPDATE "VaultSecret" SET %s WHERE id = $%d`, strings.Join(assignments, ", "), len(args))
	if _, err := tx.Exec(ctx, updateSQL, args...); err != nil {
		return credentialresolver.SecretSummary{}, fmt.Errorf("update secret: %w", err)
	}

	if input.DataSet {
		changeNote := input.ChangeNote
		if _, err := tx.Exec(ctx, `
INSERT INTO "VaultSecretVersion" (
	id,
	"secretId",
	version,
	"encryptedData",
	"dataIV",
	"dataTag",
	"changedBy",
	"changeNote"
)
SELECT $1, id, "currentVersion", "encryptedData", "dataIV", "dataTag", $2, $3
FROM "VaultSecret"
WHERE id = $4
`, uuid.NewString(), userID, changeNote, secretID); err != nil {
			return credentialresolver.SecretSummary{}, fmt.Errorf("insert updated secret version: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return credentialresolver.SecretSummary{}, fmt.Errorf("commit update secret: %w", err)
	}

	if input.DataSet {
		s.enqueuePwnedCountCheck(secretID, input.Data)
	}

	return s.resolver().LoadSecretSummary(ctx, secretID)
}

func (s Service) resolveCreateScope(ctx context.Context, userID, tenantID, scope string, teamID *string) (*string, error) {
	switch scope {
	case "PERSONAL":
		return nil, nil
	case "TEAM":
		if teamID == nil || strings.TrimSpace(*teamID) == "" {
			return nil, &credentialresolver.RequestError{Status: http.StatusBadRequest, Message: "teamId is required for team-scoped secrets"}
		}
		if err := s.requireTeamCreateAccess(ctx, userID, tenantID, *teamID); err != nil {
			return nil, err
		}
		if strings.TrimSpace(tenantID) == "" {
			return nil, nil
		}
		value := tenantID
		return &value, nil
	case "TENANT":
		if strings.TrimSpace(tenantID) == "" {
			return nil, &credentialresolver.RequestError{Status: http.StatusBadRequest, Message: "tenantId is required for tenant-scoped secrets"}
		}
		if err := s.requireTenantCreateAccess(ctx, userID, tenantID); err != nil {
			return nil, err
		}
		value := tenantID
		return &value, nil
	default:
		return nil, &credentialresolver.RequestError{Status: http.StatusBadRequest, Message: "scope must be PERSONAL, TEAM, or TENANT"}
	}
}

func (s Service) requireTeamCreateAccess(ctx context.Context, userID, tenantID, teamID string) error {
	if s.DB == nil {
		return fmt.Errorf("database is unavailable")
	}

	var (
		role       string
		teamTenant *string
	)
	err := s.DB.QueryRow(ctx, `
SELECT tm.role::text, t."tenantId"
FROM "TeamMember" tm
JOIN "Team" t ON t.id = tm."teamId"
WHERE tm."teamId" = $1
  AND tm."userId" = $2
`, teamID, userID).Scan(&role, &teamTenant)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &credentialresolver.RequestError{Status: http.StatusForbidden, Message: "Insufficient team role to create secrets"}
		}
		return fmt.Errorf("load team membership: %w", err)
	}

	if strings.TrimSpace(tenantID) != "" && teamTenant != nil && strings.TrimSpace(*teamTenant) != "" && *teamTenant != tenantID {
		return &credentialresolver.RequestError{Status: http.StatusForbidden, Message: "Access denied"}
	}

	switch role {
	case "TEAM_EDITOR", "TEAM_ADMIN":
		return nil
	default:
		return &credentialresolver.RequestError{Status: http.StatusForbidden, Message: "Insufficient team role to create secrets"}
	}
}

func (s Service) requireTenantCreateAccess(ctx context.Context, userID, tenantID string) error {
	if s.DB == nil {
		return fmt.Errorf("database is unavailable")
	}

	var (
		role   string
		status string
	)
	if err := s.DB.QueryRow(ctx, `
SELECT role::text, status::text
FROM "TenantMember"
WHERE "tenantId" = $1
  AND "userId" = $2
`, tenantID, userID).Scan(&role, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &credentialresolver.RequestError{Status: http.StatusForbidden, Message: "Only admins and owners can create tenant-scoped secrets"}
		}
		return fmt.Errorf("load tenant membership: %w", err)
	}

	if status != "ACCEPTED" || (role != "OWNER" && role != "ADMIN") {
		return &credentialresolver.RequestError{Status: http.StatusForbidden, Message: "Only admins and owners can create tenant-scoped secrets"}
	}
	return nil
}

func (s Service) enqueuePwnedCountCheck(secretID string, payload json.RawMessage) {
	password := extractPasswordFromPayload(payload)
	if password == "" {
		return
	}

	go func() {
		ctx := context.Background()
		pwnedCount, err := checkPwnedPassword(ctx, password)
		if err != nil {
			return
		}
		_ = s.updateSecretPwnedCount(ctx, secretID, pwnedCount)
	}()
}

func readBodyBytes(r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	return body, nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableMetadata(value map[string]any) any {
	if value == nil {
		return nil
	}
	return value
}
