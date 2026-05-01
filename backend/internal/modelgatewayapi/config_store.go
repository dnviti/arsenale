package modelgatewayapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type legacyConfigRow struct {
	Loaded                 bool
	Provider               contracts.AIProviderID
	EncryptedAPIKey        string
	APIKeyIV               string
	APIKeyTag              string
	ModelID                string
	BaseURL                string
	MaxTokensPerRequest    int
	DailyRequestLimit      int
	Enabled                bool
	QueryGenerationBackend string
	QueryGenerationModel   string
	QueryOptimizerEnabled  bool
	QueryOptimizerBackend  string
	QueryOptimizerModel    string
	Temperature            float64
	TimeoutMS              int
}

func defaultStoredAIConfig() storedAIConfig {
	return storedAIConfig{
		QueryGeneration: storedAIFeature{
			Enabled:             false,
			MaxTokensPerRequest: 4096,
			DailyRequestLimit:   100,
		},
		QueryOptimizer: storedAIFeature{
			Enabled:             false,
			MaxTokensPerRequest: 4096,
		},
		Temperature: 0.2,
		TimeoutMS:   60000,
	}
}

func normalizeFeatureConfig(feature storedAIFeature, defaultMaxTokens, defaultDailyLimit int) storedAIFeature {
	feature.Backend = strings.TrimSpace(feature.Backend)
	feature.ModelID = strings.TrimSpace(feature.ModelID)
	if feature.MaxTokensPerRequest <= 0 {
		feature.MaxTokensPerRequest = defaultMaxTokens
	}
	if defaultDailyLimit > 0 && feature.DailyRequestLimit <= 0 {
		feature.DailyRequestLimit = defaultDailyLimit
	}
	return feature
}

func (s Service) loadLegacyConfigRow(ctx context.Context, tenantID string) (legacyConfigRow, error) {
	if s.DB == nil {
		return legacyConfigRow{}, errors.New("database is unavailable")
	}

	row := s.DB.QueryRow(ctx, `
SELECT provider,
       COALESCE("encryptedApiKey", ''),
       COALESCE("apiKeyIV", ''),
       COALESCE("apiKeyTag", ''),
       COALESCE("modelId", ''),
       COALESCE("baseUrl", ''),
       "maxTokensPerRequest",
       "dailyRequestLimit",
       enabled,
       COALESCE("queryGenerationBackend", ''),
       COALESCE("queryGenerationModel", ''),
       "queryOptimizerEnabled",
       COALESCE("queryOptimizerBackend", ''),
       COALESCE("queryOptimizerModel", ''),
       temperature,
       "timeoutMs"
FROM "TenantAiConfig"
WHERE "tenantId" = $1
`, tenantID)

	var item legacyConfigRow
	if err := row.Scan(
		&item.Provider,
		&item.EncryptedAPIKey,
		&item.APIKeyIV,
		&item.APIKeyTag,
		&item.ModelID,
		&item.BaseURL,
		&item.MaxTokensPerRequest,
		&item.DailyRequestLimit,
		&item.Enabled,
		&item.QueryGenerationBackend,
		&item.QueryGenerationModel,
		&item.QueryOptimizerEnabled,
		&item.QueryOptimizerBackend,
		&item.QueryOptimizerModel,
		&item.Temperature,
		&item.TimeoutMS,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return legacyConfigRow{}, nil
		}
		return legacyConfigRow{}, fmt.Errorf("load tenant ai config: %w", err)
	}

	item.Loaded = true
	item.ModelID = strings.TrimSpace(item.ModelID)
	item.BaseURL = strings.TrimSpace(item.BaseURL)
	item.QueryGenerationBackend = strings.TrimSpace(item.QueryGenerationBackend)
	item.QueryGenerationModel = strings.TrimSpace(item.QueryGenerationModel)
	item.QueryOptimizerBackend = strings.TrimSpace(item.QueryOptimizerBackend)
	item.QueryOptimizerModel = strings.TrimSpace(item.QueryOptimizerModel)
	return item, nil
}

func (s Service) loadStoredBackends(ctx context.Context, tenantID string) ([]storedAIBackend, error) {
	if s.DB == nil {
		return nil, errors.New("database is unavailable")
	}

	rows, err := s.DB.Query(ctx, `
SELECT id,
       name,
       provider,
       COALESCE("encryptedApiKey", ''),
       COALESCE("apiKeyIV", ''),
       COALESCE("apiKeyTag", ''),
       COALESCE("baseUrl", ''),
       COALESCE("defaultModel", ''),
       "createdAt",
       "updatedAt"
FROM "TenantAiBackend"
WHERE "tenantId" = $1
ORDER BY name ASC
`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list tenant ai backends: %w", err)
	}
	defer rows.Close()

	items := make([]storedAIBackend, 0)
	for rows.Next() {
		var item storedAIBackend
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Provider,
			&item.EncryptedAPIKey,
			&item.APIKeyIV,
			&item.APIKeyTag,
			&item.BaseURL,
			&item.DefaultModel,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan tenant ai backend: %w", err)
		}
		item.Name = strings.TrimSpace(item.Name)
		item.BaseURL = strings.TrimSpace(item.BaseURL)
		item.DefaultModel = strings.TrimSpace(item.DefaultModel)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tenant ai backends: %w", err)
	}
	return items, nil
}

