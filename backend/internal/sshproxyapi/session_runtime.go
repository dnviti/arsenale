package sshproxyapi

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/ssh"
)

const (
	proxySessionStatePollInterval = time.Second
	proxySessionHeartbeatInterval = 10 * time.Second
)

var defaultProxyRuntimeRegistry = newProxyRuntimeRegistry()

type proxySessionState struct {
	exists bool
	closed bool
	paused bool
	status string
}

type proxySessionControl struct {
	ctx       context.Context
	cancel    context.CancelFunc
	sessionID string
	observer  *proxyObservedSession
	closeFn   func()

	pausedMu sync.Mutex
	paused   bool
	activeMu sync.Mutex
	channels map[ssh.Channel]struct{}
	sessions map[*ssh.Session]struct{}
	closeMu  sync.Once
}

type proxyRuntimeRegistry struct {
	mu       sync.Mutex
	controls map[string]*proxySessionControl
}

func newProxyRuntimeRegistry() *proxyRuntimeRegistry {
	return &proxyRuntimeRegistry{controls: make(map[string]*proxySessionControl)}
}

func (r *proxyRuntimeRegistry) register(sessionID string, control *proxySessionControl) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || control == nil {
		return
	}

	r.mu.Lock()
	r.controls[sessionID] = control
	r.mu.Unlock()
	slog.Default().Debug("registered live SSH proxy session", "session_id", sessionID)
}

func (r *proxyRuntimeRegistry) unregister(sessionID string, control *proxySessionControl) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || control == nil {
		return
	}

	r.mu.Lock()
	if current := r.controls[sessionID]; current == control {
		delete(r.controls, sessionID)
	}
	r.mu.Unlock()
}

func (r *proxyRuntimeRegistry) get(sessionID string) (*proxySessionControl, bool) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, false
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	control, ok := r.controls[sessionID]
	if !ok || control == nil {
		return nil, false
	}
	select {
	case <-control.done():
		delete(r.controls, sessionID)
		return nil, false
	default:
		return control, true
	}
}

func newProxySessionControl(parent context.Context, sessionID string, observer *proxyObservedSession, closeFn func()) *proxySessionControl {
	ctx, cancel := context.WithCancel(parent)
	return &proxySessionControl{
		ctx:       ctx,
		cancel:    cancel,
		sessionID: sessionID,
		observer:  observer,
		closeFn:   closeFn,
		channels:  make(map[ssh.Channel]struct{}),
		sessions:  make(map[*ssh.Session]struct{}),
	}
}

func (s Service) TerminateLiveSession(sessionID string) bool {
	control, ok := defaultProxyRuntimeRegistry.get(sessionID)
	if !ok {
		slog.Default().Warn("live SSH proxy session not found for terminate", "session_id", sessionID)
		return false
	}
	slog.Default().Debug("terminating live SSH proxy session", "session_id", sessionID)
	control.terminate("SESSION_TERMINATED", "Session terminated by administrator")
	return true
}

func (s Service) PauseLiveSession(sessionID string) bool {
	control, ok := defaultProxyRuntimeRegistry.get(sessionID)
	if !ok {
		slog.Default().Warn("live SSH proxy session not found for pause", "session_id", sessionID)
		return false
	}
	control.setPaused(true)
	return true
}

func (s Service) ResumeLiveSession(sessionID string) bool {
	control, ok := defaultProxyRuntimeRegistry.get(sessionID)
	if !ok {
		slog.Default().Warn("live SSH proxy session not found for resume", "session_id", sessionID)
		return false
	}
	control.setPaused(false)
	return true
}

func (c *proxySessionControl) done() <-chan struct{} {
	if c == nil {
		return nil
	}
	return c.ctx.Done()
}

func (c *proxySessionControl) stopTransport() {
	if c == nil {
		return
	}
	c.closeMu.Do(func() {
		c.cancel()
		c.closeActiveSSH()
		if c.closeFn != nil {
			c.closeFn()
		}
	})
}

func (c *proxySessionControl) finish() {
	if c == nil {
		return
	}
	if c.observer != nil {
		c.observer.close("", "")
	}
	c.stopTransport()
}

func (c *proxySessionControl) terminate(code, message string) {
	if c == nil {
		return
	}
	if c.observer != nil {
		c.observer.close(code, message)
	}
	c.stopTransport()
}

func (c *proxySessionControl) observeOutput(data []byte) {
	if c == nil || c.observer == nil {
		return
	}
	c.observer.broadcastData(data)
}

