package gateways

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/jackc/pgx/v5"
)

const (
	defaultTunnelBrokerAddress = "http://tunnel-broker:8092"
	defaultTunnelTrustDomain   = "arsenale.local"
	tunnelTokenBytes           = 32
	tunnelCAValidityDays       = 3650
	tunnelClientValidityDays   = 90
)

type tunnelTokenResponse struct {
	Token           string `json:"token"`
	TunnelEnabled   bool   `json:"tunnelEnabled"`
	TunnelConnected bool   `json:"tunnelConnected"`
}

type revokeTunnelTokenResponse struct {
	Revoked       bool `json:"revoked"`
	TunnelEnabled bool `json:"tunnelEnabled"`
}

type disconnectTunnelResponse struct {
	Disconnected bool `json:"disconnected"`
}

type tunnelOverviewResponse struct {
	Total        int  `json:"total"`
	Connected    int  `json:"connected"`
	Disconnected int  `json:"disconnected"`
	AvgRTTMS     *int `json:"avgRttMs"`
}

type tunnelEventsResponse struct {
	Events []tunnelEventResponse `json:"events"`
}

type tunnelEventResponse struct {
	Action    string         `json:"action"`
	Timestamp time.Time      `json:"timestamp"`
	Details   map[string]any `json:"details"`
	IPAddress *string        `json:"ipAddress"`
}

type tunnelMetricsResponse struct {
	Connected        bool                   `json:"connected"`
	ConnectedAt      *time.Time             `json:"connectedAt,omitempty"`
	LastHeartbeat    *time.Time             `json:"lastHeartbeat,omitempty"`
	PingPongLatency  *int                   `json:"pingPongLatency,omitempty"`
	ActiveStreams    *int                   `json:"activeStreams,omitempty"`
	BytesTransferred *int64                 `json:"bytesTransferred,omitempty"`
	ClientVersion    *string                `json:"clientVersion,omitempty"`
	ClientIP         *string                `json:"clientIp,omitempty"`
	Heartbeat        *tunnelHeartbeatMetric `json:"heartbeatMetadata,omitempty"`
}

type tunnelHeartbeatMetric struct {
	Healthy       bool `json:"healthy"`
	LatencyMS     *int `json:"latencyMs,omitempty"`
	ActiveStreams *int `json:"activeStreams,omitempty"`
}

type mtlsAuditDetails struct {
	tenantCAGenerated bool
	caFingerprint     string
	clientExpiry      time.Time
}

