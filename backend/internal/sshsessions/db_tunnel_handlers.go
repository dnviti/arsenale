package sshsessions

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/sessions"
)

func (s Service) HandleCreateDBTunnel(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	var payload dbTunnelRequest
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.openDBTunnel(r.Context(), claims, payload, requestIP(r))
	if err != nil {
		_ = s.insertAuditLog(r.Context(), claims.UserID, "DB_TUNNEL_ERROR", "Connection", strings.TrimSpace(payload.ConnectionID), requestIP(r), map[string]any{
			"protocol": "DB_TUNNEL",
			"error":    err.Error(),
		})
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

func (s Service) HandleListDBTunnels(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	app.WriteJSON(w, http.StatusOK, activeDBTunnels.listForUser(claims.UserID))
}

func (s Service) HandleDBTunnelHeartbeat(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	tunnelID := strings.TrimSpace(r.PathValue("tunnelId"))
	tunnel, ok := activeDBTunnels.get(tunnelID)
	if !ok || tunnel.UserID != claims.UserID {
		app.ErrorJSON(w, http.StatusNotFound, "Tunnel not found")
		return
	}

	tunnel.touch()
	if tunnel.SessionID != "" && s.SessionStore != nil {
		if err := s.SessionStore.HeartbeatOwnedSession(r.Context(), tunnel.SessionID, claims.UserID); err != nil &&
			!errors.Is(err, sessions.ErrSessionClosed) &&
			!errors.Is(err, sessions.ErrSessionNotFound) {
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
			return
		}
	}

	app.WriteJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"healthy": tunnel.snapshot().Healthy,
	})
}

func (s Service) HandleCloseDBTunnel(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	tunnelID := strings.TrimSpace(r.PathValue("tunnelId"))
	tunnel, ok := activeDBTunnels.closeOwned(tunnelID, claims.UserID)
	if !ok {
		app.ErrorJSON(w, http.StatusNotFound, "Tunnel not found")
		return
	}

	tunnel.close()
	if tunnel.SessionID != "" && s.SessionStore != nil {
		if err := s.SessionStore.EndOwnedSession(r.Context(), tunnel.SessionID, claims.UserID, "client_disconnect"); err != nil &&
			!errors.Is(err, sessions.ErrSessionNotFound) {
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
			return
		}
	}
	snapshot := tunnel.snapshot()
	_ = s.insertAuditLog(r.Context(), claims.UserID, "DB_TUNNEL_CLOSE", "Connection", tunnel.ConnectionID, requestIP(r), map[string]any{
		"tunnelId":     tunnel.ID,
		"durationMs":   time.Since(tunnel.CreatedAt).Milliseconds(),
		"localPort":    tunnel.LocalPort,
		"targetDbHost": tunnel.TargetDBHost,
		"targetDbPort": tunnel.TargetDBPort,
		"connectionId": tunnel.ConnectionID,
		"sessionId":    tunnel.SessionID,
		"healthy":      snapshot.Healthy,
		"lastError":    valueOrEmpty(snapshot.LastError),
	})

	app.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}
