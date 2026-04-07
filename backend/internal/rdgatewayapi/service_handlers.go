package rdgatewayapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/jackc/pgx/v5"
)

func (s Service) HandleGetConfig(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageRDGW(claims) {
		app.ErrorJSON(w, http.StatusForbidden, "forbidden")
		return
	}

	config, err := s.GetConfig(r.Context())
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, config)
}

func (s Service) HandleUpdateConfig(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanManageRDGW(claims) {
		app.ErrorJSON(w, http.StatusForbidden, "forbidden")
		return
	}

	var payload updateConfigRequest
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	current, err := s.GetConfig(r.Context())
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	updated := current
	if payload.Enabled != nil {
		updated.Enabled = *payload.Enabled
	}
	if payload.ExternalHostname != nil {
		updated.ExternalHostname = strings.TrimSpace(*payload.ExternalHostname)
	}
	if payload.Port != nil {
		updated.Port = *payload.Port
	}
	if payload.IdleTimeoutSeconds != nil {
		updated.IdleTimeoutSeconds = *payload.IdleTimeoutSeconds
	}

	if updated.ExternalHostname != "" && !hostnamePattern.MatchString(updated.ExternalHostname) {
		app.ErrorJSON(w, http.StatusBadRequest, "Invalid external hostname format")
		return
	}
	if updated.Port < 1 || updated.Port > 65535 {
		app.ErrorJSON(w, http.StatusBadRequest, "Port must be between 1 and 65535")
		return
	}
	if updated.IdleTimeoutSeconds < 0 {
		app.ErrorJSON(w, http.StatusBadRequest, "Idle timeout must be zero or greater")
		return
	}

	if err := s.UpsertConfig(r.Context(), updated); err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	_ = s.insertAuditLog(r.Context(), claims.UserID, "APP_CONFIG_UPDATE", "AppConfig", "rdGatewayConfig", map[string]any{
		"previous": current,
		"updated":  updated,
	}, requestIP(r))

	app.WriteJSON(w, http.StatusOK, updated)
}

func (s Service) HandleStatus(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if !claimsCanViewRDGWStatus(claims) {
		app.ErrorJSON(w, http.StatusForbidden, "forbidden")
		return
	}

	status, err := s.GetStatus(r.Context(), claims.TenantID)
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, status)
}

func (s Service) HandleRDPFile(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	connectionID := strings.TrimSpace(r.PathValue("connectionId"))
	if connectionID == "" {
		app.ErrorJSON(w, http.StatusBadRequest, "connectionId is required")
		return
	}

	conn, err := s.Connections.GetConnection(r.Context(), claims.UserID, claims.TenantID, connectionID)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "Connection not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}
	if !strings.EqualFold(conn.Type, "RDP") {
		app.ErrorJSON(w, http.StatusBadRequest, "RDP file generation is only available for RDP connections")
		return
	}

	config, err := s.GetConfig(r.Context())
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	if !config.Enabled {
		app.ErrorJSON(w, http.StatusBadRequest, "RD Gateway is not enabled")
		return
	}
	if strings.TrimSpace(config.ExternalHostname) == "" {
		app.ErrorJSON(w, http.StatusBadRequest, "RD Gateway external hostname is not configured")
		return
	}

	content := generateRDPFile(rdpFileParams{
		ConnectionName:  conn.Name,
		TargetHost:      conn.Host,
		TargetPort:      conn.Port,
		GatewayHostname: config.ExternalHostname,
		GatewayPort:     config.Port,
		ScreenMode:      2,
		DesktopWidth:    1920,
		DesktopHeight:   1080,
	})

	_ = s.insertAuditLog(r.Context(), claims.UserID, "SESSION_START", "Connection", connectionID, map[string]any{
		"protocol":       "RDGW",
		"operation":      "generateRdpFile",
		"connectionName": conn.Name,
		"targetHost":     conn.Host,
		"targetPort":     conn.Port,
	}, requestIP(r))

	safeFilename := sanitizeFilename(conn.Name)
	w.Header().Set("Content-Type", "application/x-rdp")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.rdp"`, safeFilename))
	_, _ = w.Write([]byte(content))
}