func (s Service) GetTunnelOverview(ctx context.Context, tenantID string) (tunnelOverviewResponse, error) {
	if s.DB == nil {
		return tunnelOverviewResponse{}, errors.New("database is unavailable")
	}

	rows, err := s.DB.Query(ctx, `
SELECT id
FROM "Gateway"
WHERE "tenantId" = $1
  AND "tunnelEnabled" = true
`, tenantID)
	if err != nil {
		return tunnelOverviewResponse{}, fmt.Errorf("list tunneled gateways: %w", err)
	}
	defer rows.Close()

	enabledGateways := make(map[string]struct{})
	for rows.Next() {
		var gatewayID string
		if err := rows.Scan(&gatewayID); err != nil {
			return tunnelOverviewResponse{}, fmt.Errorf("scan tunneled gateway: %w", err)
		}
		enabledGateways[gatewayID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return tunnelOverviewResponse{}, fmt.Errorf("iterate tunneled gateways: %w", err)
	}

	statuses, err := s.listTunnelStatuses(ctx)
	if err != nil {
		statuses = nil
	}
	return aggregateTunnelOverview(enabledGateways, statuses), nil
}

func (s Service) GenerateTunnelToken(ctx context.Context, claims authn.Claims, gatewayID, ipAddress string) (tunnelTokenResponse, error) {
	if s.DB == nil {
		return tunnelTokenResponse{}, errors.New("database is unavailable")
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return tunnelTokenResponse{}, fmt.Errorf("begin tunnel token transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	mtlsDetails, err := s.ensureTunnelMTLSMaterialTx(ctx, tx, claims.TenantID, gatewayID)
	if err != nil {
		return tunnelTokenResponse{}, err
	}

	rawToken, err := randomTunnelToken()
	if err != nil {
		return tunnelTokenResponse{}, fmt.Errorf("generate tunnel token: %w", err)
	}
	encToken, err := encryptValue(s.ServerEncryptionKey, rawToken)
	if err != nil {
		return tunnelTokenResponse{}, fmt.Errorf("encrypt tunnel token: %w", err)
	}
	tokenHash := hashToken(rawToken)

	if _, err := tx.Exec(ctx, `
UPDATE "Gateway"
SET "tunnelEnabled" = true,
    "encryptedTunnelToken" = $2,
    "tunnelTokenIV" = $3,
    "tunnelTokenTag" = $4,
    "tunnelTokenHash" = $5,
    "updatedAt" = NOW()
WHERE id = $1
`, gatewayID, encToken.Ciphertext, encToken.IV, encToken.Tag, tokenHash); err != nil {
		return tunnelTokenResponse{}, fmt.Errorf("store tunnel token: %w", err)
	}

	if mtlsDetails != nil {
		if err := s.insertAuditLogTx(ctx, tx, claims.UserID, "TUNNEL_TOKEN_GENERATE", gatewayID, map[string]any{
			"mtlsCertsGenerated": true,
			"tenantId":           claims.TenantID,
			"tenantCaGenerated":  mtlsDetails.tenantCAGenerated,
			"caFingerprint":      truncateString(mtlsDetails.caFingerprint, 16),
			"clientCertExpiry":   mtlsDetails.clientExpiry.UTC().Format(time.RFC3339),
		}, ipAddress); err != nil {
			return tunnelTokenResponse{}, err
		}
	}
	if err := s.insertAuditLogTx(ctx, tx, claims.UserID, "TUNNEL_TOKEN_GENERATE", gatewayID, nil, ipAddress); err != nil {
		return tunnelTokenResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return tunnelTokenResponse{}, fmt.Errorf("commit tunnel token transaction: %w", err)
	}

	return tunnelTokenResponse{
		Token:           rawToken,
		TunnelEnabled:   true,
		TunnelConnected: false,
	}, nil
}

func (s Service) RevokeTunnelToken(ctx context.Context, claims authn.Claims, gatewayID, ipAddress string) (revokeTunnelTokenResponse, error) {
	if s.DB == nil {
		return revokeTunnelTokenResponse{}, errors.New("database is unavailable")
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return revokeTunnelTokenResponse{}, fmt.Errorf("begin tunnel revoke transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockGatewayForTenant(ctx, tx, claims.TenantID, gatewayID); err != nil {
		return revokeTunnelTokenResponse{}, err
	}

	if _, err := tx.Exec(ctx, `
UPDATE "Gateway"
SET "tunnelEnabled" = false,
    "encryptedTunnelToken" = NULL,
    "tunnelTokenIV" = NULL,
    "tunnelTokenTag" = NULL,
    "tunnelTokenHash" = NULL,
    "tunnelConnectedAt" = NULL,
    "tunnelLastHeartbeat" = NULL,
    "updatedAt" = NOW()
WHERE id = $1
`, gatewayID); err != nil {
		return revokeTunnelTokenResponse{}, fmt.Errorf("revoke tunnel token: %w", err)
	}

	if err := s.insertAuditLogTx(ctx, tx, claims.UserID, "TUNNEL_TOKEN_ROTATE", gatewayID, map[string]any{
		"revoked": true,
	}, ipAddress); err != nil {
		return revokeTunnelTokenResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return revokeTunnelTokenResponse{}, fmt.Errorf("commit tunnel revoke transaction: %w", err)
	}

	_ = s.disconnectTunnel(ctx, gatewayID)

	return revokeTunnelTokenResponse{
		Revoked:       true,
		TunnelEnabled: false,
	}, nil
}

func (s Service) ForceDisconnectTunnel(ctx context.Context, claims authn.Claims, gatewayID, ipAddress string) (disconnectTunnelResponse, error) {
	if s.DB == nil {
		return disconnectTunnelResponse{}, errors.New("database is unavailable")
	}

	if _, err := s.loadGatewayOwnership(ctx, claims.TenantID, gatewayID); err != nil {
		return disconnectTunnelResponse{}, err
	}

	status, err := s.getTunnelStatus(ctx, gatewayID)
	if err != nil || status == nil || !status.Connected {
		return disconnectTunnelResponse{}, &requestError{status: http.StatusBadRequest, message: "Tunnel is not connected"}
	}

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return disconnectTunnelResponse{}, fmt.Errorf("begin tunnel disconnect transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := lockGatewayForTenant(ctx, tx, claims.TenantID, gatewayID); err != nil {
		return disconnectTunnelResponse{}, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE "Gateway"
SET "tunnelConnectedAt" = NULL,
    "tunnelLastHeartbeat" = NULL,
    "updatedAt" = NOW()
WHERE id = $1
`, gatewayID); err != nil {
		return disconnectTunnelResponse{}, fmt.Errorf("clear tunnel connection state: %w", err)
	}
	if err := s.insertAuditLogTx(ctx, tx, claims.UserID, "TUNNEL_DISCONNECT", gatewayID, map[string]any{
		"forced": true,
	}, ipAddress); err != nil {
		return disconnectTunnelResponse{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return disconnectTunnelResponse{}, fmt.Errorf("commit tunnel disconnect transaction: %w", err)
	}

	_ = s.disconnectTunnel(ctx, gatewayID)

	return disconnectTunnelResponse{Disconnected: true}, nil
}

func (s Service) GetTunnelEvents(ctx context.Context, tenantID, gatewayID string) (tunnelEventsResponse, error) {
	if s.DB == nil {
		return tunnelEventsResponse{}, errors.New("database is unavailable")
	}
	if _, err := s.loadGatewayOwnership(ctx, tenantID, gatewayID); err != nil {
		return tunnelEventsResponse{}, err
	}

	rows, err := s.DB.Query(ctx, `
SELECT action::text, "createdAt", details, "ipAddress"
FROM "AuditLog"
WHERE "targetId" = $1
  AND "targetType" = 'Gateway'
  AND action IN ('TUNNEL_CONNECT'::"AuditAction", 'TUNNEL_DISCONNECT'::"AuditAction")
ORDER BY "createdAt" DESC
LIMIT 20
`, gatewayID)
	if err != nil {
		return tunnelEventsResponse{}, fmt.Errorf("list tunnel events: %w", err)
	}
	defer rows.Close()

	events := make([]tunnelEventResponse, 0)
	for rows.Next() {
		var item tunnelEventResponse
		var rawDetails []byte
		if err := rows.Scan(&item.Action, &item.Timestamp, &rawDetails, &item.IPAddress); err != nil {
			return tunnelEventsResponse{}, fmt.Errorf("scan tunnel event: %w", err)
		}
		item.Details = sanitizeTunnelEventDetails(rawDetails)
		events = append(events, item)
	}
	if err := rows.Err(); err != nil {
		return tunnelEventsResponse{}, fmt.Errorf("iterate tunnel events: %w", err)
	}

	return tunnelEventsResponse{Events: events}, nil
}

func (s Service) GetTunnelMetrics(ctx context.Context, tenantID, gatewayID string) (tunnelMetricsResponse, error) {
	if s.DB == nil {
		return tunnelMetricsResponse{}, errors.New("database is unavailable")
	}
	if _, err := s.loadGatewayOwnership(ctx, tenantID, gatewayID); err != nil {
		return tunnelMetricsResponse{}, err
	}

	status, err := s.getTunnelStatus(ctx, gatewayID)
	if err != nil {
		return tunnelMetricsResponse{Connected: false}, nil
	}
	return tunnelMetricsFromStatus(status), nil
}

func (s Service) ensureTunnelMTLSMaterialTx(ctx context.Context, tx pgx.Tx, tenantID, gatewayID string) (*mtlsAuditDetails, error) {
	var tenantCACert, tenantCAKey, tenantCAKeyIV, tenantCAKeyTag, tenantCAFingerprint *string
	if err := tx.QueryRow(ctx, `
SELECT "tunnelCaCert", "tunnelCaKey", "tunnelCaKeyIV", "tunnelCaKeyTag", "tunnelCaCertFingerprint"
FROM "Tenant"
WHERE id = $1
FOR UPDATE
`, tenantID).Scan(&tenantCACert, &tenantCAKey, &tenantCAKeyIV, &tenantCAKeyTag, &tenantCAFingerprint); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &requestError{status: http.StatusNotFound, message: "Tenant not found"}
		}
		return nil, fmt.Errorf("load tenant tunnel CA: %w", err)
	}

	var tunnelClientCert, tunnelClientKey, tunnelClientKeyIV, tunnelClientKeyTag *string
	if err := tx.QueryRow(ctx, `
SELECT "tunnelClientCert", "tunnelClientKey", "tunnelClientKeyIV", "tunnelClientKeyTag"
FROM "Gateway"
WHERE id = $1
  AND "tenantId" = $2
FOR UPDATE
`, gatewayID, tenantID).Scan(&tunnelClientCert, &tunnelClientKey, &tunnelClientKeyIV, &tunnelClientKeyTag); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &requestError{status: http.StatusNotFound, message: "Gateway not found"}
		}
		return nil, fmt.Errorf("load gateway tunnel certificate: %w", err)
	}

	tenantGenerated := false
	caCertPEM := derefString(tenantCACert)
	caFingerprint := derefString(tenantCAFingerprint)
	var caKeyPEM string

	if strings.TrimSpace(caCertPEM) == "" || strings.TrimSpace(derefString(tenantCAKey)) == "" || strings.TrimSpace(derefString(tenantCAKeyIV)) == "" || strings.TrimSpace(derefString(tenantCAKeyTag)) == "" {
		certPEM, keyPEM, err := generateCACert("arsenale-tenant-" + tenantID)
		if err != nil {
			return nil, fmt.Errorf("generate tenant CA: %w", err)
		}
		encKey, err := encryptValue(s.ServerEncryptionKey, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("encrypt tenant CA key: %w", err)
		}
		fingerprint, err := certificateFingerprint(certPEM)
		if err != nil {
			return nil, fmt.Errorf("fingerprint tenant CA: %w", err)
		}
		if _, err := tx.Exec(ctx, `
UPDATE "Tenant"
SET "tunnelCaCert" = $2,
    "tunnelCaKey" = $3,
    "tunnelCaKeyIV" = $4,
    "tunnelCaKeyTag" = $5,
    "tunnelCaCertFingerprint" = $6,
    "updatedAt" = NOW()
WHERE id = $1
`, tenantID, certPEM, encKey.Ciphertext, encKey.IV, encKey.Tag, fingerprint); err != nil {
			return nil, fmt.Errorf("store tenant CA: %w", err)
		}
		tenantGenerated = true
		caCertPEM = certPEM
		caKeyPEM = keyPEM
		caFingerprint = fingerprint
	} else {
		decryptedKey, err := decryptEncryptedField(s.ServerEncryptionKey, encryptedField{
			Ciphertext: derefString(tenantCAKey),
			IV:         derefString(tenantCAKeyIV),
			Tag:        derefString(tenantCAKeyTag),
		})
		if err != nil {
			return nil, fmt.Errorf("decrypt tenant CA key: %w", err)
		}
		caKeyPEM = decryptedKey
		if strings.TrimSpace(caFingerprint) == "" {
			fingerprint, err := certificateFingerprint(caCertPEM)
			if err != nil {
				return nil, fmt.Errorf("fingerprint tenant CA: %w", err)
			}
			caFingerprint = fingerprint
		}
	}

	needsClientCert := strings.TrimSpace(derefString(tunnelClientCert)) == "" ||
		strings.TrimSpace(derefString(tunnelClientKey)) == "" ||
		strings.TrimSpace(derefString(tunnelClientKeyIV)) == "" ||
		strings.TrimSpace(derefString(tunnelClientKeyTag)) == ""
	if !needsClientCert {
		return nil, nil
	}

	clientCertPEM, clientKeyPEM, clientExpiry, err := generateClientCertificate(caCertPEM, caKeyPEM, gatewayID, buildGatewaySPIFFEID(s.tunnelTrustDomain(), gatewayID))
	if err != nil {
		return nil, fmt.Errorf("generate tunnel client certificate: %w", err)
	}
	encClientKey, err := encryptValue(s.ServerEncryptionKey, clientKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("encrypt tunnel client key: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE "Gateway"
SET "tunnelClientCert" = $2,
    "tunnelClientKey" = $3,
    "tunnelClientKeyIV" = $4,
    "tunnelClientKeyTag" = $5,
    "tunnelClientCertExp" = $6,
    "updatedAt" = NOW()
WHERE id = $1
`, gatewayID, clientCertPEM, encClientKey.Ciphertext, encClientKey.IV, encClientKey.Tag, clientExpiry.UTC()); err != nil {
		return nil, fmt.Errorf("store tunnel client certificate: %w", err)
	}

	return &mtlsAuditDetails{
		tenantCAGenerated: tenantGenerated,
		caFingerprint:     caFingerprint,
		clientExpiry:      clientExpiry,
	}, nil
}

func lockGatewayForTenant(ctx context.Context, tx pgx.Tx, tenantID, gatewayID string) error {
	var id string
	if err := tx.QueryRow(ctx, `
SELECT id
FROM "Gateway"
WHERE id = $1
  AND "tenantId" = $2
FOR UPDATE
`, gatewayID, tenantID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &requestError{status: http.StatusNotFound, message: "Gateway not found"}
		}
		return fmt.Errorf("load gateway: %w", err)
	}
	return nil
}

func (s Service) loadGatewayOwnership(ctx context.Context, tenantID, gatewayID string) (string, error) {
	if s.DB == nil {
		return "", errors.New("database is unavailable")
	}

	var id string
	if err := s.DB.QueryRow(ctx, `
SELECT id
FROM "Gateway"
WHERE id = $1
  AND "tenantId" = $2
`, gatewayID, tenantID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", &requestError{status: http.StatusNotFound, message: "Gateway not found"}
		}
		return "", fmt.Errorf("load gateway: %w", err)
	}
	return id, nil
}

func (s Service) listTunnelStatuses(ctx context.Context) ([]contracts.TunnelStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.tunnelBrokerURL(), "/")+"/v1/tunnels", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("list broker tunnels: %w", decodeBrokerError(resp))
	}

	var payload contracts.TunnelStatusesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode broker tunnels: %w", err)
	}
	return payload.Tunnels, nil
}

