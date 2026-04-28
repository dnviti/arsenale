package cmd

import "testing"

func TestSelectGatewayInstancePrefersHealthyRunningNewest(t *testing.T) {
	instances := []gatewayInstance{
		{
			ID:           "old-running",
			Status:       "RUNNING",
			HealthStatus: "healthy",
			UpdatedAt:    "2026-04-01T10:00:00Z",
		},
		{
			ID:           "new-running",
			Status:       "RUNNING",
			HealthStatus: "healthy",
			UpdatedAt:    "2026-04-01T11:00:00Z",
		},
		{
			ID:           "stopped",
			Status:       "STOPPED",
			HealthStatus: "unhealthy",
			UpdatedAt:    "2026-04-01T12:00:00Z",
		},
	}

	selected, err := selectGatewayInstance(instances, "")
	if err != nil {
		t.Fatalf("selectGatewayInstance returned error: %v", err)
	}
	if got := selected.ID; got != "new-running" {
		t.Fatalf("expected new-running, got %s", got)
	}
}

func TestSelectGatewayInstanceReturnsRequestedID(t *testing.T) {
	instances := []gatewayInstance{
		{ID: "one", Status: "RUNNING"},
		{ID: "two", Status: "STOPPED"},
	}

	selected, err := selectGatewayInstance(instances, "two")
	if err != nil {
		t.Fatalf("selectGatewayInstance returned error: %v", err)
	}
	if got := selected.ID; got != "two" {
		t.Fatalf("expected two, got %s", got)
	}
}

func TestGatewayInstanceRankOrdersRunningHealthyHighest(t *testing.T) {
	tests := []struct {
		name     string
		instance gatewayInstance
		want     int
	}{
		{
			name:     "running healthy",
			instance: gatewayInstance{Status: "RUNNING", HealthStatus: "healthy"},
			want:     3,
		},
		{
			name:     "running only",
			instance: gatewayInstance{Status: "RUNNING", HealthStatus: "starting"},
			want:     2,
		},
		{
			name:     "healthy only",
			instance: gatewayInstance{Status: "STOPPED", HealthStatus: "healthy"},
			want:     1,
		},
		{
			name:     "other",
			instance: gatewayInstance{Status: "STOPPED", HealthStatus: "unhealthy"},
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gatewayInstanceRank(tt.instance); got != tt.want {
				t.Fatalf("expected rank %d, got %d", tt.want, got)
			}
		})
	}
}
