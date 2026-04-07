package users

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/jackc/pgx/v5"
)

func (s Service) HandleChangePassword(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	oldPassword, newPassword, verificationID, err := parsePasswordChangePayload(r)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.ChangePassword(r.Context(), claims.UserID, oldPassword, newPassword, verificationID, requestIP(r))
	if err != nil {
		var reqErr *requestError
		switch {
		case errors.As(err, &reqErr):
			app.ErrorJSON(w, reqErr.status, reqErr.message)
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "User not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleInitiatePasswordChange(w http.ResponseWriter, r *http.Request, claims authn.Claims) error {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return nil
	}

	result, err := s.InitiatePasswordChange(r.Context(), claims.UserID)
	if err != nil {
		var reqErr *requestError
		switch {
		case errors.As(err, &reqErr):
			app.ErrorJSON(w, reqErr.status, reqErr.message)
		case errors.Is(err, errNoVerificationMethod):
			app.ErrorJSON(w, http.StatusBadRequest, errNoVerificationMethod.Error())
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "User not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return nil
	}

	app.WriteJSON(w, http.StatusOK, result)
	return nil
}

func (s Service) HandleInitiateIdentity(w http.ResponseWriter, r *http.Request, claims authn.Claims) error {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return nil
	}

	var payload map[string]json.RawMessage
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return nil
	}

	result, err := s.InitiateIdentity(r.Context(), claims.UserID, payload)
	if err != nil {
		var reqErr *requestError
		switch {
		case errors.As(err, &reqErr):
			app.ErrorJSON(w, reqErr.status, reqErr.message)
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "User not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return nil
	}

	app.WriteJSON(w, http.StatusOK, result)
	return nil
}

func (s Service) HandleConfirmIdentity(w http.ResponseWriter, r *http.Request, claims authn.Claims) error {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return nil
	}

	var payload map[string]json.RawMessage
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return nil
	}

	confirmed, err := s.ConfirmIdentity(r.Context(), claims.UserID, payload)
	if err != nil {
		var reqErr *requestError
		switch {
		case errors.As(err, &reqErr):
			app.ErrorJSON(w, reqErr.status, reqErr.message)
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return nil
	}

	app.WriteJSON(w, http.StatusOK, map[string]any{"confirmed": confirmed})
	return nil
}

func (s Service) HandleInitiateEmailChange(w http.ResponseWriter, r *http.Request, claims authn.Claims) error {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return nil
	}

	var payload map[string]json.RawMessage
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return nil
	}

	result, err := s.InitiateEmailChange(r.Context(), claims.UserID, payload)
	if err != nil {
		var reqErr *requestError
		switch {
		case errors.As(err, &reqErr):
			app.ErrorJSON(w, reqErr.status, reqErr.message)
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "User not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return nil
	}

	app.WriteJSON(w, http.StatusOK, result)
	return nil
}

func (s Service) HandleConfirmEmailChange(w http.ResponseWriter, r *http.Request, claims authn.Claims) error {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		app.ErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return nil
	}

	var payload map[string]json.RawMessage
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return nil
	}

	result, err := s.ConfirmEmailChange(r.Context(), claims.UserID, payload, requestIP(r))
	if err != nil {
		var reqErr *requestError
		switch {
		case errors.As(err, &reqErr):
			app.ErrorJSON(w, reqErr.status, reqErr.message)
		case errors.Is(err, pgx.ErrNoRows):
			app.ErrorJSON(w, http.StatusNotFound, "User not found")
		default:
			app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		}
		return nil
	}

	app.WriteJSON(w, http.StatusOK, result)
	return nil
}
