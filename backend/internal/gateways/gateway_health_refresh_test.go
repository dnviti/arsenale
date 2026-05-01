package gateways

import (
	"testing"
	"time"
)

func TestShouldRefreshGatewayHealth(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	stale := now.Add(-20 * time.Second)
	recent := now.Add(-2 * time.Second)

	tests := []struct {
		name   string
		record gatewayRecord
		want   bool
	}{
		{
			name: "refreshes unknown single instance without a check",
			record: gatewayRecord{
				MonitoringEnabled: true,
				DeploymentMode:    "SINGLE_INSTANCE",
				LastHealthStatus:  "UNKNOWN",
			},
			want: true,
		},
		{
			name: "refreshes stale known status",
			record: gatewayRecord{
				MonitoringEnabled: true,
				DeploymentMode:    "SINGLE_INSTANCE",
				LastHealthStatus:  "REACHABLE",
				LastCheckedAt:     &stale,
				MonitorIntervalMS: 5000,
			},
			want: true,
		},
		{
			name: "skips recent known status",
			record: gatewayRecord{
				MonitoringEnabled: true,
				DeploymentMode:    "SINGLE_INSTANCE",
				LastHealthStatus:  "REACHABLE",
				LastCheckedAt:     &recent,
				MonitorIntervalMS: 5000,
			},
			want: false,
		},
		{
			name: "skips tunnel gateways",
			record: gatewayRecord{
				MonitoringEnabled: true,
				DeploymentMode:    "MANAGED_GROUP",
				TunnelEnabled:     true,
				LastHealthStatus:  "UNKNOWN",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRefreshGatewayHealth(tt.record, now); got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
