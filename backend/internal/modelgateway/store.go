package modelgateway

import (
	"context"
	"errors"
	"fmt"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) Enabled() bool {
	return s != nil && s.db != nil
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	if !s.Enabled() {
		return nil
	}

	_, err := s.db.Exec(ctx, `
CREATE TABLE IF NOT EXISTS "TenantAiConfig" (
  id TEXT PRIMARY KEY,
  "tenantId" TEXT NOT NULL UNIQUE,
  provider TEXT NOT NULL DEFAULT 'none',
  "encryptedApiKey" TEXT,
  "apiKeyIV" TEXT,
  "apiKeyTag" TEXT,
  "modelId" TEXT NOT NULL DEFAULT 'claude-sonnet-4-20250514',
  "baseUrl" TEXT,
  "maxTokensPerRequest" INTEGER NOT NULL DEFAULT 4000,
  "dailyRequestLimit" INTEGER NOT NULL DEFAULT 100,
  enabled BOOLEAN NOT NULL DEFAULT false,
  "createdAt" TIMESTAMP NOT NULL DEFAULT now(),
  "updatedAt" TIMESTAMP NOT NULL DEFAULT now()
)
`)
	if err != nil {
		return fmt.Errorf("ensure TenantAiConfig schema: %w", err)
	}

	return nil
}

func (s *Store) GetConfig(ctx context.Context, tenantID string) (contracts.TenantAIConfig, error) {
	if !s.Enabled() {
		return contracts.TenantAIConfig{}, errors.New("model gateway store is not configured")
	}

	row := s.db.QueryRow(ctx, `
SELECT "tenantId", provider, COALESCE("encryptedApiKey", ''), "modelId", COALESCE("baseUrl", ''), "maxTokensPerRequest", "dailyRequestLimit", enabled
FROM "TenantAiConfig"
WHERE "tenantId" = $1
`, tenantID)

	config, err := scanConfig(row)
	if err != nil {
		return contracts.TenantAIConfig{}, err
	}
	return config, nil
}

func (s *Store) UpsertConfig(ctx context.Context, tenantID string, update contracts.TenantAIConfigUpdate, encryptionKey []byte) (contracts.TenantAIConfig, error) {
	if !s.Enabled() {
		return contracts.TenantAIConfig{}, errors.New("model gateway store is not configured")
	}

	current, err := s.GetConfig(ctx, tenantID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return contracts.TenantAIConfig{}, fmt.Errorf("load current tenant ai config: %w", err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		current = contracts.TenantAIConfig{
			TenantID:            tenantID,
			Provider:            contracts.AIProviderNone,
			ModelID:             "",
			MaxTokensPerRequest: 4000,
			DailyRequestLimit:   100,
			Enabled:             false,
		}
	}

	next := current
	next.TenantID = tenantID
	if update.Provider != nil {
		next.Provider = *update.Provider
	}
	if update.ModelID != nil {
		next.ModelID = *update.ModelID
	}
	if update.BaseURL != nil {
		next.BaseURL = *update.BaseURL
	}
	if update.MaxTokensPerRequest != nil {
		next.MaxTokensPerRequest = *update.MaxTokensPerRequest
	}
	if update.DailyRequestLimit != nil {
		next.DailyRequestLimit = *update.DailyRequestLimit
	}
	if update.Enabled != nil {
		next.Enabled = *update.Enabled
	}

	apiKeyProvided := current.HasAPIKey
	encryptedAPIKey := ""
	apiKeyIV := ""
	apiKeyTag := ""
	if update.APIKey != nil {
		if *update.APIKey == "" {
			apiKeyProvided = false
		} else {
			if len(encryptionKey) == 0 {
				return contracts.TenantAIConfig{}, errors.New("SERVER_ENCRYPTION_KEY is required to store apiKey")
			}
			ciphertext, iv, tag, encErr := EncryptAPIKey(*update.APIKey, encryptionKey)
			if encErr != nil {
				return contracts.TenantAIConfig{}, encErr
			}
			encryptedAPIKey = ciphertext
			apiKeyIV = iv
			apiKeyTag = tag
			apiKeyProvided = true
		}
	} else if current.HasAPIKey {
		encryptedAPIKey = preserveMarker
		apiKeyIV = preserveMarker
		apiKeyTag = preserveMarker
	}

	if validation := ValidateConfig(next, apiKeyProvided); !validation.Valid {
		return contracts.TenantAIConfig{}, fmt.Errorf("invalid tenant ai config: %s", validation.Errors[0])
	}

	if encryptedAPIKey == preserveMarker {
		row := s.db.QueryRow(ctx, `
INSERT INTO "TenantAiConfig" (
  id, "tenantId", provider, "modelId", "baseUrl", "maxTokensPerRequest", "dailyRequestLimit", enabled, "createdAt", "updatedAt"
) VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, now(), now())
ON CONFLICT ("tenantId") DO UPDATE SET
  provider = EXCLUDED.provider,
  "modelId" = EXCLUDED."modelId",
  "baseUrl" = EXCLUDED."baseUrl",
  "maxTokensPerRequest" = EXCLUDED."maxTokensPerRequest",
  "dailyRequestLimit" = EXCLUDED."dailyRequestLimit",
  enabled = EXCLUDED.enabled,
  "updatedAt" = now()
RETURNING "tenantId", provider, COALESCE("encryptedApiKey", ''), "modelId", COALESCE("baseUrl", ''), "maxTokensPerRequest", "dailyRequestLimit", enabled
`, uuid.NewString(), tenantID, next.Provider, next.ModelID, next.BaseURL, next.MaxTokensPerRequest, next.DailyRequestLimit, next.Enabled)

		return scanConfig(row)
	}

	row := s.db.QueryRow(ctx, `
INSERT INTO "TenantAiConfig" (
  id, "tenantId", provider, "encryptedApiKey", "apiKeyIV", "apiKeyTag", "modelId", "baseUrl", "maxTokensPerRequest", "dailyRequestLimit", enabled, "createdAt", "updatedAt"
) VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7, NULLIF($8, ''), $9, $10, $11, now(), now())
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
  "updatedAt" = now()
RETURNING "tenantId", provider, COALESCE("encryptedApiKey", ''), "modelId", COALESCE("baseUrl", ''), "maxTokensPerRequest", "dailyRequestLimit", enabled
`, uuid.NewString(), tenantID, next.Provider, encryptedAPIKey, apiKeyIV, apiKeyTag, next.ModelID, next.BaseURL, next.MaxTokensPerRequest, next.DailyRequestLimit, next.Enabled)

	return scanConfig(row)
}

const preserveMarker = "__preserve__"

type configScanner interface {
	Scan(dest ...any) error
}

func scanConfig(scanner configScanner) (contracts.TenantAIConfig, error) {
	var (
		config          contracts.TenantAIConfig
		encryptedAPIKey string
		baseURL         string
	)

	if err := scanner.Scan(
		&config.TenantID,
		&config.Provider,
		&encryptedAPIKey,
		&config.ModelID,
		&baseURL,
		&config.MaxTokensPerRequest,
		&config.DailyRequestLimit,
		&config.Enabled,
	); err != nil {
		return contracts.TenantAIConfig{}, err
	}

	config.HasAPIKey = encryptedAPIKey != ""
	config.BaseURL = baseURL

	return config, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
