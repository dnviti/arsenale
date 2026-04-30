package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestTunnelTokenEnvContent(t *testing.T) {
	bundle := tunnelTokenBundle{
		Token:            "tok",
		GatewayID:        "gw-1",
		TunnelLocalHost:  "127.0.0.1",
		TunnelLocalPort:  2222,
		TunnelClientCert: "cert",
		TunnelClientKey:  "key",
	}

	got := tunnelTokenEnvContent(bundle, "https://arsenale.example.com/", "./certs/client.pem", "./certs/client.key")
	for _, want := range []string{
		`TUNNEL_SERVER_URL="https://arsenale.example.com"`,
		`TUNNEL_TOKEN="tok"`,
		`TUNNEL_GATEWAY_ID="gw-1"`,
		`TUNNEL_LOCAL_HOST="127.0.0.1"`,
		`TUNNEL_LOCAL_PORT="2222"`,
		`TUNNEL_CLIENT_CERT_FILE="./certs/client.pem"`,
		`TUNNEL_CLIENT_KEY_FILE="./certs/client.key"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected env content to include %q, got:\n%s", want, got)
		}
	}
}

func TestTunnelTokenEnvContentUsesGatewayDefinitionFallback(t *testing.T) {
	bundle := tunnelTokenBundle{
		Token:            "tok",
		GatewayID:        "gw-db",
		GatewayType:      "DB_PROXY",
		TunnelClientCert: "cert",
		TunnelClientKey:  "key",
	}

	got := tunnelTokenEnvContent(bundle, "https://arsenale.example.com", "./certs/client.pem", "./certs/client.key")
	if !strings.Contains(got, `TUNNEL_LOCAL_PORT="5432"`) {
		t.Fatalf("expected DB proxy default port from gateway runtime definition, got:\n%s", got)
	}
}

func TestWriteTunnelTokenBundle(t *testing.T) {
	tempDir := t.TempDir()
	bundle := tunnelTokenBundle{
		Token:            "tok",
		GatewayID:        "gw-1",
		TunnelLocalHost:  "127.0.0.1",
		TunnelLocalPort:  4822,
		TunnelClientCert: "-----BEGIN CERTIFICATE-----\ncert\n-----END CERTIFICATE-----",
		TunnelClientKey:  "-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----",
	}

	envPath, err := writeTunnelTokenBundle(tempDir, bundle, "https://arsenale.example.com")
	if err != nil {
		t.Fatalf("writeTunnelTokenBundle returned error: %v", err)
	}
	if envPath != filepath.Join(tempDir, "tunnel.env") {
		t.Fatalf("unexpected env path %q", envPath)
	}

	for _, path := range []string{
		filepath.Join(tempDir, "certs", "tunnel-client-cert.pem"),
		filepath.Join(tempDir, "certs", "tunnel-client-key.pem"),
		filepath.Join(tempDir, "tunnel.env"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}
