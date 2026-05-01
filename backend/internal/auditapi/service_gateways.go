package auditapi

import (
	"context"
	"fmt"
)

func (s Service) ListGateways(ctx context.Context, userID string) ([]auditGateway, error) {
	rows, err := s.DB.Query(ctx, `
SELECT DISTINCT a."gatewayId", g.name
FROM "AuditLog" a
LEFT JOIN "Gateway" g ON g.id = a."gatewayId"
WHERE a."userId" = $1
  AND a."gatewayId" IS NOT NULL
ORDER BY a."gatewayId" ASC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("list audit gateways: %w", err)
	}
	defer rows.Close()
	return collectGateways(rows)
}

func (s Service) ListTenantGateways(ctx context.Context, tenantID string) ([]auditGateway, error) {
	rows, err := s.DB.Query(ctx, `
SELECT DISTINCT a."gatewayId", g.name
FROM "AuditLog" a
JOIN "TenantMember" tm ON tm."userId" = a."userId"
LEFT JOIN "Gateway" g ON g.id = a."gatewayId"
WHERE tm."tenantId" = $1
  AND a."gatewayId" IS NOT NULL
ORDER BY a."gatewayId" ASC
`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list tenant audit gateways: %w", err)
	}
	defer rows.Close()
	return collectGateways(rows)
}
