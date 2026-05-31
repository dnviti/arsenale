package tunnelbroker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/gorilla/websocket"
)

func (b *Broker) registerConnection(gatewayID string, wsConn *websocket.Conn, clientVersion, clientIP string) *tunnelConnection {
	conn := &tunnelConnection{
		broker:        b,
		gatewayID:     gatewayID,
		ws:            wsConn,
		connectedAt:   time.Now().UTC(),
		clientVersion: clientVersion,
		clientIP:      clientIP,
		streams:       make(map[uint16]*streamConn),
		pendingOpens:  make(map[uint16]*pendingOpen),
		nextStreamID:  1,
	}

	b.mu.Lock()
	existing := b.registry[gatewayID]
	b.registry[gatewayID] = conn
	b.mu.Unlock()

	// Tear the previous connection down outside the lock: closeTransport
	// re-acquires Broker.mu while destroying streams, so calling it under the
	// lock would deadlock. The registry already points at the new connection,
	// so the evicted one's read loop deregisters as a no-op.
	if existing != nil {
		existing.closeTransport("replaced")
	}

	go b.readLoop(conn)
	return conn
}

func (b *Broker) readLoop(conn *tunnelConnection) {
	defer func() {
		b.deregister(conn, conn.gatewayID, conn.clientIP, "client_closed")
	}()

	for {
		messageType, payload, err := conn.ws.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.BinaryMessage {
			continue
		}
		frame, ok := parseFrame(payload)
		if !ok {
			continue
		}

		switch frame.Type {
		case msgOpen:
			b.handleOpenAck(conn, frame.StreamID)
		case msgData:
			b.handleData(conn, frame.StreamID, frame.Payload)
		case msgClose:
			b.handleClose(conn, frame.StreamID)
		case msgPing:
			b.recordHeartbeat(conn, frame.Payload)
			_ = b.sendFrame(conn, msgPong, frame.StreamID, nil)
		case msgPong:
			now := time.Now().UTC()
			conn.statusMu.Lock()
			if !conn.lastPingSentAt.IsZero() {
				latency := now.Sub(conn.lastPingSentAt).Milliseconds()
				conn.pingLatency = &latency
				conn.lastPingSentAt = time.Time{}
			}
			conn.lastHeartbeat = now
			hb := conn.heartbeat
			conn.statusMu.Unlock()
			_ = b.config.Store.MarkTunnelHeartbeat(context.Background(), conn.gatewayID, now, hb)
		case msgHeartbeat:
			b.recordHeartbeat(conn, frame.Payload)
		case msgCertRenew:
			// The current tunnel agent only receives CERT_RENEW from the broker.
			// Ignore any peer-sent CERT_RENEW frame.
		}
	}
}

func (b *Broker) handleOpenAck(conn *tunnelConnection, streamID uint16) {
	b.mu.Lock()
	pending := conn.pendingOpens[streamID]
	if pending != nil {
		delete(conn.pendingOpens, streamID)
	}
	b.mu.Unlock()
	if pending == nil {
		return
	}

	pending.timer.Stop()
	stream := newStreamConn(conn, streamID)

	b.mu.Lock()
	conn.streams[streamID] = stream
	b.mu.Unlock()
	conn.activeStreams.Add(1)

	pending.resolve <- stream
}

func (b *Broker) handleData(conn *tunnelConnection, streamID uint16, payload []byte) {
	b.mu.RLock()
	stream := conn.streams[streamID]
	b.mu.RUnlock()
	if stream == nil {
		return
	}
	conn.bytesTransferred.Add(int64(len(payload)))
	_, _ = stream.writer.Write(payload)
}

func (b *Broker) handleClose(conn *tunnelConnection, streamID uint16) {
	b.mu.Lock()
	stream := conn.streams[streamID]
	if stream != nil {
		delete(conn.streams, streamID)
		b.mu.Unlock()
		_ = stream.close(false)
		return
	}
	pending := conn.pendingOpens[streamID]
	if pending != nil {
		delete(conn.pendingOpens, streamID)
	}
	b.mu.Unlock()
	if pending != nil {
		pending.timer.Stop()
		select {
		case pending.resolve <- nil:
		default:
		}
	}
}

