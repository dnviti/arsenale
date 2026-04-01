package sshsessions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/sessions"
	"github.com/dnviti/arsenale/backend/internal/tenantauth"
	"github.com/google/uuid"
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

func (t *activeDBTunnel) snapshot() dbTunnelListItem {
	t.mu.Lock()
	defer t.mu.Unlock()

	return dbTunnelListItem{
		TunnelID:         t.ID,
		SessionID:        t.SessionID,
		LocalHost:        "127.0.0.1",
		LocalPort:        t.LocalPort,
		TargetDBHost:     t.TargetDBHost,
		TargetDBPort:     t.TargetDBPort,
		DBType:           cloneStringPtr(t.DBType),
		ConnectionString: cloneStringPtr(t.ConnectionString),
		ConnectionID:     t.ConnectionID,
		Healthy:          t.healthy,
		CreatedAt:        t.CreatedAt,
		LastError:        cloneStringPtr(t.lastError),
		LastUsedAt:       cloneTimePtr(t.LastUsedAt),
	}
}

func (t *activeDBTunnel) setForwardError(err error) {
	if err == nil {
		return
	}
	message := err.Error()
	now := time.Now().UTC()

	t.mu.Lock()
	defer t.mu.Unlock()
	t.healthy = false
	t.lastError = &message
	t.LastUsedAt = &now
}

func (t *activeDBTunnel) touch() {
	now := time.Now().UTC()
	t.mu.Lock()
	defer t.mu.Unlock()
	t.LastUsedAt = &now
}

func (t *activeDBTunnel) close() {
	t.closeOnce.Do(func() {
		_ = t.listener.Close()
		_ = t.sshClient.Close()
	})
}

type dbTunnelRegistry struct {
	mu      sync.RWMutex
	tunnels map[string]*activeDBTunnel
}

var activeDBTunnels = &dbTunnelRegistry{tunnels: make(map[string]*activeDBTunnel)}

func (r *dbTunnelRegistry) add(tunnel *activeDBTunnel) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tunnels[tunnel.ID] = tunnel
}

func (r *dbTunnelRegistry) get(id string) (*activeDBTunnel, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tunnel, ok := r.tunnels[id]
	return tunnel, ok
}

func (r *dbTunnelRegistry) listForUser(userID string) []dbTunnelListItem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]dbTunnelListItem, 0, len(r.tunnels))
	for _, tunnel := range r.tunnels {
		if tunnel.UserID == userID {
			items = append(items, tunnel.snapshot())
		}
	}
	slices.SortFunc(items, func(a, b dbTunnelListItem) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return items
}

func (r *dbTunnelRegistry) closeOwned(tunnelID, userID string) (*activeDBTunnel, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	tunnel, ok := r.tunnels[tunnelID]
	if !ok || tunnel.UserID != userID {
		return nil, false
	}
	delete(r.tunnels, tunnelID)
	return tunnel, true
}

func (r *dbTunnelRegistry) closeByID(tunnelID string) {
	r.mu.Lock()
	tunnel, ok := r.tunnels[tunnelID]
	if ok {
		delete(r.tunnels, tunnelID)
	}
	r.mu.Unlock()
	if ok {
		tunnel.close()
	}
}

func (s Service) HandleCreateDBTunnel(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	var payload dbTunnelRequest
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.openDBTunnel(r.Context(), claims, payload, requestIP(r))
	if err != nil {
		_ = s.insertAuditLog(r.Context(), claims.UserID, "DB_TUNNEL_ERROR", "Connection", strings.TrimSpace(payload.ConnectionID), requestIP(r), map[string]any{
			"protocol": "DB_TUNNEL",
			"error":    err.Error(),
		})
		var reqErr *requestError
		if errors.As(err, &reqErr) {
			app.ErrorJSON(w, reqErr.status, reqErr.message)
			return
		}
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleListDBTunnels(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	app.WriteJSON(w, http.StatusOK, activeDBTunnels.listForUser(claims.UserID))
}

func (s Service) HandleDBTunnelHeartbeat(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	tunnelID := strings.TrimSpace(r.PathValue("tunnelId"))
	tunnel, ok := activeDBTunnels.get(tunnelID)
	if !ok || tunnel.UserID != claims.UserID {
		app.ErrorJSON(w, http.StatusNotFound, "Tunnel not found")
		return
	}

	tunnel.touch()
	if tunnel.SessionID != "" && s.SessionStore != nil {
		if err := s.SessionStore.HeartbeatOwnedSession(r.Context(), tunnel.SessionID, claims.UserID); err != nil &&
			!errors.Is(err, sessions.ErrSessionClosed) &&
			!errors.Is(err, sessions.ErrSessionNotFound) {
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
			return
		}
	}

	app.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"healthy": tunnel.snapshot().Healthy,
	})
}

