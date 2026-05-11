package sshproxyapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/dnviti/arsenale/backend/internal/connectionaccess"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

type fakeProxyTargetResolver struct {
	target connectionaccess.ResolvedFileTransferTarget
	err    error
}

func (f fakeProxyTargetResolver) ResolveConnection(context.Context, string, string, string, connectionaccess.ResolveConnectionOptions) (connectionaccess.ResolvedConnection, error) {
	return connectionaccess.ResolvedConnection{}, errors.New("unused")
}

func (f fakeProxyTargetResolver) CreateTunnelProxy(context.Context, string, string, int) (contracts.TunnelProxyResponse, error) {
	return contracts.TunnelProxyResponse{}, errors.New("unused")
}

func (f fakeProxyTargetResolver) ResolveFileTransferTarget(context.Context, string, string, string, connectionaccess.ResolveConnectionOptions) (connectionaccess.ResolvedFileTransferTarget, error) {
	return f.target, f.err
}

func TestBuildSSHProxyCommandUsesDirectProxyEndpoint(t *testing.T) {
	got := buildSSHProxyCommand("arsenale.example.test", 2222)

	for _, want := range []string{
		"ssh -p 2222",
		"-o PreferredAuthentications=none",
		"'<token>@arsenale.example.test'",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("command missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, "ProxyCommand") || strings.Contains(got, " nc ") {
		t.Fatalf("command must not depend on netcat proxy command: %s", got)
	}
}

func TestPreflightProxyTargetSurfacesResolveError(t *testing.T) {
	resolveErr := &connectionaccess.ResolveError{
		Status:  http.StatusForbidden,
		Message: "Tunnel egress denied: target 192.0.2.10:22 is not allowed by gateway egress policy",
	}
	service := Service{ConnectionResolver: fakeProxyTargetResolver{err: resolveErr}}

	err := service.preflightProxyTarget(context.Background(), "user-1", "tenant-1", "connection-1")
	if !errors.Is(err, resolveErr) {
		t.Fatalf("error = %v; want resolve error", err)
	}
	if got := proxyResolveHTTPStatus(err); got != http.StatusForbidden {
		t.Fatalf("status = %d; want %d", got, http.StatusForbidden)
	}
}

func TestPreflightProxyTargetRejectsIncompleteTarget(t *testing.T) {
	service := Service{ConnectionResolver: fakeProxyTargetResolver{
		target: connectionaccess.ResolvedFileTransferTarget{
			Target: contracts.TerminalEndpoint{
				Host: "target.example.test",
				Port: 22,
			},
		},
	}}

	err := service.preflightProxyTarget(context.Background(), "user-1", "tenant-1", "connection-1")
	if err == nil || err.Error() != "SSH target is incomplete" {
		t.Fatalf("error = %v; want incomplete target", err)
	}
	if got := proxyResolveHTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("status = %d; want %d", got, http.StatusBadRequest)
	}
}

func TestTargetConnectFailureReasonIncludesRootCause(t *testing.T) {
	err := errors.New("handshake target: ssh: unable to authenticate")

	got := targetConnectFailureReason(err)
	if got != "target_connect_failed: handshake target: ssh: unable to authenticate" {
		t.Fatalf("reason = %q", got)
	}
}
