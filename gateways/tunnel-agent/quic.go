package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/dnviti/arsenale/gateways/gateway-core/auth"
	"github.com/quic-go/quic-go"
)

// quicALPN must match the broker's tunnelbroker.TunnelALPN.
const quicALPN = "arsenale-tunnel/1"

const (
	quicKeepAlivePeriod = 15 * time.Second
	quicMaxIdleTimeout  = 45 * time.Second
	quicHandshakeWait   = 10 * time.Second
	// autoQUICCooldown is how many WSS-fallback cycles to run after a failed QUIC
	// dial before retrying QUIC, so a UDP-blocked network does not repeatedly pay
	// the QUIC handshake timeout on every reconnect.
	autoQUICCooldown = 5
)

type quicHello struct {
	GatewayID     string `json:"gatewayId"`
	Token         string `json:"token"`
	ClientVersion string `json:"clientVersion"`
}

type quicHeartbeat struct {
	Healthy       bool `json:"healthy"`
	LatencyMs     *int `json:"latencyMs,omitempty"`
	ActiveStreams *int `json:"activeStreams,omitempty"`
}

type quicControlMsg struct {
	Type       string         `json:"type"`
	Heartbeat  *quicHeartbeat `json:"heartbeat,omitempty"`
	ClientCert string         `json:"clientCert,omitempty"`
	ClientKey  string         `json:"clientKey,omitempty"`
}

// runQUIC is the QUIC transport equivalent of the WebSocket Run loop: dial out
// to the broker, authenticate on a control stream, then accept broker-opened
// streams and forward each to the configured local service. It reconnects with
// the same exponential backoff as the WebSocket path.
func (a *Agent) runQUIC(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			a.log("Agent stopped")
			return err
		}

		qc, err := a.dialQUIC(ctx)
		if err != nil {
			a.err("QUIC dial error: %v", err)
			if !a.waitReconnect(ctx) {
				a.log("Agent stopped")
				return ctx.Err()
			}
			continue
		}

		a.log("Connected to TunnelBroker over QUIC")
		a.reconnectDelay = a.cfg.ReconnectInitial
		serveErr := a.serveQUIC(ctx, qc)
		_ = qc.CloseWithError(0, "agent reconnecting")

		if errors.Is(serveErr, context.Canceled) || ctx.Err() != nil {
			a.log("Agent stopped")
			return ctx.Err()
		}
		a.warn("QUIC connection closed (%v). Reconnecting in %s", serveErr, a.reconnectDelay)
		if !a.waitReconnect(ctx) {
			a.log("Agent stopped")
			return ctx.Err()
		}
	}
}

// runAuto prefers QUIC and automatically falls back to the WebSocket transport
// when QUIC cannot be established (e.g. UDP-blocked networks). After a failed
// QUIC dial it stays on WSS for autoQUICCooldown reconnect cycles before
// retrying QUIC, so the handshake timeout is paid at most once per cooldown.
func (a *Agent) runAuto(ctx context.Context) error {
	quicCooldown := 0
	for {
		if err := ctx.Err(); err != nil {
			a.forwarder.DestroyAll()
			a.log("Agent stopped")
			return err
		}

		if quicCooldown == 0 {
			qc, err := a.dialQUIC(ctx)
			if err == nil {
				a.log("Connected to TunnelBroker over QUIC")
				a.reconnectDelay = a.cfg.ReconnectInitial
				serveErr := a.serveQUIC(ctx, qc)
				_ = qc.CloseWithError(0, "agent reconnecting")
				if errors.Is(serveErr, context.Canceled) || ctx.Err() != nil {
					a.log("Agent stopped")
					return ctx.Err()
				}
				a.warn("QUIC connection closed (%v). Reconnecting in %s", serveErr, a.reconnectDelay)
				if !a.waitReconnect(ctx) {
					return ctx.Err()
				}
				continue
			}
			a.warn("QUIC dial failed (%v); falling back to WSS for %d cycle(s)", err, autoQUICCooldown)
			quicCooldown = autoQUICCooldown
		} else {
			quicCooldown--
		}

		wsConn, err := a.connect(ctx)
		if err != nil {
			a.err("WSS fallback error: %v", err)
			if !a.waitReconnect(ctx) {
				a.forwarder.DestroyAll()
				a.log("Agent stopped")
				return ctx.Err()
			}
			continue
		}
		a.log("Connected to TunnelBroker over WSS (fallback)")
		a.reconnectDelay = a.cfg.ReconnectInitial
		readErr := a.runConnection(ctx, wsConn)
		a.forwarder.DestroyAll()
		_ = wsConn.Close()
		if errors.Is(readErr, context.Canceled) || ctx.Err() != nil {
			a.log("Agent stopped")
			return ctx.Err()
		}
		a.warn("WSS connection closed (%v). Reconnecting in %s", readErr, a.reconnectDelay)
		if !a.waitReconnect(ctx) {
			return ctx.Err()
		}
	}
}

