package tunnelbroker

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/quic-go/quic-go"
)

// TunnelALPN is the QUIC ALPN protocol identifier negotiated between the agent
// and broker. It must match on both ends.
const TunnelALPN = "arsenale-tunnel/1"

const (
	// quicErrShutdown closes a connection during normal teardown/eviction.
	quicErrShutdown quic.ApplicationErrorCode = 0
	// quicErrAuth closes a connection that failed authentication.
	quicErrAuth quic.ApplicationErrorCode = 1

	quicControlAcceptTimeout = 10 * time.Second
	quicKeepAlivePeriod      = 15 * time.Second
	quicMaxIdleTimeout       = 45 * time.Second
	quicMaxIncomingStreams   = int64(maxStreamID)

	// quicDefaultListenAddr matches the tunnel-broker command default
	// (TUNNEL_QUIC_LISTEN_ADDR) and the Ansible compose UDP publish.
	quicDefaultListenAddr = ":8092"

	// quicMaxHelloBytes bounds the unauthenticated hello line so a reachable UDP
	// listener cannot be forced to buffer an unbounded newline-less payload
	// before any token/certificate validation.
	quicMaxHelloBytes = 4096

	// quicStreamOK is the single-byte acknowledgement the agent writes back on a
	// proxy stream once it has successfully dialed the local target. It must
	// match the agent's constant of the same name. Any other byte (or a stream
	// reset) signals the target was refused, so createTCPProxy can fall back to
	// the next candidate port.
	quicStreamOK = byte('1')
)

// quicHello is the first newline-delimited JSON message the agent sends on the
// control stream, carrying the bearer token and gateway identity. The client
// certificate itself is validated from the QUIC/TLS handshake, not this message.
type quicHello struct {
	GatewayID     string `json:"gatewayId"`
	Token         string `json:"token"`
	ClientVersion string `json:"clientVersion"`
}

// quicControlMsg is a newline-delimited JSON message exchanged on the control
// stream after the hello: heartbeats (agent->broker), an ack (broker->agent),
// and certificate renewals (broker->agent).
type quicControlMsg struct {
	Type       string             `json:"type"`
	Heartbeat  *HeartbeatMetadata `json:"heartbeat,omitempty"`
	ClientCert string             `json:"clientCert,omitempty"`
	ClientKey  string             `json:"clientKey,omitempty"`
}

// quicConnection is the QUIC implementation of tunnelConn. One quic.Conn per
// gateway carries N independent bidirectional streams, so the WebSocket
// frame-mux (stream-ID allocation, io.Pipe, sendMu serialization) is gone:
// each logical stream is a native quic.Stream.
type quicConnection struct {
	broker        *Broker
	gatewayID     string
	qconn         *quic.Conn
	connectedAt   time.Time
	clientVersion string
	clientIP      string

	mu            sync.Mutex
	lastHeartbeat time.Time
	heartbeat     *HeartbeatMetadata
	pingLatency   *int64

	control          *quic.Stream
	activeStreams    atomic.Int64
	bytesTransferred atomic.Int64
	closeOnce        sync.Once
}

var _ tunnelConn = (*quicConnection)(nil)

// quicStream wraps a quic.Stream to track byte counts and the active-stream
// gauge for status reporting. It is an io.ReadWriteCloser, so it drops into the
// existing createTCPProxy io.Copy loops unchanged.
type quicStream struct {
	*quic.Stream
	conn      *quicConnection
	closeOnce sync.Once
}

func (s *quicStream) Read(p []byte) (int, error) {
	n, err := s.Stream.Read(p)
	if n > 0 {
		s.conn.bytesTransferred.Add(int64(n))
	}
	return n, err
}

func (s *quicStream) Write(p []byte) (int, error) {
	n, err := s.Stream.Write(p)
	if n > 0 {
		s.conn.bytesTransferred.Add(int64(n))
	}
	return n, err
}

func (s *quicStream) Close() error {
	s.closeOnce.Do(func() {
		s.conn.activeStreams.Add(-1)
	})
	return s.Stream.Close()
}

func (c *quicConnection) openStream(ctx context.Context, host string, port int, timeout time.Duration) (io.ReadWriteCloser, error) {
	octx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stream, err := c.qconn.OpenStreamSync(octx)
	if err != nil {
		return nil, fmt.Errorf("open quic stream: %w", err)
	}
	if _, err := stream.Write([]byte(net.JoinHostPort(host, strconv.Itoa(port)) + "\n")); err != nil {
		stream.CancelWrite(0)
		return nil, fmt.Errorf("write tunnel target: %w", err)
	}

	// Wait for the agent to confirm it dialed the local target before treating
	// the stream as open. This mirrors the WebSocket open-ack and lets
	// createTCPProxy fall back to the next candidate port when this one is
	// refused, instead of stopping at the first candidate.
	if deadline, ok := octx.Deadline(); ok {
		_ = stream.SetReadDeadline(deadline)
	}
	ack := make([]byte, 1)
	if _, err := io.ReadFull(stream, ack); err != nil {
		stream.CancelRead(0)
		stream.CancelWrite(0)
		return nil, fmt.Errorf("await tunnel target ack for %s:%d: %w", host, port, err)
	}
	_ = stream.SetReadDeadline(time.Time{})
	if ack[0] != quicStreamOK {
		stream.CancelRead(0)
		stream.CancelWrite(0)
		return nil, fmt.Errorf("agent refused tunnel target %s:%d", host, port)
	}

	c.activeStreams.Add(1)
	return &quicStream{Stream: stream, conn: c}, nil
}

