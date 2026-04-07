package notifications

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandleList(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	limit := 50
	offset := 0
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if value := strings.TrimSpace(r.URL.Query().Get("offset")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	result, err := s.ListNotifications(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleMarkRead(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.MarkRead(r.Context(), claims.UserID, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s Service) HandleMarkAllRead(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.MarkAllRead(r.Context(), claims.UserID); err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s Service) HandleDelete(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.DeleteNotification(r.Context(), claims.UserID, r.PathValue("id")); err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s Service) HandleGetPreferences(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.GetPreferences(r.Context(), claims.UserID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleUpdatePreference(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	var payload preferenceUpdatePayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.UpsertPreference(r.Context(), claims.UserID, r.PathValue("type"), payload.InApp, payload.Email)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleBulkUpdatePreferences(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	var payload bulkPreferencesPayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.BulkUpsertPreferences(r.Context(), claims.UserID, payload.Preferences)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}
