package gateways

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	gatewayOperationalHealthy   = "HEALTHY"
	gatewayOperationalDegraded  = "DEGRADED"
	gatewayOperationalUnhealthy = "UNHEALTHY"
	gatewayOperationalUnknown   = "UNKNOWN"
)

type tunnelStatusSnapshot struct {
	Connected      bool
	ConnectedAt    *time.Time
	LastHeartbeat  *time.Time
	HeartbeatKnown bool
	HeartbeatOk    bool
}

type gatewayReportedHealth struct {
	Status    string
	CheckedAt *time.Time
	LatencyMS *int
	Error     *string
}

func (s Service) loadTunnelStatusSnapshots(ctx context.Context) (map[string]tunnelStatusSnapshot, bool) {
	statuses, err := s.listTunnelStatuses(ctx)
	if err != nil {
		return map[string]tunnelStatusSnapshot{}, false
	}

	result := make(map[string]tunnelStatusSnapshot, len(statuses))
	for _, status := range statuses {
		snapshot := tunnelStatusSnapshot{
			Connected:     status.Connected,
			ConnectedAt:   parseBrokerTime(status.ConnectedAt),
			LastHeartbeat: parseBrokerTime(status.LastHeartbeatAt),
		}
		if status.Heartbeat != nil {
			snapshot.HeartbeatKnown = true
			snapshot.HeartbeatOk = status.Heartbeat.Healthy
		}
		result[strings.TrimSpace(status.GatewayID)] = snapshot
	}
	return result, true
}

func deriveGatewayOperationalState(
	item gatewayRecord,
	tunnelStatus tunnelStatusSnapshot,
	hasTunnelStatus bool,
	tunnelBrokerAvailable bool,
) (status string, reason string, tunnelConnected bool, tunnelConnectedAt *time.Time) {
	if item.TunnelEnabled {
		if tunnelBrokerAvailable {
			tunnelConnected = hasTunnelStatus && tunnelStatus.Connected
			tunnelConnectedAt = tunnelStatus.ConnectedAt

			switch {
			case !tunnelConnected:
				return gatewayOperationalUnhealthy, "Tunnel is enabled but not connected.", tunnelConnected, tunnelConnectedAt
			case !tunnelStatus.HeartbeatKnown:
				return gatewayOperationalDegraded, "Tunnel is connected but has not reported a health heartbeat yet.", tunnelConnected, tunnelConnectedAt
			case !tunnelStatus.HeartbeatOk:
				return gatewayOperationalUnhealthy, "Tunnel heartbeat reports the gateway as unhealthy.", tunnelConnected, tunnelConnectedAt
			default:
				return gatewayOperationalHealthy, "Tunnel is connected and reporting a healthy heartbeat.", tunnelConnected, tunnelConnectedAt
			}
		}

		return gatewayOperationalUnknown, "Tunnel broker status is unavailable.", item.TunnelConnected, item.TunnelConnectedAt
	}

	if deploymentModeIsGroup(item.DeploymentMode) {
		switch {
		case item.TotalInstances == 0 && item.DesiredReplicas == 0:
			return gatewayOperationalUnknown, "No managed instances are currently deployed.", item.TunnelConnected, item.TunnelConnectedAt
		case item.TotalInstances == 0:
			return gatewayOperationalUnhealthy, "No managed instances are currently registered.", item.TunnelConnected, item.TunnelConnectedAt
		case item.HealthyInstances == item.TotalInstances:
			return gatewayOperationalHealthy, fmt.Sprintf("%d/%d managed instances are healthy.", item.HealthyInstances, item.TotalInstances), item.TunnelConnected, item.TunnelConnectedAt
		case item.HealthyInstances > 0:
			return gatewayOperationalDegraded, fmt.Sprintf("%d/%d managed instances are healthy.", item.HealthyInstances, item.TotalInstances), item.TunnelConnected, item.TunnelConnectedAt
		default:
			return gatewayOperationalUnhealthy, fmt.Sprintf("0/%d managed instances are healthy.", item.TotalInstances), item.TunnelConnected, item.TunnelConnectedAt
		}
	}

	switch strings.TrimSpace(item.LastHealthStatus) {
	case "REACHABLE":
		if item.LastLatencyMS != nil {
			return gatewayOperationalHealthy, fmt.Sprintf("Gateway responded in %d ms.", *item.LastLatencyMS), item.TunnelConnected, item.TunnelConnectedAt
		}
		return gatewayOperationalHealthy, "Gateway responded to the latest health check.", item.TunnelConnected, item.TunnelConnectedAt
	case "UNREACHABLE":
		if item.LastError != nil && strings.TrimSpace(*item.LastError) != "" {
			return gatewayOperationalUnhealthy, strings.TrimSpace(*item.LastError), item.TunnelConnected, item.TunnelConnectedAt
		}
		return gatewayOperationalUnhealthy, "The latest health check failed.", item.TunnelConnected, item.TunnelConnectedAt
	default:
		if item.MonitoringEnabled {
			return gatewayOperationalUnknown, "Health monitoring is enabled but no completed check is available yet.", item.TunnelConnected, item.TunnelConnectedAt
		}
		return gatewayOperationalUnknown, "Automatic monitoring is disabled for this gateway.", item.TunnelConnected, item.TunnelConnectedAt
	}
}

func deriveGatewayReportedHealth(
	item gatewayRecord,
	tunnelStatus tunnelStatusSnapshot,
	hasTunnelStatus bool,
	tunnelBrokerAvailable bool,
	operationalStatus string,
	operationalReason string,
	tunnelConnectedAt *time.Time,
) gatewayReportedHealth {
	result := gatewayReportedHealth{
		Status:    item.LastHealthStatus,
		CheckedAt: item.LastCheckedAt,
		LatencyMS: item.LastLatencyMS,
		Error:     item.LastError,
	}
	if !item.TunnelEnabled || !tunnelBrokerAvailable {
		return result
	}

	result.CheckedAt = tunnelConnectedAt
	if tunnelStatus.LastHeartbeat != nil {
		result.CheckedAt = tunnelStatus.LastHeartbeat
	}
	result.LatencyMS = nil
	result.Error = nil

	switch operationalStatus {
	case gatewayOperationalHealthy:
		result.Status = "REACHABLE"
	case gatewayOperationalUnhealthy:
		result.Status = "UNREACHABLE"
		reason := operationalReason
		result.Error = &reason
	default:
		result.Status = "UNKNOWN"
		if !hasTunnelStatus {
			result.CheckedAt = item.LastCheckedAt
		}
	}
	return result
}

func deriveGatewayReportedInstanceCounts(
	item gatewayRecord,
	tunnelStatus tunnelStatusSnapshot,
	hasTunnelStatus bool,
	tunnelBrokerAvailable bool,
) (total int, healthy int, running int) {
	total = item.TotalInstances
	healthy = item.HealthyInstances
	running = item.RunningInstances

	if !item.TunnelEnabled || !tunnelBrokerAvailable || !hasTunnelStatus || !tunnelStatus.Connected {
		return total, healthy, running
	}

	if total < 1 {
		total = 1
	}
	if running < 1 {
		running = 1
	}
	if tunnelStatus.HeartbeatKnown && tunnelStatus.HeartbeatOk && healthy < 1 {
		healthy = 1
	}
	return total, healthy, running
}
