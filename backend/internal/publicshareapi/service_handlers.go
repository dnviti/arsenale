package publicshareapi

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
)

func (s Service) HandleGetInfo(w http.ResponseWriter, r *http.Request) {
	info, err := s.GetInfo(r.Context(), strings.TrimSpace(r.PathValue("token")))
	if err != nil {
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	app.WriteJSON(w, http.StatusOK, info)
}

func (s Service) HandleAccess(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Pin string `json:"pin"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := app.ReadJSON(r, &payload); err != nil && !errors.Is(err, io.EOF) {
			app.ErrorJSON(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	payload.Pin = strings.TrimSpace(payload.Pin)
	if payload.Pin != "" && !pinPattern.MatchString(payload.Pin) {
		app.ErrorJSON(w, http.StatusBadRequest, "PIN must be 4-8 digits")
		return
	}

	result, err := s.Access(r.Context(), strings.TrimSpace(r.PathValue("token")), payload.Pin, clientIP(r))
	if err != nil {
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