func (s Service) HandleCloseDBTunnel(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	tunnelID := strings.TrimSpace(r.PathValue("tunnelId"))
	tunnel, ok := activeDBTunnels.closeOwned(tunnelID, claims.UserID)
	if !ok {
		app.ErrorJSON(w, http.StatusNotFound, "Tunnel not found")
		return
	}

	tunnel.close()
	if tunnel.SessionID != "" && s.SessionStore != nil {
		if err := s.SessionStore.EndOwnedSession(r.Context(), tunnel.SessionID, claims.UserID, "client_disconnect"); err != nil &&
			!errors.Is(err, sessions.ErrSessionNotFound) {
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
			return
		}
	}
	snapshot := tunnel.snapshot()
	_ = s.insertAuditLog(r.Context(), claims.UserID, "DB_TUNNEL_CLOSE", "Connection", tunnel.ConnectionID, requestIP(r), map[string]any{
		"tunnelId":     tunnel.ID,
		"durationMs":   time.Since(tunnel.CreatedAt).Milliseconds(),
		"localPort":    tunnel.LocalPort,
		"targetDbHost": tunnel.TargetDBHost,
		"targetDbPort": tunnel.TargetDBPort,
		"connectionId": tunnel.ConnectionID,
		"sessionId":    tunnel.SessionID,
		"healthy":      snapshot.Healthy,
		"lastError":    valueOrEmpty(snapshot.LastError),
	})

	app.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s Service) openDBTunnel(ctx context.Context, claims authn.Claims, payload dbTunnelRequest, ipAddress string) (dbTunnelResponse, error) {
	if s.DB == nil || s.SessionStore == nil {
		return dbTunnelResponse{}, fmt.Errorf("database session dependencies are unavailable")
	}

	connectionID := strings.TrimSpace(payload.ConnectionID)
	if connectionID == "" {
		return dbTunnelResponse{}, &requestError{status: http.StatusBadRequest, message: "connectionId is required"}
	}

	if claims.TenantID != "" {
		membership, err := s.TenantAuth.ResolveMembership(ctx, claims.UserID, claims.TenantID)
		if err != nil {
			return dbTunnelResponse{}, fmt.Errorf("resolve tenant membership: %w", err)
		}
		if membership == nil || !membership.Permissions[tenantauth.CanConnect] {
			return dbTunnelResponse{}, &requestError{status: http.StatusForbidden, message: "Not allowed to start sessions in this tenant"}
		}
	}

	access, err := s.loadAccess(ctx, claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		return dbTunnelResponse{}, err
	}
	if !strings.EqualFold(access.Connection.Type, "DB_TUNNEL") {
		return dbTunnelResponse{}, &requestError{status: http.StatusBadRequest, message: "Not a DB_TUNNEL connection"}
	}
	if access.Connection.TargetDBHost == nil || strings.TrimSpace(*access.Connection.TargetDBHost) == "" || access.Connection.TargetDBPort == nil || *access.Connection.TargetDBPort <= 0 {
		return dbTunnelResponse{}, &requestError{status: http.StatusBadRequest, message: "Target database host and port are required"}
	}

	credentials, err := s.resolveCredentials(ctx, claims.UserID, claims.TenantID, createRequest{}, access)
	if err != nil {
		return dbTunnelResponse{}, err
	}
	if strings.TrimSpace(credentials.Username) == "" {
		return dbTunnelResponse{}, &requestError{status: http.StatusBadRequest, message: "Connection has no credentials configured"}
	}

	dbType := firstNonEmptyString(access.Connection.DBType, stringPtr(payload.DBType))
	dbUsername := strings.TrimSpace(payload.DBUsername)
	dbPassword := strings.TrimSpace(payload.DBPassword)
	dbName := strings.TrimSpace(payload.DBName)
	targetDBHost := strings.TrimSpace(*access.Connection.TargetDBHost)
	targetDBPort := *access.Connection.TargetDBPort

	tunnel, err := startDBTunnel(credentials, dbTunnelStartOptions{
		UserID:       claims.UserID,
		ConnectionID: access.Connection.ID,
		BastionHost:  strings.TrimSpace(access.Connection.Host),
		BastionPort:  access.Connection.Port,
		TargetDBHost: targetDBHost,
		TargetDBPort: targetDBPort,
		DBType:       dbType,
	})
	if err != nil {
		return dbTunnelResponse{}, &requestError{status: http.StatusBadGateway, message: fmt.Sprintf("SSH tunnel failed: %v", err)}
	}
	tunnel.ConnectionString = buildDBTunnelConnectionString(dbType, "127.0.0.1", tunnel.LocalPort, dbUsername, dbPassword, dbName)

	if _, err := s.SessionStore.CloseStaleSessionsForConnection(ctx, claims.UserID, access.Connection.ID, "DB_TUNNEL"); err != nil {
		tunnel.close()
		return dbTunnelResponse{}, fmt.Errorf("close stale DB tunnel sessions: %w", err)
	}

	sessionID, err := s.SessionStore.StartSession(ctx, sessions.StartSessionParams{
		UserID:       claims.UserID,
		ConnectionID: access.Connection.ID,
		Protocol:     "DB_TUNNEL",
		IPAddress:    ipAddress,
		Metadata: map[string]any{
			"tunnelId":          tunnel.ID,
			"localHost":         "127.0.0.1",
			"localPort":         tunnel.LocalPort,
			"targetDbHost":      targetDBHost,
			"targetDbPort":      targetDBPort,
			"dbType":            valueOrEmpty(dbType),
			"transport":         "db-tunnel",
			"credentialSource":  credentials.CredentialSource,
			"connectionString":  valueOrEmpty(tunnel.ConnectionString),
			"bastionHost":       strings.TrimSpace(access.Connection.Host),
			"bastionPort":       access.Connection.Port,
			"bastionCredential": credentials.CredentialSource,
		},
	})
	if err != nil {
		tunnel.close()
		return dbTunnelResponse{}, fmt.Errorf("start DB tunnel session: %w", err)
	}

	tunnel.SessionID = sessionID
	activeDBTunnels.add(tunnel)
	_ = s.insertAuditLog(ctx, claims.UserID, "DB_TUNNEL_OPEN", "Connection", access.Connection.ID, ipAddress, map[string]any{
		"tunnelId":     tunnel.ID,
		"sessionId":    sessionID,
		"localPort":    tunnel.LocalPort,
		"targetDbHost": targetDBHost,
		"targetDbPort": targetDBPort,
		"bastionHost":  strings.TrimSpace(access.Connection.Host),
		"dbType":       valueOrEmpty(dbType),
		"connectionId": access.Connection.ID,
	})
	go tunnel.serve()

	return dbTunnelResponse{
		TunnelID:         tunnel.ID,
		SessionID:        sessionID,
		LocalHost:        "127.0.0.1",
		LocalPort:        tunnel.LocalPort,
		ConnectionString: cloneStringPtr(tunnel.ConnectionString),
		TargetDBHost:     targetDBHost,
		TargetDBPort:     targetDBPort,
		DBType:           cloneStringPtr(dbType),
	}, nil
}

