package tunnelbroker

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/gorilla/websocket"
)

const (
	frameHeaderSize         = 4
	maxFramePayloadSize     = 10 * 1024 * 1024
	maxStreamID             = 0xffff
	defaultOpenTimeout      = 10 * time.Second
	defaultProxyIdleTimeout = 60 * time.Second
	defaultTrustDomain      = "arsenale.local"
	aesKeyBytes             = 32
	aesIVBytes              = 16
)

type msgType byte

const (
	msgOpen      msgType = 1
	msgData      msgType = 2
	msgClose     msgType = 3
	msgPing      msgType = 4
	msgPong      msgType = 5
	msgHeartbeat msgType = 6
	msgCertRenew msgType = 7
)

type HeartbeatMetadata struct {
	Healthy       bool `json:"healthy"`
	LatencyMs     *int `json:"latencyMs,omitempty"`
	ActiveStreams *int `json:"activeStreams,omitempty"`
}

type BrokerConfig struct {
	Store               Store
	Logger              *slog.Logger
	ServerEncryptionKey []byte
	SpiffeTrustDomain   string
	ProxyBindHost       string
	ProxyAdvertiseHost  string

	// QUICTLSConfig, when non-nil, enables the QUIC transport listener. It must
	// carry the broker's server certificate; the listener forces TLS 1.3, the
	// tunnel ALPN, and client-certificate requests (verified per-gateway against
	// the tenant CA after the handshake). When nil the broker is WebSocket-only.
	QUICTLSConfig  *tls.Config
	QUICListenAddr string
}

type Broker struct {
	config   BrokerConfig
	upgrader websocket.Upgrader

	mu       sync.RWMutex
	registry map[string]tunnelConn
}

// tunnelConn is a transport-agnostic per-gateway tunnel connection. Both the
// WebSocket frame multiplexer (*tunnelConnection) and the QUIC transport
// (*quicConnection) satisfy it, so createTCPProxy and the status/list/disconnect
// handlers stay transport-blind and the registry holds either kind.
type tunnelConn interface {
	// openStream opens a logical byte stream to host:port over the tunnel.
	openStream(ctx context.Context, host string, port int, timeout time.Duration) (io.ReadWriteCloser, error)
	// describe returns a status snapshot for the API and gateway monitor.
	describe() contracts.TunnelStatus
	// closeTransport tears the connection down — closing the underlying
	// transport and every logical stream — with a human-readable reason. It is
	// idempotent and must NOT be called while holding Broker.mu.
	closeTransport(reason string)
}

type tunnelConnection struct {
	broker        *Broker
	gatewayID     string
	ws            *websocket.Conn
	connectedAt   time.Time
	clientVersion string
	clientIP      string

	// statusMu guards the mutable status fields below, which are written by the
	// read loop and read by describe() from other goroutines (API/monitor).
	statusMu       sync.Mutex
	lastHeartbeat  time.Time
	lastPingSentAt time.Time
	pingLatency    *int64
	heartbeat      *HeartbeatMetadata

	// bytesTransferred and activeStreams are read by describe() from other
	// goroutines while mutated on the data path, so they are atomic.
	bytesTransferred atomic.Int64
	activeStreams    atomic.Int64

	sendMu       sync.Mutex
	streams      map[uint16]*streamConn
	pendingOpens map[uint16]*pendingOpen
	nextStreamID uint16

	closeOnce sync.Once
}

type pendingOpen struct {
	resolve chan *streamConn
	timer   *time.Timer
}

type streamConn struct {
	parent *tunnelConnection
	id     uint16

	reader *io.PipeReader
	writer *io.PipeWriter

	closeOnce sync.Once
	closed    chan struct{}
}

func NewBroker(config BrokerConfig) *Broker {
	if config.Store == nil {
		config.Store = NoopStore{}
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if strings.TrimSpace(config.SpiffeTrustDomain) == "" {
		config.SpiffeTrustDomain = defaultTrustDomain
	}
	if strings.TrimSpace(config.ProxyBindHost) == "" {
		config.ProxyBindHost = "0.0.0.0"
	}
	if strings.TrimSpace(config.ProxyAdvertiseHost) == "" {
		config.ProxyAdvertiseHost = strings.TrimSpace(os.Getenv("HOSTNAME"))
		if config.ProxyAdvertiseHost == "" {
			config.ProxyAdvertiseHost = "tunnel-broker"
		}
	}

	return &Broker{
		config: config,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
		registry: make(map[string]tunnelConn),
	}
}
