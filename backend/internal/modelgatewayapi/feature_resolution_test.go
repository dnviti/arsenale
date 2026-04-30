package modelgatewayapi

import (
	"testing"
	"time"

	"github.com/dnviti/arsenale/backend/internal/dbsessions"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestResolveFeatureExecutionUsesConnectionOverrides(t *testing.T) {
	t.Parallel()

	enabled := true
	platform := aiPlatformConfig{
		Backends: []runtimeAIBackend{
			{Name: "primary", Provider: contracts.AIProviderOpenAI, APIKey: "key", DefaultModel: "gpt-4o"},
			{Name: "local", Provider: contracts.AIProviderOllama, BaseURL: "http://ollama:11434", DefaultModel: "llama3.1:8b"},
		},
		QueryGeneration: storedAIFeature{
			Enabled:             true,
			Backend:             "primary",
			ModelID:             "gpt-4o",
			MaxTokensPerRequest: 1024,
			DailyRequestLimit:   25,
		},
		Temperature: 0.3,
		Timeout:     5 * time.Second,
	}
	context := dbsessions.OwnedAIContext{
		AIQueryGenerationEnabled: &enabled,
		AIQueryGenerationBackend: "local",
		AIQueryGenerationModel:   "llama3.1:70b",
	}

	execution, err := resolveFeatureExecution(platform, context, "query-generation")
	if err != nil {
		t.Fatalf("resolveFeatureExecution() error = %v", err)
	}
	if execution.Backend.Name != "local" || execution.ModelID != "llama3.1:70b" {
		t.Fatalf("unexpected execution: %#v", execution)
	}
	if execution.MaxTokens != 1024 || execution.DailyRequestLimit != 25 {
		t.Fatalf("unexpected limits: %#v", execution)
	}
}

func TestResolveFeatureExecutionReturnsDisabledWhenOverrideDisablesFeature(t *testing.T) {
	t.Parallel()

	disabled := false
	execution, err := resolveFeatureExecution(aiPlatformConfig{
		QueryOptimizer: storedAIFeature{
			Enabled:             true,
			Backend:             "primary",
			MaxTokensPerRequest: 1024,
		},
	}, dbsessions.OwnedAIContext{
		AIQueryOptimizerEnabled: &disabled,
	}, "query-optimizer")
	if err != nil {
		t.Fatalf("resolveFeatureExecution() error = %v", err)
	}
	if execution.Enabled {
		t.Fatalf("expected disabled execution, got %#v", execution)
	}
}

func TestBuildFeatureExecutionRequiresProviderCredentials(t *testing.T) {
	t.Parallel()

	_, err := buildFeatureExecution(aiPlatformConfig{
		Backends: []runtimeAIBackend{{
			Name:         "primary",
			Provider:     contracts.AIProviderOpenAI,
			DefaultModel: "gpt-4o",
		}},
	}, true, "primary", "", 0, 0)
	reqErr, ok := err.(*requestError)
	if !ok {
		t.Fatalf("expected requestError, got %T", err)
	}
	if reqErr.status != 503 || reqErr.message != "AI backend API key is not configured." {
		t.Fatalf("unexpected error: %#v", reqErr)
	}
}
