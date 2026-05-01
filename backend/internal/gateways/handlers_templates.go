package gateways

import (
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandleListTemplates(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.ListGatewayTemplates(r.Context(), claims.TenantID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleCreateTemplate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	var payload createTemplatePayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.CreateGatewayTemplate(r.Context(), claims, payload, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleUpdateTemplate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	var payload updateTemplatePayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.UpdateGatewayTemplate(r.Context(), claims, r.PathValue("templateId"), payload, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleDeleteTemplate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.DeleteGatewayTemplate(r.Context(), claims, r.PathValue("templateId"), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleDeployTemplate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !requireGatewayManager(w, claims) {
		return
	}
	result, err := s.DeployGatewayTemplate(r.Context(), claims, r.PathValue("templateId"), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}
