package dbsessions

import (
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func (s Service) applyOwnedSessionConfig(w http.ResponseWriter, r *http.Request, userID string, sessionConfig *contracts.DatabaseSessionConfig, target *contracts.DatabaseTarget) {
	state, err := s.Store.LoadOwnedSessionState(r.Context(), r.PathValue("sessionId"), userID)
	if err != nil {
		writeLifecycleError(w, err, false)
		return
	}
	if state.Record.Status == "CLOSED" {
		app.ErrorJSON(w, http.StatusGone, "session already closed")
		return
	}

	if target != nil {
		gatewayID := ""
		if state.Record.GatewayID != nil {
			gatewayID = strings.TrimSpace(*state.Record.GatewayID)
		}
		instanceID := ""
		if state.Record.InstanceID != nil {
			instanceID = strings.TrimSpace(*state.Record.InstanceID)
		}
		if err := s.validateTargetViaDBProxy(r.Context(), gatewayID, instanceID, target); err != nil {
			app.ErrorJSON(w, classifyConnectivityStatus(err), err.Error())
			return
		}
	}

	metadata := normalizeMetadata(state.Metadata)
	if sessionConfig == nil || isEmptySessionConfig(*sessionConfig) {
		delete(metadata, "sessionConfig")
	} else {
		metadata["sessionConfig"] = normalizeSessionConfig(*sessionConfig)
	}

	if err := s.Store.UpdateOwnedSessionMetadata(r.Context(), state.Record.ID, userID, metadata); err != nil {
		writeLifecycleError(w, err, false)
		return
	}

	activeDatabase := ""
	if target != nil {
		activeDatabase = strings.TrimSpace(target.Database)
	}
	if activeDatabase == "" && sessionConfig != nil {
		activeDatabase = strings.TrimSpace(sessionConfig.ActiveDatabase)
	}
	if activeDatabase == "" {
		activeDatabase = stringValue(metadata["databaseName"])
	}

	app.WriteJSON(w, http.StatusOK, map[string]any{
		"applied":        true,
		"activeDatabase": activeDatabase,
		"sessionConfig":  metadata["sessionConfig"],
	})
}

func (s Service) writeOwnedConfig(w http.ResponseWriter, r *http.Request, userID string) {
	state, err := s.Store.LoadOwnedSessionState(r.Context(), r.PathValue("sessionId"), userID)
	if err != nil {
		writeLifecycleError(w, err, false)
		return
	}

	sessionConfig := map[string]any{}
	if raw, ok := state.Metadata["sessionConfig"]; ok {
		if normalized, ok := raw.(map[string]any); ok {
			sessionConfig = normalized
		}
	}

	app.WriteJSON(w, http.StatusOK, sessionConfig)
}