type dbTunnelStartOptions struct {
	UserID       string
	ConnectionID string
	BastionHost  string
	BastionPort  int
	TargetDBHost string
	TargetDBPort int
	DBType       *string
}

func startDBTunnel(credentials resolvedCredentials, options dbTunnelStartOptions) (*activeDBTunnel, error) {
	config, err := dbTunnelSSHClientConfig(credentials)
	if err != nil {
		return nil, err
	}

	bastionAddr := net.JoinHostPort(options.BastionHost, strconv.Itoa(options.BastionPort))
	client, err := ssh.Dial("tcp", bastionAddr, config)
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = client.Close()
		return nil, err
	}

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		_ = client.Close()
		return nil, errors.New("failed to allocate local tunnel port")
	}

	return &activeDBTunnel{
		ID:           "dbt-" + uuid.NewString(),
		UserID:       options.UserID,
		ConnectionID: options.ConnectionID,
		LocalPort:    addr.Port,
		TargetDBHost: options.TargetDBHost,
		TargetDBPort: options.TargetDBPort,
		DBType:       cloneStringPtr(options.DBType),
		CreatedAt:    time.Now().UTC(),
		listener:     listener,
		sshClient:    client,
		healthy:      true,
	}, nil
}

func (t *activeDBTunnel) serve() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			t.setForwardError(err)
			return
		}
		t.touch()
		go t.forward(conn)
	}
}

