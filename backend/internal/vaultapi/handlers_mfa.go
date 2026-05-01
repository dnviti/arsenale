package vaultapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/webauthnflow"
)

func (s Service) HandleUnlockWithTOTP(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	var payload codePayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	code, err := ParseTOTPCode(payload.Code)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.UnlockWithTOTP(r.Context(), claims.UserID, code, requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleRequestWebAuthnOptions(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	result, err := s.RequestWebAuthnOptions(r.Context(), claims.UserID)
	if err != nil {
		s.writeError(w, err)
		return
	}

	webauthnflow.SetChallengeCookie(w, r, webauthnflow.AuthChallengeCookieName, result.Challenge)
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleUnlockWithWebAuthn(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	var payload struct {
		Credential        json.RawMessage `json:"credential"`
		ExpectedChallenge string          `json:"expectedChallenge"`
	}
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.UnlockWithWebAuthn(
		r.Context(),
		claims.UserID,
		payload.Credential,
		strings.TrimSpace(payload.ExpectedChallenge),
		requestIP(r),
	)
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleRequestSMSCode(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := s.RequestSMSCode(r.Context(), claims.UserID); err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, map[string]any{"sent": true})
}

func (s Service) HandleUnlockWithSMS(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	var payload codePayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.UnlockWithSMS(r.Context(), claims.UserID, strings.TrimSpace(payload.Code), requestIP(r))
	if err != nil {
		s.writeError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}
