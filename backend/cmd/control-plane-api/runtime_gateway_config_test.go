package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadGatewayRuntimeConfigDerivesGRPCFallbackPaths(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "tls-cert.pem")
	keyPath := filepath.Join(dir, "tls-key.pem")
	for _, path := range []string{certPath, keyPath, filepath.Join(dir, "client-ca.pem"), filepath.Join(dir, "server-cert.pem"), filepath.Join(dir, "server-key.pem")} {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	t.Setenv("GATEWAY_GRPC_TLS_CERT", certPath)
	t.Setenv("GATEWAY_GRPC_TLS_KEY", keyPath)
	t.Setenv("GATEWAY_GRPC_CLIENT_CA", "")
	t.Setenv("GATEWAY_GRPC_SERVER_CERT", "")
	t.Setenv("GATEWAY_GRPC_SERVER_KEY", "")

	cfg := loadGatewayRuntimeConfig()
	if cfg.GRPCClientCA != filepath.Join(dir, "client-ca.pem") {
		t.Fatalf("unexpected client ca %q", cfg.GRPCClientCA)
	}
	if cfg.GRPCServerCert != filepath.Join(dir, "server-cert.pem") {
		t.Fatalf("unexpected server cert %q", cfg.GRPCServerCert)
	}
	if cfg.GRPCServerKey != filepath.Join(dir, "server-key.pem") {
		t.Fatalf("unexpected server key %q", cfg.GRPCServerKey)
	}
}

func TestCloseRuntimeResourcesRunsInReverseOrder(t *testing.T) {
	t.Parallel()

	calls := []int{}
	closeRuntimeResources([]func(){
		func() { calls = append(calls, 1) },
		nil,
		func() { calls = append(calls, 3) },
	})

	if !reflect.DeepEqual(calls, []int{3, 1}) {
		t.Fatalf("unexpected close order %#v", calls)
	}
}
