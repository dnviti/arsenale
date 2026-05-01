package keystrokepolicies

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) List(ctx context.Context, tenantID string) ([]policyResponse, error) {
	if s.DB == nil {
		return nil, errors.New("database is unavailable")
	}
	rows, err := s.DB.Query(ctx, `
SELECT id, "tenantId", name, description, action::text, "regexPatterns", enabled, "createdAt", "updatedAt"
FROM "KeystrokePolicy"
WHERE "tenantId" = $1
ORDER BY "createdAt" DESC
`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list keystroke policies: %w", err)
	}
	defer rows.Close()

	items := make([]policyResponse, 0)
	for rows.Next() {
		item, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate keystroke policies: %w", err)
	}
	return items, nil
}

func (s Service) Get(ctx context.Context, tenantID, policyID string) (policyResponse, error) {
	if s.DB == nil {
		return policyResponse{}, errors.New("database is unavailable")
	}
	row := s.DB.QueryRow(ctx, `
SELECT id, "tenantId", name, description, action::text, "regexPatterns", enabled, "createdAt", "updatedAt"
FROM "KeystrokePolicy"
WHERE id = $1 AND "tenantId" = $2
`, policyID, tenantID)
	item, err := scanPolicy(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return policyResponse{}, &requestError{status: http.StatusNotFound, message: "Keystroke policy not found"}
		}
		return policyResponse{}, err
	}
	return item, nil
}

func (s Service) Create(ctx context.Context, claims authn.Claims, payload createPayload) (policyResponse, error) {
	if s.DB == nil {
		return policyResponse{}, errors.New("database is unavailable")
	}
	if err := validateName(payload.Name); err != nil {
		return policyResponse{}, err
	}
	action, err := normalizeAction(payload.Action)
	if err != nil {
		return policyResponse{}, err
	}
	patterns, err := validatePatterns(payload.RegexPatterns)
	if err != nil {
		return policyResponse{}, err
	}

	description := normalizeOptionalString(payload.Description)
	enabled := true
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}
	now := time.Now().UTC()
	item := policyResponse{}
	if err := s.DB.QueryRow(ctx, `
INSERT INTO "KeystrokePolicy" (
	id, "tenantId", name, description, action, "regexPatterns", enabled, "createdAt", "updatedAt"
)
VALUES ($1, $2, $3, $4, $5::"KeystrokePolicyAction", $6, $7, $8, $9)
RETURNING id, "tenantId", name, description, action::text, "regexPatterns", enabled, "createdAt", "updatedAt"
`, uuid.NewString(), claims.TenantID, strings.TrimSpace(payload.Name), description, action, patterns, enabled, now, now).Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Description,
		&item.Action,
		&item.RegexPatterns,
		&item.Enabled,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return policyResponse{}, fmt.Errorf("create keystroke policy: %w", err)
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "KEYSTROKE_POLICY_CREATE", item.ID, map[string]any{
		"name":         item.Name,
		"action":       item.Action,
		"patternCount": len(item.RegexPatterns),
	})
	return item, nil
}

func (s Service) Update(ctx context.Context, claims authn.Claims, policyID string, payload updatePayload) (policyResponse, error) {
	if s.DB == nil {
		return policyResponse{}, errors.New("database is unavailable")
	}
	if _, err := s.Get(ctx, claims.TenantID, policyID); err != nil {
		return policyResponse{}, err
	}

	setClauses := []string{}
	args := []any{policyID, claims.TenantID}
	add := func(clause string, value any) {
		args = append(args, value)
		setClauses = append(setClauses, fmt.Sprintf(clause, len(args)))
	}

	if payload.Name != nil {
		if err := validateName(*payload.Name); err != nil {
			return policyResponse{}, err
		}
		add(`name = $%d`, strings.TrimSpace(*payload.Name))
	}
	if payload.Description != nil {
		add(`description = $%d`, normalizeOptionalString(*payload.Description))
	}
	if payload.Action != nil {
		action, err := normalizeAction(*payload.Action)
		if err != nil {
			return policyResponse{}, err
		}
		add(`action = $%d::"KeystrokePolicyAction"`, action)
	}
	if payload.RegexPatterns != nil {
		patterns, err := validatePatterns(*payload.RegexPatterns)
		if err != nil {
			return policyResponse{}, err
		}
		add(`"regexPatterns" = $%d`, patterns)
	}
	if payload.Enabled != nil {
		add(`enabled = $%d`, *payload.Enabled)
	}
	if len(setClauses) == 0 {
		return s.Get(ctx, claims.TenantID, policyID)
	}

	args = append(args, time.Now().UTC())
	setClauses = append(setClauses, fmt.Sprintf(`"updatedAt" = $%d`, len(args)))
	query := fmt.Sprintf(`
UPDATE "KeystrokePolicy"
SET %s
WHERE id = $1 AND "tenantId" = $2
RETURNING id, "tenantId", name, description, action::text, "regexPatterns", enabled, "createdAt", "updatedAt"
`, strings.Join(setClauses, ", "))

	item := policyResponse{}
	if err := s.DB.QueryRow(ctx, query, args...).Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Description,
		&item.Action,
		&item.RegexPatterns,
		&item.Enabled,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return policyResponse{}, &requestError{status: http.StatusNotFound, message: "Keystroke policy not found"}
		}
		return policyResponse{}, fmt.Errorf("update keystroke policy: %w", err)
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "KEYSTROKE_POLICY_UPDATE", item.ID, map[string]any{
		"name":   item.Name,
		"action": item.Action,
	})
	return item, nil
}

func (s Service) Delete(ctx context.Context, claims authn.Claims, policyID string) error {
	if s.DB == nil {
		return errors.New("database is unavailable")
	}
	cmd, err := s.DB.Exec(ctx, `DELETE FROM "KeystrokePolicy" WHERE id = $1 AND "tenantId" = $2`, policyID, claims.TenantID)
	if err != nil {
		return fmt.Errorf("delete keystroke policy: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return &requestError{status: http.StatusNotFound, message: "Keystroke policy not found"}
	}
	_ = s.insertAuditLog(ctx, claims.UserID, "KEYSTROKE_POLICY_DELETE", policyID, nil)
	return nil
}

func scanPolicy(row rowScanner) (policyResponse, error) {
	var item policyResponse
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Description,
		&item.Action,
		&item.RegexPatterns,
		&item.Enabled,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return policyResponse{}, fmt.Errorf("scan keystroke policy: %w", err)
	}
	return item, nil
}
