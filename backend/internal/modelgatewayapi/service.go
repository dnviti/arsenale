package modelgatewayapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/dbsessions"
	"github.com/dnviti/arsenale/backend/internal/modelgateway"
	"github.com/dnviti/arsenale/backend/internal/tenantauth"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	Store               *modelgateway.Store
	DB                  *pgxpool.Pool
	TenantAuth          tenantauth.Service
	DatabaseSessions    dbsessions.Service
	ServerEncryptionKey []byte
	AIState             *aiState
}

type requestError struct {
	status  int
	message string
}

type configResponse struct {
	Provider            contracts.AIProviderID `json:"provider"`
	HasAPIKey           bool                   `json:"hasApiKey"`
	ModelID             string                 `json:"modelId"`
	BaseURL             *string                `json:"baseUrl"`
	MaxTokensPerRequest int                    `json:"maxTokensPerRequest"`
	DailyRequestLimit   int                    `json:"dailyRequestLimit"`
	Enabled             bool                   `json:"enabled"`
}

func (e *requestError) Error() string {
	return e.message
}

func (s Service) HandleGetConfig(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireDatabaseProxyFeature(); err != nil {
		s.writeError(w, err)
		return
	}
	if err := s.requireTenantRole(r.Context(), claims, "ADMIN"); err != nil {
		s.writeError(w, err)
		return
	}

	cfg, err := s.getConfig(r.Context(), claims.TenantID)
	if err != nil {
		s.writeError(w, err)
		return
	}

	app.WriteJSON(w, http.StatusOK, normalizeConfigResponse(cfg))
}

func (s Service) HandleUpdateConfig(w http.ResponseWriter, r *http.Request, claims authn.Claims) {
	if err := requireDatabaseProxyFeature(); err != nil {
		s.writeError(w, err)
		return
	}
	if err := s.requireTenantRole(r.Context(), claims, "OWNER"); err != nil {
		s.writeError(w, err)
		return
	}

	var payload map[string]json.RawMessage
	if err := app.ReadJSON(r, &payload); err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	update, err := parseUpdatePayload(payload)
	if err != nil {
		app.ErrorJSON(w, http.StatusBadRequest, err.Error())
		return
	}

	if s.Store == nil {
		s.writeError(w, errors.New("model gateway store is not configured"))
		return
	}

	cfg, err := s.Store.UpsertConfig(r.Context(), claims.TenantID, update, s.ServerEncryptionKey)
	if err != nil {
		s.writeError(w, classifyStoreError(err))
		return
	}

	app.WriteJSON(w, http.StatusOK, normalizeConfigResponse(cfg))
}

func (s Service) getConfig(ctx context.Context, tenantID string) (contracts.TenantAIConfig, error) {
	if s.Store == nil {
		return contracts.TenantAIConfig{}, errors.New("model gateway store is not configured")
	}

	cfg, err := s.Store.GetConfig(ctx, tenantID)
	if err == nil {
		return cfg, nil
	}
	if modelgateway.IsNotFound(err) {
		return defaultTenantAIConfig(tenantID), nil
	}
	return contracts.TenantAIConfig{}, err
}

func (s Service) requireTenantRole(ctx context.Context, claims authn.Claims, minimum string) error {
	if strings.TrimSpace(claims.UserID) == "" || strings.TrimSpace(claims.TenantID) == "" {
		return &requestError{status: http.StatusForbidden, message: "Tenant membership required"}
	}

	membership, err := s.TenantAuth.ResolveMembership(ctx, claims.UserID, claims.TenantID)
	if err != nil {
		return fmt.Errorf("resolve tenant membership: %w", err)
	}
	if membership == nil {
		return &requestError{status: http.StatusForbidden, message: "Tenant membership required"}
	}
	if !roleAtLeast(membership.Role, minimum) {
		return &requestError{status: http.StatusForbidden, message: "Insufficient tenant role"}
	}
	return nil
}

func (s Service) writeError(w http.ResponseWriter, err error) {
	var reqErr *requestError
	if errors.As(err, &reqErr) {
		app.ErrorJSON(w, reqErr.status, reqErr.message)
		return
	}
	app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
}

func requireDatabaseProxyFeature() error {
	if os.Getenv("FEATURE_DATABASE_PROXY_ENABLED") == "false" {
		return &requestError{status: http.StatusForbidden, message: "The Database SQL Proxy feature is currently disabled."}
	}
	return nil
}

func defaultTenantAIConfig(tenantID string) contracts.TenantAIConfig {
	return contracts.TenantAIConfig{
		TenantID:            tenantID,
		Provider:            contracts.AIProviderNone,
		HasAPIKey:           false,
		ModelID:             "",
		BaseURL:             "",
		MaxTokensPerRequest: 4000,
		DailyRequestLimit:   100,
		Enabled:             false,
	}
}

