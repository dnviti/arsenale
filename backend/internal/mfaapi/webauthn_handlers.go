package mfaapi

import (
	"encoding/json"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/webauthnflow"
)

func (s Service) HandleWebAuthnRegistrationOptions(w http.ResponseWriter, r *http.Request, userID string) {
	result, err := s.GenerateWebAuthnRegistrationOptions(r.Context(), userID)
	if err != nil {
		s.writeError(w, err)
		return
	}
	webauthnflow.SetChallengeCookie(w, r, webauthnflow.RegistrationChallengeCookieName, result.Challenge)
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleRegisterWebAuthn(w http.ResponseWriter, r *http.Request, userID string) {
	var payload struct {
		Credential        json.RawMessage `json:"credential"`
		FriendlyName      string          `json:"friendlyName"`
		ExpectedChallenge string          `json:"expectedChallenge"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.RegisterWebAuthnCredential(
		r.Context(),
		userID,
		payload.Credential,
		payload.FriendlyName,
		payload.ExpectedChallenge,
		requestIP(r),
	)
	if err != nil {
		s.writeError(w, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}
