package gateways

import (
	"context"
	"strings"
	"time"
)

const (
	gatewayReadProbeTimeout     = 2 * time.Second
	gatewayReadRefreshMinimum   = 15 * time.Second
	gatewayUnknownRefreshWindow = 5 * time.Second
)

func (s Service) refreshGatewayHealthIfNeeded(ctx context.Context, record gatewayRecord, now time.Time) (gatewayRecord, bool) {
	if !shouldRefreshGatewayHealth(record, now) {
		return record, false
	}

	result, err := s.testGatewayRecordConnectivity(ctx, record, gatewayReadProbeTimeout)
	if err != nil {
		return record, false
	}
	return applyConnectivityResult(record, result, now), true
}

func shouldRefreshGatewayHealth(record gatewayRecord, now time.Time) bool {
	if !record.MonitoringEnabled || record.TunnelEnabled || deploymentModeIsGroup(record.DeploymentMode) {
		return false
	}
	if record.LastCheckedAt == nil {
		return true
	}

	age := now.Sub(*record.LastCheckedAt)
	if age < 0 {
		return false
	}

	if !gatewayHealthStatusKnown(record.LastHealthStatus) {
		return age >= maxDuration(normalizedGatewayHealthInterval(record.MonitorIntervalMS), gatewayUnknownRefreshWindow)
	}

	return age >= maxDuration(normalizedGatewayHealthInterval(record.MonitorIntervalMS), gatewayReadRefreshMinimum)
}

func gatewayHealthStatusKnown(status string) bool {
	switch strings.TrimSpace(status) {
	case "REACHABLE", "UNREACHABLE":
		return true
	default:
		return false
	}
}

func normalizedGatewayHealthInterval(intervalMS int) time.Duration {
	if intervalMS <= 0 {
		return 5 * time.Second
	}
	return time.Duration(intervalMS) * time.Millisecond
}

func applyConnectivityResult(record gatewayRecord, result connectivityResult, checkedAt time.Time) gatewayRecord {
	record.LastCheckedAt = &checkedAt
	record.LastLatencyMS = result.LatencyMS
	record.LastError = result.Error

	if result.Reachable {
		record.LastHealthStatus = "REACHABLE"
		record.LastError = nil
		return record
	}

	record.LastHealthStatus = "UNREACHABLE"
	record.LastLatencyMS = nil
	return record
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
