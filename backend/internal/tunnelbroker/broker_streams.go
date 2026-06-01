package tunnelbroker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func (c *tunnelConnection) openStream(ctx context.Context, host string, port int, timeout time.Duration) (io.ReadWriteCloser, error) {
	b := c.broker

	b.mu.Lock()
	streamID, ok := allocateStreamID(c)
	if !ok {
		b.mu.Unlock()
		return nil, errors.New("no available tunnel stream IDs")
	}

	wait := &pendingOpen{resolve: make(chan *streamConn, 1)}
	wait.timer = time.AfterFunc(timeout, func() {
		b.mu.Lock()
		current := c.pendingOpens[streamID]
		if current == wait {
			delete(c.pendingOpens, streamID)
		}
		b.mu.Unlock()
		if current == wait {
			select {
			case wait.resolve <- nil:
			default:
			}
		}
	})
	c.pendingOpens[streamID] = wait
	b.mu.Unlock()

	target := []byte(net.JoinHostPort(host, strconv.Itoa(port)))
	if err := b.sendFrame(c, msgOpen, streamID, target); err != nil {
		wait.timer.Stop()
		b.mu.Lock()
		delete(c.pendingOpens, streamID)
		b.mu.Unlock()
		return nil, err
	}

	select {
	case stream := <-wait.resolve:
		if stream == nil {
			return nil, fmt.Errorf("openStream timeout for gateway %s -> %s:%d", c.gatewayID, host, port)
		}
		return stream, nil
	case <-ctx.Done():
		wait.timer.Stop()
		b.mu.Lock()
		delete(c.pendingOpens, streamID)
		b.mu.Unlock()
		return nil, ctx.Err()
	}
}

func (b *Broker) createTCPProxy(req contracts.TunnelProxyRequest) (contracts.TunnelProxyResponse, error) {
	timeout := defaultOpenTimeout
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	idleTimeout := defaultProxyIdleTimeout
	if req.IdleTimeout > 0 {
		idleTimeout = time.Duration(req.IdleTimeout) * time.Millisecond
	}

	b.mu.RLock()
	conn := b.registry[req.GatewayID]
	b.mu.RUnlock()
	if conn == nil {
		return contracts.TunnelProxyResponse{}, fmt.Errorf("no active tunnel for gateway %s", req.GatewayID)
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(b.config.ProxyBindHost, "0"))
	if err != nil {
		return contracts.TunnelProxyResponse{}, fmt.Errorf("listen tunnel proxy: %w", err)
	}

	proxyID := uuid.NewString()
	idleTimer := time.AfterFunc(idleTimeout, func() {
		_ = listener.Close()
	})

	go func() {
		defer idleTimer.Stop()
		defer listener.Close()

		socket, err := listener.Accept()
		if err != nil {
			return
		}
		idleTimer.Stop()
		defer socket.Close()

		// Re-resolve the tunnel at accept time: the gateway may have reconnected
		// (evicting the transport captured when this endpoint was created) before
		// the client connected, so the captured conn could be closed.
		b.mu.RLock()
		conn := b.registry[req.GatewayID]
		b.mu.RUnlock()
		if conn == nil {
			return
		}

		var stream io.ReadWriteCloser
		for _, targetPort := range targetPortCandidates(req.TargetPort, req.TargetPorts) {
			stream, err = conn.openStream(context.Background(), req.TargetHost, targetPort, timeout)
			if err == nil {
				break
			}
		}
		if stream == nil {
			return
		}
		defer stream.Close()

		done := make(chan struct{}, 2)
		go func() {
			_, _ = io.Copy(stream, socket)
			_ = stream.Close()
			done <- struct{}{}
		}()
		go func() {
			_, _ = io.Copy(socket, stream)
			_ = socket.Close()
			done <- struct{}{}
		}()
		<-done
	}()

	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return contracts.TunnelProxyResponse{}, errors.New("unexpected tunnel proxy listener type")
	}

	return contracts.TunnelProxyResponse{
		ID:        proxyID,
		Host:      b.config.ProxyAdvertiseHost,
		Port:      address.Port,
		ExpiresIn: int(idleTimeout / time.Millisecond),
	}, nil
}

func targetPortCandidates(primary int, additional []int) []int {
	seen := map[int]struct{}{}
	ports := make([]int, 0, len(additional)+1)
	for _, port := range append([]int{primary}, additional...) {
		if port <= 0 {
			continue
		}
		if _, ok := seen[port]; ok {
			continue
		}
		seen[port] = struct{}{}
		ports = append(ports, port)
	}
	return ports
}

func (b *Broker) sendFrame(conn *tunnelConnection, frameType msgType, streamID uint16, payload []byte) error {
	frame := buildFrame(frameType, streamID, payload)
	conn.sendMu.Lock()
	defer conn.sendMu.Unlock()
	return conn.ws.WriteMessage(websocket.BinaryMessage, frame)
}

func newStreamConn(parent *tunnelConnection, streamID uint16) *streamConn {
	reader, writer := io.Pipe()
	return &streamConn{
		parent: parent,
		id:     streamID,
		reader: reader,
		writer: writer,
		closed: make(chan struct{}),
	}
}

func (s *streamConn) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *streamConn) Write(p []byte) (int, error) {
	select {
	case <-s.closed:
		return 0, io.ErrClosedPipe
	default:
	}
	if err := s.parent.broker.sendFrame(s.parent, msgData, s.id, p); err != nil {
		return 0, err
	}
	s.parent.bytesTransferred.Add(int64(len(p)))
	return len(p), nil
}

func (s *streamConn) Close() error {
	return s.close(true)
}

func (s *streamConn) close(sendClose bool) error {
	var err error
	s.closeOnce.Do(func() {
		close(s.closed)
		if sendClose {
			err = s.parent.broker.sendFrame(s.parent, msgClose, s.id, nil)
		}
		s.parent.broker.mu.Lock()
		delete(s.parent.streams, s.id)
		s.parent.broker.mu.Unlock()
		s.parent.activeStreams.Add(-1)
		_ = s.writer.Close()
		_ = s.reader.Close()
	})
	return err
}

func allocateStreamID(conn *tunnelConnection) (uint16, bool) {
	streamID := conn.nextStreamID
	for attempts := 0; attempts < maxStreamID; attempts++ {
		if streamID == 0 {
			streamID = 1
		}
		if conn.streams[streamID] == nil && conn.pendingOpens[streamID] == nil {
			conn.nextStreamID = streamID + 1
			return streamID, true
		}
		streamID++
		if streamID > maxStreamID {
			streamID = 1
		}
	}
	return 0, false
}
