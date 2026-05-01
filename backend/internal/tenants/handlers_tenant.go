package tenants

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/jackc/pgx/v5"
)

func (s Service) HandleGetMine(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if strings.TrimSpace(claims.TenantID) == "" {
		app.ErrorJSON(w, http.StatusForbidden, "You must belong to an organization to perform this action")
		return
	}

	result, err := s.GetTenant(r.Context(), claims.TenantID)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			app.ErrorJSON(w, http.StatusNotFound, "Organization not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleListMine(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.ListUserTenants(r.Context(), claims.UserID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetIPAllowlist(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	if !claimsCanAdminTenant(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient tenant role")
		return
	}

	result, err := s.GetIPAllowlist(r.Context(), claims.TenantID)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			app.ErrorJSON(w, http.StatusNotFound, "Organization not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleUpdateIPAllowlist(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	if !claimsCanAdminTenant(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient tenant role")
		return
	}

	var payload ipAllowlistResponse
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	switch payload.Mode {
	case "flag", "block":
	default:
		app.ErrorJSON(w, http.StatusBadRequest, "mode must be either 'flag' or 'block'")
		return
	}
	if payload.Entries == nil {
		payload.Entries = []string{}
	}

	result, err := s.UpdateIPAllowlist(r.Context(), claims.TenantID, payload)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetMFAStats(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	if !claimsCanAdminTenant(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient tenant role")
		return
	}

	result, err := s.GetTenantMFAStats(r.Context(), claims.TenantID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleUpdate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	if !claimsCanAdminTenant(claims.TenantRole) {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient tenant role")
		return
	}

	var payload map[string]json.RawMessage
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.UpdateTenant(r.Context(), claims.TenantID, payload)
	if err != nil {
		var reqErr *requestError
		if errors.As(err, &reqErr) {
			app.ErrorJSON(w, reqErr.status, reqErr.message)
			return
		}
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleDelete(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	if !strings.EqualFold(strings.TrimSpace(claims.TenantRole), "OWNER") {
		app.ErrorJSON(w, http.StatusForbidden, "Insufficient tenant role")
		return
	}

	result, err := s.DeleteTenant(r.Context(), claims.TenantID)
	if err != nil {
		var reqErr *requestError
		if errors.As(err, &reqErr) {
			app.ErrorJSON(w, reqErr.status, reqErr.message)
			return
		}
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}
