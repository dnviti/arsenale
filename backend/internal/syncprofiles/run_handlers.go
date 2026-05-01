package syncprofiles

import (
	"errors"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/jackc/pgx/v5"
)

func (s Service) HandleTestConnection(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireTenantAdmin(claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return
	}
	result, err := s.TestConnection(r.Context(), r.PathValue("id"), claims.TenantID)
	if err != nil {
		var reqErr *requestError
		switch {
		case errors.As(err, &reqErr):
			app.ErrorJSON(w, reqErr.status, reqErr.message)
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "Sync profile not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleTriggerSync(w http.ResponseWriter, r *http.Request, claims authn.Claims) error {
	if err := requireTenantAdmin(claims); err != nil {
		app.ErrorJSON(w, err.status, err.message)
		return nil
	}
	var payload struct {
		DryRun bool `json:"dryRun"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return nil
	}

	result, err := s.TriggerSync(r.Context(), claims.UserID, claims.TenantID, r.PathValue("id"), payload.DryRun)
	if err != nil {
		var reqErr *requestError
		switch {
		case errors.As(err, &reqErr):
			app.ErrorJSON(w, reqErr.status, reqErr.message)
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "Sync profile not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return nil
	}
	app.WriteJSON(w, http.StatusOK, result)
	return nil
}
