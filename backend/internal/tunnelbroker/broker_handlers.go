package tunnelbroker

import (
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func (b *Broker) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tunnel/connect", b.HandleTunnelConnect)
	mux.HandleFunc("GET /v1/tunnels", b.HandleTunnelList)
	mux.HandleFunc("GET /v1/tunnels/{gatewayId}", b.HandleTunnelGet)
	mux.HandleFunc("DELETE /v1/tunnels/{gatewayId}", b.HandleTunnelDelete)
	mux.HandleFunc("POST /v1/tcp-proxies", b.HandleCreateTCPProxy)
}

func (b *Broker) HandleTunnelConnect(w http.ResponseWriter, r *http.Request) {
	gatewayID := strings.TrimSpace(r.Header.Get("X-Gateway-Id"))
	bearerToken := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	clientVersion := strings.TrimSpace(r.Header.Get("X-Agent-Version"))
	clientIP := extractClientIP(r)
	clientCertPEM, err := parseClientCertHeader(r.Header.Get("X-Client-Cert"))
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, "invalid x-client-cert header")
		return
	}
	if gatewayID == "" || bearerToken == "" || clientCertPEM == "" {
		app.ErrorJSON(w, http.StatusUnauthorized, "missing tunnel authentication headers")
		return
	}

	if _, err := b.authenticateTunnel(r.Context(), gatewayID, bearerToken, clientCertPEM); err != nil {
		b.config.Logger.Warn("tunnel authentication failed", "gateway_id", gatewayID, "error", err)
		_ = b.config.Store.InsertTunnelAudit(r.Context(), "TUNNEL_MTLS_REJECTED", gatewayID, clientIP, map[string]any{
			"reason": err.Error(),
		})
		app.ErrorJSON(w, http.StatusForbidden, "tunnel authentication failed")
		return
	}

	wsConn, err := b.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	conn := b.registerConnection(gatewayID, wsConn, clientVersion, clientIP)
	if err := b.config.Store.MarkTunnelConnected(r.Context(), gatewayID, conn.connectedAt, clientVersion, clientIP); err != nil {
		b.config.Logger.Warn("persist tunnel connect failed", "gateway_id", gatewayID, "error", err)
	}
	if err := b.config.Store.InsertTunnelAudit(r.Context(), "TUNNEL_CONNECT", gatewayID, clientIP, map[string]any{
		"clientVersion": clientVersion,
		"clientIp":      clientIP,
	}); err != nil {
		b.config.Logger.Warn("insert tunnel connect audit failed", "gateway_id", gatewayID, "error", err)
	}
}

func (b *Broker) HandleTunnelList(w http.ResponseWriter, _ *http.Request) {
	b.mu.RLock()
	statuses := make([]contracts.TunnelStatus, 0, len(b.registry))
	for _, conn := range b.registry {
		statuses = append(statuses, describeConnection(conn))
	}
	b.mu.RUnlock()
	app.WriteJSON(w, http.StatusOK, contracts.TunnelStatusesResponse{Tunnels: statuses})
}

func (b *Broker) HandleTunnelGet(w http.ResponseWriter, r *http.Request) {
	gatewayID := strings.TrimSpace(r.PathValue("gatewayId"))
	status, ok := b.getStatus(gatewayID)
	if !ok {
		app.ErrorJSON(w, http.StatusNotFound, "tunnel not found")
		return
	}
	app.WriteJSON(w, http.StatusOK, status)
}

func (b *Broker) HandleTunnelDelete(w http.ResponseWriter, r *http.Request) {
	gatewayID := strings.TrimSpace(r.PathValue("gatewayId"))
	if gatewayID == "" {
		app.ErrorJSON(w, http.StatusBadRequest, "gatewayId is required")
		return
	}
	if !b.disconnectTunnel(gatewayID, "revoked") {
		app.ErrorJSON(w, http.StatusNotFound, "tunnel not found")
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"disconnected": true, "gatewayId": gatewayID})
}

func (b *Broker) HandleCreateTCPProxy(w http.ResponseWriter, r *http.Request) {
	var req contracts.TunnelProxyRequest
	if err := app.ReadJSON(r, &req); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.GatewayID) == "" || strings.TrimSpace(req.TargetHost) == "" || req.TargetPort <= 0 {
		app.ErrorJSON(w, http.StatusBadRequest, "gatewayId, targetHost and targetPort are required")
		return
	}

	proxy, err := b.createTCPProxy(req)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	app.WriteJSON(w, http.StatusOK, proxy)
}
