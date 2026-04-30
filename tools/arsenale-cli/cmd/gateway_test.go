package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