func normalizeConfigResponse(cfg contracts.TenantAIConfig) configResponse {
	response := configResponse{
		Provider:            cfg.Provider,
		HasAPIKey:           cfg.HasAPIKey,
		ModelID:             cfg.ModelID,
		MaxTokensPerRequest: cfg.MaxTokensPerRequest,
		DailyRequestLimit:   cfg.DailyRequestLimit,
		Enabled:             cfg.Enabled,
	}
	if trimmed := strings.TrimSpace(cfg.BaseURL); trimmed != "" {
		response.BaseURL = &trimmed
	}
	return response
}

func parseUpdatePayload(payload map[string]json.RawMessage) (contracts.TenantAIConfigUpdate, error) {
	var update contracts.TenantAIConfigUpdate
	allowed := map[string]struct{}{
		"provider":            {},
		"apiKey":              {},
		"modelId":             {},
		"baseUrl":             {},
		"maxTokensPerRequest": {},
		"dailyRequestLimit":   {},
		"enabled":             {},
	}
	for key := range payload {
		if _, ok := allowed[key]; !ok {
			return contracts.TenantAIConfigUpdate{}, fmt.Errorf("unknown field %q", key)
		}
	}

	if raw, ok := payload["provider"]; ok {
		var provider contracts.AIProviderID
		if err := json.Unmarshal(raw, &provider); err != nil {
			return contracts.TenantAIConfigUpdate{}, fmt.Errorf("provider must be a string")
		}
		update.Provider = &provider
	}
	if raw, ok := payload["apiKey"]; ok {
		var apiKey string
		if err := json.Unmarshal(raw, &apiKey); err != nil {
			return contracts.TenantAIConfigUpdate{}, fmt.Errorf("apiKey must be a string")
		}
		update.APIKey = &apiKey
	}
	if raw, ok := payload["modelId"]; ok {
		var modelID string
		if err := json.Unmarshal(raw, &modelID); err != nil {
			return contracts.TenantAIConfigUpdate{}, fmt.Errorf("modelId must be a string")
		}
		update.ModelID = &modelID
	}
	if raw, ok := payload["baseUrl"]; ok {
		baseURL, err := decodeNullableString(raw, "baseUrl")
		if err != nil {
			return contracts.TenantAIConfigUpdate{}, err
		}
		update.BaseURL = &baseURL
	}
	if raw, ok := payload["maxTokensPerRequest"]; ok {
		var value int
		if err := json.Unmarshal(raw, &value); err != nil {
			return contracts.TenantAIConfigUpdate{}, fmt.Errorf("maxTokensPerRequest must be a number")
		}
		update.MaxTokensPerRequest = &value
	}
	if raw, ok := payload["dailyRequestLimit"]; ok {
		var value int
		if err := json.Unmarshal(raw, &value); err != nil {
			return contracts.TenantAIConfigUpdate{}, fmt.Errorf("dailyRequestLimit must be a number")
		}
		update.DailyRequestLimit = &value
	}
	if raw, ok := payload["enabled"]; ok {
		var enabled bool
		if err := json.Unmarshal(raw, &enabled); err != nil {
			return contracts.TenantAIConfigUpdate{}, fmt.Errorf("enabled must be a boolean")
		}
		update.Enabled = &enabled
	}

	return update, nil
}

func decodeNullableString(raw json.RawMessage, field string) (string, error) {
	if strings.TrimSpace(string(raw)) == "null" {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s must be a string or null", field)
	}
	return value, nil
}

func classifyStoreError(err error) error {
	lowered := strings.ToLower(err.Error())
	switch {
	case strings.HasPrefix(lowered, "invalid tenant ai config:"):
		return &requestError{status: http.StatusBadRequest, message: strings.TrimPrefix(err.Error(), "invalid tenant ai config: ")}
	case strings.Contains(lowered, "unsupported provider"):
		return &requestError{status: http.StatusBadRequest, message: err.Error()}
	case strings.Contains(lowered, "server_encryption_key"):
		return &requestError{status: http.StatusServiceUnavailable, message: err.Error()}
	default:
		return err
	}
}

func roleAtLeast(role, minimum string) bool {
	order := map[string]int{
		"GUEST":      1,
		"AUDITOR":    2,
		"CONSULTANT": 3,
		"MEMBER":     4,
		"OPERATOR":   5,
		"ADMIN":      6,
		"OWNER":      7,
	}
	return order[strings.ToUpper(strings.TrimSpace(role))] >= order[strings.ToUpper(strings.TrimSpace(minimum))]
}
