package ldapapi

import (
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandleGetStatus(w http.ResponseWriter, _ *http.Request, claims authn.Claims) {
	if err := requireTenantAdmin(claims); err != nil {
		app.ErrorJSON(w, http.StatusForbidden, err.Error())
		return
	}

	cfg := loadConfig()
	app.WriteJSON(w, http.StatusOK, statusResponse{
		Enabled:       cfg.isEnabled(),
		ProviderName:  cfg.ProviderName,
		ServerURL:     redactLDAPURL(cfg.ServerURL),
		BaseDN:        cfg.BaseDN,
		SyncEnabled:   cfg.SyncEnabled,
		SyncCron:      cfg.SyncCron,
		AutoProvision: cfg.AutoProvision,
	})
}

func (s Service) HandleTestConnection(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireTenantAdmin(claims); err != nil {
		app.ErrorJSON(w, http.StatusForbidden, err.Error())
		return
	}

	result := s.testConnection(r.Context())
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleTriggerSync(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireTenantAdmin(claims); err != nil {
		app.ErrorJSON(w, http.StatusForbidden, err.Error())
		return
	}

	result := s.syncUsers(r.Context())
	app.WriteJSON(w, http.StatusOK, result)
}