func (a *Agent) dialQUIC(ctx context.Context) (*quic.Conn, error) {
	tlsConfig, err := auth.BuildTLSConfig(a.cfg.CACert, a.cfg.ClientCert, a.cfg.ClientKey)
	if err != nil {
		return nil, err
	}
	tlsConfig.MinVersion = tls.VersionTLS13
	tlsConfig.NextProtos = []string{quicALPN}
	if a.cfg.QUICServerName != "" {
		tlsConfig.ServerName = a.cfg.QUICServerName
	}

	dialCtx, cancel := context.WithTimeout(ctx, quicHandshakeWait)
	defer cancel()
	return quic.DialAddr(dialCtx, a.cfg.QUICServerAddr, tlsConfig, &quic.Config{
		KeepAlivePeriod: quicKeepAlivePeriod,
		MaxIdleTimeout:  quicMaxIdleTimeout,
	})
}

func (a *Agent) serveQUIC(ctx context.Context, qc *quic.Conn) error {
	ctrl, err := qc.OpenStreamSync(ctx)
	if err != nil {
		return err
	}
	hello := quicHello{GatewayID: a.cfg.GatewayID, Token: a.cfg.Token, ClientVersion: a.cfg.AgentVersion}
	if err := writeQUICLine(ctrl, quicControlMsg{}, hello); err != nil {
		return err
	}

	reader := bufio.NewReader(ctrl)
	ack, err := reader.ReadBytes('\n')
	if err != nil {
		return err
	}
	var ackMsg quicControlMsg
	if err := json.Unmarshal(trimQUICLine(ack), &ackMsg); err != nil || ackMsg.Type != "ack" {
		return errors.New("broker did not acknowledge tunnel connection")
	}

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go a.quicHeartbeatLoop(connCtx, ctrl)
	go a.quicControlLoop(connCtx, qc, reader)

	for {
		stream, err := qc.AcceptStream(connCtx)
		if err != nil {
			return err
		}
		go a.handleQUICStream(stream)
	}
}

// quicControlLoop reads broker-initiated control messages (currently certificate
// renewals) until the control stream errors.
func (a *Agent) quicControlLoop(ctx context.Context, qc *quic.Conn, reader *bufio.Reader) {
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}
		var msg quicControlMsg
		if err := json.Unmarshal(trimQUICLine(line), &msg); err != nil {
			continue
		}
		if msg.Type == "certRenew" && msg.ClientCert != "" && msg.ClientKey != "" {
			a.cfg.ClientCert = msg.ClientCert
			a.cfg.ClientKey = msg.ClientKey
			a.log("Tunnel client certificate renewed - reconnecting")
			_ = qc.CloseWithError(0, "client certificate renewed")
			return
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (a *Agent) quicHeartbeatLoop(ctx context.Context, ctrl *quic.Stream) {
	ticker := time.NewTicker(a.cfg.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			health := a.probeLocalService()
			active := int(a.quicActive.Load())
			latency := int(health.LatencyMs)
			msg := quicControlMsg{Type: "heartbeat", Heartbeat: &quicHeartbeat{
				Healthy:       health.Healthy,
				LatencyMs:     &latency,
				ActiveStreams: &active,
			}}
			if err := writeQUICLine(ctrl, msg, nil); err != nil {
				a.warn("Failed to send QUIC heartbeat: %v", err)
				return
			}
		}
	}
}

// handleQUICStream forwards a single broker-opened stream to the local service.
// The broker writes "host:port\n" as the leading line; the agent enforces the
// same localhost-only SSRF guard as the WebSocket forwarder before dialing.
func (a *Agent) handleQUICStream(stream *quic.Stream) {
	br := bufio.NewReader(stream)
	target, err := br.ReadBytes('\n')
	if err != nil {
		_ = stream.Close()
		return
	}
	host, port, ok := parseTarget(string(trimQUICLine(target)))
	if !ok || port < 1 || port > 65535 {
		a.warn("QUIC stream has invalid target: %q", string(trimQUICLine(target)))
		stream.CancelWrite(0)
		stream.CancelRead(0)
		return
	}
	if !isAllowedLocalHost(host) {
		a.warn("QUIC stream rejected: non-localhost host %q is not allowed", host)
		stream.CancelWrite(0)
		stream.CancelRead(0)
		return
	}

	local, err := net.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		a.warn("Local dial error for %s:%d: %v", host, port, err)
		stream.CancelWrite(0)
		stream.CancelRead(0)
		return
	}

	a.quicActive.Add(1)
	defer a.quicActive.Add(-1)

	done := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(local, br)
		_ = local.Close()
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(stream, local)
		_ = stream.Close()
		done <- struct{}{}
	}()
	<-done
	<-done
	_ = local.Close()
	_ = stream.Close()
}

// writeQUICLine marshals either a hello (when hello != nil) or a control message
// to a single newline-delimited JSON line.
func writeQUICLine(w io.Writer, msg quicControlMsg, hello any) error {
	var (
		payload []byte
		err     error
	)
	if hello != nil {
		payload, err = json.Marshal(hello)
	} else {
		payload, err = json.Marshal(msg)
	}
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	_, err = w.Write(payload)
	return err
}

func trimQUICLine(line []byte) []byte {
	for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
		line = line[:len(line)-1]
	}
	return line
}