func (s Service) getTunnelStatus(ctx context.Context, gatewayID string) (*contracts.TunnelStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(s.tunnelBrokerURL(), "/")+"/v1/tunnels/"+url.PathEscape(gatewayID), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, nil
	case resp.StatusCode >= 400:
		return nil, fmt.Errorf("get broker tunnel: %w", decodeBrokerError(resp))
	}

	var payload contracts.TunnelStatus
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode broker tunnel: %w", err)
	}
	if !payload.Connected {
		return nil, nil
	}
	return &payload, nil
}

func (s Service) disconnectTunnel(ctx context.Context, gatewayID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, strings.TrimRight(s.tunnelBrokerURL(), "/")+"/v1/tunnels/"+url.PathEscape(gatewayID), nil)
	if err != nil {
		return err
	}

	resp, err := s.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 400 {
		return decodeBrokerError(resp)
	}
	return nil
}

func (s Service) client() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (s Service) tunnelBrokerURL() string {
	if value := strings.TrimSpace(s.TunnelBrokerURL); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("GO_TUNNEL_BROKER_URL")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("TUNNEL_BROKER_URL")); value != "" {
		return value
	}
	return defaultTunnelBrokerAddress
}

func (s Service) tunnelTrustDomain() string {
	if value := strings.TrimSpace(s.TunnelTrustDomain); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("SPIFFE_TRUST_DOMAIN")); value != "" {
		return value
	}
	return defaultTunnelTrustDomain
}

