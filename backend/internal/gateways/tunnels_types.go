package gateways

import "time"

const (
	defaultTunnelBrokerAddress = "http://tunnel-broker:8092"
	defaultTunnelTrustDomain   = "arsenale.local"
	tunnelTokenBytes           = 32
	tunnelCAValidityDays       = 3650
	tunnelClientValidityDays   = 90
)

type tunnelTokenResponse struct {
	Token            string     `json:"token"`
	TunnelEnabled    bool       `json:"tunnelEnabled"`
	TunnelConnected  bool       `json:"tunnelConnected"`
	GatewayID        string     `json:"gatewayId"`
	GatewayType      string     `json:"gatewayType"`
	TunnelLocalHost  string     `json:"tunnelLocalHost"`
	TunnelLocalPort  int        `json:"tunnelLocalPort"`
	TunnelClientCert string     `json:"tunnelClientCert"`
	TunnelClientKey  string     `json:"tunnelClientKey"`
	TunnelClientExp  *time.Time `json:"tunnelClientCertExp,omitempty"`
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
