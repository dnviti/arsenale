package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestQUICHelloWireFormat pins the on-the-wire JSON the agent sends so it stays
// compatible with the broker's quicHello parser in another module (the broker
// reads gatewayId/token/clientVersion).
func TestQUICHelloWireFormat(t *testing.T) {
	var buf bytes.Buffer
	hello := quicHello{GatewayID: "gw-1", Token: "tok", ClientVersion: "1.2.3"}
	if err := writeQUICLine(&buf, quicControlMsg{}, hello); err != nil {
		t.Fatalf("write hello: %v", err)
	}
	if buf.Bytes()[buf.Len()-1] != '\n' {
		t.Fatal("hello line is not newline-terminated")
	}

	var decoded map[string]any
	if err := json.Unmarshal(trimQUICLine(buf.Bytes()), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"gatewayId", "token", "clientVersion"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("hello missing key %q: %v", key, decoded)
		}
	}
}

// TestQUICHeartbeatWireFormat pins the heartbeat JSON shape the broker decodes
// into its HeartbeatMetadata (healthy/latencyMs/activeStreams).
func TestQUICHeartbeatWireFormat(t *testing.T) {
	latency, active := 7, 2
	var buf bytes.Buffer
	msg := quicControlMsg{Type: "heartbeat", Heartbeat: &quicHeartbeat{Healthy: true, LatencyMs: &latency, ActiveStreams: &active}}
	if err := writeQUICLine(&buf, msg, nil); err != nil {
		t.Fatalf("write heartbeat: %v", err)
	}

	var decoded struct {
		Type      string `json:"type"`
		Heartbeat struct {
			Healthy       bool `json:"healthy"`
			LatencyMs     *int `json:"latencyMs"`
			ActiveStreams *int `json:"activeStreams"`
		} `json:"heartbeat"`
	}
	if err := json.Unmarshal(trimQUICLine(buf.Bytes()), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Type != "heartbeat" || !decoded.Heartbeat.Healthy ||
		decoded.Heartbeat.LatencyMs == nil || *decoded.Heartbeat.LatencyMs != 7 ||
		decoded.Heartbeat.ActiveStreams == nil || *decoded.Heartbeat.ActiveStreams != 2 {
		t.Fatalf("unexpected heartbeat wire format: %s", buf.String())
	}
}

func TestNormalizeTransport(t *testing.T) {
	cases := map[string]string{"": transportWSS, "wss": transportWSS, "QUIC": transportQUIC, "quic": transportQUIC, "  quic  ": transportQUIC, "AUTO": transportAuto, "auto": transportAuto, "tcp": transportWSS}
	for in, want := range cases {
		if got := normalizeTransport(in); got != want {
			t.Fatalf("normalizeTransport(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestLoadConfigAutoRequiresBothEndpoints verifies that auto mode requires both
// the QUIC address (to attempt) and the WSS URL (to fall back to).
func TestLoadConfigAutoRequiresBothEndpoints(t *testing.T) {
	base := func() {
		t.Setenv("TUNNEL_TRANSPORT", "auto")
		t.Setenv("TUNNEL_TOKEN", "tok")
		t.Setenv("TUNNEL_GATEWAY_ID", "gw-1")
		t.Setenv("TUNNEL_LOCAL_PORT", "4822")
	}

	// Missing the WSS fallback URL → error.
	base()
	t.Setenv("TUNNEL_QUIC_SERVER_ADDR", "broker:8092")
	t.Setenv("TUNNEL_SERVER_URL", "")
	if _, _, err := LoadConfigFromEnv("test"); err == nil {
		t.Fatal("expected error when auto mode is missing TUNNEL_SERVER_URL")
	}

	// Both present → ok, transport=auto.
	base()
	t.Setenv("TUNNEL_QUIC_SERVER_ADDR", "broker:8092")
	t.Setenv("TUNNEL_SERVER_URL", "https://arsenale.example.com")
	cfg, dormant, err := LoadConfigFromEnv("test")
	if err != nil || dormant {
		t.Fatalf("unexpected: err=%v dormant=%v", err, dormant)
	}
	if cfg.Transport != transportAuto || cfg.QUICServerAddr != "broker:8092" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