func (c *proxySessionControl) registerActiveSSH(channel ssh.Channel, session *ssh.Session) func() {
	if c == nil {
		return func() {}
	}

	c.activeMu.Lock()
	if channel != nil {
		c.channels[channel] = struct{}{}
	}
	if session != nil {
		c.sessions[session] = struct{}{}
	}
	c.activeMu.Unlock()

	return func() {
		c.activeMu.Lock()
		if channel != nil {
			delete(c.channels, channel)
		}
		if session != nil {
			delete(c.sessions, session)
		}
		c.activeMu.Unlock()
	}
}

func (c *proxySessionControl) closeActiveSSH() {
	c.activeMu.Lock()
	channels := make([]ssh.Channel, 0, len(c.channels))
	for channel := range c.channels {
		channels = append(channels, channel)
	}
	sessions := make([]*ssh.Session, 0, len(c.sessions))
	for session := range c.sessions {
		sessions = append(sessions, session)
	}
	c.activeMu.Unlock()

	for _, session := range sessions {
		_ = session.Close()
	}
	for _, channel := range channels {
		_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(struct {
			Status uint32
		}{Status: 255}))
		_ = channel.Close()
	}
}

func (c *proxySessionControl) setPaused(paused bool) {
	if c == nil {
		return
	}
	c.pausedMu.Lock()
	c.paused = paused
	c.pausedMu.Unlock()
}

func (c *proxySessionControl) isPaused() bool {
	if c == nil {
		return false
	}
	c.pausedMu.Lock()
	defer c.pausedMu.Unlock()
	return c.paused
}

func (c *proxySessionControl) waitUntilResumed() bool {
	if c == nil {
		return true
	}
	for c.isPaused() {
		select {
		case <-c.ctx.Done():
			return false
		case <-time.After(100 * time.Millisecond):
		}
	}
	return true
}

func (s Service) watchProxySessionState(control *proxySessionControl) {
	if control == nil || control.sessionID == "" {
		return
	}

	ticker := time.NewTicker(proxySessionStatePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-control.done():
			return
		case <-ticker.C:
			queryCtx, cancel := context.WithTimeout(control.ctx, 3*time.Second)
			state, err := s.loadProxySessionState(queryCtx, control.sessionID)
			cancel()
			if err != nil {
				select {
				case <-control.done():
					return
				default:
				}
				slog.Default().Warn("load SSH proxy session state failed", "session_id", control.sessionID, "error", err)
				continue
			}
			if !state.exists {
				control.terminate("SESSION_CLOSED", "Session closed")
				return
			}
			if state.closed {
				control.terminate("SESSION_TERMINATED", "Session terminated by administrator")
				return
			}
			control.setPaused(state.paused)
		}
	}
}

func (s Service) runProxySessionHeartbeat(control *proxySessionControl) {
	if control == nil || control.sessionID == "" {
		return
	}

	ticker := time.NewTicker(proxySessionHeartbeatInterval)
	defer ticker.Stop()
	s.heartbeatProxySession(control.sessionID)

	for {
		select {
		case <-control.done():
			return
		case <-ticker.C:
			s.heartbeatProxySession(control.sessionID)
		}
	}
}

func (s Service) loadProxySessionState(ctx context.Context, sessionID string) (proxySessionState, error) {
	if s.DB == nil {
		return proxySessionState{}, fmt.Errorf("database is unavailable")
	}

	row := s.DB.QueryRow(
		ctx,
		`SELECT status::text
		   FROM "ActiveSession"
		  WHERE id = $1
		    AND protocol = 'SSH_PROXY'::"SessionProtocol"`,
		sessionID,
	)

	var status string
	if err := row.Scan(&status); err != nil {
		if err == pgx.ErrNoRows {
			return proxySessionState{}, nil
		}
		return proxySessionState{}, fmt.Errorf("load SSH proxy session state: %w", err)
	}

	status = strings.ToUpper(strings.TrimSpace(status))
	return proxySessionState{
		exists: true,
		closed: status == "CLOSED",
		paused: status == "PAUSED",
		status: status,
	}, nil
}

func (s Service) heartbeatProxySession(sessionID string) {
	if s.DB == nil || strings.TrimSpace(sessionID) == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := s.DB.Exec(
		ctx,
		`UPDATE "ActiveSession"
		    SET "lastActivityAt" = NOW(),
		        status = CASE
		            WHEN status = 'PAUSED'::"SessionStatus" THEN 'PAUSED'::"SessionStatus"
		            ELSE 'ACTIVE'::"SessionStatus"
		        END
		  WHERE id = $1
		    AND protocol = 'SSH_PROXY'::"SessionProtocol"
		    AND status <> 'CLOSED'::"SessionStatus"`,
		sessionID,
	); err != nil {
		slog.Default().Warn("heartbeat SSH proxy session failed", "session_id", sessionID, "error", err)
	}
}
