package gateways

import (
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandleTunnelOverview(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.GetTunnelOverview(r.Context(), claims.TenantID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGenerateTunnelToken(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.GenerateTunnelToken(r.Context(), claims, r.PathValue("id"), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleRevokeTunnelToken(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.RevokeTunnelToken(r.Context(), claims, r.PathValue("id"), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleForceDisconnectTunnel(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.ForceDisconnectTunnel(r.Context(), claims, r.PathValue("id"), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetTunnelEvents(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.GetTunnelEvents(r.Context(), claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetTunnelMetrics(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.GetTunnelMetrics(r.Context(), claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}
