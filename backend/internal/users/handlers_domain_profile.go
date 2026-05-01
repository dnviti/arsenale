package users

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/jackc/pgx/v5"
)

func (s Service) HandleGetDomainProfile(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	result, err := s.GetDomainProfile(r.Context(), claims.UserID)
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

func (s Service) HandleUpdateDomainProfile(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var payload map[string]json.RawMessage
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	patch, fields, err := parseDomainProfilePatch(payload)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.UpdateDomainProfile(r.Context(), claims.UserID, patch, fields, requestIP(r))
	if err != nil {
		switch {
		case errors.Is(err, errVaultLocked):
			app.ErrorJSON(w, http.StatusForbidden, errVaultLocked.Error())
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "User not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleClearDomainProfile(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if r.Method != http.MethodDelete {
		w.Header().Set("Allow", http.MethodDelete)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := s.ClearDomainProfile(r.Context(), claims.UserID, requestIP(r)); err != nil {
		switch err {
		case pgx.ErrNoRows:
			app.ErrorJSON(w, http.StatusNotFound, "User not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}

	app.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}