func (s Service) getConfig(ctx context.Context, tenantID string) (configResponse, error) {
	legacyRow, err := s.loadLegacyConfigRow(ctx, tenantID)
	if err != nil {
		return configResponse{}, err
	}
	backends, err := s.loadStoredBackends(ctx, tenantID)
	if err != nil {
		return configResponse{}, err
	}
	return buildConfigResponse(buildStoredConfig(legacyRow, backends), backends, legacyRow), nil
}

func (s Service) loadPlatformConfig(ctx context.Context, tenantID string) (aiPlatformConfig, error) {
	legacyRow, err := s.loadLegacyConfigRow(ctx, tenantID)
	if err != nil {
		return aiPlatformConfig{}, err
	}
	backends, err := s.loadStoredBackends(ctx, tenantID)
	if err != nil {
		return aiPlatformConfig{}, err
	}

	config := buildStoredConfig(legacyRow, backends)
	runtimeBackends := make([]runtimeAIBackend, 0, len(backends)+1)
	for _, backend := range backends {
		runtimeBackends = append(runtimeBackends, decryptStoredBackend(backend, s.ServerEncryptionKey))
	}
	if len(runtimeBackends) == 0 && hasLegacyProvider(legacyRow) {
		runtimeBackends = append(runtimeBackends, decryptLegacyBackend(legacyRow, s.ServerEncryptionKey))
	}
	if len(runtimeBackends) == 0 && !legacyRow.Loaded {
		if envPlatform, ok := buildEnvPlatformConfig(); ok {
			return envPlatform, nil
		}
	}

	return runtimePlatformFromStored(config, runtimeBackends), nil
}

func (s Service) saveConfig(ctx context.Context, tenantID, userID string, update configUpdate) (configResponse, error) {
	if s.DB == nil {
		return configResponse{}, errors.New("database is unavailable")
	}

	existingBackends, err := s.loadStoredBackends(ctx, tenantID)
	if err != nil {
		return configResponse{}, err
	}
	existingByName := backendMapByName(existingBackends)

	normalizedBackends := make([]storedAIBackend, 0, len(update.Backends))
	seenNames := make(map[string]struct{}, len(update.Backends))
	for _, backend := range update.Backends {
		normalized, err := normalizeBackendUpdate(backend, existingByName, s.ServerEncryptionKey)
		if err != nil {
			return configResponse{}, err
		}
		if _, exists := seenNames[normalized.Name]; exists {
			return configResponse{}, fmt.Errorf("backend name %q is duplicated", normalized.Name)
		}
		seenNames[normalized.Name] = struct{}{}
		normalizedBackends = append(normalizedBackends, normalized)
	}

	queryGeneration := normalizeFeatureUpdate(update.QueryGeneration, 4096, 100)
	queryOptimizer := normalizeFeatureUpdate(update.QueryOptimizer, 4096, 0)
	if err := validateFeatureBackend("queryGeneration", queryGeneration, normalizedBackends); err != nil {
		return configResponse{}, err
	}
	if err := validateFeatureBackend("queryOptimizer", queryOptimizer, normalizedBackends); err != nil {
		return configResponse{}, err
	}

	temperature := update.Temperature
	if temperature < 0 || temperature > 2 {
		return configResponse{}, errors.New("temperature must be between 0 and 2")
	}
	timeoutMS := update.TimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = 60000
	}

	legacy := legacyColumnsForFeature(queryGeneration, normalizedBackends)
	legacy.QueryGenerationBackend = queryGeneration.Backend
	legacy.QueryGenerationModel = queryGeneration.ModelID
	legacy.QueryOptimizerEnabled = queryOptimizer.Enabled
	legacy.QueryOptimizerBackend = queryOptimizer.Backend
	legacy.QueryOptimizerModel = queryOptimizer.ModelID
	legacy.Temperature = temperature
	legacy.TimeoutMS = timeoutMS

	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return configResponse{}, fmt.Errorf("begin ai config update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
INSERT INTO "TenantAiConfig" (
	id,
	"tenantId",
	provider,
	"encryptedApiKey",
	"apiKeyIV",
	"apiKeyTag",
	"modelId",
	"baseUrl",
	"maxTokensPerRequest",
	"dailyRequestLimit",
	enabled,
	"queryGenerationBackend",
	"queryGenerationModel",
	"queryOptimizerEnabled",
	"queryOptimizerBackend",
	"queryOptimizerModel",
	temperature,
	"timeoutMs",
	"createdAt",
	"updatedAt"
)
VALUES (
	$1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''),
	$9, $10, $11, NULLIF($12, ''), NULLIF($13, ''), $14, NULLIF($15, ''), NULLIF($16, ''), $17, $18, NOW(), NOW()
)
ON CONFLICT ("tenantId") DO UPDATE SET
	provider = EXCLUDED.provider,
	"encryptedApiKey" = EXCLUDED."encryptedApiKey",
	"apiKeyIV" = EXCLUDED."apiKeyIV",
	"apiKeyTag" = EXCLUDED."apiKeyTag",
	"modelId" = EXCLUDED."modelId",
	"baseUrl" = EXCLUDED."baseUrl",
	"maxTokensPerRequest" = EXCLUDED."maxTokensPerRequest",
	"dailyRequestLimit" = EXCLUDED."dailyRequestLimit",
	enabled = EXCLUDED.enabled,
	"queryGenerationBackend" = EXCLUDED."queryGenerationBackend",
	"queryGenerationModel" = EXCLUDED."queryGenerationModel",
	"queryOptimizerEnabled" = EXCLUDED."queryOptimizerEnabled",
	"queryOptimizerBackend" = EXCLUDED."queryOptimizerBackend",
	"queryOptimizerModel" = EXCLUDED."queryOptimizerModel",
	temperature = EXCLUDED.temperature,
	"timeoutMs" = EXCLUDED."timeoutMs",
	"updatedAt" = NOW()
