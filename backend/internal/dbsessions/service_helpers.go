package dbsessions

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/sessions"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func validateSessionIssueRequest(req SessionIssueRequest) error {
	if strings.TrimSpace(req.UserID) == "" {
		return errors.New("userId is required")
	}
	if strings.TrimSpace(req.ConnectionID) == "" {
		return errors.New("connectionId is required")
	}
	protocol := strings.ToUpper(strings.TrimSpace(req.Protocol))
	if protocol != "DATABASE" {
		return fmt.Errorf("unsupported protocol %q", req.Protocol)
	}
	if strings.TrimSpace(req.ProxyHost) == "" {
		return errors.New("proxyHost is required")
	}
	if req.ProxyPort <= 0 || req.ProxyPort > 65535 {
		return errors.New("proxyPort must be between 1 and 65535")
	}
	if req.Target == nil {
		return errors.New("target is required")
	}
	return nil
}

func classifyConnectivityStatus(err error) int {
	lowered := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lowered, "authentication"), strings.Contains(lowered, "password"):
		return http.StatusUnauthorized
	case strings.Contains(lowered, "timeout"), strings.Contains(lowered, "timed out"):
		return http.StatusGatewayTimeout
	default:
		return http.StatusBadGateway
	}
}

func writeLifecycleError(w http.ResponseWriter, err error, heartbeat bool) {
	switch {
	case errors.Is(err, sessions.ErrSessionNotFound):
		app.ErrorJSON(w, http.StatusNotFound, "session not found")
	case heartbeat && errors.Is(err, sessions.ErrSessionClosed):
		app.ErrorJSON(w, http.StatusGone, "session already closed")
	default:
		app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
	}
}

func normalizeMetadata(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}

	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = normalizeValue(value)
	}
	return out
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return normalizeMetadata(typed)
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, normalizeValue(item))
		}
		return items
	case nil:
		return nil
	default:
		return typed
	}
}

func normalizeSessionConfig(config contracts.DatabaseSessionConfig) map[string]any {
	result := map[string]any{}
	if strings.TrimSpace(config.ActiveDatabase) != "" {
		result["activeDatabase"] = strings.TrimSpace(config.ActiveDatabase)
	}
	if strings.TrimSpace(config.Timezone) != "" {
		result["timezone"] = strings.TrimSpace(config.Timezone)
	}
	if strings.TrimSpace(config.SearchPath) != "" {
		result["searchPath"] = strings.TrimSpace(config.SearchPath)
	}
	if strings.TrimSpace(config.Encoding) != "" {
		result["encoding"] = strings.TrimSpace(config.Encoding)
	}
	if len(config.InitCommands) > 0 {
		commands := make([]string, 0, len(config.InitCommands))
		for _, command := range config.InitCommands {
			command = strings.TrimSpace(command)
			if command == "" {
				continue
			}
			commands = append(commands, command)
		}
		if len(commands) > 0 {
			result["initCommands"] = commands
		}
	}
	return result
}

func isEmptySessionConfig(config contracts.DatabaseSessionConfig) bool {
	return strings.TrimSpace(config.ActiveDatabase) == "" &&
		strings.TrimSpace(config.Timezone) == "" &&
		strings.TrimSpace(config.SearchPath) == "" &&
		strings.TrimSpace(config.Encoding) == "" &&
		len(config.InitCommands) == 0
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}
