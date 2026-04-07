package desktopsessions

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/sshsessions"
	"github.com/jackc/pgx/v5"
)

func (s Service) handleCreateDesktopSession(w http.ResponseWriter, r *http.Request, claims authn.Claims, protocol string) {
	var payload createRequest
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	protocol = strings.ToUpper(strings.TrimSpace(protocol))
	ipAddress := requestIP(r)
	errorCtx := sessionErrorContext{
		ConnectionID: strings.TrimSpace(payload.ConnectionID),
	}

	result, err := s.createDesktopSession(r.Context(), claims, payload, protocol, ipAddress, &errorCtx)
	if err != nil {
		s.recordSessionError(r.Context(), claims.UserID, protocol, errorCtx, ipAddress, err)

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

		if errors.Is(err, pgx.ErrNoRows) {
			app.ErrorJSON(w, http.StatusNotFound, "Connection not found")
			return
		}

		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	app.WriteJSON(w, http.StatusOK, result)
}
