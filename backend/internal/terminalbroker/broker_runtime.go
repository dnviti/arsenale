package terminalbroker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/sessionrecording"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

func (r *terminalRuntime) readWebSocket() {
	for {
		_, payload, err := r.wsConn.ReadMessage()
		if err != nil {
			r.close()
			return
		}

		var message clientMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			_ = r.send(serverMessage{Type: "error", Code: "PROTOCOL_ERROR", Message: "invalid websocket payload"})
			r.close()
			return
		}

		switch message.Type {
		case "input":
			if _, err := io.WriteString(r.stdin, message.Data); err != nil {
				_ = r.send(serverMessage{Type: "error", Code: "WRITE_ERROR", Message: "failed to send terminal input"})
				r.close()
				return
			}
			r.noteActivity(false)
		case "resize":
			if message.Cols > 0 && message.Rows > 0 {
				_ = r.session.WindowChange(message.Rows, message.Cols)
			}
			r.noteActivity(false)
		case "ping":
			r.noteActivity(true)
			if err := r.send(serverMessage{Type: "pong"}); err != nil {
				r.close()
				return
			}
		case "close":
			r.close()
			return
		default:
			_ = r.send(serverMessage{Type: "error", Code: "PROTOCOL_ERROR", Message: "unsupported terminal message"})
			r.close()
			return
		}
	}
}

func (r *terminalRuntime) streamOutput(reader io.Reader) {
	defer r.outputWG.Done()

	buffer := make([]byte, 8192)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			output := string(buffer[:n])
			r.appendRecordingOutput(output)
			if writeErr := r.send(serverMessage{Type: "data", Data: output}); writeErr != nil {
				r.close()
				return
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				r.logger.Warn("terminal stream read error", "error", err)
			}
			return
		}
	}
}

func (r *terminalRuntime) waitForSession() {
	if err := r.session.Wait(); err != nil {
		var exitErr *ssh.ExitError
		if !errors.As(err, &exitErr) {
			_ = r.send(serverMessage{Type: "error", Code: "SESSION_ERROR", Message: "terminal session ended unexpectedly"})
		}
	}

	outputDone := make(chan struct{})
	go func() {
		r.outputWG.Wait()
		close(outputDone)
	}()

	select {
	case <-outputDone:
	case <-time.After(250 * time.Millisecond):
	}

	r.close()
}

func (r *terminalRuntime) monitorSessionState() {
	if r.sessionID == "" {
		return
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Session shutdown can happen outside this websocket process, so the broker
	// keeps polling the persisted state and mirrors the same terminal close code.
	for {
		select {
		case <-r.closed:
			return
		case <-ticker.C:
			state, err := r.sessionStore.GetTerminalSessionState(context.Background(), r.sessionID)
			if err != nil {
				r.logger.Warn("load terminal session state failed", "error", err)
				continue
			}
			if !state.Exists || !state.Closed {
				continue
			}

			r.markExternalClose()
			switch state.Reason {
			case "admin_terminated":
				_ = r.send(serverMessage{Type: "error", Code: "SESSION_TERMINATED", Message: "Session terminated by administrator"})
			case "timeout":
				_ = r.send(serverMessage{Type: "error", Code: "SESSION_TIMEOUT", Message: "Session expired due to inactivity"})
			default:
				_ = r.send(serverMessage{Type: "error", Code: "SESSION_CLOSED", Message: "Session closed"})
			}
			r.close()
			return
		}
	}
}

func (r *terminalRuntime) send(message serverMessage) error {
	r.wsWriteMu.Lock()
	defer r.wsWriteMu.Unlock()
	return sendWebsocketMessage(r.wsConn, message)
}

func (r *terminalRuntime) close() {
	r.closeOnce.Do(func() {
		_ = r.stdin.Close()
		_ = r.session.Close()
		_ = r.send(serverMessage{Type: "closed"})
		closeWebSocketConnection(r.wsConn, websocket.CloseNormalClosure, "")
		if !r.wasExternallyClosed() {
			if err := r.sessionStore.FinalizeTerminalSession(context.Background(), r.sessionID, r.recordingID()); err != nil {
				r.logger.Warn("finalize terminal session failed", "error", err)
			}
		}
		close(r.closed)
	})
}

func (r *terminalRuntime) noteActivity(force bool) {
	if r.sessionID == "" {
		return
	}

	r.activityMu.Lock()
	now := time.Now().UTC()
	if !force && !r.lastActivityAt.IsZero() && now.Sub(r.lastActivityAt) < 10*time.Second {
		r.activityMu.Unlock()
		return
	}
	r.lastActivityAt = now
	r.activityMu.Unlock()

	if err := r.sessionStore.HeartbeatTerminalSession(context.Background(), r.sessionID); err != nil {
		r.logger.Warn("terminal session heartbeat failed", "error", err)
	}
}

func (r *terminalRuntime) markExternalClose() {
	r.externalCloseMu.Lock()
	r.externalCloseSet = true
	r.externalCloseMu.Unlock()
}

func (r *terminalRuntime) wasExternallyClosed() bool {
	r.externalCloseMu.Lock()
	defer r.externalCloseMu.Unlock()
	return r.externalCloseSet
}

func (r *terminalRuntime) appendRecordingOutput(output string) {
	if r.recording == nil || output == "" {
		return
	}

	r.recordingMu.Lock()
	defer r.recordingMu.Unlock()

	if err := sessionrecording.AppendAsciicastOutputAt(r.recording.FilePath, r.recording.StartedAt, time.Now().UTC(), output); err != nil {
		r.logger.Warn("append terminal recording output failed", "error", err)
	}
}

func (r *terminalRuntime) recordingID() string {
	if r.recording == nil {
		return ""
	}
	return strings.TrimSpace(r.recording.ID)
}
