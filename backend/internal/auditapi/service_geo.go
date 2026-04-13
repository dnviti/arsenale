package auditapi

import (
	"context"
	"fmt"
	"time"
)

func (s Service) ListCountries(ctx context.Context, userID string) ([]string, error) {
	return s.listCountriesByQuery(ctx, `
SELECT DISTINCT "geoCountry"
FROM "AuditLog"
WHERE "userId" = $1
  AND "geoCountry" IS NOT NULL
ORDER BY "geoCountry" ASC
`, userID)
}

func (s Service) ListTenantCountries(ctx context.Context, tenantID string) ([]string, error) {
	return s.listCountriesByQuery(ctx, `
SELECT DISTINCT a."geoCountry"
FROM "AuditLog" a
JOIN "TenantMember" tm ON tm."userId" = a."userId"
WHERE tm."tenantId" = $1
  AND a."geoCountry" IS NOT NULL
ORDER BY a."geoCountry" ASC
`, tenantID)
}

func (s Service) GetTenantGeoSummary(ctx context.Context, tenantID string, query auditQuery, days int) ([]geoSummaryPoint, error) {
	if query.StartDate == nil && query.EndDate == nil && days > 0 {
		start := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
		query.StartDate = &start
	}

	baseArgs := []any{tenantID}
	filterSQL, filterArgs := buildAuditFilters("a", query, len(baseArgs)+1)
	args := append(baseArgs, filterArgs...)

	rows, err := s.DB.Query(ctx, `
SELECT a."geoCountry", COALESCE(a."geoCity", ''), a."geoCoords", COUNT(*)::int, MAX(a."createdAt")
FROM "AuditLog" a
JOIN "TenantMember" tm
  ON tm."userId" = a."userId"
 AND tm."tenantId" = $1
 AND tm.status = 'ACCEPTED'
WHERE a."geoCountry" IS NOT NULL
  AND cardinality(a."geoCoords") >= 2
`+filterSQL+`
GROUP BY a."geoCountry", a."geoCity", a."geoCoords"
ORDER BY MAX(a."createdAt") DESC
`, args...)
	if err != nil {
		return nil, fmt.Errorf("list tenant geo summary: %w", err)
	}
	defer rows.Close()

	points := make([]geoSummaryPoint, 0)
	for rows.Next() {
		var (
			item   geoSummaryPoint
			coords []float64
		)
		if err := rows.Scan(&item.Country, &item.City, &coords, &item.Count, &item.LastSeen); err != nil {
			return nil, fmt.Errorf("scan tenant geo summary: %w", err)
		}
		if len(coords) < 2 {
			continue
		}
		item.Lat = coords[0]
		item.Lng = coords[1]
		points = append(points, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenant geo summary: %w", err)
	}
	return points, nil
}
