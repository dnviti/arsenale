package gateways

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProbeTunnelGatewayConnectivityUsesBrokerProxy(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		conn, err := listener.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tcp-proxies" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["gatewayId"] != "gateway-1" {
			t.Fatalf("unexpected gatewayId: %#v", body["gatewayId"])
		}
		if body["targetHost"] != "127.0.0.1" {
			t.Fatalf("unexpected targetHost: %#v", body["targetHost"])
		}
		if int(body["targetPort"].(float64)) != 2222 {
			t.Fatalf("unexpected targetPort: %#v", body["targetPort"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"host": "127.0.0.1",
			"port": listener.Addr().(*net.TCPAddr).Port,
		})
	}))
	defer server.Close()

	svc := Service{
		TunnelBrokerURL: server.URL,
		HTTPClient:      server.Client(),
	}

	result := svc.probeTunnelGatewayConnectivity(context.Background(), gatewayRecord{
		ID:            "gateway-1",
		TunnelEnabled: true,
		Port:          2222,
	}, time.Second)

	if !result.Reachable {
		t.Fatalf("expected reachable result, got %#v", result)
	}
	<-acceptDone
}
