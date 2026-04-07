package dbsessions

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
)

func (s Service) HandleIssue(w http.ResponseWriter, r *http.Request) {
	var req SessionIssueRequest
	if err := app.ReadJSON(r, &req); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.issueSession(r.Context(), req, true)
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

func (s Service) HandleHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req OwnedSessionRequest
	if err := app.ReadJSON(r, &req); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		app.ErrorJSON(w, http.StatusBadRequest, "userId is required")
		return
	}

	if err := s.Store.HeartbeatOwnedSession(r.Context(), r.PathValue("sessionId"), req.UserID); err != nil {
		writeLifecycleError(w, err, true)
		return
	}

	app.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s Service) HandleOwnedHeartbeat(w http.ResponseWriter, r *http.Request, userID string) {
	if strings.TrimSpace(userID) == "" {
		app.ErrorJSON(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}
	if err := s.Store.HeartbeatOwnedSession(r.Context(), r.PathValue("sessionId"), userID); err != nil {
		writeLifecycleError(w, err, true)
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s Service) HandleEnd(w http.ResponseWriter, r *http.Request) {
	var req OwnedSessionRequest
	if err := app.ReadJSON(r, &req); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		app.ErrorJSON(w, http.StatusBadRequest, "userId is required")
		return
	}

	if err := s.Store.EndOwnedSession(r.Context(), r.PathValue("sessionId"), req.UserID, strings.TrimSpace(req.Reason)); err != nil {
		writeLifecycleError(w, err, false)
		return
	}

	app.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s Service) HandleOwnedEnd(w http.ResponseWriter, r *http.Request, userID, reason string) {
	if strings.TrimSpace(userID) == "" {
		app.ErrorJSON(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}
	if strings.TrimSpace(reason) == "" {
		reason = "client_disconnect"
	}
	if err := s.Store.EndOwnedSession(r.Context(), r.PathValue("sessionId"), userID, strings.TrimSpace(reason)); err != nil {
		writeLifecycleError(w, err, false)
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s Service) HandleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	var req SessionConfigRequest
	if err := app.ReadJSON(r, &req); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.UserID) == "" {
		app.ErrorJSON(w, http.StatusBadRequest, "userId is required")
		return
	}

	s.applyOwnedSessionConfig(w, r, req.UserID, req.SessionConfig, req.Target)
}

func (s Service) HandleOwnedConfigUpdate(w http.ResponseWriter, r *http.Request, userID string) {
	if strings.TrimSpace(userID) == "" {
		app.ErrorJSON(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	var payload ownedSessionConfigPayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	s.applyOwnedSessionConfig(w, r, userID, payload.SessionConfig, payload.Target)
}

func (s Service) HandleConfigGet(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.URL.Query().Get("userId"))
	if userID == "" {
		app.ErrorJSON(w, http.StatusBadRequest, "userId is required")
		return
	}

	s.writeOwnedConfig(w, r, userID)
}

func (s Service) HandleOwnedConfigGet(w http.ResponseWriter, r *http.Request, userID string) {
	if strings.TrimSpace(userID) == "" {
		app.ErrorJSON(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}
	s.writeOwnedConfig(w, r, userID)
}

func (s Service) HandleHistory(w http.ResponseWriter, r *http.Request, userID string) {
	if strings.TrimSpace(userID) == "" {
		app.ErrorJSON(w, http.StatusUnauthorized, "Invalid or expired token")
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			app.ErrorJSON(w, http.StatusBadRequest, "limit must be between 1 and 200")
			return
		}
		limit = parsed
	}

	items, err := s.GetQueryHistory(r.Context(), userID, r.PathValue("sessionId"), limit, strings.TrimSpace(r.URL.Query().Get("search")))
	if err != nil {
		writeLifecycleError(w, err, false)
		return
	}

	app.WriteJSON(w, http.StatusOK, items)
}
