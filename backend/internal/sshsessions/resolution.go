package sshsessions

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

type ResolveError struct {
	Status  int
	Message string
}

func (e *ResolveError) Error() string {
	return e.Message
}

type ResolveConnectionOptions struct {
	ExpectedType     string
	OverrideUsername string
	OverridePassword string
	OverrideDomain   string
	CredentialMode   string
}

type ConnectionSnapshot struct {
	ID           string
	Type         string
	Host         string
	Port         int
	TeamID       *string
	GatewayID    *string
	TargetDBHost *string
	TargetDBPort *int
	DBType       *string
	DBSettings   json.RawMessage
	DLPPolicy    json.RawMessage
}

type ResolvedConnection struct {
	Connection  ConnectionSnapshot
	AccessType  string
	Credentials ResolvedCredentials
}

type ResolvedCredentials struct {
	Username         string
	Password         string
	Domain           string
	PrivateKey       string
	Passphrase       string
	CredentialSource string
}

func (s Service) ResolveConnection(ctx context.Context, userID, tenantID, connectionID string, opts ResolveConnectionOptions) (ResolvedConnection, error) {
	access, err := s.loadAccess(ctx, userID, tenantID, strings.TrimSpace(connectionID))
	if err != nil {
		return ResolvedConnection{}, mapResolveError(err)
	}

	expectedType := strings.ToUpper(strings.TrimSpace(opts.ExpectedType))
	if expectedType != "" && !strings.EqualFold(access.Connection.Type, expectedType) {
		return ResolvedConnection{}, &ResolveError{
			Status:  http.StatusBadRequest,
			Message: "Not a " + expectedType + " connection",
		}
	}

	payload := createRequest{
		CredentialMode: normalizeCredentialMode(opts.CredentialMode),
	}
	overrideUsername := strings.TrimSpace(opts.OverrideUsername)
	overridePassword := strings.TrimSpace(opts.OverridePassword)
	if overrideUsername != "" && overridePassword != "" {
		payload.Username = overrideUsername
		payload.Password = overridePassword
		payload.Domain = strings.TrimSpace(opts.OverrideDomain)
	}

	credentials, err := s.resolveCredentials(ctx, userID, tenantID, payload, access)
	if err != nil {
		return ResolvedConnection{}, mapResolveError(err)
	}

	return ResolvedConnection{
		Connection: ConnectionSnapshot{
			ID:           access.Connection.ID,
			Type:         access.Connection.Type,
			Host:         access.Connection.Host,
			Port:         access.Connection.Port,
			TeamID:       cloneStringPtr(access.Connection.TeamID),
			GatewayID:    cloneStringPtr(access.Connection.GatewayID),
			TargetDBHost: cloneStringPtr(access.Connection.TargetDBHost),
			TargetDBPort: cloneIntPtr(access.Connection.TargetDBPort),
			DBType:       cloneStringPtr(access.Connection.DBType),
			DBSettings:   cloneRawJSON(access.Connection.DBSettings),
			DLPPolicy:    cloneRawJSON(access.Connection.DLPPolicy),
		},
		AccessType: access.AccessType,
		Credentials: ResolvedCredentials{
			Username:         credentials.Username,
			Password:         credentials.Password,
			Domain:           credentials.Domain,
			PrivateKey:       credentials.PrivateKey,
			Passphrase:       credentials.Passphrase,
			CredentialSource: credentials.CredentialSource,
		},
	}, nil
}

func (s Service) CreateTunnelProxy(ctx context.Context, gatewayID, targetHost string, targetPort int) (contracts.TunnelProxyResponse, error) {
	proxy, err := s.createTunnelProxy(ctx, gatewayID, targetHost, targetPort)
	if err != nil {
		return contracts.TunnelProxyResponse{}, mapResolveError(err)
	}
	return contracts.TunnelProxyResponse{
		ID:        proxy.ID,
		Host:      proxy.Host,
		Port:      proxy.Port,
		ExpiresIn: proxy.ExpiresIn,
	}, nil
}

func mapResolveError(err error) error {
	if err == nil {
		return nil
	}

	var reqErr *requestError
	if errors.As(err, &reqErr) {
		return &ResolveError{
			Status:  reqErr.status,
			Message: reqErr.message,
		}
	}

	return err
}

func cloneRawJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	cloned := make([]byte, len(value))
	copy(cloned, value)
	return cloned
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
