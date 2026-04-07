package terminalbroker

import (
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/dnviti/arsenale/backend/internal/sessionrecording"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

type BrokerConfig struct {
	Secret       string
	SessionStore SessionStore
	Logger       *slog.Logger
}

type Broker struct {
	config   BrokerConfig
	upgrader websocket.Upgrader
}

type clientMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

type serverMessage struct {
	Type    string `json:"type"`
	Data    string `json:"data,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type terminalRuntime struct {
	logger       *slog.Logger
	wsConn       *websocket.Conn
	session      *ssh.Session
	stdin        io.WriteCloser
	stdout       io.Reader
	stderr       io.Reader
	sessionStore SessionStore
	sessionID    string
	recording    *sessionrecording.Reference

	wsWriteMu   sync.Mutex
	recordingMu sync.Mutex
	closeOnce   sync.Once
	closed      chan struct{}
	outputWG    sync.WaitGroup

	activityMu       sync.Mutex
	lastActivityAt   time.Time
	externalCloseMu  sync.Mutex
	externalCloseSet bool
}
