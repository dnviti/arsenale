package tunnelbroker

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

// These tests build and run the REAL tunnel-agent binary against the real broker,
// proving the two modules interoperate over the wire (not via an in-test mirror).

var (
	agentBinOnce sync.Once
	agentBinPath string
	agentBinErr  error
)

func buildAgentBinary(t *testing.T) string {
	t.Helper()
	agentBinOnce.Do(func() {
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			agentBinErr = errString("cannot resolve test file path")
			return
		}
		agentDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "gateways", "tunnel-agent")
		dir, err := os.MkdirTemp("", "tunnel-agent-e2e")
		if err != nil {
			agentBinErr = err
			return
		}
		bin := filepath.Join(dir, "tunnel-agent")
		// -ldflags=-w avoids the DWARF relocation linker error on large cmd binaries.
		cmd := exec.Command("go", "build", "-ldflags=-w", "-o", bin, ".")
		cmd.Dir = agentDir
		cmd.Env = os.Environ()
		if out, err := cmd.CombinedOutput(); err != nil {
			agentBinErr = errString("build tunnel-agent: " + err.Error() + "\n" + string(out))
			return
		}
		agentBinPath = bin
	})
	if agentBinErr != nil {
		t.Fatalf("%v", agentBinErr)
	}
	return agentBinPath
}

type errString string

func (e errString) Error() string { return string(e) }

// safeBuffer is a concurrency-safe buffer for capturing subprocess output.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}
func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func runAgent(t *testing.T, ctx context.Context, env map[string]string) *safeBuffer {
	t.Helper()
	bin := buildAgentBinary(t)
	out := &safeBuffer{}
	cmd := exec.CommandContext(ctx, bin)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start agent: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	return out
}

func writeFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

// newTestServerCert issues a server certificate signed by the CA with the given
// SANs (so the real agent, which verifies against the CA, accepts the broker).
func newTestServerCert(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey, dnsNames []string, ips []net.IP) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("server key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(10),
		Subject:      pkix.Name{CommonName: "arsenale-tunnel-broker"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dnsNames,
		IPAddresses:  ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("server cert: %v", err)
	}
	keyDER, _ := x509.MarshalECPrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

func startEchoServerOn(t *testing.T, host string) string {
	t.Helper()
	ln, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		t.Fatalf("echo listen on %s: %v", host, err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) { defer c.Close(); _, _ = io.Copy(c, c) }(c)
		}
	}()
	return ln.Addr().String()
}

func brokerWithSignedCert(t *testing.T, store Store, caCert *x509.Certificate, caKey *ecdsa.PrivateKey) (*Broker, string) {
	t.Helper()
	certPEM, keyPEM := newTestServerCert(t, caCert, caKey, []string{"tunnel-broker", "localhost"}, []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback})
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("server keypair: %v", err)
	}
	broker := NewBroker(BrokerConfig{
		Store:              store,
		SpiffeTrustDomain:  deepTrustDomain,
		ProxyBindHost:      "127.0.0.1",
		ProxyAdvertiseHost: "127.0.0.1",
		QUICListenAddr:     "127.0.0.1:0",
		QUICTLSConfig:      &tls.Config{Certificates: []tls.Certificate{pair}},
	})
	listener, err := broker.quicListener()
	if err != nil {
		t.Fatalf("quic listener: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = broker.serveQUIC(ctx, listener) }()
	return broker, listener.Addr().String()
}

func TestE2E_RealAgentQUIC(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the agent binary; skipped in -short")
	}
	const (
		gatewayID = "gw-e2e-quic"
		token     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)
	caPEM, caKey, caCert := newTestCA(t)
	clientCertPEM, clientKeyPEM := newTestLeaf(t, caCert, caKey, spiffeURI(deepTrustDomain, gatewayID), true)

	store := &recordingStore{record: authRecord(gatewayID, string(caPEM), token)}
	broker, quicAddr := brokerWithSignedCert(t, store, caCert, caKey)

	echoAddr := startEchoServerOn(t, "127.0.0.1")
	_, echoPort, _ := net.SplitHostPort(echoAddr)

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	out := runAgent(t, ctx, map[string]string{
		"TUNNEL_TRANSPORT":        "quic",
		"TUNNEL_QUIC_SERVER_ADDR": quicAddr,
		"TUNNEL_QUIC_SERVER_NAME": "127.0.0.1",
		"TUNNEL_GATEWAY_ID":       gatewayID,
		"TUNNEL_TOKEN":            token,
		"TUNNEL_LOCAL_HOST":       "127.0.0.1",
		"TUNNEL_LOCAL_PORT":       echoPort,
		"TUNNEL_CA_CERT_FILE":     writeFile(t, dir, "ca.pem", caPEM),
		"TUNNEL_CLIENT_CERT_FILE": writeFile(t, dir, "client-cert.pem", clientCertPEM),
		"TUNNEL_CLIENT_KEY_FILE":  writeFile(t, dir, "client-key.pem", clientKeyPEM),
		"TUNNEL_PING_INTERVAL_MS": "1000",
	})

	waitForAgent(t, 20*time.Second, func() bool { _, ok := broker.getStatus(gatewayID); return ok }, out)

	assertEchoThroughProxy(t, broker, gatewayID, "127.0.0.1", echoPort, []byte("real-agent-over-quic"))

	if !strings.Contains(out.String(), "over QUIC") {
		t.Fatalf("agent did not report a QUIC connection:\n%s", out.String())
	}
}

