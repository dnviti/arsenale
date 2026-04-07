package tenants

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/jackc/pgx/v5"
)

func (s Service) HandleListUsers(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	result, err := s.ListTenantUsers(r.Context(), claims.TenantID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetUserProfile(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	result, err := s.GetUserProfile(r.Context(), claims.TenantID, r.PathValue("userId"), claims.TenantRole)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			app.ErrorJSON(w, http.StatusNotFound, "User not found in this organization")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleGetUserPermissions(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	if err := s.requireManageUsersPermission(r.Context(), claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	result, err := s.GetUserPermissions(r.Context(), claims.TenantID, r.PathValue("userId"))
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "User not found in this organization")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleUpdateUserPermissions(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	if err := s.requireManageUsersPermission(r.Context(), claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	rawPayload := map[string]json.RawMessage{}
	if err := app.ReadJSON(r, &rawPayload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	rawOverrides, ok := rawPayload["overrides"]
	if !ok {
		app.ErrorJSON(w, http.StatusBadRequest, "Missing required field: overrides")
		return
	}

	var overrides map[string]bool
	if string(bytesTrimSpace(rawOverrides)) != "null" {
		if err := json.Unmarshal(rawOverrides, &overrides); err != nil {
			app.ErrorJSON(w, http.StatusBadRequest, "overrides must be an object or null")
			return
		}
	}

	result, err := s.UpdateUserPermissions(r.Context(), claims.TenantID, r.PathValue("userId"), overrides)
	if err != nil {
		var reqErr *requestError
		if errors.As(err, &reqErr) {
			app.ErrorJSON(w, reqErr.status, reqErr.message)
			return
		}
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "User not found in this organization")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleUpdateUserRole(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	if err := s.requireManageUsersPermission(r.Context(), claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	var payload struct {
		Role string `json:"role"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.UpdateUserRole(r.Context(), claims.TenantID, r.PathValue("userId"), payload.Role, claims.UserID)
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

func (s Service) HandleRemoveUser(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	if err := s.requireManageUsersPermission(r.Context(), claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	result, err := s.RemoveUser(r.Context(), claims.TenantID, r.PathValue("userId"), claims.UserID)
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

func (s Service) HandleToggleUserEnabled(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	if err := s.requireManageUsersPermission(r.Context(), claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	var payload struct {
		Enabled bool `json:"enabled"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.ToggleUserEnabled(r.Context(), claims.TenantID, r.PathValue("userId"), payload.Enabled, claims.UserID)
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

func (s Service) HandleUpdateMembershipExpiry(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireOwnTenant(claims, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	if err := s.requireManageUsersPermission(r.Context(), claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}

	var payload struct {
		ExpiresAt *string `json:"expiresAt"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	var expiresAt *time.Time
	if payload.ExpiresAt != nil {
		parsed, err := time.Parse(time.RFC3339, *payload.ExpiresAt)
		if err != nil {
			app.ErrorJSON(w, http.StatusBadRequest, "expiresAt must be a valid ISO-8601 date-time")
			return
		}
		expiresAt = &parsed
	}

	result, err := s.UpdateMembershipExpiry(r.Context(), claims.TenantID, r.PathValue("userId"), expiresAt)
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
