package adminapi

import (
	"net/http"
	"net/mail"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/emaildelivery"
)

func (s Service) HandleGetEmailStatus(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.requireTenantAdmin(r.Context(), claims); err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, buildEmailStatus())
}

func (s Service) HandleSendTestEmail(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.requireTenantAdmin(r.Context(), claims); err != nil {
		s.writeError(w, err)
		return
	}

	var payload struct {
		To string `json:"to"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(payload.To)); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, "invalid recipient email")
		return
	}

	status := buildEmailStatus()
	if err := emaildelivery.Send(r.Context(), emaildelivery.Message{
		To:      strings.TrimSpace(payload.To),
		Subject: "Arsenale test email",
		HTML:    "<p>This is a test email from Arsenale.</p>",
		Text:    "This is a test email from Arsenale.",
	}); err != nil {
		s.writeError(w, err)
		return
	}

	if err := s.insertStandaloneAuditLog(r.Context(), claims.UserID, "EMAIL_TEST_SEND", map[string]any{
		"to":       strings.TrimSpace(payload.To),
		"provider": status.Provider,
	}); err != nil {
		s.writeError(w, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Test email sent successfully",
	})
}

func (s Service) HandleGetAppConfig(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.requireTenantAdmin(r.Context(), claims); err != nil {
		s.writeError(w, err)
		return
	}

	selfSignupEnabled, err := s.getSelfSignupEnabled(r.Context())
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	app.WriteJSON(w, http.StatusOK, appConfigResponse{
		SelfSignupEnabled:   selfSignupEnabled,
		SelfSignupEnvLocked: selfSignupEnvLocked(),
	})
}

func (s Service) HandleSetSelfSignup(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.requireTenantAdmin(r.Context(), claims); err != nil {
		s.writeError(w, err)
		return
	}

	var payload struct {
		Enabled bool `json:"enabled"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.setSelfSignupEnabled(r.Context(), payload.Enabled, claims.UserID); err != nil {
		s.writeError(w, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, appConfigResponse{
		SelfSignupEnabled:   payload.Enabled,
		SelfSignupEnvLocked: selfSignupEnvLocked(),
	})
}

func (s Service) HandleGetAuthProviders(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.requireTenantAdmin(r.Context(), claims); err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, buildAuthProviderDetails())
}

func (s Service) HandleGetSystemSettingsDBStatus(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.requireTenantAdmin(r.Context(), claims); err != nil {
		s.writeError(w, err)
		return
	}

	status, err := s.getDBStatus(r.Context())
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, status)
}
