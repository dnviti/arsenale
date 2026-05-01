package importexportapi

import (
	"fmt"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) HandleExport(w http.ResponseWriter, r *http.Request, claims authn.Claims) error {
	var payload exportPayload
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return nil
	}

	filename, contentType, body, err := s.ExportConnections(r.Context(), claims, payload, requestIP(r))
	if err != nil {
		writeError(w, err)
		return nil
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("write export response: %w", err)
	}
	return nil
}

func (s Service) HandleImport(w http.ResponseWriter, r *http.Request, claims authn.Claims) error {
	result, err := s.ImportConnections(r.Context(), r, claims)
	if err != nil {
		writeError(w, err)
		return nil
	}
	app.WriteJSON(w, http.StatusOK, result)
	return nil
}
