package modelgatewayapi

import (
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/modelgateway"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func decryptStoredBackend(item storedAIBackend, key []byte) runtimeAIBackend {
	runtime := runtimeAIBackend{
		Name:         item.Name,
		Provider:     item.Provider,
		BaseURL:      item.BaseURL,
		DefaultModel: item.DefaultModel,
	}
	if item.EncryptedAPIKey != "" && item.APIKeyIV != "" && item.APIKeyTag != "" && len(key) > 0 {
		if decrypted, err := modelgateway.DecryptAPIKey(item.EncryptedAPIKey, item.APIKeyIV, item.APIKeyTag, key); err == nil {
			runtime.APIKey = strings.TrimSpace(decrypted)
		}
	}
	return runtime
}

func decryptLegacyBackend(row legacyConfigRow, key []byte) runtimeAIBackend {
	runtime := runtimeAIBackend{
		Name:         "default",
		Provider:     row.Provider,
		BaseURL:      row.BaseURL,
		DefaultModel: row.ModelID,
	}
	if row.EncryptedAPIKey != "" && row.APIKeyIV != "" && row.APIKeyTag != "" && len(key) > 0 {
		if decrypted, err := modelgateway.DecryptAPIKey(row.EncryptedAPIKey, row.APIKeyIV, row.APIKeyTag, key); err == nil {
			runtime.APIKey = strings.TrimSpace(decrypted)
		}
	}
	return runtime
}

func hasLegacyProvider(row legacyConfigRow) bool {
	provider := strings.TrimSpace(string(row.Provider))
	return provider != "" && row.Provider != contracts.AIProviderNone
}

func buildStoredConfig(row legacyConfigRow, backends []storedAIBackend) storedAIConfig {
	config := defaultStoredAIConfig()
	config.QueryGeneration = storedAIFeature{
		Enabled:             row.Enabled,
		Backend:             row.QueryGenerationBackend,
		ModelID:             row.QueryGenerationModel,
		MaxTokensPerRequest: row.MaxTokensPerRequest,
		DailyRequestLimit:   row.DailyRequestLimit,
	}
	config.QueryOptimizer = storedAIFeature{
		Enabled:             row.QueryOptimizerEnabled,
		Backend:             row.QueryOptimizerBackend,
		ModelID:             row.QueryOptimizerModel,
		MaxTokensPerRequest: row.MaxTokensPerRequest,
	}
	if row.Temperature != 0 {
		config.Temperature = row.Temperature
	}
	if row.TimeoutMS > 0 {
		config.TimeoutMS = row.TimeoutMS
	}

	if len(backends) == 0 && hasLegacyProvider(row) {
		if config.QueryGeneration.Backend == "" {
			config.QueryGeneration.Backend = "default"
		}
		if config.QueryGeneration.ModelID == "" {
			config.QueryGeneration.ModelID = row.ModelID
		}
		if !config.QueryOptimizer.Enabled {
			config.QueryOptimizer.Enabled = true
		}
		if config.QueryOptimizer.Backend == "" {
			config.QueryOptimizer.Backend = "default"
		}
		if config.QueryOptimizer.ModelID == "" {
			config.QueryOptimizer.ModelID = row.ModelID
		}
	}

	config.QueryGeneration = normalizeFeatureConfig(config.QueryGeneration, 4096, 100)
	config.QueryOptimizer = normalizeFeatureConfig(config.QueryOptimizer, 4096, 0)
	return config
}

