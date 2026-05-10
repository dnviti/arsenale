package cmd

import (
	"encoding/json"
	"testing"
)

func TestNormalizeAIConfigPayloadStripsReadOnlyFields(t *testing.T) {
	body, err := normalizeAIConfigPayload([]byte(`{
		"backends": [
			{
				"name": "default",
				"provider": "openai",
				"hasApiKey": true,
				"baseUrl": "https://api.openai.com/v1",
				"defaultModel": "gpt-4o-mini"
			}
		],
		"queryGeneration": {
			"enabled": true,
			"backend": "default",
			"modelId": "gpt-4o-mini",
			"maxTokensPerRequest": 2048,
			"dailyRequestLimit": 50
		},
		"queryOptimizer": {
			"enabled": true,
			"backend": "default",
			"modelId": "gpt-4o-mini",
			"maxTokensPerRequest": 2048
		},
		"temperature": 0.2,
		"timeoutMs": 60000,
		"provider": "openai",
		"hasApiKey": true,
		"modelId": "gpt-4o-mini",
		"baseUrl": "https://api.openai.com/v1",
		"maxTokensPerRequest": 2048,
		"dailyRequestLimit": 50,
		"enabled": true
	}`))
	if err != nil {
		t.Fatalf("normalizeAIConfigPayload() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	for _, key := range []string{"provider", "hasApiKey", "modelId", "baseUrl", "maxTokensPerRequest", "dailyRequestLimit", "enabled"} {
		if _, ok := got[key]; ok {
			t.Fatalf("top-level read-only field %q was not stripped", key)
		}
	}

	backends := got["backends"].([]any)
	backend := backends[0].(map[string]any)
	if _, ok := backend["hasApiKey"]; ok {
		t.Fatal("backend hasApiKey field was not stripped")
	}
	if backend["name"] != "default" || backend["provider"] != "openai" {
		t.Fatalf("backend payload changed unexpectedly: %#v", backend)
	}
}
