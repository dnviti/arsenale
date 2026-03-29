package contracts

type AIProviderID string

const (
	AIProviderNone             AIProviderID = "none"
	AIProviderAnthropic        AIProviderID = "anthropic"
	AIProviderOpenAI           AIProviderID = "openai"
	AIProviderOllama           AIProviderID = "ollama"
	AIProviderOpenAICompatible AIProviderID = "openai-compatible"
)

type AIProviderDescriptor struct {
	ID                 AIProviderID `json:"id"`
	SupportsEmbeddings bool         `json:"supportsEmbeddings"`
	SupportsResponses  bool         `json:"supportsResponses"`
	RequiresAPIKey     bool         `json:"requiresApiKey"`
	RequiresBaseURL    bool         `json:"requiresBaseUrl"`
	DefaultModel       string       `json:"defaultModel"`
}

type TenantAIConfig struct {
	TenantID            string       `json:"tenantId"`
	Provider            AIProviderID `json:"provider"`
	HasAPIKey           bool         `json:"hasApiKey"`
	ModelID             string       `json:"modelId"`
	BaseURL             string       `json:"baseUrl,omitempty"`
	MaxTokensPerRequest int          `json:"maxTokensPerRequest"`
	DailyRequestLimit   int          `json:"dailyRequestLimit"`
	Enabled             bool         `json:"enabled"`
}

type TenantAIConfigUpdate struct {
	Provider            *AIProviderID `json:"provider,omitempty"`
	APIKey              *string       `json:"apiKey,omitempty"`
	ModelID             *string       `json:"modelId,omitempty"`
	BaseURL             *string       `json:"baseUrl,omitempty"`
	MaxTokensPerRequest *int          `json:"maxTokensPerRequest,omitempty"`
	DailyRequestLimit   *int          `json:"dailyRequestLimit,omitempty"`
	Enabled             *bool         `json:"enabled,omitempty"`
}

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}