func buildConfigResponse(config storedAIConfig, backends []storedAIBackend, legacyRow legacyConfigRow) configResponse {
	items := buildBackendResponses(backends, legacyRow)
	response := configResponse{
		Backends: items,
		QueryGeneration: aiFeatureResponse{
			Enabled:             config.QueryGeneration.Enabled,
			Backend:             config.QueryGeneration.Backend,
			ModelID:             config.QueryGeneration.ModelID,
			MaxTokensPerRequest: config.QueryGeneration.MaxTokensPerRequest,
			DailyRequestLimit:   config.QueryGeneration.DailyRequestLimit,
		},
		QueryOptimizer: aiFeatureResponse{
			Enabled:             config.QueryOptimizer.Enabled,
			Backend:             config.QueryOptimizer.Backend,
			ModelID:             config.QueryOptimizer.ModelID,
			MaxTokensPerRequest: config.QueryOptimizer.MaxTokensPerRequest,
		},
		Temperature: config.Temperature,
		TimeoutMS:   config.TimeoutMS,
	}

	legacyBackendName := strings.TrimSpace(config.QueryGeneration.Backend)
	for _, backend := range response.Backends {
		if backend.Name != legacyBackendName {
			continue
		}
		response.Provider = backend.Provider
		response.HasAPIKey = backend.HasAPIKey
		response.BaseURL = backend.BaseURL
		if response.ModelID == "" {
			response.ModelID = backend.DefaultModel
		}
		break
	}
	response.ModelID = firstNonEmpty(response.QueryGeneration.ModelID, response.ModelID)
	if response.Provider == "" {
		response.Provider = contracts.AIProviderNone
	}
	response.MaxTokensPerRequest = response.QueryGeneration.MaxTokensPerRequest
	response.DailyRequestLimit = response.QueryGeneration.DailyRequestLimit
	response.Enabled = response.QueryGeneration.Enabled
	return response
}

func buildBackendResponses(backends []storedAIBackend, legacyRow legacyConfigRow) []aiBackendResponse {
	items := make([]aiBackendResponse, 0, len(backends)+1)
	for _, backend := range backends {
		items = append(items, backendResponse(backend.Name, backend.Provider, backend.EncryptedAPIKey, backend.BaseURL, backend.DefaultModel))
	}
	if len(items) == 0 && hasLegacyProvider(legacyRow) {
		items = append(items, backendResponse("default", legacyRow.Provider, legacyRow.EncryptedAPIKey, legacyRow.BaseURL, legacyRow.ModelID))
	}
	return items
}

func backendResponse(name string, provider contracts.AIProviderID, encryptedAPIKey, baseURL, defaultModel string) aiBackendResponse {
	item := aiBackendResponse{
		Name:         name,
		Provider:     provider,
		HasAPIKey:    encryptedAPIKey != "",
		DefaultModel: defaultModel,
	}
	if baseURL != "" {
		item.BaseURL = &baseURL
	}
	return item
}

func buildEnvPlatformConfig() (aiPlatformConfig, bool) {
	envCfg := loadAIEnvConfig()
	if strings.TrimSpace(string(envCfg.Provider)) == "" || envCfg.Provider == contracts.AIProviderNone {
		return aiPlatformConfig{}, false
	}

	defaultModel := firstNonEmpty(strings.TrimSpace(envCfg.Model), defaultModelForProvider(envCfg.Provider))
	queryGenerationModel := firstNonEmpty(strings.TrimSpace(envCfg.QueryGenerationModel), defaultModel)

	return aiPlatformConfig{
		Backends: []runtimeAIBackend{{
			Name:         "environment",
			Provider:     envCfg.Provider,
			APIKey:       strings.TrimSpace(envCfg.APIKey),
			BaseURL:      strings.TrimSpace(envCfg.BaseURL),
			DefaultModel: defaultModel,
		}},
		QueryGeneration: normalizeFeatureConfig(storedAIFeature{
			Enabled:             envCfg.QueryGenerationEnabled,
			Backend:             "environment",
			ModelID:             queryGenerationModel,
			MaxTokensPerRequest: envCfg.MaxTokens,
			DailyRequestLimit:   envCfg.MaxRequestsPerDay,
		}, 4096, 100),
		QueryOptimizer: normalizeFeatureConfig(storedAIFeature{
			Enabled:             true,
			Backend:             "environment",
			ModelID:             defaultModel,
			MaxTokensPerRequest: envCfg.MaxTokens,
		}, 4096, 0),
		Temperature: envCfg.Temperature,
		Timeout:     envCfg.Timeout,
	}, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func runtimePlatformFromStored(config storedAIConfig, backends []runtimeAIBackend) aiPlatformConfig {
	return aiPlatformConfig{
		Backends:        backends,
		QueryGeneration: config.QueryGeneration,
		QueryOptimizer:  config.QueryOptimizer,
		Temperature:     config.Temperature,
		Timeout:         time.Duration(config.TimeoutMS) * time.Millisecond,
	}
}
