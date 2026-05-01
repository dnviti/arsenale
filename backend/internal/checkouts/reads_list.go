package checkouts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

func (s Service) List(ctx context.Context, userID, role, status string, limit, offset int) (paginatedResponse, error) {
	if s.DB == nil {
		return paginatedResponse{}, errors.New("database is unavailable")
	}

	whereSQL, args, err := s.buildListFilter(ctx, userID, role, status)
	if err != nil {
		return paginatedResponse{}, err
	}
	if whereSQL == "" {
		return paginatedResponse{Data: []checkoutEntry{}, Total: 0}, nil
	}

	countSQL := `SELECT COUNT(*)::int FROM "SecretCheckoutRequest" cr WHERE ` + whereSQL
	var total int
	if err := s.DB.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return paginatedResponse{}, fmt.Errorf("count checkouts: %w", err)
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	rows, err := s.DB.Query(ctx, fmt.Sprintf(`
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
WHERE %s
ORDER BY cr."createdAt" DESC
LIMIT $%d OFFSET $%d
`, whereSQL, len(args)+1, len(args)+2), listArgs...)
	if err != nil {
		return paginatedResponse{}, fmt.Errorf("list checkouts: %w", err)
	}
	defer rows.Close()

	items := make([]checkoutEntry, 0)
	for rows.Next() {
		entry, err := scanCheckout(rows)
		if err != nil {
			return paginatedResponse{}, err
		}
		items = append(items, entry)
	}
	if err := rows.Err(); err != nil {
		return paginatedResponse{}, fmt.Errorf("iterate checkouts: %w", err)
	}

	if err := s.attachResourceNames(ctx, items); err != nil {
		return paginatedResponse{}, err
	}
	return paginatedResponse{Data: items, Total: total}, nil
}

func (s Service) buildListFilter(ctx context.Context, userID, role, status string) (string, []any, error) {
	args := make([]any, 0)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	var whereSQL string
	switch role {
	case "requester":
		whereSQL = `"requesterId" = ` + addArg(userID)
	case "approver":
		secretIDs, connectionIDs, err := s.approvableResourceIDs(ctx, userID)
		if err != nil {
			return "", nil, err
		}
		orConditions := make([]string, 0, 2)
		requesterArg := addArg(userID)
		if len(secretIDs) > 0 {
			orConditions = append(orConditions, fmt.Sprintf(`("secretId" = ANY(%s) AND "requesterId" <> %s)`, addArg(secretIDs), requesterArg))
		}
		if len(connectionIDs) > 0 {
			orConditions = append(orConditions, fmt.Sprintf(`("connectionId" = ANY(%s) AND "requesterId" <> %s)`, addArg(connectionIDs), requesterArg))
		}
		if len(orConditions) == 0 {
			return "", nil, nil
		}
		whereSQL = "(" + strings.Join(orConditions, " OR ") + ")"
	default:
		userArg := addArg(userID)
		whereSQL = fmt.Sprintf(`("requesterId" = %s OR "approverId" = %s)`, userArg, userArg)
	}

	if status != "" {
		whereSQL += ` AND status = ` + addArg(status) + `::"CheckoutStatus"`
	}
	return whereSQL, args, nil
}

func (s Service) approvableResourceIDs(ctx context.Context, userID string) ([]string, []string, error) {
	ownedSecretIDs, err := s.listIDs(ctx, `SELECT id FROM "VaultSecret" WHERE "userId" = $1`, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("list owned secrets: %w", err)
	}

	adminTenantIDs, err := s.listIDs(ctx, `SELECT "tenantId" FROM "TenantMember" WHERE "userId" = $1 AND role IN ('OWNER', 'ADMIN')`, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("list admin tenants: %w", err)
	}
	tenantSecretIDs := make([]string, 0)
	if len(adminTenantIDs) > 0 {
		tenantSecretIDs, err = s.listIDs(ctx, `SELECT id FROM "VaultSecret" WHERE "tenantId" = ANY($1)`, adminTenantIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("list tenant secrets: %w", err)
		}
	}

	ownedConnectionIDs, err := s.listIDs(ctx, `SELECT id FROM "Connection" WHERE "userId" = $1`, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("list owned connections: %w", err)
	}

	adminTeamIDs, err := s.listIDs(ctx, `SELECT "teamId" FROM "TeamMember" WHERE "userId" = $1 AND role = 'TEAM_ADMIN'`, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("list admin teams: %w", err)
	}
	teamConnectionIDs := make([]string, 0)
	if len(adminTeamIDs) > 0 {
		teamConnectionIDs, err = s.listIDs(ctx, `SELECT id FROM "Connection" WHERE "teamId" = ANY($1)`, adminTeamIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("list team connections: %w", err)
		}
	}

	return uniqueStrings(append(ownedSecretIDs, tenantSecretIDs...)), uniqueStrings(append(ownedConnectionIDs, teamConnectionIDs...)), nil
}

func (s Service) userCanApproveResource(ctx context.Context, userID string, secretID, connectionID *string) (bool, error) {
	if secretID != nil {
		var ownerID string
		var tenantID sql.NullString
		if err := s.DB.QueryRow(ctx, `SELECT "userId", "tenantId" FROM "VaultSecret" WHERE id = $1`, *secretID).Scan(&ownerID, &tenantID); err == nil {
			if ownerID == userID {
				return true, nil
			}
			if tenantID.Valid {
				var exists bool
				if err := s.DB.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM "TenantMember"
  WHERE "tenantId" = $1 AND "userId" = $2 AND role IN ('OWNER', 'ADMIN')
)
`, tenantID.String, userID).Scan(&exists); err != nil {
					return false, fmt.Errorf("check secret approver membership: %w", err)
				}
				if exists {
					return true, nil
				}
			}
		}
	}
	if connectionID != nil {
		var ownerID string
		var teamID sql.NullString
		if err := s.DB.QueryRow(ctx, `SELECT "userId", "teamId" FROM "Connection" WHERE id = $1`, *connectionID).Scan(&ownerID, &teamID); err == nil {
			if ownerID == userID {
				return true, nil
			}
			if teamID.Valid {
				var exists bool
				if err := s.DB.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM "TeamMember"
  WHERE "teamId" = $1 AND "userId" = $2 AND role = 'TEAM_ADMIN'
)
`, teamID.String, userID).Scan(&exists); err != nil {
					return false, fmt.Errorf("check connection approver membership: %w", err)
				}
				if exists {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