func (c *quicConnection) describe() contracts.TunnelStatus {
	c.mu.Lock()
	lastHB := c.lastHeartbeat
	hb := c.heartbeat
	lat := c.pingLatency
	c.mu.Unlock()

	status := contracts.TunnelStatus{
		GatewayID:        c.gatewayID,
		Connected:        true,
		ConnectedAt:      c.connectedAt.Format(time.RFC3339),
		ClientVersion:    c.clientVersion,
		ClientIP:         c.clientIP,
		ActiveStreams:    int(c.activeStreams.Load()),
		BytesTransferred: c.bytesTransferred.Load(),
	}
	if !lastHB.IsZero() {
		status.LastHeartbeatAt = lastHB.Format(time.RFC3339)
	}
	if lat != nil {
		value := *lat
		status.PingPongLatencyMs = &value
	}
	if hb != nil {
		status.Heartbeat = &contracts.TunnelHeartbeat{
			Healthy:       hb.Healthy,
			LatencyMs:     hb.LatencyMs,
			ActiveStreams: hb.ActiveStreams,
		}
	}
	return status
}

func (c *quicConnection) closeTransport(reason string) {
	c.closeOnce.Do(func() {
		_ = c.qconn.CloseWithError(quicErrShutdown, reason)
	})
}

// sendCertRenew delivers a certificate rotation to the agent over the control
// stream, mirroring the WebSocket msgCertRenew path.
func (c *quicConnection) sendCertRenew(certPEM, keyPEM string) error {
	c.mu.Lock()
	ctrl := c.control
	c.mu.Unlock()
	if ctrl == nil {
		return errors.New("control stream unavailable")
	}
	return writeJSONLine(ctrl, quicControlMsg{Type: "certRenew", ClientCert: certPEM, ClientKey: keyPEM})
}

// quicServerConfig returns a hardened copy of the configured server TLS config:
// TLS 1.3, the tunnel ALPN, and a client-certificate request (verified
// per-gateway against the tenant CA after the handshake, in authenticateTunnel).
func (b *Broker) quicServerConfig() *tls.Config {
	cfg := b.config.QUICTLSConfig.Clone()
	cfg.MinVersion = tls.VersionTLS13
	// Always negotiate the tunnel ALPN. A caller-supplied NextProtos that omits
	// TunnelALPN would otherwise let the listener start but fail every agent
	// handshake with an ALPN mismatch.
	cfg.NextProtos = []string{TunnelALPN}
	if cfg.ClientAuth < tls.RequireAnyClientCert {
		cfg.ClientAuth = tls.RequireAnyClientCert
	}
	return cfg
}

func (b *Broker) quicConfig() *quic.Config {
	return &quic.Config{
		MaxIncomingStreams: quicMaxIncomingStreams,
		KeepAlivePeriod:    quicKeepAlivePeriod,
		MaxIdleTimeout:     quicMaxIdleTimeout,
	}
}

// StartQUIC binds the QUIC tunnel listener and returns a serve function to run
// in the background. Binding happens synchronously so a failure (for example the
// UDP port is already in use) is returned to the caller instead of being lost in
// a goroutine — the broker can then refuse to start rather than appear healthy
// with no QUIC socket. Returns (nil, nil) when QUICTLSConfig is not configured,
// so the broker stays WebSocket-only by default.
func (b *Broker) StartQUIC() (func(context.Context) error, error) {
	listener, err := b.quicListener()
	if err != nil {
		return nil, err
	}
	if listener == nil {
		return nil, nil
	}
	b.config.Logger.Info("tunnel QUIC listener started", "addr", listener.Addr().String())
	return func(ctx context.Context) error { return b.serveQUIC(ctx, listener) }, nil
}

// quicListener binds the QUIC listener, or returns (nil, nil) when the QUIC
// transport is not configured (WebSocket-only mode).
func (b *Broker) quicListener() (*quic.Listener, error) {
	if b.config.QUICTLSConfig == nil {
		return nil, nil
	}
	addr := b.config.QUICListenAddr
	if addr == "" {
		addr = quicDefaultListenAddr
	}
	listener, err := quic.ListenAddr(addr, b.quicServerConfig(), b.quicConfig())
	if err != nil {
		return nil, fmt.Errorf("listen quic: %w", err)
	}
	return listener, nil
}

// serveQUIC runs the accept loop until ctx is cancelled, closing the listener on
// exit. Each accepted connection is authenticated and served in its own
// goroutine.
func (b *Broker) serveQUIC(ctx context.Context, listener *quic.Listener) error {
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		qc, err := listener.Accept(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			b.config.Logger.Warn("quic accept failed", "error", err)
			continue
		}
		go b.handleQUICConn(ctx, qc)
	}
}