func (b *Broker) recordHeartbeat(conn *tunnelConnection, payload []byte) {
	now := time.Now().UTC()

	var heartbeat *HeartbeatMetadata
	if len(payload) > 0 {
		var parsed HeartbeatMetadata
		if err := json.Unmarshal(payload, &parsed); err == nil {
			heartbeat = &parsed
		} else {
			heartbeat = &HeartbeatMetadata{Healthy: true}
		}
	} else {
		heartbeat = &HeartbeatMetadata{Healthy: true}
	}

	conn.statusMu.Lock()
	conn.lastHeartbeat = now
	conn.heartbeat = heartbeat
	conn.statusMu.Unlock()

	if err := b.config.Store.MarkTunnelHeartbeat(context.Background(), conn.gatewayID, now, heartbeat); err != nil {
		b.config.Logger.Warn("persist tunnel heartbeat failed", "gateway_id", conn.gatewayID, "error", err)
	}
}

// deregister removes a connection from the registry (only if it is still the
// current one for its gateway), tears its transport down, and persists the
// disconnect. It is transport-agnostic and safe to call from any goroutine.
func (b *Broker) deregister(conn tunnelConn, gatewayID, clientIP, reason string) {
	b.mu.Lock()
	if b.registry[gatewayID] != conn {
		b.mu.Unlock()
		return
	}
	delete(b.registry, gatewayID)
	b.mu.Unlock()

	conn.closeTransport(reason)

	if err := b.config.Store.MarkTunnelDisconnected(context.Background(), gatewayID); err != nil {
		b.config.Logger.Warn("persist tunnel disconnect failed", "gateway_id", gatewayID, "error", err)
	}
	if err := b.config.Store.InsertTunnelAudit(context.Background(), "TUNNEL_DISCONNECT", gatewayID, clientIP, map[string]any{
		"reason": reason,
	}); err != nil {
		b.config.Logger.Warn("insert tunnel disconnect audit failed", "gateway_id", gatewayID, "error", err)
	}
}

// closeTransport closes the WebSocket and destroys every logical stream. It is
// idempotent (guarded by closeOnce) and must not be called while holding
// Broker.mu, because stream teardown re-acquires it.
func (c *tunnelConnection) closeTransport(reason string) {
	c.closeOnce.Do(func() {
		_ = c.ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason), time.Now().Add(2*time.Second))

		b := c.broker
		b.mu.Lock()
		streams := c.streams
		pending := c.pendingOpens
		c.streams = make(map[uint16]*streamConn)
		c.pendingOpens = make(map[uint16]*pendingOpen)
		b.mu.Unlock()

		for _, stream := range streams {
			_ = stream.close(false)
		}
		for _, wait := range pending {
			wait.timer.Stop()
			select {
			case wait.resolve <- nil:
			default:
			}
		}
		_ = c.ws.Close()
	})
}

func (b *Broker) getStatus(gatewayID string) (contracts.TunnelStatus, bool) {
	b.mu.RLock()
	conn := b.registry[gatewayID]
	b.mu.RUnlock()
	if conn == nil {
		return contracts.TunnelStatus{}, false
	}
	return conn.describe(), true
}

func (b *Broker) disconnectTunnel(gatewayID, reason string) bool {
	b.mu.RLock()
	conn := b.registry[gatewayID]
	b.mu.RUnlock()
	if conn == nil {
		return false
	}
	b.deregister(conn, gatewayID, conn.describe().ClientIP, reason)
	return true
}

func (c *tunnelConnection) describe() contracts.TunnelStatus {
	c.statusMu.Lock()
	lastHB := c.lastHeartbeat
	lat := c.pingLatency
	hb := c.heartbeat
	c.statusMu.Unlock()

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
