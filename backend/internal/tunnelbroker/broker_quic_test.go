package tunnelbroker

import (
	"bufio"
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
	"net/url"
	"testing"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/quic-go/quic-go"
)

// fakeStore returns a single canned gateway auth record and records nothing else.
type fakeStore struct {
	NoopStore
	record GatewayAuthRecord
}

func (s fakeStore) LoadGatewayAuth(context.Context, string) (GatewayAuthRecord, error) {
	return s.record, nil
}

// TestQUICTransportEndToEnd proves the QUIC transport: a real QUIC client
// authenticates via the existing authenticateTunnel (SPIFFE URI + tenant-CA
// chain validated from the handshake certificate), the broker opens a proxied
// stream through createTCPProxy, and bytes flow end-to-end to a local service.
func TestQUICTransportEndToEnd(t *testing.T) {
	const (
		gatewayID   = "gw-quic-test"
		token       = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		trustDomain = "arsenale.local"
	)

	caCertPEM, caKey, caCert := newTestCA(t)
	clientCertPEM, clientKeyPEM := newTestLeaf(t, caCert, caKey, spiffeURI(trustDomain, gatewayID), true)
	serverCertPEM, serverKeyPEM := newSelfSigned(t, "tunnel-broker")

	serverPair, err := tls.X509KeyPair(serverCertPEM, serverKeyPEM)
	if err != nil {
		t.Fatalf("server keypair: %v", err)
	}

	broker := NewBroker(BrokerConfig{
		Store:              fakeStore{record: GatewayAuthRecord{GatewayID: gatewayID, TenantID: "tenant-1", TunnelEnabled: true, TunnelTokenHash: hashToken(token), TenantTunnelCACertPEM: string(caCertPEM)}},
		SpiffeTrustDomain:  trustDomain,
		ProxyBindHost:      "127.0.0.1",
		ProxyAdvertiseHost: "127.0.0.1",
		QUICListenAddr:     "127.0.0.1:0",
		QUICTLSConfig:      &tls.Config{Certificates: []tls.Certificate{serverPair}},
	})

	listener, err := broker.quicListener()
	if err != nil {
		t.Fatalf("quic listener: %v", err)
	}
	if listener == nil {
		t.Fatal("expected a QUIC listener")
	}
	brokerAddr := listener.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = broker.serveQUIC(ctx, listener) }()

	// Local "service" the agent forwards to: a TCP echo server.
	echoAddr := startEchoServer(t)
	_, echoPortStr, _ := net.SplitHostPort(echoAddr)
	echoPort := atoiOrFail(t, echoPortStr)

	// The test agent: dial QUIC, authenticate, then accept broker-opened streams
	// and forward them to the echo server (mirroring the real tunnel agent).
	dialTestAgent(t, ctx, brokerAddr, gatewayID, token, clientCertPEM, clientKeyPEM)

	// Wait for registration.
	waitFor(t, 5*time.Second, func() bool {
		_, ok := broker.getStatus(gatewayID)
		return ok
	})

	// Open a proxied stream via the transport-blind createTCPProxy path.
	resp, err := broker.createTCPProxy(contracts.TunnelProxyRequest{
		GatewayID:  gatewayID,
		TargetHost: "127.0.0.1",
		TargetPort: echoPort,
	})
	if err != nil {
		t.Fatalf("createTCPProxy: %v", err)
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(resp.Host, itoa(resp.Port)), 3*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer conn.Close()

	payload := []byte("hello-over-quic")
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("echo mismatch: got %q want %q", got, payload)
	}

	// Status should reflect an active connection and counted bytes.
	status, ok := broker.getStatus(gatewayID)
	if !ok || !status.Connected {
		t.Fatalf("expected connected status, got %+v (ok=%v)", status, ok)
	}
}

