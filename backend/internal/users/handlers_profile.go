package users

import (
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/jackc/pgx/v5"
)

func (s Service) HandleProfile(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	profile, err := s.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			app.ErrorJSON(w, http.StatusNotFound, "User not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}

	app.WriteJSON(w, http.StatusOK, profile)
}

func (s Service) HandlePermissions(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	result, err := s.GetCurrentPermissions(r.Context(), claims)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleUpdateProfile(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var payload struct {
		Username *string `json:"username"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if payload.Username != nil {
		username := strings.TrimSpace(*payload.Username)
		if len(username) < 1 || len(username) > 50 {
			app.ErrorJSON(w, http.StatusBadRequest, "username must be between 1 and 50 characters")
			return
		}
		payload.Username = &username
	}

	result, err := s.UpdateProfile(r.Context(), claims.UserID, payload.Username, requestIP(r))
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			app.ErrorJSON(w, http.StatusNotFound, "User not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleSearch(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if claims.TenantID == "" {
		app.ErrorJSON(w, http.StatusForbidden, "You must belong to an organization to perform this action")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(query) < 1 || len(query) > 100 {
		app.ErrorJSON(w, http.StatusBadRequest, "q must be between 1 and 100 characters")
		return
	}

	rawScope := r.URL.Query().Get("scope")
	if rawScope == "" {
		rawScope = "team"
	}
	if rawScope != "tenant" && rawScope != "team" {
		app.ErrorJSON(w, http.StatusBadRequest, "scope must be one of tenant, team")
		return
	}

	teamID := strings.TrimSpace(r.URL.Query().Get("teamId"))
	if rawScope == "team" && teamID == "" {
		app.ErrorJSON(w, http.StatusBadRequest, "teamId is required when scope is team")
		return
	}

	effectiveScope := rawScope
	if rawScope == "tenant" {
		if _, ok := adminRoles[claims.TenantRole]; !ok {
			effectiveScope = "team"
		}
	}

	results, err := s.SearchUsers(r.Context(), claims.UserID, claims.TenantID, query, effectiveScope, teamID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	app.WriteJSON(w, http.StatusOK, results)
}
