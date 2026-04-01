package dbsessions

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

var ErrAISchemaUnsupported = errors.New("ai schema fetch is unsupported for this session")

type aiConnectionSnapshot struct {
	Host       string
	Port       int
	DBSettings json.RawMessage
}

func (s Service) ResolveOwnedAITarget(ctx context.Context, userID, tenantID, sessionID string) (*contracts.DatabaseTarget, string, error) {
	runtime, err := s.resolveOwnedQueryRuntime(ctx, userID, tenantID, sessionID)
	if err != nil {
		if errors.Is(err, ErrQueryRuntimeUnsupported) {
			return nil, "", ErrAISchemaUnsupported
		}
		return nil, "", err
	}
	if runtime == nil {
		return nil, "", ErrAISchemaUnsupported
	}
	return runtime.Target, runtime.Protocol, nil
}

func resolveAITarget(metadata map[string]any, fallbackHost string, fallbackPort int) (string, int) {
	host := strings.TrimSpace(fallbackHost)
	port := fallbackPort

	if value, ok := metadata["resolvedHost"].(string); ok && strings.TrimSpace(value) != "" {
		host = strings.TrimSpace(value)
	}
	switch value := metadata["resolvedPort"].(type) {
	case float64:
		if value > 0 {
			port = int(value)
		}
	case int:
		if value > 0 {
			port = value
		}
	}

	return host, port
}

func sessionConfigFromMetadata(metadata map[string]any) *contracts.DatabaseSessionConfig {
	raw, ok := metadata["sessionConfig"]
	if !ok || raw == nil {
		return nil
	}

	payload, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var config contracts.DatabaseSessionConfig
	if err := json.Unmarshal(payload, &config); err != nil {
		return nil
	}
	if isEmptySessionConfig(config) {
		return nil
	}
	return &config
}

func metadataBool(metadata map[string]any, key string) bool {
	value, ok := metadata[key]
	if !ok {
		return false
	}
	flag, ok := value.(bool)
	return ok && flag
}
