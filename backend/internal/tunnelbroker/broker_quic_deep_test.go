package tunnelbroker

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/quic-go/quic-go"
)

const deepTrustDomain = "arsenale.local"

// recordingStore records the Store calls the broker makes so tests can assert on
// connect/disconnect/heartbeat persistence and audit actions.
type recordingStore struct {
	mu            sync.Mutex
	record        GatewayAuthRecord
	connected     int
	disconnected  int
	heartbeats    int
	lastHeartbeat *HeartbeatMetadata
	audits        []string
}

func (s *recordingStore) LoadGatewayAuth(context.Context, string) (GatewayAuthRecord, error) {
	return s.record, nil
}
func (s *recordingStore) MarkTunnelConnected(context.Context, string, time.Time, string, string) error {
	s.mu.Lock()
	s.connected++
	s.mu.Unlock()
	return nil
}
func (s *recordingStore) MarkTunnelDisconnected(context.Context, string) error {
	s.mu.Lock()
	s.disconnected++
	s.mu.Unlock()
	return nil
}
func (s *recordingStore) MarkTunnelHeartbeat(_ context.Context, _ string, _ time.Time, hb *HeartbeatMetadata) error {
	s.mu.Lock()
	s.heartbeats++
	s.lastHeartbeat = hb
	s.mu.Unlock()
	return nil
}
func (s *recordingStore) InsertTunnelAudit(_ context.Context, action, _, _ string, _ map[string]any) error {
	s.mu.Lock()
	s.audits = append(s.audits, action)
	s.mu.Unlock()
	return nil
}
func (s *recordingStore) snapshot() (conn, disc, hb int, last *HeartbeatMetadata, audits []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connected, s.disconnected, s.heartbeats, s.lastHeartbeat, append([]string(nil), s.audits...)
}

