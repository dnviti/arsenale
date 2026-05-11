package sshproxyapi

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/dnviti/arsenale/backend/internal/connectionaccess"
	"github.com/dnviti/arsenale/backend/internal/sessions"
	"github.com/dnviti/arsenale/backend/internal/tenantauth"
	"golang.org/x/crypto/ssh"
)

func (s Service) Start(ctx context.Context) (func(), error) {
	if !strings.EqualFold(getenv("SSH_PROXY_ENABLED", "false"), "true") {
		return nil, nil
	}
	if s.DB == nil || s.ConnectionResolver == nil || s.SessionStore == nil {
		return nil, errors.New("SSH proxy is enabled but dependencies are unavailable")
	}

	port := parsePositiveInt(getenv("SSH_PROXY_PORT", "2222"), 2222)
	listenAddr := getenv("SSH_PROXY_LISTEN_ADDR", net.JoinHostPort("", strconv.Itoa(port)))
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("start SSH proxy listener on %s: %w", listenAddr, err)
	}

	config, err := newServerConfig()
	if err != nil {
		_ = listener.Close()
		return nil, err
	}

	var wg sync.WaitGroup
	closed := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = listener.Close()
		case <-closed:
		}
	}()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				s.handleProxyConnection(ctx, conn, config)
			}()
		}
	}()

	var stopOnce sync.Once
	return func() {
		stopOnce.Do(func() {
			close(closed)
			_ = listener.Close()
			wg.Wait()
		})
	}, nil
}

func newServerConfig() (*ssh.ServerConfig, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate SSH proxy host key: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("create SSH proxy host signer: %w", err)
	}
	config := &ssh.ServerConfig{NoClientAuth: true}
	config.AddHostKey(signer)
	return config, nil
}

func (s Service) handleProxyConnection(parentCtx context.Context, conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	ipAddress := connRemoteIP(conn)

	serverConn, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer serverConn.Close()

	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	grant, err := s.redeemProxyGrant(ctx, serverConn.User(), ipAddress)
	if err != nil {
		return
	}

	target, err := s.resolveProxyTarget(ctx, grant, ipAddress)
	if err != nil {
		_ = s.insertAuditLog(ctx, grant.UserID, "SSH_PROXY_AUTH_FAILURE", grant.ConnectionID, map[string]any{
			"reason": err.Error(),
		}, stringToPtr(ipAddress))
		return
	}

	targetClient, err := connectSSHProxyTarget(ctx, target.Target, target.Bastion)
	if err != nil {
		_ = s.insertAuditLog(ctx, grant.UserID, "SSH_PROXY_AUTH_FAILURE", grant.ConnectionID, map[string]any{
			"reason": targetConnectFailureReason(err),
		}, stringToPtr(ipAddress))
		return
	}
	defer targetClient.Close()

	sessionID, err := s.startProxySession(ctx, grant, target, ipAddress)
	if err != nil {
		return
	}
	_ = s.attachProxyGrantSession(ctx, grant.ID, sessionID)
	defer func() {
		_ = s.SessionStore.EndOwnedSession(context.Background(), sessionID, grant.UserID, "ssh_proxy_disconnect")
	}()

	go ssh.DiscardRequests(requests)

	var wg sync.WaitGroup
	for newChannel := range channels {
		wg.Add(1)
		go func(ch ssh.NewChannel) {
			defer wg.Done()
			handleProxyChannel(ch, targetClient.client)
		}(newChannel)
	}
	wg.Wait()
}

func (s Service) resolveProxyTarget(ctx context.Context, grant proxyGrantRecord, ipAddress string) (connectionaccess.ResolvedFileTransferTarget, error) {
	if strings.TrimSpace(grant.TenantID) != "" {
		membership, err := s.TenantAuth.ResolveMembership(ctx, grant.UserID, grant.TenantID)
		if err != nil {
			return connectionaccess.ResolvedFileTransferTarget{}, fmt.Errorf("resolve tenant membership: %w", err)
		}
		if membership == nil || !membership.Permissions[tenantauth.CanConnect] {
			return connectionaccess.ResolvedFileTransferTarget{}, errors.New("not allowed to start sessions in this tenant")
		}
	}

	allowed, err := s.checkLateralMovement(ctx, grant.UserID, grant.ConnectionID, ipAddress)
	if err != nil {
		return connectionaccess.ResolvedFileTransferTarget{}, err
	}
	if !allowed {
		return connectionaccess.ResolvedFileTransferTarget{}, errors.New("anomalous lateral movement detected")
	}

	target, err := s.ConnectionResolver.ResolveFileTransferTarget(ctx, grant.UserID, grant.TenantID, grant.ConnectionID, connectionaccess.ResolveConnectionOptions{
		ExpectedType: "SSH",
	})
	if err != nil {
		return connectionaccess.ResolvedFileTransferTarget{}, err
	}
	if err := validateResolvedProxyTarget(target); err != nil {
		return connectionaccess.ResolvedFileTransferTarget{}, err
	}
	return target, nil
}

func (s Service) startProxySession(ctx context.Context, grant proxyGrantRecord, target connectionaccess.ResolvedFileTransferTarget, ipAddress string) (string, error) {
	gatewayID := ""
	if target.Connection.GatewayID != nil {
		gatewayID = strings.TrimSpace(*target.Connection.GatewayID)
	}
	sessionID, err := s.SessionStore.StartSession(ctx, sessions.StartSessionParams{
		TenantID:     grant.TenantID,
		UserID:       grant.UserID,
		ConnectionID: target.Connection.ID,
		GatewayID:    gatewayID,
		Protocol:     "SSH_PROXY",
		IPAddress:    ipAddress,
		Metadata: map[string]any{
			"transport":   "ssh-proxy",
			"host":        target.Connection.Host,
			"port":        target.Connection.Port,
			"accessType":  target.AccessType,
			"grantId":     grant.ID,
			"hasBastion":  target.Bastion != nil,
			"viewerRoute": "native-openssh",
		},
	})
	if err != nil {
		return "", fmt.Errorf("start SSH proxy session: %w", err)
	}
	return sessionID, nil
}

func connRemoteIP(conn net.Conn) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(conn.RemoteAddr().String()))
	if err == nil {
		return host
	}
	return strings.TrimSpace(conn.RemoteAddr().String())
}

func targetConnectFailureReason(err error) string {
	reason := strings.TrimSpace(err.Error())
	if reason == "" {
		return "target_connect_failed"
	}
	return "target_connect_failed: " + reason
}
