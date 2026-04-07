package auditapi

import (
	"context"
	"fmt"
)

func (s Service) ListAuditLogs(ctx context.Context, userID string, query auditQuery) (paginatedAuditLogs, error) {
	baseSQL, args := buildAuditFilters("a", auditQuery{UserID: userID}, 1)
	filterSQL, filterArgs := buildAuditFilters("a", query, len(args)+1)
	args = append(args, filterArgs...)

	orderBy := orderByClause(query)
	limitOffsetSQL := fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)

	rows, err := s.DB.Query(ctx, `
SELECT
	a.id,
	a.action::text,
	a."targetType",
	a."targetId",
	COALESCE(a.details, 'null'::jsonb),
	a."ipAddress",
	a."gatewayId",
	a."geoCountry",
	a."geoCity",
	a."geoCoords",
	a.flags,
	a."createdAt"
FROM "AuditLog" a
WHERE 1 = 1`+baseSQL+filterSQL+orderBy+limitOffsetSQL, append(args, query.Limit, (query.Page-1)*query.Limit)...)
	if err != nil {
		return paginatedAuditLogs{}, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	items, err := collectAuditLogs(rows)
	if err != nil {
		return paginatedAuditLogs{}, err
	}

	var total int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*)::int FROM "AuditLog" a WHERE 1 = 1`+baseSQL+filterSQL, args...).Scan(&total); err != nil {
		return paginatedAuditLogs{}, fmt.Errorf("count audit logs: %w", err)
	}

	return paginatedAuditLogs{
		Data:       items,
		Total:      total,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages(total, query.Limit),
	}, nil
}

func (s Service) ListTenantAuditLogs(ctx context.Context, tenantID string, query auditQuery) (paginatedTenantAuditLogs, error) {
	baseArgs := []any{tenantID}
	filterSQL, filterArgs := buildAuditFilters("a", query, len(baseArgs)+1)
	args := append(baseArgs, filterArgs...)
	orderBy := orderByClause(query)
	limitOffsetSQL := fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)

	rows, err := s.DB.Query(ctx, `
SELECT
	a.id,
	a."userId",
	u.username,
	u.email,
	a.action::text,
	a."targetType",
	a."targetId",
	COALESCE(a.details, 'null'::jsonb),
	a."ipAddress",
	a."gatewayId",
	a."geoCountry",
	a."geoCity",
	a."geoCoords",
	a.flags,
	a."createdAt"
FROM "AuditLog" a
JOIN "TenantMember" tm
  ON tm."userId" = a."userId"
 AND tm."tenantId" = $1
 AND tm.status = 'ACCEPTED'
LEFT JOIN "User" u ON u.id = a."userId"
WHERE 1 = 1`+filterSQL+orderBy+limitOffsetSQL, append(args, query.Limit, (query.Page-1)*query.Limit)...)
	if err != nil {
		return paginatedTenantAuditLogs{}, fmt.Errorf("list tenant audit logs: %w", err)
	}
	defer rows.Close()

	items, err := collectTenantAuditLogs(rows)
	if err != nil {
		return paginatedTenantAuditLogs{}, err
	}

	var total int
	if err := s.DB.QueryRow(ctx, `
SELECT COUNT(*)::int
FROM "AuditLog" a
JOIN "TenantMember" tm
  ON tm."userId" = a."userId"
 AND tm."tenantId" = $1
 AND tm.status = 'ACCEPTED'
WHERE 1 = 1`+filterSQL, args...).Scan(&total); err != nil {
		return paginatedTenantAuditLogs{}, fmt.Errorf("count tenant audit logs: %w", err)
	}

	return paginatedTenantAuditLogs{
		Data:       items,
		Total:      total,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages(total, query.Limit),
	}, nil
}

func (s Service) ListConnectionAuditLogs(ctx context.Context, connectionID string, query auditQuery) (paginatedTenantAuditLogs, error) {
	baseArgs := []any{connectionID}
	connectionFilter := auditQuery{
		UserID:      query.UserID,
		Action:      query.Action,
		StartDate:   query.StartDate,
		EndDate:     query.EndDate,
		Search:      query.Search,
		IPAddress:   query.IPAddress,
		GatewayID:   query.GatewayID,
		GeoCountry:  query.GeoCountry,
		SortBy:      query.SortBy,
		SortOrder:   query.SortOrder,
		FlaggedOnly: query.FlaggedOnly,
	}
	filterSQL, filterArgs := buildAuditFilters("a", connectionFilter, len(baseArgs)+1)
	args := append(baseArgs, filterArgs...)
	orderBy := orderByClause(query)
	limitOffsetSQL := fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)

	rows, err := s.DB.Query(ctx, `
SELECT
	a.id,
	a."userId",
	u.username,
	u.email,
	a.action::text,
	a."targetType",
	a."targetId",
	COALESCE(a.details, 'null'::jsonb),
	a."ipAddress",
	a."gatewayId",
	a."geoCountry",
	a."geoCity",
	a."geoCoords",
	a.flags,
	a."createdAt"
FROM "AuditLog" a
LEFT JOIN "User" u ON u.id = a."userId"
WHERE a."targetId" = $1`+filterSQL+orderBy+limitOffsetSQL, append(args, query.Limit, (query.Page-1)*query.Limit)...)
	if err != nil {
		return paginatedTenantAuditLogs{}, fmt.Errorf("list connection audit logs: %w", err)
	}
	defer rows.Close()

	items, err := collectTenantAuditLogs(rows)
	if err != nil {
		return paginatedTenantAuditLogs{}, err
	}

	var total int
	if err := s.DB.QueryRow(ctx, `SELECT COUNT(*)::int FROM "AuditLog" a WHERE a."targetId" = $1`+filterSQL, args...).Scan(&total); err != nil {
		return paginatedTenantAuditLogs{}, fmt.Errorf("count connection audit logs: %w", err)
	}

	return paginatedTenantAuditLogs{
		Data:       items,
		Total:      total,
		Page:       query.Page,
		Limit:      query.Limit,
		TotalPages: totalPages(total, query.Limit),
	}, nil
}

func (s Service) ListConnectionUsers(ctx context.Context, connectionID string) ([]connectionAuditUser, error) {
	rows, err := s.DB.Query(ctx, `
SELECT DISTINCT u.id, u.username, u.email
FROM "AuditLog" a
JOIN "User" u ON u.id = a."userId"
WHERE a."targetId" = $1
  AND a."userId" IS NOT NULL
ORDER BY u.email ASC
`, connectionID)
	if err != nil {
		return nil, fmt.Errorf("list connection audit users: %w", err)
	}
	defer rows.Close()

	items := make([]connectionAuditUser, 0)
	for rows.Next() {
		var (
			item     connectionAuditUser
			username *string
		)
		if err := rows.Scan(&item.ID, &username, &item.Email); err != nil {
			return nil, fmt.Errorf("scan connection audit user: %w", err)
		}
		item.Username = username
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connection audit users: %w", err)
	}
	return items, nil
}