func (b *Broker) handleQUICConn(ctx context.Context, qc *quic.Conn) {
	clientIP := hostOnly(qc.RemoteAddr())

	acceptCtx, cancel := context.WithTimeout(ctx, quicControlAcceptTimeout)
	ctrl, err := qc.AcceptStream(acceptCtx)
	cancel()
	if err != nil {
		_ = qc.CloseWithError(quicErrAuth, "control stream not opened")
		return
	}

	// Bound the unauthenticated hello: an attacker who can reach the UDP listener
	// could otherwise open the control stream and send a newline-less payload
	// that ReadBytes buffers without limit, before any auth. A well-behaved agent
	// sends only the small hello and then waits for the ack, so the bounded read
	// never truncates a legitimate control stream.
	line, err := bufio.NewReader(io.LimitReader(ctrl, quicMaxHelloBytes)).ReadBytes('\n')
	if err != nil {
		_ = qc.CloseWithError(quicErrAuth, "read hello failed")
		return
	}
	var hello quicHello
	if err := json.Unmarshal(trimLine(line), &hello); err != nil {
		_ = qc.CloseWithError(quicErrAuth, "invalid hello")
		return
	}

	certs := qc.ConnectionState().TLS.PeerCertificates
	if len(certs) == 0 {
		_ = b.config.Store.InsertTunnelAudit(ctx, "TUNNEL_MTLS_REJECTED", hello.GatewayID, clientIP, map[string]any{"reason": "missing client certificate", "transport": "quic"})
		_ = qc.CloseWithError(quicErrAuth, "missing client certificate")
		return
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certs[0].Raw})

	if _, err := b.authenticateTunnel(ctx, hello.GatewayID, hello.Token, string(certPEM)); err != nil {
		b.config.Logger.Warn("quic tunnel authentication failed", "gateway_id", hello.GatewayID, "error", err)
		_ = b.config.Store.InsertTunnelAudit(ctx, "TUNNEL_MTLS_REJECTED", hello.GatewayID, clientIP, map[string]any{"reason": err.Error(), "transport": "quic"})
		_ = qc.CloseWithError(quicErrAuth, "authentication failed")
		return
	}

	conn := &quicConnection{
		broker:        b,
		gatewayID:     hello.GatewayID,
		qconn:         qc,
		connectedAt:   time.Now().UTC(),
		clientVersion: hello.ClientVersion,
		clientIP:      clientIP,
		control:       ctrl,
	}
	b.registerQUIC(conn)

	if err := writeJSONLine(ctrl, quicControlMsg{Type: "ack"}); err != nil {
		b.deregister(conn, conn.gatewayID, clientIP, "ack failed")
		return
	}

	if err := b.config.Store.MarkTunnelConnected(ctx, conn.gatewayID, conn.connectedAt, conn.clientVersion, clientIP); err != nil {
		b.config.Logger.Warn("persist tunnel connect failed", "gateway_id", conn.gatewayID, "error", err)
	}
	if err := b.config.Store.InsertTunnelAudit(ctx, "TUNNEL_CONNECT", conn.gatewayID, clientIP, map[string]any{
		"clientVersion": conn.clientVersion,
		"clientIp":      clientIP,
		"transport":     "quic",
	}); err != nil {
		b.config.Logger.Warn("insert tunnel connect audit failed", "gateway_id", conn.gatewayID, "error", err)
	}

	b.runQUICControl(conn, bufio.NewReader(ctrl))
	b.deregister(conn, conn.gatewayID, clientIP, "client_closed")
}

func (b *Broker) registerQUIC(conn *quicConnection) {
	b.mu.Lock()
	existing := b.registry[conn.gatewayID]
	b.registry[conn.gatewayID] = conn
	b.mu.Unlock()
	if existing != nil {
		existing.closeTransport("replaced")
	}
}

// runQUICControl reads heartbeat messages off the control stream until it errors
// (the agent disconnected), persisting them through the same Store path the
// WebSocket PING/PONG handler uses.
func (b *Broker) runQUICControl(conn *quicConnection, reader *bufio.Reader) {
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}
		var msg quicControlMsg
		if err := json.Unmarshal(trimLine(line), &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "heartbeat":
			now := time.Now().UTC()
			conn.mu.Lock()
			conn.lastHeartbeat = now
			if msg.Heartbeat != nil {
				conn.heartbeat = msg.Heartbeat
			} else {
				conn.heartbeat = &HeartbeatMetadata{Healthy: true}
			}
			hb := conn.heartbeat
			conn.mu.Unlock()
			if err := b.config.Store.MarkTunnelHeartbeat(context.Background(), conn.gatewayID, now, hb); err != nil {
				b.config.Logger.Warn("persist tunnel heartbeat failed", "gateway_id", conn.gatewayID, "error", err)
			}
		}
	}
}

func writeJSONLine(w io.Writer, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	_, err = w.Write(payload)
	return err
}

func trimLine(line []byte) []byte {
	for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
		line = line[:len(line)-1]
	}
	return line
}

func hostOnly(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}
