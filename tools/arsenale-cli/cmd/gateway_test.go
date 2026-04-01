package cmd

import "testing"

func TestSelectGatewayInstancePrefersHealthyRunningNewest(t *testing.T) {
	instances := []map[string]any{
		{
			"id":           "old-running",
			"status":       "RUNNING",
			"healthStatus": "healthy",
			"updatedAt":    "2026-04-01T10:00:00Z",
		},
		{
			"id":           "new-running",
			"status":       "RUNNING",
			"healthStatus": "healthy",
			"updatedAt":    "2026-04-01T11:00:00Z",
		},
		{
			"id":           "stopped",
			"status":       "STOPPED",
			"healthStatus": "unhealthy",
			"updatedAt":    "2026-04-01T12:00:00Z",
		},
	}

	selected, err := selectGatewayInstance(instances, "")
	if err != nil {
		t.Fatalf("selectGatewayInstance returned error: %v", err)
	}
	if got := formatValue(selected["id"]); got != "new-running" {
		t.Fatalf("expected new-running, got %s", got)
	}
}

func TestSelectGatewayInstanceReturnsRequestedID(t *testing.T) {
	instances := []map[string]any{
		{"id": "one", "status": "RUNNING"},
		{"id": "two", "status": "STOPPED"},
	}

	selected, err := selectGatewayInstance(instances, "two")
	if err != nil {
		t.Fatalf("selectGatewayInstance returned error: %v", err)
	}
	if got := formatValue(selected["id"]); got != "two" {
		t.Fatalf("expected two, got %s", got)
	}
}

func TestGatewayInstanceRankOrdersRunningHealthyHighest(t *testing.T) {
	tests := []struct {
		name     string
		instance map[string]any
		want     int
	}{
		{
			name:     "running healthy",
			instance: map[string]any{"status": "RUNNING", "healthStatus": "healthy"},
			want:     3,
		},
		{
			name:     "running only",
			instance: map[string]any{"status": "RUNNING", "healthStatus": "starting"},
			want:     2,
		},
		{
			name:     "healthy only",
			instance: map[string]any{"status": "STOPPED", "healthStatus": "healthy"},
			want:     1,
		},
		{
			name:     "other",
			instance: map[string]any{"status": "STOPPED", "healthStatus": "unhealthy"},
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
