package sshsessions

import (
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type dbTunnelRequest struct {
	ConnectionID string `json:"connectionId"`
	DBUsername   string `json:"dbUsername,omitempty"`
	DBPassword   string `json:"dbPassword,omitempty"`
	DBName       string `json:"dbName,omitempty"`
	DBType       string `json:"dbType,omitempty"`
}

type dbTunnelResponse struct {
	TunnelID         string  `json:"tunnelId"`
	SessionID        string  `json:"sessionId,omitempty"`
	LocalHost        string  `json:"localHost"`
	LocalPort        int     `json:"localPort"`
	ConnectionString *string `json:"connectionString"`
	TargetDBHost     string  `json:"targetDbHost"`
	TargetDBPort     int     `json:"targetDbPort"`
	DBType           *string `json:"dbType"`
}

type dbTunnelListItem struct {
	TunnelID         string     `json:"tunnelId"`
	SessionID        string     `json:"sessionId,omitempty"`
	LocalHost        string     `json:"localHost"`
	LocalPort        int        `json:"localPort"`
	TargetDBHost     string     `json:"targetDbHost"`
	TargetDBPort     int        `json:"targetDbPort"`
	DBType           *string    `json:"dbType"`
	ConnectionString *string    `json:"connectionString"`
	ConnectionID     string     `json:"connectionId"`
	Healthy          bool       `json:"healthy"`
	CreatedAt        time.Time  `json:"createdAt"`
	LastError        *string    `json:"lastError,omitempty"`
	LastUsedAt       *time.Time `json:"lastUsedAt,omitempty"`
}

type activeDBTunnel struct {
	ID               string
	SessionID        string
	UserID           string
	ConnectionID     string
	LocalPort        int
	TargetDBHost     string
	TargetDBPort     int
	DBType           *string
	ConnectionString *string
	CreatedAt        time.Time
	LastUsedAt       *time.Time

	listener  net.Listener
	sshClient *ssh.Client

	mu        sync.Mutex
	healthy   bool
	lastError *string
	closeOnce sync.Once
}

type dbTunnelRegistry struct {
	mu      sync.RWMutex
	tunnels map[string]*activeDBTunnel
}

var activeDBTunnels = &dbTunnelRegistry{tunnels: make(map[string]*activeDBTunnel)}

type dbTunnelStartOptions struct {
	UserID       string
	ConnectionID string
	BastionHost  string
	BastionPort  int
	TargetDBHost string
	TargetDBPort int
	DBType       *string
}