func TestE2E_RealAgentAutoFallbackToWSS(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the agent binary and waits out the QUIC handshake timeout; skipped in -short")
	}
	const (
		gatewayID = "gw-e2e-auto"
		token     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)
	caPEM, caKey, caCert := newTestCA(t)
	clientCertPEM, clientKeyPEM := newTestLeaf(t, caCert, caKey, spiffeURI(deepTrustDomain, gatewayID), true)

	store := &recordingStore{record: authRecord(gatewayID, string(caPEM), token)}
	// Broker with WebSocket routes only (QUIC will be pointed at a dead UDP port).
	broker := NewBroker(BrokerConfig{Store: store, SpiffeTrustDomain: deepTrustDomain, ProxyBindHost: "127.0.0.1", ProxyAdvertiseHost: "127.0.0.1"})
	mux := http.NewServeMux()
	broker.RegisterRoutes(mux)
	wsSrv := httptest.NewServer(mux)
	t.Cleanup(wsSrv.Close)

	echoAddr := startEchoServerOn(t, "127.0.0.1")
	_, echoPort, _ := net.SplitHostPort(echoAddr)

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	out := runAgent(t, ctx, map[string]string{
		"TUNNEL_TRANSPORT":        "auto",
		"TUNNEL_QUIC_SERVER_ADDR": deadUDPAddr(t), // QUIC handshake will time out
		"TUNNEL_QUIC_SERVER_NAME": "127.0.0.1",
		"TUNNEL_SERVER_URL":       wsSrv.URL, // WSS fallback target
		"TUNNEL_GATEWAY_ID":       gatewayID,
		"TUNNEL_TOKEN":            token,
		"TUNNEL_LOCAL_HOST":       "127.0.0.1",
		"TUNNEL_LOCAL_PORT":       echoPort,
		"TUNNEL_CA_CERT_FILE":     writeFile(t, dir, "ca.pem", caPEM),
		"TUNNEL_CLIENT_CERT_FILE": writeFile(t, dir, "client-cert.pem", clientCertPEM),
		"TUNNEL_CLIENT_KEY_FILE":  writeFile(t, dir, "client-key.pem", clientKeyPEM),
		"TUNNEL_PING_INTERVAL_MS": "1000",
	})

	// QUIC dial times out (~10s) then the agent falls back to WSS and registers.
	waitForAgent(t, 30*time.Second, func() bool { _, ok := broker.getStatus(gatewayID); return ok }, out)

	assertEchoThroughProxy(t, broker, gatewayID, "127.0.0.1", echoPort, []byte("real-agent-fellback-to-wss"))

	if !strings.Contains(out.String(), "WSS") {
		t.Fatalf("agent did not fall back to WSS:\n%s", out.String())
	}
}