func aggregateTunnelOverview(enabledGateways map[string]struct{}, statuses []contracts.TunnelStatus) tunnelOverviewResponse {
	statusesByGateway := make(map[string]contracts.TunnelStatus, len(statuses))
	rttSum := 0
	rttCount := 0
	for _, status := range statuses {
		if !status.Connected {
			continue
		}
		if _, ok := enabledGateways[status.GatewayID]; !ok {
			continue
		}
		statusesByGateway[status.GatewayID] = status
		if status.PingPongLatencyMs != nil {
			rttSum += int(*status.PingPongLatencyMs)
			rttCount++
		}
	}

	total := len(enabledGateways)
	connected := len(statusesByGateway)
	result := tunnelOverviewResponse{
		Total:        total,
		Connected:    connected,
		Disconnected: total - connected,
	}
	if rttCount > 0 {
		avg := rttSum / rttCount
		result.AvgRTTMS = &avg
	}
	return result
}

func tunnelMetricsFromStatus(status *contracts.TunnelStatus) tunnelMetricsResponse {
	if status == nil || !status.Connected {
		return tunnelMetricsResponse{Connected: false}
	}

	result := tunnelMetricsResponse{
		Connected: true,
	}
	if connectedAt := parseBrokerTime(status.ConnectedAt); connectedAt != nil {
		result.ConnectedAt = connectedAt
	}
	if heartbeatAt := parseBrokerTime(status.LastHeartbeatAt); heartbeatAt != nil {
		result.LastHeartbeat = heartbeatAt
	}
	if status.PingPongLatencyMs != nil {
		latency := int(*status.PingPongLatencyMs)
		result.PingPongLatency = &latency
	}
	activeStreams := status.ActiveStreams
	result.ActiveStreams = &activeStreams
	bytesTransferred := status.BytesTransferred
	result.BytesTransferred = &bytesTransferred
	if value := strings.TrimSpace(status.ClientVersion); value != "" {
		result.ClientVersion = &value
	}
	if value := strings.TrimSpace(status.ClientIP); value != "" {
		result.ClientIP = &value
	}
	if status.Heartbeat != nil {
		result.Heartbeat = &tunnelHeartbeatMetric{
			Healthy:       status.Heartbeat.Healthy,
			LatencyMS:     status.Heartbeat.LatencyMs,
			ActiveStreams: status.Heartbeat.ActiveStreams,
		}
	}
	return result
}

