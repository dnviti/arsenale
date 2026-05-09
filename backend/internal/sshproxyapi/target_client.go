package sshproxyapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"golang.org/x/crypto/ssh"
)

const sshProxyTargetTimeout = 30 * time.Second

type proxyTargetClient struct {
	client  *ssh.Client
	bastion *ssh.Client
}

func (c *proxyTargetClient) Close() {
	if c == nil {
		return
	}
	if c.client != nil {
		_ = c.client.Close()
	}
	if c.bastion != nil {
		_ = c.bastion.Close()
	}
}

func connectSSHProxyTarget(ctx context.Context, target contracts.TerminalEndpoint, bastion *contracts.TerminalEndpoint) (*proxyTargetClient, error) {
	if bastion == nil {
		client, err := dialSSHEndpoint(ctx, target)
		if err != nil {
			return nil, err
		}
		return &proxyTargetClient{client: client}, nil
	}

	bastionClient, err := dialSSHEndpoint(ctx, *bastion)
	if err != nil {
		return nil, fmt.Errorf("connect bastion: %w", err)
	}

	targetAddr := endpointAddr(target)
	targetConn, err := bastionClient.Dial("tcp", targetAddr)
	if err != nil {
		_ = bastionClient.Close()
		return nil, fmt.Errorf("connect target through bastion: %w", err)
	}

	config, err := sshClientConfig(target)
	if err != nil {
		_ = targetConn.Close()
		_ = bastionClient.Close()
		return nil, err
	}
	conn, chans, reqs, err := ssh.NewClientConn(targetConn, targetAddr, config)
	if err != nil {
		_ = targetConn.Close()
		_ = bastionClient.Close()
		return nil, fmt.Errorf("handshake target through bastion: %w", err)
	}
	return &proxyTargetClient{
		client:  ssh.NewClient(conn, chans, reqs),
		bastion: bastionClient,
	}, nil
}

func dialSSHEndpoint(ctx context.Context, endpoint contracts.TerminalEndpoint) (*ssh.Client, error) {
	config, err := sshClientConfig(endpoint)
	if err != nil {
		return nil, err
	}
	var dialer net.Dialer
	dialer.Timeout = sshProxyTargetTimeout
	conn, err := dialer.DialContext(ctx, "tcp", endpointAddr(endpoint))
	if err != nil {
		return nil, fmt.Errorf("dial target: %w", err)
	}
	clientConn, chans, reqs, err := ssh.NewClientConn(conn, endpointAddr(endpoint), config)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("handshake target: %w", err)
	}
	return ssh.NewClient(clientConn, chans, reqs), nil
}

func sshClientConfig(endpoint contracts.TerminalEndpoint) (*ssh.ClientConfig, error) {
	username := strings.TrimSpace(endpoint.Username)
	if username == "" {
		return nil, errors.New("SSH username is required")
	}
	authMethods, err := sshAuthMethods(endpoint)
	if err != nil {
		return nil, err
	}
	if len(authMethods) == 0 {
		return nil, errors.New("SSH credentials are required")
	}
	return &ssh.ClientConfig{
		User:            username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         sshProxyTargetTimeout,
	}, nil
}

func sshAuthMethods(endpoint contracts.TerminalEndpoint) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if strings.TrimSpace(endpoint.PrivateKey) != "" {
		signer, err := signerFromPrivateKey(endpoint.PrivateKey, endpoint.Passphrase)
		if err != nil {
			return nil, err
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if endpoint.Password != "" {
		password := endpoint.Password
		methods = append(methods, ssh.Password(password))
		methods = append(methods, ssh.KeyboardInteractive(func(_ string, _ string, questions []string, _ []bool) ([]string, error) {
			answers := make([]string, len(questions))
			for i := range answers {
				answers[i] = password
			}
			return answers, nil
		}))
	}
	return methods, nil
}

func signerFromPrivateKey(privateKey, passphrase string) (ssh.Signer, error) {
	key := []byte(privateKey)
	if strings.TrimSpace(passphrase) != "" {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("parse SSH private key: %w", err)
		}
		return signer, nil
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("parse SSH private key: %w", err)
	}
	return signer, nil
}

func endpointAddr(endpoint contracts.TerminalEndpoint) string {
	return net.JoinHostPort(strings.TrimSpace(endpoint.Host), strconv.Itoa(endpoint.Port))
}
