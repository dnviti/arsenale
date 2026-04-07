package dbsessions

import (
	"context"
	"errors"
	"net/http"

	"github.com/dnviti/arsenale/backend/internal/app"
)

func (s Service) ShouldHandleOwnedQueryRuntime(ctx context.Context, userID, tenantID, sessionID string) (bool, error) {
	_, err := s.resolveOwnedQueryRuntime(ctx, userID, tenantID, sessionID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrQueryRuntimeUnsupported) {
		return false, nil
	}
	return false, err
}

func (s Service) HandleOwnedQuery(w http.ResponseWriter, r *http.Request, userID, tenantID, tenantRole, ipAddress string) {
	var payload ownedQueryRequest
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.executeOwnedQuery(r.Context(), userID, tenantID, tenantRole, r.PathValue("sessionId"), payload.SQL, ipAddress)
	if err != nil {
		writeOwnedQueryError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleOwnedSchema(w http.ResponseWriter, r *http.Request, userID, tenantID string) {
	result, err := s.fetchOwnedSchema(r.Context(), userID, tenantID, r.PathValue("sessionId"))
	if err != nil {
		writeOwnedQueryError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleOwnedExplain(w http.ResponseWriter, r *http.Request, userID, tenantID, tenantRole, ipAddress string) {
	var payload ownedQueryRequest
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.explainOwnedQuery(r.Context(), userID, tenantID, tenantRole, r.PathValue("sessionId"), payload.SQL, ipAddress)
	if err != nil {
		writeOwnedQueryError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}

func (s Service) HandleOwnedIntrospect(w http.ResponseWriter, r *http.Request, userID, tenantID string) {
	var payload ownedIntrospectionRequest
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := s.introspectOwnedQuery(r.Context(), userID, tenantID, r.PathValue("sessionId"), payload.Type, payload.Target)
	if err != nil {
		writeOwnedQueryError(w, err)
		return
	}
	app.WriteJSON(w, http.StatusOK, result)
}
