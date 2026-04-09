package gateways

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

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

type tunnelTCPProxy struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (s Service) createTunnelTCPProxy(ctx context.Context, gatewayID, targetHost string, targetPort int) (tunnelTCPProxy, error) {
	body, err := json.Marshal(map[string]any{
		"gatewayId":  gatewayID,
		"targetHost": targetHost,
		"targetPort": targetPort,
	})
	if err != nil {
		return tunnelTCPProxy{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(s.tunnelBrokerURL(), "/")+"/v1/tcp-proxies", bytes.NewReader(body))
	if err != nil {
		return tunnelTCPProxy{}, err
	}
	req.Header.Set("content-type", "application/json")

	resp, err := s.client().Do(req)
	if err != nil {
		return tunnelTCPProxy{}, fmt.Errorf("create tunnel proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return tunnelTCPProxy{}, fmt.Errorf("create tunnel proxy: %w", decodeBrokerError(resp))
	}

	var proxy tunnelTCPProxy
	if err := json.NewDecoder(resp.Body).Decode(&proxy); err != nil {
		return tunnelTCPProxy{}, fmt.Errorf("decode tunnel proxy response: %w", err)
	}
	return proxy, nil
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

// sanitizeTunnelEventDetails keeps audit event payloads narrow so the API only returns operator-visible tunnel metadata.
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
