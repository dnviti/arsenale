package gateways

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
)

func (s Service) buildManagedGatewayTunnelEnv(ctx context.Context, record gatewayRecord) (map[string]string, error) {
	if !record.TunnelEnabled {
		return nil, nil
	}

	gateway := record
	if gateway.TunnelClientCert == nil || gateway.TunnelClientKey == nil || gateway.TunnelClientKeyIV == nil || gateway.TunnelClientKeyTag == nil {
		if err := s.ensureTunnelMaterial(ctx, record.TenantID, record.ID); err != nil {
			return nil, err
		}
		reloaded, err := s.loadGateway(ctx, record.TenantID, record.ID)
		if err != nil {
			return nil, err
		}
		gateway = reloaded
	}

	if gateway.EncryptedTunnelToken == nil || gateway.TunnelTokenIV == nil || gateway.TunnelTokenTag == nil {
		return nil, &requestError{status: http.StatusBadRequest, message: "Tunnel is enabled but no tunnel token is configured for this gateway"}
	}

	token, err := decryptEncryptedField(s.ServerEncryptionKey, encryptedField{
		Ciphertext: *gateway.EncryptedTunnelToken,
		IV:         *gateway.TunnelTokenIV,
		Tag:        *gateway.TunnelTokenTag,
	})
	if err != nil {
		return nil, fmt.Errorf("decrypt tunnel token: %w", err)
	}

	clientKey, err := decryptEncryptedField(s.ServerEncryptionKey, encryptedField{
		Ciphertext: derefString(gateway.TunnelClientKey),
		IV:         derefString(gateway.TunnelClientKeyIV),
		Tag:        derefString(gateway.TunnelClientKeyTag),
	})
	if err != nil {
		return nil, fmt.Errorf("decrypt tunnel client key: %w", err)
	}

	env := map[string]string{
		"TUNNEL_SERVER_URL":  buildManagedTunnelConnectURL(s.TunnelBrokerURL),
		"TUNNEL_TOKEN":       token,
		"TUNNEL_GATEWAY_ID":  gateway.ID,
		"TUNNEL_LOCAL_HOST":  "127.0.0.1",
		"TUNNEL_LOCAL_PORT":  strconv.Itoa(s.managedGatewayPrimaryPort(gateway.Type)),
		"TUNNEL_CLIENT_CERT": derefString(gateway.TunnelClientCert),
		"TUNNEL_CLIENT_KEY":  clientKey,
	}
	return env, nil
}

func (s Service) ensureTunnelMaterial(ctx context.Context, tenantID, gatewayID string) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tunnel material transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := s.ensureTunnelMTLSMaterialTx(ctx, tx, tenantID, gatewayID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tunnel material transaction: %w", err)
	}
	return nil
}

func buildManagedTunnelConnectURL(rawBaseURL string) string {
	rawBaseURL = strings.TrimSpace(rawBaseURL)
	if rawBaseURL == "" {
		return "ws://tunnel-broker:8092/api/tunnel/connect"
	}

	parsed, err := url.Parse(rawBaseURL)
	if err != nil {
		return "ws://tunnel-broker:8092/api/tunnel/connect"
	}

	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	case "ws", "wss":
	default:
		parsed.Scheme = "ws"
	}

	path := strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(path, "/api/tunnel/connect") {
		if path == "" {
			path = "/api/tunnel/connect"
		} else {
			path += "/api/tunnel/connect"
		}
	}
	parsed.Path = path
	parsed.RawPath = ""
	return parsed.String()
}

func buildServiceSPIFFEID(trustDomain, service string) string {
	trustDomain = strings.TrimSpace(trustDomain)
	if trustDomain == "" {
		trustDomain = defaultTunnelTrustDomain
	}
	service = strings.TrimSpace(service)
	if service == "" {
		service = "control-plane-api"
	}
	return "spiffe://" + trustDomain + "/service/" + service
}

func defaultGatewayGRPCClientCAPath(certPath string) string {
	certPath = strings.TrimSpace(certPath)
	if certPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(certPath), "client-ca.pem")
}
