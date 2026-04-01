package modelgatewayapi

import (
	"encoding/json"
	"testing"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestNormalizeConfigResponseOmitsEmptyBaseURLAsNull(t *testing.T) {
	t.Parallel()

	response := normalizeConfigResponse(contracts.TenantAIConfig{
		Provider:            contracts.AIProviderOpenAI,
		HasAPIKey:           true,
		ModelID:             "gpt-4o",
		BaseURL:             "",
		MaxTokensPerRequest: 2048,
		DailyRequestLimit:   25,
		Enabled:             true,
	})

	if response.BaseURL != nil {
		t.Fatalf("expected nil baseUrl, got %q", *response.BaseURL)
	}
}

func TestParseUpdatePayloadSupportsNullBaseURL(t *testing.T) {
	t.Parallel()

	payload := map[string]json.RawMessage{
		"provider":            json.RawMessage(`"openai-compatible"`),
		"apiKey":              json.RawMessage(`"secret"`),
		"modelId":             json.RawMessage(`"gpt-4.1"`),
		"baseUrl":             json.RawMessage(`null`),
		"maxTokensPerRequest": json.RawMessage(`512`),
		"dailyRequestLimit":   json.RawMessage(`42`),
		"enabled":             json.RawMessage(`true`),
	}

	update, err := parseUpdatePayload(payload)
	if err != nil {
		t.Fatalf("parseUpdatePayload returned error: %v", err)
	}
	if update.Provider == nil || *update.Provider != contracts.AIProviderOpenAICompatible {
		t.Fatalf("unexpected provider: %#v", update.Provider)
	}
	if update.BaseURL == nil || *update.BaseURL != "" {
		t.Fatalf("expected empty baseUrl for null input, got %#v", update.BaseURL)
	}
	if update.Enabled == nil || !*update.Enabled {
		t.Fatalf("expected enabled=true, got %#v", update.Enabled)
	}
	if update.MaxTokensPerRequest == nil || *update.MaxTokensPerRequest != 512 {
		t.Fatalf("unexpected maxTokensPerRequest: %#v", update.MaxTokensPerRequest)
	}
}

func TestParseUpdatePayloadRejectsUnknownField(t *testing.T) {
	t.Parallel()

	_, err := parseUpdatePayload(map[string]json.RawMessage{
		"provider": json.RawMessage(`"openai"`),
		"extra":    json.RawMessage(`true`),
	})
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestRoleAtLeast(t *testing.T) {
	t.Parallel()

	if !roleAtLeast("OWNER", "ADMIN") {
		t.Fatal("expected OWNER to satisfy ADMIN")
	}
	if roleAtLeast("MEMBER", "ADMIN") {
		t.Fatal("did not expect MEMBER to satisfy ADMIN")
	}
}
