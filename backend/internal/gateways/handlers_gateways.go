package gateways

import (
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/pkg/gatewayruntime"
)

// gatewayTypesResponse is the catalog of gateway types with human-readable
// metadata, so the UI and CLI can explain what each type deploys.
type gatewayTypesResponse struct {
	Types []gatewayruntime.TypeInfo `json:"types"`
}

// HandleTypes returns the gateway type catalog. It is tenant-independent (static
// metadata), so any authenticated user may read it.
func (s Service) HandleTypes(w http.ResponseWriter, _ *http.Request, _ authn.Claims) {
	app.WriteJSON(w, http.StatusOK, gatewayTypesResponse{Types: gatewayruntime.Catalog()})
}

func (s Service) HandleList(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.ListGateways(r.Context(), claims.TenantID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleCreate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	var payload createPayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.CreateGateway(r.Context(), claims, payload, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleUpdate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	var payload updatePayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.UpdateGateway(r.Context(), claims, r.PathValue("id"), payload, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleDelete(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	force := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("force")), "true")
	result, err := s.DeleteGateway(r.Context(), claims, r.PathValue("id"), force, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleTestConnectivity(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.TestGatewayConnectivity(r.Context(), claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetScalingStatus(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.GetScalingStatus(r.Context(), claims.TenantID, r.PathValue("id"))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}