// startBrokerQUIC stands up a broker QUIC listener on an ephemeral loopback port
// with a self-signed server cert (in-test agents skip server verification).
func startBrokerQUIC(t *testing.T, store Store) (*Broker, string, context.CancelFunc) {
	t.Helper()
	serverCertPEM, serverKeyPEM := newSelfSigned(t, "tunnel-broker")
	serverPair, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("server keypair: %v", err)
	}
	broker := NewBroker(BrokerConfig{
		Store:              store,
		SpiffeTrustDomain:  deepTrustDomain,
		ProxyBindHost:      "127.0.0.1",
		ProxyAdvertiseHost: "127.0.0.1",
		QUICListenAddr:     "127.0.0.1:0",
		QUICTLSConfig:      &tls.Config{Certificates: []tls.Certificate{serverPair}},
	})
	listener, err := broker.quicListener()
	if err != nil {
		t.Fatalf("quic listener: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = broker.serveQUIC(ctx, listener) }()
	t.Cleanup(cancel)
	return broker, listener.Addr().String(), cancel
}

func authRecord(gatewayID, caPEM, token string) GatewayAuthRecord {
	return GatewayAuthRecord{
		GatewayID:             gatewayID,
		TenantID:              "tenant-1",
		TunnelEnabled:         true,
		TunnelTokenHash:       hashToken(token),
		TenantTunnelCACertPEM: caPEM,
	}
}

// TestQUICConcurrentStreams proves many independent proxied streams over a single
// QUIC connection do not deadlock, do not cross-talk, and each round-trips.
func TestQUICConcurrentStreams(t *testing.T) {
	const (
		gatewayID = "gw-concurrent"
		token     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		streams   = 30
	)
	caPEM, caKey, caCert := newTestCA(t)
	clientCertPEM, clientKeyPEM := newTestLeaf(t, caCert, caKey, spiffeURI(deepTrustDomain, gatewayID), true)

	broker, addr, _ := startBrokerQUIC(t, &recordingStore{record: authRecord(gatewayID, string(caPEM), token)})
	echoAddr := startEchoServer(t)
	_, echoPortStr, _ := net.SplitHostPort(echoAddr)
	echoPort := atoiOrFail(t, echoPortStr)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	dialTestAgent(t, ctx, addr, gatewayID, token, clientCertPEM, clientKeyPEM)
	waitFor(t, 5*time.Second, func() bool { _, ok := broker.getStatus(gatewayID); return ok })

	var wg sync.WaitGroup
	errs := make(chan error, streams)
	for i := 0; i < streams; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, err := broker.createTCPProxy(contracts.TunnelProxyRequest{GatewayID: gatewayID, TargetHost: "127.0.0.1", TargetPort: echoPort})
			if err != nil {
				errs <- fmt.Errorf("proxy %d: %w", i, err)
				return
			}
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(resp.Host, itoa(resp.Port)), 5*time.Second)
			if err != nil {
				errs <- fmt.Errorf("dial %d: %w", i, err)
				return
			}
			defer conn.Close()
			// Distinct payload per stream catches cross-talk between streams.
			payload := []byte(fmt.Sprintf("stream-%d-payload-%d", i, i*7919))
			if _, err := conn.Write(payload); err != nil {
				errs <- fmt.Errorf("write %d: %w", i, err)
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			got := make([]byte, len(payload))
			if _, err := io.ReadFull(conn, got); err != nil {
				errs <- fmt.Errorf("read %d: %w", i, err)
				return
			}
			if string(got) != string(payload) {
				errs <- fmt.Errorf("stream %d cross-talk: got %q want %q", i, got, payload)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// TestQUICLargePayload proves a multi-megabyte bidirectional transfer streams
// correctly (flow control, no truncation).
func TestQUICLargePayload(t *testing.T) {
	const (
		gatewayID = "gw-large"
		token     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		size      = 4 << 20 // 4 MiB
	)
	caPEM, caKey, caCert := newTestCA(t)
	clientCertPEM, clientKeyPEM := newTestLeaf(t, caCert, caKey, spiffeURI(deepTrustDomain, gatewayID), true)

	broker, addr, _ := startBrokerQUIC(t, &recordingStore{record: authRecord(gatewayID, string(caPEM), token)})
	echoAddr := startEchoServer(t)
	_, echoPortStr, _ := net.SplitHostPort(echoAddr)
	echoPort := atoiOrFail(t, echoPortStr)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	dialTestAgent(t, ctx, addr, gatewayID, token, clientCertPEM, clientKeyPEM)
	waitFor(t, 5*time.Second, func() bool { _, ok := broker.getStatus(gatewayID); return ok })

	resp, err := broker.createTCPProxy(contracts.TunnelProxyRequest{GatewayID: gatewayID, TargetHost: "127.0.0.1", TargetPort: echoPort})
	if err != nil {
		t.Fatalf("createTCPProxy: %v", err)
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(resp.Host, itoa(resp.Port)), 5*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	payload := make([]byte, size)
	if _, err := rand.Read(payload); err != nil {
		t.Fatalf("rand: %v", err)
	}
	got := make([]byte, size)
	var readErr error
	done := make(chan struct{})
	go func() {
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		_, readErr = io.ReadFull(conn, got)
		close(done)
	}()
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	<-done
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	for i := range payload {
		if payload[i] != got[i] {
			t.Fatalf("payload mismatch at byte %d", i)
		}
	}
}

// TestQUICEvictsPreviousConnection proves a second connection for the same
// gateway evicts the first (the path that previously dead-locked), leaving
// exactly one registry entry that still serves proxies.
func TestQUICEvictsPreviousConnection(t *testing.T) {
	const (
		gatewayID = "gw-evict"
		token     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)
	caPEM, caKey, caCert := newTestCA(t)
	clientCertPEM, clientKeyPEM := newTestLeaf(t, caCert, caKey, spiffeURI(deepTrustDomain, gatewayID), true)

	store := &recordingStore{record: authRecord(gatewayID, string(caPEM), token)}
	broker, addr, _ := startBrokerQUIC(t, store)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	dialTestAgent(t, ctx, addr, gatewayID, token, clientCertPEM, clientKeyPEM)
	waitFor(t, 5*time.Second, func() bool { _, ok := broker.getStatus(gatewayID); return ok })

	// Second connection for the same gateway.
	dialTestAgent(t, ctx, addr, gatewayID, token, clientCertPEM, clientKeyPEM)
	waitFor(t, 5*time.Second, func() bool {
		broker.mu.RLock()
		defer broker.mu.RUnlock()
		return len(broker.registry) == 1
	})

	broker.mu.RLock()
	n := len(broker.registry)
	broker.mu.RUnlock()
	if n != 1 {
		t.Fatalf("registry has %d entries, want 1 after eviction", n)
	}
	if _, ok := broker.getStatus(gatewayID); !ok {
		t.Fatal("surviving connection not registered")
	}
}

// TestQUICDisconnectCleansUp proves that when the agent goes away the broker
// deregisters it and persists the disconnect.
func TestQUICDisconnectCleansUp(t *testing.T) {
	const (
		gatewayID = "gw-disconnect"
		token     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)
	caPEM, caKey, caCert := newTestCA(t)
	clientCertPEM, clientKeyPEM := newTestLeaf(t, caCert, caKey, spiffeURI(deepTrustDomain, gatewayID), true)

	store := &recordingStore{record: authRecord(gatewayID, string(caPEM), token)}
	broker, addr, _ := startBrokerQUIC(t, store)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	qc := dialClosableAgent(t, ctx, addr, gatewayID, token, clientCertPEM, clientKeyPEM)
	waitFor(t, 5*time.Second, func() bool { _, ok := broker.getStatus(gatewayID); return ok })

	// Explicitly close the QUIC connection (cancelling the dial context does not
	// tear down an already-established connection).
	_ = qc.CloseWithError(0, "agent going away")

	waitFor(t, 5*time.Second, func() bool { _, ok := broker.getStatus(gatewayID); return !ok })
	waitFor(t, 5*time.Second, func() bool {
		_, disc, _, _, _ := store.snapshot()
		return disc >= 1
	})
	_, _, _, _, audits := store.snapshot()
	if !contains(audits, "TUNNEL_CONNECT") || !contains(audits, "TUNNEL_DISCONNECT") {
		t.Fatalf("expected connect+disconnect audits, got %v", audits)
	}
}

// TestQUICRejectsForeignCA proves a client certificate that does not chain to the
// gateway's tenant CA is rejected.
func TestQUICRejectsForeignCA(t *testing.T) {
	const (
		gatewayID = "gw-foreign-ca"
		token     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)
	tenantCAPEM, _, _ := newTestCA(t)         // the CA the gateway is bound to
	_, foreignKey, foreignCA := newTestCA(t)  // a different CA signs the client cert
	clientCertPEM, clientKeyPEM := newTestLeaf(t, foreignCA, foreignKey, spiffeURI(deepTrustDomain, gatewayID), true)

	broker, addr, _ := startBrokerQUIC(t, &recordingStore{record: authRecord(gatewayID, string(tenantCAPEM), token)})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	dialTestAgent(t, ctx, addr, gatewayID, token, clientCertPEM, clientKeyPEM)

	time.Sleep(500 * time.Millisecond)
	if _, ok := broker.getStatus(gatewayID); ok {
		t.Fatal("gateway registered with a cert from a foreign CA")
	}
}

// TestQUICRejectsSPIFFEMismatch proves a certificate whose SPIFFE ID does not
// match the claimed gateway ID is rejected.
func TestQUICRejectsSPIFFEMismatch(t *testing.T) {
	const (
		realGateway   = "gw-real"
		claimedGateway = "gw-claimed"
		token         = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)
	caPEM, caKey, caCert := newTestCA(t)
	// Cert carries the SPIFFE ID for realGateway, but the hello claims claimedGateway.
	clientCertPEM, clientKeyPEM := newTestLeaf(t, caCert, caKey, spiffeURI(deepTrustDomain, realGateway), true)

	broker, addr, _ := startBrokerQUIC(t, &recordingStore{record: authRecord(claimedGateway, string(caPEM), token)})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	dialTestAgent(t, ctx, addr, claimedGateway, token, clientCertPEM, clientKeyPEM)

	time.Sleep(500 * time.Millisecond)
	if _, ok := broker.getStatus(claimedGateway); ok {
		t.Fatal("gateway registered despite SPIFFE ID mismatch")
	}
}

// TestQUICHeartbeatPersisted proves heartbeat control messages are persisted via
// the Store with their health metadata.
func TestQUICHeartbeatPersisted(t *testing.T) {
	const (
		gatewayID = "gw-heartbeat"
		token     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)
	caPEM, caKey, caCert := newTestCA(t)
	clientCertPEM, clientKeyPEM := newTestLeaf(t, caCert, caKey, spiffeURI(deepTrustDomain, gatewayID), true)

	store := &recordingStore{record: authRecord(gatewayID, string(caPEM), token)}
	_, addr, _ := startBrokerQUIC(t, store)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Bespoke client: connect, hello, read ack, send one heartbeat.
	pair, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}
	qc, err := quic.DialAddr(ctx, addr, &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS13, NextProtos: []string{TunnelALPN}, Certificates: []tls.Certificate{pair}}, &quic.Config{KeepAlivePeriod: 5 * time.Second})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = qc.CloseWithError(0, "done") })
	ctrl, err := qc.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("control stream: %v", err)
	}
	if err := writeJSONLine(ctrl, quicHello{GatewayID: gatewayID, Token: token, ClientVersion: "deep-test"}); err != nil {
		t.Fatalf("hello: %v", err)
	}
	reader := bufio.NewReader(ctrl)
	if _, err := reader.ReadBytes('\n'); err != nil {
		t.Fatalf("ack: %v", err)
	}
	latency := 12
	if err := writeJSONLine(ctrl, quicControlMsg{Type: "heartbeat", Heartbeat: &HeartbeatMetadata{Healthy: true, LatencyMs: &latency}}); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	waitFor(t, 5*time.Second, func() bool {
		_, _, hb, last, _ := store.snapshot()
		return hb >= 1 && last != nil && last.Healthy
	})
}

// dialClosableAgent dials and authenticates like dialTestAgent but returns the
// connection so the caller can close it mid-test (to exercise disconnect paths).
func dialClosableAgent(t *testing.T, ctx context.Context, addr, gatewayID, token string, clientCertPEM, clientKeyPEM []byte) *quic.Conn {
	t.Helper()
	pair, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
	if err != nil {
		t.Fatalf("client keypair: %v", err)
	}
	qc, err := quic.DialAddr(ctx, addr, &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS13, NextProtos: []string{TunnelALPN}, Certificates: []tls.Certificate{pair}}, &quic.Config{KeepAlivePeriod: 5 * time.Second, MaxIdleTimeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("quic dial: %v", err)
	}
	t.Cleanup(func() { _ = qc.CloseWithError(0, "test done") })
	ctrl, err := qc.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open control stream: %v", err)
	}
	if err := writeJSONLine(ctrl, quicHello{GatewayID: gatewayID, Token: token, ClientVersion: "deep-test"}); err != nil {
		t.Fatalf("write hello: %v", err)
	}
	reader := bufio.NewReader(ctrl)
	if _, err := reader.ReadBytes('\n'); err != nil {
		t.Fatalf("read ack: %v", err)
	}
	return qc
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
