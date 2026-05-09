package desktopsessions

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/sessions"
	"github.com/dnviti/arsenale/backend/internal/sshsessions"
	"github.com/jackc/pgx/v5"
)

func (s Service) HandleCreateDesktopLaunch(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	var payload desktopLaunchRequest
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.CreateDesktopLaunch(r, claims, payload)
	if err != nil {
		s.writeDesktopLaunchError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleRedeemDesktopLaunch(w http.ResponseWriter, r *http.Request) {
	var payload desktopLaunchRedeemRequest
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.RedeemDesktopLaunch(r, strings.TrimSpace(payload.Grant))
	if err != nil {
		s.writeDesktopLaunchError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleDesktopViewerHeartbeat(w http.ResponseWriter, r *http.Request) {
	if s.Store == nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, "session store is unavailable")
		return
	}
	control, err := s.authorizeViewerControlRequest(r)
	if err != nil {
		s.writeDesktopLaunchError(w, err)
		return
	}
	if err := s.Store.HeartbeatOwnedSession(r.Context(), r.PathValue("sessionId"), control.UserID); err != nil {
		s.writeLifecycleError(w, err, true)
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s Service) HandleDesktopViewerEnd(w http.ResponseWriter, r *http.Request) {
	if s.Store == nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, "session store is unavailable")
		return
	}
	control, err := s.authorizeViewerControlRequest(r)
	if err != nil {
		s.writeDesktopLaunchError(w, err)
		return
	}
	if err := s.Store.EndOwnedSession(r.Context(), r.PathValue("sessionId"), control.UserID, "cli_viewer_disconnect"); err != nil {
		s.writeLifecycleError(w, err, false)
		return
	}
	_ = s.revokeViewerControlToken(r.Context(), control.ID)
	app.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s Service) authorizeViewerControlRequest(r *http.Request) (desktopViewerControlRecord, error) {
	var payload desktopViewerControlRequest
	if err := app.ReadJSON(r, &payload); err != nil {
		return desktopViewerControlRecord{}, &requestError{status: http.StatusBadRequest, message: err.Error()}
	}
	return s.AuthorizeViewerControl(r.Context(), r.PathValue("sessionId"), strings.TrimSpace(payload.ControlToken))
}

func (s Service) writeDesktopLaunchError(w http.ResponseWriter, err error) {
	var reqErr *requestError
	if errors.As(err, &reqErr) {
		app.ErrorJSON(w, reqErr.status, reqErr.message)
		return
	}

	var resolveErr *sshsessions.ResolveError
	if errors.As(err, &resolveErr) {
		app.ErrorJSON(w, resolveErr.Status, resolveErr.Message)
		return
	}

	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sessions.ErrSessionNotFound) {
		app.ErrorJSON(w, http.StatusNotFound, "not found")
		return
	}

	app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
}
