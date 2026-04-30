package modelgatewayapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/dbsessions"
	"github.com/dnviti/arsenale/backend/internal/modelgateway"
)

func llmOverridesFromExecution(execution aiFeatureExecution) *llmOverrides {
	if !execution.Enabled {
		return nil
	}
	return &llmOverrides{
		Provider:    execution.Backend.Provider,
		APIKey:      strings.TrimSpace(execution.Backend.APIKey),
		Model:       strings.TrimSpace(execution.ModelID),
		BaseURL:     strings.TrimSpace(execution.Backend.BaseURL),
		MaxTokens:   execution.MaxTokens,
		Temperature: &execution.Temperature,
		Timeout:     execution.Timeout,
	}
}

func providerAndModelFromOverrides(overrides *llmOverrides) (string, string) {
	if overrides == nil {
		return "none", ""
	}
	provider := strings.TrimSpace(string(overrides.Provider))
	if provider == "" {
		provider = "none"
	}
	modelID := strings.TrimSpace(overrides.Model)
	return provider, modelID
}

func findRuntimeBackend(backends []runtimeAIBackend, name string) (runtimeAIBackend, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return runtimeAIBackend{}, false
	}
	for _, backend := range backends {
		if backend.Name == name {
			return backend, true
		}
	}
	return runtimeAIBackend{}, false
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func resolveFeatureExecution(platform aiPlatformConfig, context dbsessions.OwnedAIContext, featureName string) (aiFeatureExecution, error) {
	switch featureName {
	case "query-generation":
		enabled := boolOrDefault(context.AIQueryGenerationEnabled, platform.QueryGeneration.Enabled)
		backendName := strings.TrimSpace(context.AIQueryGenerationBackend)
		if backendName == "" {
			backendName = platform.QueryGeneration.Backend
		}
		modelID := strings.TrimSpace(context.AIQueryGenerationModel)
		if modelID == "" {
			modelID = platform.QueryGeneration.ModelID
		}
		return buildFeatureExecution(platform, enabled, backendName, modelID, platform.QueryGeneration.MaxTokensPerRequest, platform.QueryGeneration.DailyRequestLimit)
	case "query-optimizer":
		enabled := boolOrDefault(context.AIQueryOptimizerEnabled, platform.QueryOptimizer.Enabled)
		backendName := strings.TrimSpace(context.AIQueryOptimizerBackend)
		if backendName == "" {
			backendName = platform.QueryOptimizer.Backend
		}
		modelID := strings.TrimSpace(context.AIQueryOptimizerModel)
		if modelID == "" {
			modelID = platform.QueryOptimizer.ModelID
		}
		return buildFeatureExecution(platform, enabled, backendName, modelID, platform.QueryOptimizer.MaxTokensPerRequest, platform.QueryOptimizer.DailyRequestLimit)
	default:
		return aiFeatureExecution{}, fmt.Errorf("unsupported ai feature %q", featureName)
	}
}

func buildFeatureExecution(platform aiPlatformConfig, enabled bool, backendName, modelID string, maxTokens, dailyLimit int) (aiFeatureExecution, error) {
	if !enabled {
		return aiFeatureExecution{}, nil
	}
	backend, ok := findRuntimeBackend(platform.Backends, backendName)
	if !ok {
		return aiFeatureExecution{}, &requestError{status: 503, message: "AI backend is not configured or is unavailable."}
	}
	if modelID == "" {
		modelID = strings.TrimSpace(backend.DefaultModel)
	}
	if modelID == "" {
		modelID = defaultModelForProvider(backend.Provider)
	}
	if modelID == "" {
		return aiFeatureExecution{}, &requestError{status: 503, message: "AI model is not configured and no default is available for the selected backend."}
	}

	providerMeta, ok := modelgateway.LookupProvider(backend.Provider)
	if !ok {
		return aiFeatureExecution{}, &requestError{status: 503, message: "AI backend provider is not supported."}
	}
	if providerMeta.RequiresAPIKey && strings.TrimSpace(backend.APIKey) == "" {
		return aiFeatureExecution{}, &requestError{status: 503, message: "AI backend API key is not configured."}
	}
	if providerMeta.RequiresBaseURL && strings.TrimSpace(backend.BaseURL) == "" {
		return aiFeatureExecution{}, &requestError{status: 503, message: fmt.Sprintf("AI backend base URL is required for %s.", backend.Provider)}
	}

	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return aiFeatureExecution{
		Enabled:           true,
		Backend:           backend,
		ModelID:           modelID,
		MaxTokens:         maxTokens,
		DailyRequestLimit: dailyLimit,
		Temperature:       platform.Temperature,
		Timeout:           platform.Timeout,
	}, nil
}

func (s Service) resolveFeatureExecutionForSession(ctx context.Context, userID, tenantID, sessionID, featureName string) (dbsessions.OwnedAIContext, aiFeatureExecution, error) {
	aiContext, err := s.DatabaseSessions.ResolveOwnedAIContext(ctx, userID, tenantID, sessionID)
	if err != nil {
		return dbsessions.OwnedAIContext{}, aiFeatureExecution{}, err
	}
	platform, err := s.loadPlatformConfig(ctx, tenantID)
	if err != nil {
		return dbsessions.OwnedAIContext{}, aiFeatureExecution{}, err
	}
	execution, err := resolveFeatureExecution(platform, aiContext, featureName)
	if err != nil {
		return dbsessions.OwnedAIContext{}, aiFeatureExecution{}, err
	}
	if execution.Enabled {
		return aiContext, execution, nil
	}

	message := "AI feature is not enabled"
	switch featureName {
	case "query-generation":
		message = "AI query generation is not enabled"
	case "query-optimizer":
		message = "AI query optimization is not enabled"
	}
	return aiContext, aiFeatureExecution{}, &requestError{status: http.StatusForbidden, message: message}
}
