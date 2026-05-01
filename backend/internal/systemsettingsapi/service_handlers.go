package systemsettingsapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandleList(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	membership, err := s.requireReader(r.Context(), claims)
	if err != nil {
		s.writeError(w, err)
		return
	}

	settings, err := s.listSettings(r.Context(), membership.Role)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	app.WriteJSON(w, http.StatusOK, map[string]any{
		"settings": settings,
		"groups":   loadedGroups(),
	})
}

func (s Service) HandleUpdateSingle(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	membership, err := s.requireWriter(r.Context(), claims)
	if err != nil {
		s.writeError(w, err)
		return
	}

	var payload struct {
		Value any `json:"value"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.setSetting(r.Context(), strings.TrimSpace(r.PathValue("key")), payload.Value, claims.UserID, membership.Role)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleBulkUpdate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	membership, err := s.requireWriter(r.Context(), claims)
	if err != nil {
		s.writeError(w, err)
		return
	}

	var payload struct {
		Updates []struct {
			Key   string `json:"key"`
			Value any    `json:"value"`
		} `json:"updates"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(payload.Updates) == 0 || len(payload.Updates) > 100 {
		app.ErrorJSON(w, http.StatusBadRequest, "updates must contain between 1 and 100 items")
		return
	}

	results := make([]updateResult, 0, len(payload.Updates))
	for _, update := range payload.Updates {
		key := strings.TrimSpace(update.Key)
		if key == "" {
			results = append(results, updateResult{Key: key, Success: false, Error: "Unknown setting key."})
			continue
		}
		if _, err := s.setSetting(r.Context(), key, update.Value, claims.UserID, membership.Role); err != nil {
			var reqErr *requestError
			if errors.As(err, &reqErr) {
				results = append(results, updateResult{Key: key, Success: false, Error: reqErr.message})
				continue
			}
			results = append(results, updateResult{Key: key, Success: false, Error: "Unknown error"})
			continue
		}
		results = append(results, updateResult{Key: key, Success: true})
	}

	app.WriteJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s Service) writeError(w http.ResponseWriter, err error) {
	var reqErr *requestError
	if errors.As(err, &reqErr) {
		app.ErrorJSON(w, reqErr.status, reqErr.message)
		return
	}
	app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
}
