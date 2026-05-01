package modelgateway

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

var providers = []contracts.AIProviderDescriptor{
	{ID: contracts.AIProviderAnthropic, SupportsEmbeddings: false, SupportsResponses: true, RequiresAPIKey: true, RequiresBaseURL: false, DefaultModel: "claude-sonnet-4-20250514"},
	{ID: contracts.AIProviderOpenAI, SupportsEmbeddings: true, SupportsResponses: true, RequiresAPIKey: true, RequiresBaseURL: false, DefaultModel: "gpt-4o"},
	{ID: contracts.AIProviderOllama, SupportsEmbeddings: true, SupportsResponses: true, RequiresAPIKey: false, RequiresBaseURL: true, DefaultModel: "llama3.1:8b"},
	{ID: contracts.AIProviderOpenAICompatible, SupportsEmbeddings: true, SupportsResponses: true, RequiresAPIKey: true, RequiresBaseURL: true, DefaultModel: ""},
}

const (
	aesKeyBytes = 32
	ivBytes     = 16
)

func Providers() []contracts.AIProviderDescriptor {
	out := make([]contracts.AIProviderDescriptor, len(providers))
	copy(out, providers)
	return out
}

func LookupProvider(id contracts.AIProviderID) (contracts.AIProviderDescriptor, bool) {
	if id == contracts.AIProviderNone {
		return contracts.AIProviderDescriptor{ID: contracts.AIProviderNone}, true
	}
	for _, provider := range providers {
		if provider.ID == id {
			return provider, true
		}
	}
	return contracts.AIProviderDescriptor{}, false
}

func ValidateConfig(config contracts.TenantAIConfig, apiKeyProvided bool) contracts.ValidationResult {
	var errs []string
	var warnings []string

	provider, ok := LookupProvider(config.Provider)
	if !ok {
		errs = append(errs, fmt.Sprintf("unsupported provider %q", config.Provider))
		return contracts.ValidationResult{Valid: false, Errors: errs}
	}

	if config.MaxTokensPerRequest <= 0 {
		errs = append(errs, "maxTokensPerRequest must be greater than zero")
	}
	if config.DailyRequestLimit <= 0 {
		errs = append(errs, "dailyRequestLimit must be greater than zero")
	}

	if config.Provider == contracts.AIProviderNone {
		if config.Enabled {
			errs = append(errs, "provider 'none' cannot be enabled")
		}
		return contracts.ValidationResult{Valid: len(errs) == 0, Errors: errs, Warnings: warnings}
	}

	if provider.RequiresAPIKey && !apiKeyProvided {
		errs = append(errs, "provider requires an API key")
	}
	if provider.RequiresBaseURL && strings.TrimSpace(config.BaseURL) == "" {
		errs = append(errs, "provider requires baseUrl")
	}
	if strings.TrimSpace(config.ModelID) == "" && provider.DefaultModel == "" {
		warnings = append(warnings, "provider has no default model; modelId should be set explicitly")
	}

	return contracts.ValidationResult{
		Valid:    len(errs) == 0,
		Errors:   errs,
		Warnings: warnings,
	}
}

func LoadServerEncryptionKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv("SERVER_ENCRYPTION_KEY"))
	if raw == "" {
		if path := strings.TrimSpace(os.Getenv("SERVER_ENCRYPTION_KEY_FILE")); path != "" {
			payload, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("read SERVER_ENCRYPTION_KEY_FILE: %w", err)
			}
			raw = strings.TrimSpace(string(payload))
		}
	}
	if raw == "" {
		return nil, nil
	}
	if len(raw) != aesKeyBytes*2 {
		return nil, fmt.Errorf("SERVER_ENCRYPTION_KEY must be exactly %d hex characters", aesKeyBytes*2)
	}
	key, err := hex.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("decode SERVER_ENCRYPTION_KEY: %w", err)
	}
	return key, nil
}

func EncryptAPIKey(apiKey string, key []byte) (ciphertext, iv, tag string, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", "", fmt.Errorf("create cipher: %w", err)
	}
	nonce := make([]byte, ivBytes)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", "", fmt.Errorf("read nonce: %w", err)
	}
	aead, err := cipher.NewGCMWithNonceSize(block, ivBytes)
	if err != nil {
		return "", "", "", fmt.Errorf("create gcm: %w", err)
	}
	sealed := aead.Seal(nil, nonce, []byte(apiKey), nil)
	tagOffset := len(sealed) - aead.Overhead()
	return hex.EncodeToString(sealed[:tagOffset]), hex.EncodeToString(nonce), hex.EncodeToString(sealed[tagOffset:]), nil
}

func DecryptAPIKey(ciphertext, iv, tag string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}

	nonce, err := hex.DecodeString(strings.TrimSpace(iv))
	if err != nil {
		return "", fmt.Errorf("decode iv: %w", err)
	}
	payload, err := hex.DecodeString(strings.TrimSpace(ciphertext))
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	authTag, err := hex.DecodeString(strings.TrimSpace(tag))
	if err != nil {
		return "", fmt.Errorf("decode tag: %w", err)
	}

	aead, err := cipher.NewGCMWithNonceSize(block, len(nonce))
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}

	plaintext, err := aead.Open(nil, nonce, append(payload, authTag...), nil)
	if err != nil {
		return "", fmt.Errorf("decrypt api key: %w", err)
	}
	return string(plaintext), nil
}