func sanitizeTunnelEventDetails(raw []byte) map[string]any {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" || strings.TrimSpace(string(raw)) == "null" {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}

	safe := make(map[string]any)
	if value, ok := payload["clientVersion"]; ok && value != nil {
		safe["clientVersion"] = fmt.Sprint(value)
	}
	if forced, ok := coerceBool(payload["forced"]); ok {
		safe["forced"] = forced
	}
	if len(safe) == 0 {
		return nil
	}
	return safe
}

func coerceBool(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		typed = strings.TrimSpace(strings.ToLower(typed))
		switch typed {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		}
	case float64:
		return typed != 0, true
	}
	return false, false
}

func parseBrokerTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}

func decodeBrokerError(resp *http.Response) error {
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil {
		if message, _ := payload["error"].(string); strings.TrimSpace(message) != "" {
			return errors.New(message)
		}
	}
	return fmt.Errorf("status %d", resp.StatusCode)
}

func randomTunnelToken() (string, error) {
	raw := make([]byte, tunnelTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func generateCACert(commonName string) (string, string, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate CA key pair: %w", err)
	}

	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber:          randomSerialNumber(),
		Subject:               pkix.Name{CommonName: commonName, Organization: []string{"Arsenale"}},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(tunnelCAValidityDays * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, privateKey.Public(), privateKey)
	if err != nil {
		return "", "", fmt.Errorf("create CA certificate: %w", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", fmt.Errorf("marshal CA private key: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return string(certPEM), string(keyPEM), nil
}

func generateClientCertificate(caCertPEM, caKeyPEM, commonName, spiffeID string) (string, string, time.Time, error) {
	caCertBlock, _ := pem.Decode([]byte(caCertPEM))
	if caCertBlock == nil {
		return "", "", time.Time{}, errors.New("decode CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("parse CA certificate: %w", err)
	}

	caKeyBlock, _ := pem.Decode([]byte(caKeyPEM))
	if caKeyBlock == nil {
		return "", "", time.Time{}, errors.New("decode CA private key PEM")
	}
	caKey, err := x509.ParsePKCS8PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("parse CA private key: %w", err)
	}

	uri, err := url.Parse(spiffeID)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("parse SPIFFE ID: %w", err)
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("generate client key pair: %w", err)
	}

	now := time.Now().UTC()
	expiry := now.Add(tunnelClientValidityDays * 24 * time.Hour)
	template := &x509.Certificate{
		SerialNumber:          randomSerialNumber(),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              expiry,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		URIs:                  []*url.URL{uri},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, caCert, privateKey.Public(), caKey)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("create client certificate: %w", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("marshal client private key: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	return string(certPEM), string(keyPEM), expiry, nil
}

func certificateFingerprint(certPEM string) (string, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return "", errors.New("decode certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parse certificate: %w", err)
	}
	sum := sha256.Sum256(cert.Raw)
	return hex.EncodeToString(sum[:]), nil
}

func randomSerialNumber() *big.Int {
	limit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, limit)
	if err != nil {
		return big.NewInt(time.Now().UnixNano())
	}
	return serial
}

func buildGatewaySPIFFEID(trustDomain, gatewayID string) string {
	domain := strings.ToLower(strings.TrimSpace(trustDomain))
	if domain == "" {
		domain = defaultTunnelTrustDomain
	}
	return fmt.Sprintf("spiffe://%s/gateway/%s", domain, url.PathEscape(strings.TrimSpace(gatewayID)))
}

func truncateString(value string, limit int) string {
	if len(value) <= limit || limit <= 0 {
		return value
	}
	return value[:limit]
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
