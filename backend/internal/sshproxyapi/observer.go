package sshproxyapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dnviti/arsenale/backend/internal/sessionadmin"
	"github.com/dnviti/arsenale/backend/internal/terminalbroker"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/gorilla/websocket"
)

const (
	sshProxyObserverGrantTTL      = 5 * time.Minute
	sshProxyObserverWebSocketPath = "/api/sessions/ssh-proxy/observe/ws"
	sshProxyObserverRingLimit     = 64 * 1024
	sshProxyObserverQueueSize     = 128
)

var (
	defaultProxyObserverHub = newProxyObserverHub()
	proxyObserverUpgrader   = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
)

type observerRequestError struct {
	status  int
	message string
}

func (e *observerRequestError) Error() string {
	return e.message
}

func (e *observerRequestError) StatusCode() int {
	return e.status
}

type proxyObserverServerMessage struct {
	Type    string `json:"type"`
	Data    string `json:"data,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type proxyObserverClientMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

type proxyObserverSocketInput struct {
	message proxyObserverClientMessage
	err     error
}

type proxyObserverEvent struct {
	typ     string
	data    string
	code    string
	message string
}

type proxyObserverHub struct {
	mu       sync.Mutex
	sessions map[string]*proxyObservedSession
}

type proxyObservedSession struct {
	sessionID   string
	mu          sync.Mutex
	ring        []byte
	closed      bool
	subscribers map[chan proxyObserverEvent]struct{}
}

func newProxyObserverHub() *proxyObserverHub {
	return &proxyObserverHub{sessions: make(map[string]*proxyObservedSession)}
}

func (h *proxyObserverHub) register(sessionID string) *proxyObservedSession {
	sessionID = strings.TrimSpace(sessionID)
	observed := &proxyObservedSession{
		sessionID:   sessionID,
		subscribers: make(map[chan proxyObserverEvent]struct{}),
	}
	if sessionID == "" {
		return observed
	}

	h.mu.Lock()
	h.sessions[sessionID] = observed
	h.mu.Unlock()
	return observed
}

func (h *proxyObserverHub) get(sessionID string) (*proxyObservedSession, bool) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, false
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	observed, ok := h.sessions[sessionID]
	if !ok || observed == nil || observed.isClosed() {
		return nil, false
	}
	return observed, true
}

func (h *proxyObserverHub) unregister(sessionID string, observed *proxyObservedSession) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || observed == nil {
		return
	}

	h.mu.Lock()
	if current := h.sessions[sessionID]; current == observed {
		delete(h.sessions, sessionID)
	}
	h.mu.Unlock()
}

func (s *proxyObservedSession) subscribe() (string, <-chan proxyObserverEvent, func(), bool) {
	if s == nil {
		return "", nil, nil, false
	}

	ch := make(chan proxyObserverEvent, sshProxyObserverQueueSize)
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		close(ch)
		return "", ch, func() {}, false
	}
	snapshot := string(append([]byte(nil), s.ring...))
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()

	unsubscribe := func() {
		s.mu.Lock()
		if _, ok := s.subscribers[ch]; ok {
			delete(s.subscribers, ch)
			close(ch)
		}
		s.mu.Unlock()
	}
	return snapshot, ch, unsubscribe, true
}

func (s *proxyObservedSession) broadcastData(data []byte) {
	if s == nil || len(data) == 0 {
		return
	}

	event := proxyObserverEvent{typ: "data", data: string(data)}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.ring = append(s.ring, data...)
	if len(s.ring) > sshProxyObserverRingLimit {
		s.ring = append([]byte(nil), s.ring[len(s.ring)-sshProxyObserverRingLimit:]...)
	}
	for subscriber := range s.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (s *proxyObservedSession) close(code, message string) {
	if s == nil {
		return
	}

	eventType := "closed"
	if strings.TrimSpace(code) != "" {
		eventType = "error"
	}
	event := proxyObserverEvent{typ: eventType, code: code, message: message}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	for subscriber := range s.subscribers {
		select {
		case subscriber <- event:
		default:
		}
		close(subscriber)
	}
	s.subscribers = make(map[chan proxyObserverEvent]struct{})
	s.mu.Unlock()
}

func (s *proxyObservedSession) isClosed() bool {
	if s == nil {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (s Service) IssueSSHProxyObserverGrant(ctx context.Context, sessionID, observerUserID string, request *http.Request) (sessionadmin.SSHObserveGrantResponse, error) {
	sessionID = strings.TrimSpace(sessionID)
	observerUserID = strings.TrimSpace(observerUserID)
	_ = ctx
	_ = request
	if sessionID == "" {
		return sessionadmin.SSHObserveGrantResponse{}, &observerRequestError{status: http.StatusBadRequest, message: "sessionId is required"}
	}

	secret, err := terminalbroker.LoadSecret()
	if err != nil {
		return sessionadmin.SSHObserveGrantResponse{}, fmt.Errorf("load SSH proxy observer secret: %w", err)
	}
	expiresAt := time.Now().UTC().Add(sshProxyObserverGrantTTL)
	token, err := terminalbroker.IssueGrant(secret, contracts.TerminalSessionGrant{
		Mode:      contracts.TerminalSessionModeObserve,
		SessionID: sessionID,
		UserID:    observerUserID,
		ExpiresAt: expiresAt,
		Metadata: map[string]string{
			"transport": "ssh-proxy",
		},
	})
	if err != nil {
		return sessionadmin.SSHObserveGrantResponse{}, fmt.Errorf("issue SSH proxy observer grant: %w", err)
	}

	return sessionadmin.SSHObserveGrantResponse{
		SessionID:     sessionID,
		Token:         token,
		ExpiresAt:     expiresAt,
		WebSocketPath: sshProxyObserverWebSocketPath,
		Mode:          contracts.TerminalSessionModeObserve,
		ReadOnly:      true,
	}, nil
}

func (s Service) HandleObserveWebSocket(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	conn, err := proxyObserverUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	secret, err := terminalbroker.LoadSecret()
	if err != nil {
		_ = sendProxyObserverMessage(conn, proxyObserverServerMessage{Type: "error", Code: "INVALID_TOKEN", Message: "observer secret is unavailable"})
		return
	}
	grant, err := terminalbroker.ValidateGrant(secret, token, time.Now().UTC())
	if err != nil || grant.Mode != contracts.TerminalSessionModeObserve {
		message := "invalid observer token"
		if err != nil {
			message = err.Error()
		}
		_ = sendProxyObserverMessage(conn, proxyObserverServerMessage{Type: "error", Code: "INVALID_TOKEN", Message: message})
		return
	}

	observed, ok := defaultProxyObserverHub.get(grant.SessionID)
	if !ok {
		_ = sendProxyObserverMessage(conn, proxyObserverServerMessage{Type: "error", Code: "SESSION_NOT_FOUND", Message: "active SSH proxy session is not available for observation"})
		return
	}
	snapshot, events, unsubscribe, ok := observed.subscribe()
	if !ok {
		_ = sendProxyObserverMessage(conn, proxyObserverServerMessage{Type: "error", Code: "SESSION_NOT_FOUND", Message: "active SSH proxy session is not available for observation"})
		return
	}
	defer unsubscribe()

	if err := sendProxyObserverMessage(conn, proxyObserverServerMessage{Type: "ready"}); err != nil {
		return
	}
	if snapshot != "" {
		if err := sendProxyObserverMessage(conn, proxyObserverServerMessage{Type: "data", Data: snapshot}); err != nil {
			return
		}
	}

	inputs := make(chan proxyObserverSocketInput, 1)
	go readProxyObserverMessages(conn, inputs)

	for {
		select {
		case event, ok := <-events:
			if !ok {
				_ = sendProxyObserverMessage(conn, proxyObserverServerMessage{Type: "closed"})
				return
			}
			if err := sendProxyObserverMessage(conn, event.toServerMessage()); err != nil {
				return
			}
			if event.typ == "closed" || event.typ == "error" {
				return
			}
		case input, ok := <-inputs:
			if !ok {
				return
			}
			if input.err != nil {
				if errors.Is(input.err, errProxyObserverClosed) {
					return
				}
				_ = sendProxyObserverMessage(conn, proxyObserverServerMessage{Type: "error", Code: "PROTOCOL_ERROR", Message: input.err.Error()})
				return
			}
			if !handleProxyObserverInput(conn, input.message) {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

var errProxyObserverClosed = errors.New("observer websocket closed")

func readProxyObserverMessages(conn *websocket.Conn, inputs chan<- proxyObserverSocketInput) {
	defer close(inputs)
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			sendProxyObserverInput(inputs, proxyObserverSocketInput{err: errProxyObserverClosed})
			return
		}
		var message proxyObserverClientMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			sendProxyObserverInput(inputs, proxyObserverSocketInput{err: errors.New("invalid websocket payload")})
			return
		}
		sendProxyObserverInput(inputs, proxyObserverSocketInput{message: message})
	}
}

func sendProxyObserverInput(inputs chan<- proxyObserverSocketInput, input proxyObserverSocketInput) {
	select {
	case inputs <- input:
	default:
	}
}

func handleProxyObserverInput(conn *websocket.Conn, message proxyObserverClientMessage) bool {
	switch message.Type {
	case "ping":
		return sendProxyObserverMessage(conn, proxyObserverServerMessage{Type: "pong"}) == nil
	case "close":
		return false
	case "input", "resize":
		return sendProxyObserverMessage(conn, proxyObserverServerMessage{Type: "error", Code: "READ_ONLY", Message: "observer connection is read-only"}) == nil
	default:
		_ = sendProxyObserverMessage(conn, proxyObserverServerMessage{Type: "error", Code: "PROTOCOL_ERROR", Message: "unsupported terminal message"})
		return false
	}
}

func sendProxyObserverMessage(conn *websocket.Conn, message proxyObserverServerMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, payload)
}

func (e proxyObserverEvent) toServerMessage() proxyObserverServerMessage {
	return proxyObserverServerMessage{
		Type:    e.typ,
		Data:    e.data,
		Code:    e.code,
		Message: e.message,
	}
}
