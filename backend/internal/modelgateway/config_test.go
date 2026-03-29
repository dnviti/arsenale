package modelgateway

import (
	"encoding/hex"
	"testing"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestValidateConfigRequiresBaseURLForOllama(t *testing.T) {
	result := ValidateConfig(contracts.TenantAIConfig{
		TenantID:            "tenant-1",
		Provider:            contracts.AIProviderOllama,
		ModelID:             "llama3.1:8b",
		MaxTokensPerRequest: 4000,
		DailyRequestLimit:   100,
		Enabled:             true,
	}, false)

	if result.Valid {
		t.Fatalf("expected invalid result, got %+v", result)
	}
}

func TestValidateConfigAcceptsOpenAIWithAPIKey(t *testing.T) {
	result := ValidateConfig(contracts.TenantAIConfig{
		TenantID:            "tenant-1",
		Provider:            contracts.AIProviderOpenAI,
		ModelID:             "gpt-4o",
		MaxTokensPerRequest: 4000,
		DailyRequestLimit:   100,
		Enabled:             true,
	}, true)

	if !result.Valid {
		t.Fatalf("expected valid result, got %+v", result)
	}
}

func TestEncryptAPIKeyProducesExpectedFieldShapes(t *testing.T) {
	key, err := hex.DecodeString("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff")
	if err != nil {
		t.Fatalf("decode key: %v", err)
	}

	ciphertext, iv, tag, err := EncryptAPIKey("secret-token", key)
	if err != nil {
		t.Fatalf("EncryptAPIKey() returned error: %v", err)
	}
	if ciphertext == "" || iv == "" || tag == "" {
		t.Fatalf("expected non-empty encrypted fields, got ciphertext=%q iv=%q tag=%q", ciphertext, iv, tag)
	}
}
