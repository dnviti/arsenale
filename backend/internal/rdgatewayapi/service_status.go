package rdgatewayapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (s Service) GetStatus(ctx context.Context, tenantID string) (Status, error) {
	if s.DB == nil {
		return Status{}, errors.New("database is unavailable")
	}

	query := `
SELECT COUNT(*)
FROM "ActiveSession" s
JOIN "Connection" c ON c.id = s."connectionId"
LEFT JOIN "Team" t ON t.id = c."teamId"
WHERE s.status <> 'CLOSED'
  AND s.protocol = 'RDP'
  AND COALESCE(s.metadata->>'transport', '') = 'rdgw'
`
	args := []any{}
	if strings.TrimSpace(tenantID) != "" {
		query += ` AND (c."teamId" IS NULL OR t."tenantId" = $1 OR EXISTS (
SELECT 1
FROM "TenantMember" tm
WHERE tm."userId" = c."userId"
  AND tm."tenantId" = $1
  AND tm."isActive" = true
))`
		args = append(args, tenantID)
	}

	var active int
	if err := s.DB.QueryRow(ctx, query, args...).Scan(&active); err != nil {
		return Status{}, fmt.Errorf("count rd gateway sessions: %w", err)
	}
	return Status{
		ActiveTunnels:  active,
		ActiveChannels: active,
	}, nil
}
