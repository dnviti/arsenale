package accesspolicies

import (
	"errors"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandleList(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireTenantAdmin(claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	items, err := s.ListPolicies(r.Context(), claims.TenantID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, items)
}

func (s Service) HandleCreate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireTenantAdmin(claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	var payload struct {
		TargetType           string  `json:"targetType"`
		TargetID             string  `json:"targetId"`
		AllowedTimeWindows   *string `json:"allowedTimeWindows"`
		RequireTrustedDevice *bool   `json:"requireTrustedDevice"`
		RequireMFAStepUp     *bool   `json:"requireMfaStepUp"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.CreatePolicy(r.Context(), claims.TenantID, payload.TargetType, payload.TargetID, payload.AllowedTimeWindows, payload.RequireTrustedDevice, payload.RequireMFAStepUp)
	if err != nil {
		writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusCreated, result)
}

func (s Service) HandleUpdate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireTenantAdmin(claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	var payload struct {
		AllowedTimeWindows   *string `json:"allowedTimeWindows"`
		RequireTrustedDevice *bool   `json:"requireTrustedDevice"`
		RequireMFAStepUp     *bool   `json:"requireMfaStepUp"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.UpdatePolicy(r.Context(), claims.TenantID, r.PathValue("id"), payload.AllowedTimeWindows, payload.RequireTrustedDevice, payload.RequireMFAStepUp)
	if err != nil {
		writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleDelete(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireTenantAdmin(claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	if err := s.DeletePolicy(r.Context(), claims.TenantID, r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func writeError(w http.ResponseWriter, err error) {
	var reqErr *requestError
	if errors.As(err, &reqErr) {
		app.ErrorJSON(w, reqErr.status, reqErr.message)
		return
	}
	app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
}