func TestE2E_RealAgentSSRFGuard(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the agent binary; skipped in -short")
	}
	const (
		gatewayID = "gw-e2e-ssrf"
		token     = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)
	caPEM, caKey, caCert := newTestCA(t)
	clientCertPEM, clientKeyPEM := newTestLeaf(t, caCert, caKey, spiffeURI(deepTrustDomain, gatewayID), true)

	store := &recordingStore{record: authRecord(gatewayID, string(caPEM), token)}
	broker, quicAddr := brokerWithSignedCert(t, store, caCert, caKey)

	// Echo server reachable on a NON-allowlisted loopback address (127.0.0.2).
	// The agent must refuse it via isAllowedLocalHost even though it is reachable.
	evilEcho := startEchoServerOn(t, "127.0.0.2")
	evilHost, evilPort, _ := net.SplitHostPort(evilEcho)

	// And a legitimate localhost service to confirm the agent is otherwise healthy.
	okEcho := startEchoServerOn(t, "127.0.0.1")
	_, okPort, _ := net.SplitHostPort(okEcho)

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	out := runAgent(t, ctx, map[string]string{
		"TUNNEL_TRANSPORT":        "quic",
		"TUNNEL_QUIC_SERVER_ADDR": quicAddr,
		"TUNNEL_QUIC_SERVER_NAME": "127.0.0.1",
		"TUNNEL_GATEWAY_ID":       gatewayID,
		"TUNNEL_TOKEN":            token,
		"TUNNEL_LOCAL_HOST":       "127.0.0.1",
		"TUNNEL_LOCAL_PORT":       okPort,
		"TUNNEL_CA_CERT_FILE":     writeFile(t, dir, "ca.pem", caPEM),
		"TUNNEL_CLIENT_CERT_FILE": writeFile(t, dir, "client-cert.pem", clientCertPEM),
		"TUNNEL_CLIENT_KEY_FILE":  writeFile(t, dir, "client-key.pem", clientKeyPEM),
	})
	waitForAgent(t, 20*time.Second, func() bool { _, ok := broker.getStatus(gatewayID); return ok }, out)

	// Legitimate localhost target works.
	assertEchoThroughProxy(t, broker, gatewayID, "127.0.0.1", okPort, []byte("allowed"))

	// Non-allowlisted (but reachable) target must NOT echo — the SSRF guard refuses.
	resp, err := broker.createTCPProxy(contracts.TunnelProxyRequest{GatewayID: gatewayID, TargetHost: evilHost, TargetPort: atoiOrFail(t, evilPort)})
	if err != nil {
		t.Fatalf("createTCPProxy(evil): %v", err)
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(resp.Host, itoa(resp.Port)), 3*time.Second)
	if err != nil {
		t.Fatalf("dial evil proxy: %v", err)
	}
	defer conn.Close()
	_, _ = conn.Write([]byte("should-be-refused"))
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	buf := make([]byte, 32)
	n, _ := conn.Read(buf)
	if n > 0 {
		t.Fatalf("SSRF guard failed: got %d bytes %q from a non-localhost target", n, buf[:n])
	}
}

func assertEchoThroughProxy(t *testing.T, broker *Broker, gatewayID, targetHost, targetPort string, payload []byte) {
	t.Helper()
	resp, err := broker.createTCPProxy(contracts.TunnelProxyRequest{GatewayID: gatewayID, TargetHost: targetHost, TargetPort: atoiOrFail(t, targetPort)})
	if err != nil {
		t.Fatalf("createTCPProxy: %v", err)
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(resp.Host, itoa(resp.Port)), 5*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("echo mismatch: got %q want %q", got, payload)
	}
}

func waitForAgent(t *testing.T, timeout time.Duration, cond func() bool, out *safeBuffer) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("agent condition not met within %s; agent output:\n%s", timeout, out.String())
}

// deadUDPAddr returns a loopback host:port that has no UDP listener (a UDP socket
// is bound to grab a free port, then closed), so a QUIC dial there times out.
func deadUDPAddr(t *testing.T) string {
	t.Helper()
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve udp port: %v", err)
	}
	addr := c.LocalAddr().String()
	_ = c.Close()
	return addr
}
