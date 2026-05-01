package modelgateway

import (
	"context"
	"errors"
	"fmt"

	modelgatewaydb "github.com/dnviti/arsenale/backend/internal/modelgateway/dbgen"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db      *pgxpool.Pool
	queries *modelgatewaydb.Queries
}

func NewStore(db *pgxpool.Pool) *Store {
	store := &Store{db: db}
	if db != nil {
		store.queries = modelgatewaydb.New(db)
	}
	return store
}

func (s *Store) Enabled() bool {
	return s != nil && s.db != nil
}

func (s *Store) GetConfig(ctx context.Context, tenantID string) (contracts.TenantAIConfig, error) {
	if !s.Enabled() {
		return contracts.TenantAIConfig{}, errors.New("model gateway store is not configured")
	}

	row, err := s.queries.GetConfig(ctx, tenantID)
	if err != nil {
		return contracts.TenantAIConfig{}, err
	}
	return mapConfig(
		row.TenantId,
		row.Provider,
		row.EncryptedApiKey,
		row.ModelId,
		row.BaseUrl,
		row.MaxTokensPerRequest,
		row.DailyRequestLimit,
		row.Enabled,
	), nil
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
		row, err := s.queries.UpsertConfigPreserveKey(ctx, modelgatewaydb.UpsertConfigPreserveKeyParams{
			ID:                  uuid.NewString(),
			TenantID:            tenantID,
			Provider:            string(next.Provider),
			ModelID:             next.ModelID,
			BaseUrl:             nullableValue(next.BaseURL),
			MaxTokensPerRequest: int32(next.MaxTokensPerRequest),
			DailyRequestLimit:   int32(next.DailyRequestLimit),
			Enabled:             next.Enabled,
		})
		if err != nil {
			return contracts.TenantAIConfig{}, err
		}
		return mapConfig(
			row.TenantId,
			row.Provider,
			row.EncryptedApiKey,
			row.ModelId,
			row.BaseUrl,
			row.MaxTokensPerRequest,
			row.DailyRequestLimit,
			row.Enabled,
		), nil
	}

	row, err := s.queries.UpsertConfig(ctx, modelgatewaydb.UpsertConfigParams{
		ID:                  uuid.NewString(),
		TenantID:            tenantID,
		Provider:            string(next.Provider),
		EncryptedApiKey:     nullableValue(encryptedAPIKey),
		ApiKeyIv:            nullableValue(apiKeyIV),
		ApiKeyTag:           nullableValue(apiKeyTag),
		ModelID:             next.ModelID,
		BaseUrl:             nullableValue(next.BaseURL),
		MaxTokensPerRequest: int32(next.MaxTokensPerRequest),
		DailyRequestLimit:   int32(next.DailyRequestLimit),
		Enabled:             next.Enabled,
	})
	if err != nil {
		return contracts.TenantAIConfig{}, err
	}

	return mapConfig(
		row.TenantId,
		row.Provider,
		row.EncryptedApiKey,
		row.ModelId,
		row.BaseUrl,
		row.MaxTokensPerRequest,
		row.DailyRequestLimit,
		row.Enabled,
	), nil
}

const preserveMarker = "__preserve__"

func nullableValue(value string) any {
	return value
}

func mapConfig(tenantID, provider, encryptedAPIKey, modelID, baseURL string, maxTokensPerRequest, dailyRequestLimit int32, enabled bool) contracts.TenantAIConfig {
	return contracts.TenantAIConfig{
		TenantID:            tenantID,
		Provider:            contracts.AIProviderID(provider),
		HasAPIKey:           encryptedAPIKey != "",
		ModelID:             modelID,
		BaseURL:             baseURL,
		MaxTokensPerRequest: int(maxTokensPerRequest),
		DailyRequestLimit:   int(dailyRequestLimit),
		Enabled:             enabled,
	}
}

func IsNotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
