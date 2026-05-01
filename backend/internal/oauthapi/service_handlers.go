package oauthapi

import (
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandleProviders(w http.ResponseWriter, _ *http.Request) {
	app.WriteJSON(w, http.StatusOK, availableProviders())
}

func (s Service) HandleInitiateProviderPathValue(w http.ResponseWriter, r *http.Request) {
	s.HandleInitiateProvider(w, r, r.PathValue("provider"))
}

func (s Service) HandleInitiateProvider(w http.ResponseWriter, r *http.Request, provider string) {
	target, err := s.buildAuthURL(r.Context(), provider, providerAuthOptions{})
	if err != nil {
		s.writeError(w, err)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (s Service) HandleGenerateLinkCode(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	code, err := s.GenerateLinkCode(r.Context(), claims.UserID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"code": code})
}

func (s Service) HandleExchangeCode(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Code string `json:"code"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(payload.Code) == "" {
		app.ErrorJSON(w, http.StatusBadRequest, "Missing authorization code")
		return
	}

	entry, err := s.ConsumeAuthCode(r.Context(), payload.Code)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, entry)
}

func (s Service) HandleSetupVault(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	var payload struct {
		VaultPassword string `json:"vaultPassword"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(payload.VaultPassword) == "" {
		app.ErrorJSON(w, http.StatusBadRequest, "vaultPassword is required")
		return
	}
	if err := validatePassword(payload.VaultPassword); err != nil {
		s.writeError(w, err)
		return
	}

	if err := s.SetupVaultForOAuthUser(r.Context(), claims.UserID, payload.VaultPassword); err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "vaultSetupComplete": true})
}

func (s Service) HandleAccounts(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	items, err := s.ListAccounts(r.Context(), claims.UserID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, items)
}

func (s Service) HandleUnlink(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.UnlinkAccount(r.Context(), claims.UserID, r.PathValue("provider")); err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"success": true})
}
