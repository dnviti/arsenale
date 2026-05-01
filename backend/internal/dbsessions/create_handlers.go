package dbsessions

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/sshsessions"
)

func (s Service) HandleCreate(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	var payload createRequest
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.createSession(r.Context(), claims, payload, requestIP(r))
	if err != nil {
		s.recordSessionError(r.Context(), claims.UserID, strings.TrimSpace(payload.ConnectionID), requestIP(r), err)

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

		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}
