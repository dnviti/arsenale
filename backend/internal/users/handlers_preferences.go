package users

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/jackc/pgx/v5"
)

func (s Service) HandleUpdateSSHDefaults(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	s.handleJSONPreferenceUpdate(w, r, claims.UserID, "sshDefaults")
}

func (s Service) HandleUpdateRDPDefaults(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	s.handleJSONPreferenceUpdate(w, r, claims.UserID, "rdpDefaults")
}

func (s Service) HandleUploadAvatar(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var payload struct {
		AvatarData string `json:"avatarData"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.UpdateAvatar(r.Context(), claims.UserID, payload.AvatarData)
	if err != nil {
		if strings.Contains(err.Error(), "invalid image format") || strings.Contains(err.Error(), "too large") {
			app.ErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}
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

func (s Service) HandleGetNotificationSchedule(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	result, err := s.GetNotificationSchedule(r.Context(), claims.UserID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleUpdateNotificationSchedule(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
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

	patch, err := parseNotificationSchedulePatch(payload)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.UpdateNotificationSchedule(r.Context(), claims.UserID, patch)
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

func (s Service) handleJSONPreferenceUpdate(w http.ResponseWriter, r *http.Request, userID, column string) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var payload map[string]any
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.UpdateJSONPreference(r.Context(), userID, column, payload)
	if err != nil {
		switch err {
		case pgx.ErrNoRows:
			app.ErrorJSON(w, http.StatusNotFound, "User not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}

	app.WriteJSON(w, http.StatusOK, map[string]any{
		"id":   result.ID,
		column: result.Preference,
	})
}
