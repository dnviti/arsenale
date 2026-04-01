package gateways

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

type gatewayProbeTarget struct {
	Host string
	Port int
}

func (s Service) TestGatewayConnectivity(ctx context.Context, tenantID, gatewayID string) (connectivityResult, error) {
	if s.DB == nil {
		return connectivityResult{}, fmt.Errorf("database is unavailable")
	}

	record, err := s.loadGateway(ctx, tenantID, gatewayID)
	if err != nil {
		return connectivityResult{}, err
	}
	host := record.Host
	port := record.Port
	deploymentMode := record.DeploymentMode

	if strings.EqualFold(strings.TrimSpace(deploymentMode), "MANAGED_GROUP") {
		var instanceHost, instanceContainerID, instanceContainerName string
		var instancePort int
		err := s.DB.QueryRow(ctx, `
SELECT host, port, "containerId", "containerName"
  FROM "ManagedGatewayInstance"
 WHERE "gatewayId" = $1
   AND status = 'RUNNING'
 ORDER BY "createdAt" ASC
 LIMIT 1
`, gatewayID).Scan(&instanceHost, &instancePort, &instanceContainerID, &instanceContainerName)
		if err == nil {
			instanceIPAddress := ""
			if runtimeClient, _, runtimeErr := s.managedGatewayRuntime(ctx); runtimeErr == nil && strings.TrimSpace(instanceContainerID) != "" {
				if info, inspectErr := runtimeClient.inspectContainer(ctx, instanceContainerID); inspectErr == nil {
					instanceIPAddress = info.IPAddress
				}
			}
			targets := buildManagedGatewayProbeTargets(record.Host, record.Port, instanceHost, instancePort, instanceContainerName, instanceIPAddress)
			var result connectivityResult
			for _, target := range targets {
				result = tcpProbe(ctx, target.Host, target.Port, 5*time.Second)
				if result.Reachable {
					return s.recordGatewayConnectivity(ctx, gatewayID, result)
				}
			}
			return s.recordGatewayConnectivity(ctx, gatewayID, result)
		} else {
			message := "No deployed instances for this gateway group"
			result := connectivityResult{
				Reachable: false,
				Error:     &message,
			}
			if _, updateErr := s.DB.Exec(ctx, `
UPDATE "Gateway"
   SET "lastHealthStatus" = 'UNREACHABLE'::"GatewayHealthStatus",
       "lastCheckedAt" = NOW(),
       "lastLatencyMs" = NULL,
       "lastError" = $2,
       "updatedAt" = NOW()
 WHERE id = $1
`, gatewayID, message); updateErr != nil {
				return connectivityResult{}, fmt.Errorf("update gateway health: %w", updateErr)
			}
			return result, nil
		}
	}

	result := tcpProbe(ctx, host, port, 5*time.Second)
	return s.recordGatewayConnectivity(ctx, gatewayID, result)
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

func buildManagedGatewayProbeTargets(gatewayHost string, gatewayPort int, instanceHost string, instancePort int, instanceContainerName, instanceIPAddress string) []gatewayProbeTarget {
	targets := make([]gatewayProbeTarget, 0, 3)
	seen := make(map[string]struct{})

	add := func(host string, port int) {
		host = strings.TrimSpace(host)
		if host == "" || port < 1 {
			return
		}
		key := fmt.Sprintf("%s:%d", host, port)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		targets = append(targets, gatewayProbeTarget{Host: host, Port: port})
	}

	add(instanceIPAddress, gatewayPort)
	add(instanceHost, instancePort)
	add(instanceContainerName, gatewayPort)
	add(gatewayHost, gatewayPort)

	return targets
}

func tcpProbe(ctx context.Context, host string, port int, timeout time.Duration) connectivityResult {
	start := time.Now()
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)))
	if err != nil {
		message := err.Error()
		return connectivityResult{
			Reachable: false,
			Error:     &message,
		}
	}
	_ = conn.Close()
	latency := int(time.Since(start).Milliseconds())
	return connectivityResult{
		Reachable: true,
		LatencyMS: &latency,
	}
}
