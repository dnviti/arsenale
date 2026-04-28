package modelgatewayapi

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/modelgateway"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/google/uuid"
)

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func backendMapByName(items []storedAIBackend) map[string]storedAIBackend {
	result := make(map[string]storedAIBackend, len(items))
	for _, item := range items {
		result[item.Name] = item
	}
	return result
}

func normalizeBackendUpdate(input aiBackendUpdate, existing map[string]storedAIBackend, encryptionKey []byte) (storedAIBackend, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return storedAIBackend{}, errors.New("backend name is required")
	}
	if input.Provider == "" || input.Provider == contracts.AIProviderNone {
		return storedAIBackend{}, fmt.Errorf("backend %q requires a provider", name)
	}
	providerMeta, ok := modelgateway.LookupProvider(input.Provider)
	if !ok || input.Provider == contracts.AIProviderNone {
		return storedAIBackend{}, fmt.Errorf("backend %q has an unsupported provider %q", name, input.Provider)
	}

	item := storedAIBackend{
		ID:           uuid.NewString(),
		Name:         name,
		Provider:     input.Provider,
		BaseURL:      strings.TrimSpace(stringOrEmpty(input.BaseURL)),
		DefaultModel: strings.TrimSpace(stringOrEmpty(input.DefaultModel)),
	}
	if current, ok := existing[name]; ok {
		item.ID = current.ID
		item.CreatedAt = current.CreatedAt
		item.UpdatedAt = current.UpdatedAt
		item.EncryptedAPIKey = current.EncryptedAPIKey
		item.APIKeyIV = current.APIKeyIV
		item.APIKeyTag = current.APIKeyTag
	}

	if input.ClearAPIKey {
		item.EncryptedAPIKey = ""
		item.APIKeyIV = ""
		item.APIKeyTag = ""
	}
	if input.APIKey != nil {
		apiKey := strings.TrimSpace(*input.APIKey)
		if apiKey == "" {
			item.EncryptedAPIKey = ""
			item.APIKeyIV = ""
			item.APIKeyTag = ""
		} else {
			if len(encryptionKey) == 0 {
				return storedAIBackend{}, errors.New("SERVER_ENCRYPTION_KEY is required to store apiKey")
			}
			ciphertext, iv, tag, err := modelgateway.EncryptAPIKey(apiKey, encryptionKey)
			if err != nil {
				return storedAIBackend{}, err
			}
			item.EncryptedAPIKey = ciphertext
			item.APIKeyIV = iv
			item.APIKeyTag = tag
		}
	}

	if providerMeta.RequiresBaseURL && item.BaseURL == "" {
		return storedAIBackend{}, fmt.Errorf("backend %q requires baseUrl", name)
	}
	if providerMeta.RequiresAPIKey && item.EncryptedAPIKey == "" {
		return storedAIBackend{}, fmt.Errorf("backend %q requires an apiKey", name)
	}
	if item.DefaultModel == "" {
		item.DefaultModel = providerMeta.DefaultModel
	}
	return item, nil
}

func normalizeFeatureUpdate(feature aiFeatureUpdate, defaultMaxTokens, defaultDailyLimit int) storedAIFeature {
	item := storedAIFeature{
		Enabled:             feature.Enabled,
		Backend:             strings.TrimSpace(feature.Backend),
		ModelID:             strings.TrimSpace(feature.ModelID),
		MaxTokensPerRequest: feature.MaxTokensPerRequest,
		DailyRequestLimit:   feature.DailyRequestLimit,
	}
	return normalizeFeatureConfig(item, defaultMaxTokens, defaultDailyLimit)
}

func validateFeatureBackend(featureName string, feature storedAIFeature, backends []storedAIBackend) error {
	if !feature.Enabled {
		return nil
	}
	if strings.TrimSpace(feature.Backend) == "" {
		return fmt.Errorf("%s backend is required when the feature is enabled", featureName)
	}
	for _, backend := range backends {
		if backend.Name == feature.Backend {
			return nil
		}
	}
	return fmt.Errorf("%s backend %q is not configured", featureName, feature.Backend)
}

func legacyColumnsForFeature(feature storedAIFeature, backends []storedAIBackend) legacyConfigRow {
	row := legacyConfigRow{
		Provider:            contracts.AIProviderNone,
		MaxTokensPerRequest: feature.MaxTokensPerRequest,
		DailyRequestLimit:   feature.DailyRequestLimit,
		Enabled:             feature.Enabled,
		ModelID:             feature.ModelID,
	}
	for _, backend := range backends {
		if backend.Name != feature.Backend {
			continue
		}
		row.Provider = backend.Provider
		row.EncryptedAPIKey = backend.EncryptedAPIKey
		row.APIKeyIV = backend.APIKeyIV
		row.APIKeyTag = backend.APIKeyTag
		row.BaseURL = backend.BaseURL
		if row.ModelID == "" {
			row.ModelID = backend.DefaultModel
		}
		break
	}
	return row
}