`, uuid.NewString(), tenantID, legacy.Provider, legacy.EncryptedAPIKey, legacy.APIKeyIV, legacy.APIKeyTag, legacy.ModelID, legacy.BaseURL, queryGeneration.MaxTokensPerRequest, queryGeneration.DailyRequestLimit, queryGeneration.Enabled, queryGeneration.Backend, queryGeneration.ModelID, queryOptimizer.Enabled, queryOptimizer.Backend, queryOptimizer.ModelID, temperature, timeoutMS); err != nil {
		return configResponse{}, fmt.Errorf("upsert tenant ai config: %w", err)
	}

	names := make([]string, 0, len(normalizedBackends))
	for _, backend := range normalizedBackends {
		names = append(names, backend.Name)
	}
	if _, err := tx.Exec(ctx, `
DELETE FROM "TenantAiBackend"
WHERE "tenantId" = $1
  AND NOT (name = ANY($2))
`, tenantID, names); err != nil {
		return configResponse{}, fmt.Errorf("delete stale ai backends: %w", err)
	}
	if len(normalizedBackends) == 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM "TenantAiBackend" WHERE "tenantId" = $1`, tenantID); err != nil {
			return configResponse{}, fmt.Errorf("clear ai backends: %w", err)
		}
	}

	for _, backend := range normalizedBackends {
		if _, err := tx.Exec(ctx, `
INSERT INTO "TenantAiBackend" (
	id,
	"tenantId",
	name,
	provider,
	"encryptedApiKey",
	"apiKeyIV",
	"apiKeyTag",
	"baseUrl",
	"defaultModel",
	"createdAt",
	"updatedAt"
)
VALUES (
	$1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NOW(), NOW()
)
ON CONFLICT ("tenantId", name) DO UPDATE SET
	provider = EXCLUDED.provider,
	"encryptedApiKey" = EXCLUDED."encryptedApiKey",
	"apiKeyIV" = EXCLUDED."apiKeyIV",
	"apiKeyTag" = EXCLUDED."apiKeyTag",
	"baseUrl" = EXCLUDED."baseUrl",
	"defaultModel" = EXCLUDED."defaultModel",
	"updatedAt" = NOW()
`, backend.ID, tenantID, backend.Name, backend.Provider, backend.EncryptedAPIKey, backend.APIKeyIV, backend.APIKeyTag, backend.BaseURL, backend.DefaultModel); err != nil {
			return configResponse{}, fmt.Errorf("upsert ai backend %q: %w", backend.Name, err)
		}
	}

	if err := s.insertAuditLog(ctx, userID, "APP_CONFIG_UPDATE", "ai_config", tenantID, map[string]any{
		"backendNames":       names,
		"queryGeneration":    map[string]any{"enabled": queryGeneration.Enabled, "backend": queryGeneration.Backend, "modelId": queryGeneration.ModelID},
		"queryOptimizer":     map[string]any{"enabled": queryOptimizer.Enabled, "backend": queryOptimizer.Backend, "modelId": queryOptimizer.ModelID},
		"temperature":        temperature,
		"timeoutMs":          timeoutMS,
		"configuredBackends": len(normalizedBackends),
	}, ""); err != nil {
		return configResponse{}, fmt.Errorf("audit ai config update: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return configResponse{}, fmt.Errorf("commit ai config update: %w", err)
	}

	return buildConfigResponse(storedAIConfig{
		QueryGeneration: queryGeneration,
		QueryOptimizer:  queryOptimizer,
		Temperature:     temperature,
		TimeoutMS:       timeoutMS,
	}, normalizedBackends, legacy), nil
}