// TestQUICTransportRejectsBadToken ensures the bearer-token check still gates
// the connection over QUIC.
func TestQUICTransportRejectsBadToken(t *testing.T) {
	const (
		gatewayID   = "gw-quic-bad"
		goodToken   = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		trustDomain = "arsenale.local"
	)
	caCertPEM, caKey, caCert := newTestCA(t)
	clientCertPEM, clientKeyPEM := newTestLeaf(t, caCert, caKey, spiffeURI(trustDomain, gatewayID), true)
	serverCertPEM, serverKeyPEM := newSelfSigned(t, "tunnel-broker")
	serverPair, _ := tls.X509KeyPair(serverCertPEM, serverKeyPEM)

	broker := NewBroker(BrokerConfig{
		Store:             fakeStore{record: GatewayAuthRecord{GatewayID: gatewayID, TunnelEnabled: true, TunnelTokenHash: hashToken(goodToken), TenantTunnelCACertPEM: string(caCertPEM)}},
		SpiffeTrustDomain: trustDomain,
		QUICListenAddr:    "127.0.0.1:0",
		QUICTLSConfig:     &tls.Config{Certificates: []tls.Certificate{serverPair}},
	})
	listener, err := broker.quicListener()
	if err != nil {
		t.Fatalf("listener: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = broker.serveQUIC(ctx, listener) }()

	// Dial with the WRONG token; the broker must reject and never register.
	dialTestAgent(t, ctx, listener.Addr().String(), gatewayID, "wrong-token", clientCertPEM, clientKeyPEM)

	time.Sleep(500 * time.Millisecond)
	if _, ok := broker.getStatus(gatewayID); ok {
		t.Fatal("gateway registered despite bad token")
	}
}

// --- helpers ---

func startEchoServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("echo listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(c)
		}
	}()
	return ln.Addr().String()
}

func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func spiffeURI(trustDomain, gatewayID string) *url.URL {
	return &url.URL{Scheme: "spiffe", Host: trustDomain, Path: "/gateway/" + gatewayID}
}

func newTestCA(t *testing.T) (certPEM []byte, key *ecdsa.PrivateKey, cert *x509.Certificate) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ca key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-tenant-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("ca cert: %v", err)
	}
	cert, _ = x509.ParseCertificate(der)
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return certPEM, key, cert
}

func newTestLeaf(t *testing.T, caCert *x509.Certificate, caKey *ecdsa.PrivateKey, uri *url.URL, client bool) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("leaf key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "gateway"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		URIs:         []*url.URL{uri},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatalf("leaf cert: %v", err)
	}
	keyDER, _ := x509.MarshalECPrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

func newSelfSigned(t *testing.T, cn string) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("self key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{CommonName: cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{cn, "localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("self cert: %v", err)
	}
	keyDER, _ := x509.MarshalECPrivateKey(key)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
}

func dialTestAgent(t *testing.T, ctx context.Context, addr string, gatewayID, token string, clientCertPEM, clientKeyPEM []byte) {
	t.Helper()
	pair, err := tls.X509KeyPair(clientCertPEM, clientKeyPEM)
	if err != nil {
		t.Fatalf("client keypair: %v", err)
	}
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
		NextProtos:         []string{TunnelALPN},
		Certificates:       []tls.Certificate{pair},
	}
	qc, err := quic.DialAddr(ctx, addr, tlsConf, &quic.Config{KeepAlivePeriod: 5 * time.Second, MaxIdleTimeout: 30 * time.Second})
	if err != nil {
		t.Fatalf("quic dial: %v", err)
	}
	t.Cleanup(func() { _ = qc.CloseWithError(0, "test done") })

	ctrl, err := qc.OpenStreamSync(ctx)
	if err != nil {
		t.Fatalf("open control stream: %v", err)
	}
	if err := writeJSONLine(ctrl, quicHello{GatewayID: gatewayID, Token: token, ClientVersion: "test"}); err != nil {
		t.Fatalf("write hello: %v", err)
	}

	go func() {
		reader := bufio.NewReader(ctrl)
		_, _ = reader.ReadBytes('\n') // ack (or EOF on rejection)
		for {
			stream, err := qc.AcceptStream(ctx)
			if err != nil {
				return
			}
			go forwardTestStream(stream)
		}
	}()
}

func forwardTestStream(stream *quic.Stream) {
	br := bufio.NewReader(stream)
	target, err := br.ReadBytes('\n')
	if err != nil {
		_ = stream.Close()
		return
	}
	local, err := net.DialTimeout("tcp", string(trimLine(target)), 3*time.Second)
	if err != nil {
		stream.CancelWrite(0)
		stream.CancelRead(0)
		return
	}
	if _, err := stream.Write([]byte{quicStreamOK}); err != nil {
		_ = local.Close()
		stream.CancelWrite(0)
		stream.CancelRead(0)
		return
	}
	go func() {
		_, _ = io.Copy(local, br)
		_ = local.Close()
	}()
	_, _ = io.Copy(stream, local)
	_ = stream.Close()
}

func atoiOrFail(t *testing.T, s string) int {
	t.Helper()
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			t.Fatalf("bad port %q", s)
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