func (t *activeDBTunnel) forward(localConn net.Conn) {
	targetAddr := net.JoinHostPort(t.TargetDBHost, strconv.Itoa(t.TargetDBPort))
	remoteConn, err := t.sshClient.Dial("tcp", targetAddr)
	if err != nil {
		t.setForwardError(err)
		_ = localConn.Close()
		return
	}

	go func() {
		_, _ = io.Copy(remoteConn, localConn)
		_ = remoteConn.Close()
	}()
	go func() {
		_, _ = io.Copy(localConn, remoteConn)
		_ = localConn.Close()
	}()
}

func dbTunnelSSHClientConfig(credentials resolvedCredentials) (*ssh.ClientConfig, error) {
	authMethods := make([]ssh.AuthMethod, 0, 2)
	if strings.TrimSpace(credentials.Password) != "" {
		authMethods = append(authMethods, ssh.Password(credentials.Password))
	}
	if strings.TrimSpace(credentials.PrivateKey) != "" {
		var (
			signer ssh.Signer
			err    error
		)
		if strings.TrimSpace(credentials.Passphrase) != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(credentials.PrivateKey), []byte(credentials.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(credentials.PrivateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("parse private key for %s: %w", credentials.Username, err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if len(authMethods) == 0 {
		return nil, errors.New("ssh credentials are required")
	}

	return &ssh.ClientConfig{
		User:            credentials.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}, nil
}

func buildDBTunnelConnectionString(dbType *string, host string, port int, username, password, dbName string) *string {
	host = strings.TrimSpace(host)
	if host == "" || port <= 0 {
		return nil
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))

	var value string
	userPass := ""
	if strings.TrimSpace(username) != "" && strings.TrimSpace(password) != "" {
		userPass = urlEncode(username) + ":" + urlEncode(password) + "@"
	} else if strings.TrimSpace(username) != "" {
		userPass = urlEncode(username) + "@"
	}
	db := ""
	if strings.TrimSpace(dbName) != "" {
		db = "/" + urlEncode(dbName)
	}

	switch normalized := strings.ToLower(strings.TrimSpace(valueOrEmpty(dbType))); normalized {
	case "postgresql", "postgres":
		value = "postgresql://" + userPass + address + db
	case "mysql", "mariadb":
		value = "mysql://" + userPass + address + db
	case "mongodb", "mongo":
		value = "mongodb://" + userPass + address + db
	case "redis":
		if strings.TrimSpace(password) != "" {
			value = "redis://:" + urlEncode(password) + "@" + address
		} else {
			value = "redis://" + address
		}
	case "mssql", "sqlserver":
		var builder strings.Builder
		builder.WriteString("Server=")
		builder.WriteString(host)
		builder.WriteString(",")
		builder.WriteString(strconv.Itoa(port))
		builder.WriteString(";")
		if strings.TrimSpace(dbName) != "" {
			builder.WriteString("Database=")
			builder.WriteString(dbName)
			builder.WriteString(";")
		}
		if strings.TrimSpace(username) != "" {
			builder.WriteString("User Id=")
			builder.WriteString(username)
			builder.WriteString(";")
		}
		if strings.TrimSpace(password) != "" {
			builder.WriteString("Password=")
			builder.WriteString(password)
			builder.WriteString(";")
		}
		value = builder.String()
	case "oracle":
		base := address
		if strings.TrimSpace(dbName) != "" {
			value = base + "/" + strings.TrimSpace(dbName)
		} else {
			value = base + "/ORCL"
		}
	default:
		value = address
	}

	return &value
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmptyString(values ...*string) *string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			trimmed := strings.TrimSpace(*value)
			return &trimmed
		}
	}
	return nil
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func stringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func urlEncode(value string) string {
	replacer := strings.NewReplacer(
		"%", "%25",
		" ", "%20",
		"!", "%21",
		"#", "%23",
		"$", "%24",
		"&", "%26",
		"'", "%27",
		"(", "%28",
		")", "%29",
		"*", "%2A",
		"+", "%2B",
		",", "%2C",
		"/", "%2F",
		":", "%3A",
		";", "%3B",
		"=", "%3D",
		"?", "%3F",
		"@", "%40",
		"[", "%5B",
		"]", "%5D",
	)
	return replacer.Replace(value)
}
