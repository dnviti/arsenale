package gateways

import (
	"testing"
	"time"
)

func TestDeriveGatewayOperationalStateTunnelHealthy(t *testing.T) {
	connectedAt := time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC)
	item := gatewayRecord{TunnelEnabled: true}

	status, reason, connected, gotConnectedAt := deriveGatewayOperationalState(
		item,
		tunnelStatusSnapshot{
			Connected:      true,
			ConnectedAt:    &connectedAt,
			HeartbeatKnown: true,
			HeartbeatOk:    true,
		},
		true,
		true,
	)

	if status != gatewayOperationalHealthy {
		t.Fatalf("expected healthy, got %q", status)
	}
	if reason == "" {
		t.Fatal("expected non-empty reason")
	}
	if !connected {
		t.Fatal("expected connected tunnel")
	}
	if gotConnectedAt == nil || !gotConnectedAt.Equal(connectedAt) {
		t.Fatalf("unexpected connectedAt: %#v", gotConnectedAt)
	}
}

func TestDeriveGatewayOperationalStateTunnelHeartbeatMissing(t *testing.T) {
	item := gatewayRecord{TunnelEnabled: true}

	status, _, _, _ := deriveGatewayOperationalState(
		item,
		tunnelStatusSnapshot{Connected: true},
		true,
		true,
	)

	if status != gatewayOperationalDegraded {
		t.Fatalf("expected degraded, got %q", status)
	}
}

func TestDeriveGatewayOperationalStateTunnelHeartbeatUnhealthy(t *testing.T) {
	item := gatewayRecord{TunnelEnabled: true}

	status, _, _, _ := deriveGatewayOperationalState(
		item,
		tunnelStatusSnapshot{
			Connected:      true,
			HeartbeatKnown: true,
			HeartbeatOk:    false,
		},
		true,
		true,
	)

	if status != gatewayOperationalUnhealthy {
		t.Fatalf("expected unhealthy, got %q", status)
	}
}

func TestDeriveGatewayOperationalStateTunnelDisconnected(t *testing.T) {
	item := gatewayRecord{TunnelEnabled: true}

	status, _, connected, _ := deriveGatewayOperationalState(item, tunnelStatusSnapshot{}, false, true)

	if status != gatewayOperationalUnhealthy {
		t.Fatalf("expected unhealthy, got %q", status)
	}
	if connected {
		t.Fatal("expected disconnected tunnel")
	}
}

func TestDeriveGatewayReportedHealthTunnelHealthy(t *testing.T) {
	heartbeatAt := time.Date(2026, 4, 9, 11, 0, 0, 0, time.UTC)
	item := gatewayRecord{
		TunnelEnabled:    true,
		LastHealthStatus: "UNKNOWN",
	}

	health := deriveGatewayReportedHealth(
		item,
		tunnelStatusSnapshot{
			Connected:      true,
			LastHeartbeat:  &heartbeatAt,
			HeartbeatKnown: true,
			HeartbeatOk:    true,
		},
		true,
		true,
		gatewayOperationalHealthy,
		"Tunnel is healthy.",
		&heartbeatAt,
	)

	if health.Status != "REACHABLE" {
		t.Fatalf("expected reachable, got %q", health.Status)
	}
	if health.CheckedAt == nil || !health.CheckedAt.Equal(heartbeatAt) {
		t.Fatalf("unexpected checkedAt: %#v", health.CheckedAt)
	}
	if health.Error != nil {
		t.Fatalf("expected nil error, got %#v", health.Error)
	}
}

func TestDeriveGatewayReportedInstanceCountsTunnelHealthy(t *testing.T) {
	total, healthy, running := deriveGatewayReportedInstanceCounts(
		gatewayRecord{TunnelEnabled: true},
		tunnelStatusSnapshot{
			Connected:      true,
			HeartbeatKnown: true,
			HeartbeatOk:    true,
		},
		true,
		true,
	)

	if total != 1 || healthy != 1 || running != 1 {
		t.Fatalf("unexpected tunnel counts: total=%d healthy=%d running=%d", total, healthy, running)
	}
}

func TestDeriveGatewayOperationalStateManagedGroupStates(t *testing.T) {
	tests := []struct {
		name   string
		record gatewayRecord
		want   string
	}{
		{
			name: "all healthy",
			record: gatewayRecord{
				DeploymentMode:   "MANAGED_GROUP",
				DesiredReplicas:  2,
				TotalInstances:   2,
				HealthyInstances: 2,
			},
			want: gatewayOperationalHealthy,
		},
		{
			name: "partial healthy",
			record: gatewayRecord{
				DeploymentMode:   "MANAGED_GROUP",
				DesiredReplicas:  2,
				TotalInstances:   2,
				HealthyInstances: 1,
			},
			want: gatewayOperationalDegraded,
		},
		{
			name: "no healthy with desired replicas",
			record: gatewayRecord{
				DeploymentMode:   "MANAGED_GROUP",
				DesiredReplicas:  2,
				TotalInstances:   2,
				HealthyInstances: 0,
			},
			want: gatewayOperationalUnhealthy,
		},
		{
			name: "not deployed",
			record: gatewayRecord{
				DeploymentMode:   "MANAGED_GROUP",
				DesiredReplicas:  0,
				TotalInstances:   0,
				HealthyInstances: 0,
			},
			want: gatewayOperationalUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _, _, _ := deriveGatewayOperationalState(tt.record, tunnelStatusSnapshot{}, false, false)
			if status != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, status)
			}
		})
	}
}

func TestDeriveGatewayOperationalStateSingleInstanceStates(t *testing.T) {
	latency := 12
	errMessage := "connection refused"

	tests := []struct {
		name   string
		record gatewayRecord
		want   string
	}{
		{
			name: "reachable",
			record: gatewayRecord{
				LastHealthStatus:  "REACHABLE",
				LastLatencyMS:     &latency,
				MonitoringEnabled: true,
			},
			want: gatewayOperationalHealthy,
		},
		{
			name: "unreachable",
			record: gatewayRecord{
				LastHealthStatus:  "UNREACHABLE",
				LastError:         &errMessage,
				MonitoringEnabled: true,
			},
			want: gatewayOperationalUnhealthy,
		},
		{
			name: "unknown",
			record: gatewayRecord{
				LastHealthStatus:  "UNKNOWN",
				MonitoringEnabled: true,
			},
			want: gatewayOperationalUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _, _, _ := deriveGatewayOperationalState(tt.record, tunnelStatusSnapshot{}, false, false)
			if status != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, status)
			}
		})
	}
}
