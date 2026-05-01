package gateways

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type managedGatewayInstanceResponse struct {
	ID                  string     `json:"id"`
	GatewayID           string     `json:"gatewayId"`
	ContainerID         string     `json:"containerId"`
	ContainerName       string     `json:"containerName"`
	Host                string     `json:"host"`
	Port                int        `json:"port"`
	APIPort             *int       `json:"apiPort"`
	Status              string     `json:"status"`
	OrchestratorType    string     `json:"orchestratorType"`
	HealthStatus        *string    `json:"healthStatus"`
	LastHealthCheck     *time.Time `json:"lastHealthCheck"`
	ErrorMessage        *string    `json:"errorMessage"`
	ConsecutiveFailures int        `json:"consecutiveFailures"`
	TunnelProxyHost     *string    `json:"tunnelProxyHost"`
	TunnelProxyPort     *int       `json:"tunnelProxyPort"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

func (s Service) ListGatewayInstances(ctx context.Context, tenantID, gatewayID string) ([]managedGatewayInstanceResponse, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("database is unavailable")
	}

	if _, err := s.loadGateway(ctx, tenantID, gatewayID); err != nil {
		return nil, err
	}

	rows, err := s.DB.Query(ctx, `
SELECT
	id,
	"gatewayId",
	"containerId",
	"containerName",
	host,
	port,
	"apiPort",
	status::text,
	"orchestratorType",
	"healthStatus",
	"lastHealthCheck",
	"errorMessage",
	"consecutiveFailures",
	"tunnelProxyHost",
	"tunnelProxyPort",
	"createdAt",
	"updatedAt"
FROM "ManagedGatewayInstance"
WHERE "gatewayId" = $1
ORDER BY "createdAt" ASC
`, gatewayID)
	if err != nil {
		return nil, fmt.Errorf("list managed gateway instances: %w", err)
	}
	defer rows.Close()

	result := make([]managedGatewayInstanceResponse, 0)
	for rows.Next() {
		item, err := scanManagedGatewayInstance(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate managed gateway instances: %w", err)
	}
	return result, nil
}

func scanManagedGatewayInstance(row rowScanner) (managedGatewayInstanceResponse, error) {
	var item managedGatewayInstanceResponse
	var apiPort, tunnelProxyPort sql.NullInt32
	var healthStatus, errorMessage, tunnelProxyHost sql.NullString
	var lastHealthCheck sql.NullTime

	if err := row.Scan(
		&item.ID,
		&item.GatewayID,
		&item.ContainerID,
		&item.ContainerName,
		&item.Host,
		&item.Port,
		&apiPort,
		&item.Status,
		&item.OrchestratorType,
		&healthStatus,
		&lastHealthCheck,
		&errorMessage,
		&item.ConsecutiveFailures,
		&tunnelProxyHost,
		&tunnelProxyPort,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return managedGatewayInstanceResponse{}, fmt.Errorf("scan managed gateway instance: %w", err)
	}

	item.APIPort = nullIntPtr(apiPort)
	item.HealthStatus = nullStringPtr(healthStatus)
	item.LastHealthCheck = nullTimePtr(lastHealthCheck)
	item.ErrorMessage = nullStringPtr(errorMessage)
	item.TunnelProxyHost = nullStringPtr(tunnelProxyHost)
	item.TunnelProxyPort = nullIntPtr(tunnelProxyPort)

	return item, nil
}
