package gateways

import (
	"context"
	"fmt"
	"time"
)

func (s Service) TestGatewayConnectivity(ctx context.Context, tenantID, gatewayID string) (connectivityResult, error) {
	if s.DB == nil {
		return connectivityResult{}, fmt.Errorf("database is unavailable")
	}

	record, err := s.loadGateway(ctx, tenantID, gatewayID)
	if err != nil {
		return connectivityResult{}, err
	}
	return s.testGatewayRecordConnectivity(ctx, record, 5*time.Second)
}

func (s Service) testGatewayRecordConnectivity(ctx context.Context, record gatewayRecord, timeout time.Duration) (connectivityResult, error) {
	result, err := s.probeGatewayRecordConnectivity(ctx, record, timeout)
	if err != nil {
		return connectivityResult{}, err
	}
	return s.recordGatewayConnectivity(ctx, record.ID, result)
}

func (s Service) recordGatewayConnectivity(ctx context.Context, gatewayID string, result connectivityResult) (connectivityResult, error) {
	status := "UNREACHABLE"
	if result.Reachable {
		status = "REACHABLE"
	}
	if _, err := s.DB.Exec(ctx, `
UPDATE "Gateway"
   SET "lastHealthStatus" = $2::"GatewayHealthStatus",
       "lastCheckedAt" = NOW(),
       "lastLatencyMs" = $3,
       "lastError" = $4,
       "updatedAt" = NOW()
 WHERE id = $1
`, gatewayID, status, result.LatencyMS, result.Error); err != nil {
		return connectivityResult{}, fmt.Errorf("update gateway health: %w", err)
	}
	return result, nil
}
