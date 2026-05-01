package externalvaultapi

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedAuthMethods = map[string]map[string]struct{}{
	"HASHICORP_VAULT": {
		"TOKEN":   {},
		"APPROLE": {},
	},
	"AWS_SECRETS_MANAGER": {
		"IAM_ACCESS_KEY": {},
		"IAM_ROLE":       {},
	},
	"AZURE_KEY_VAULT": {
		"CLIENT_CREDENTIALS": {},
		"MANAGED_IDENTITY":   {},
	},
	"GCP_SECRET_MANAGER": {
		"SERVICE_ACCOUNT_KEY": {},
		"WORKLOAD_IDENTITY":   {},
	},
	"CYBERARK_CONJUR": {
		"CONJUR_API_KEY":   {},
		"CONJUR_AUTHN_K8S": {},
	},
}

type Service struct {
	DB                  *pgxpool.Pool
	ServerEncryptionKey []byte
}

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string {
	return e.message
}

type providerResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	ProviderType    string    `json:"providerType"`
	ServerURL       string    `json:"serverUrl"`
	AuthMethod      string    `json:"authMethod"`
	Namespace       *string   `json:"namespace"`
	MountPath       string    `json:"mountPath"`
	CacheTTLSeconds int       `json:"cacheTtlSeconds"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	CACertificate   *string   `json:"caCertificate,omitempty"`
	HasAuthPayload  bool      `json:"hasApiToken,omitempty"`
}

type providerRecord struct {
	ID                   string
	Name                 string
	ProviderType         string
	ServerURL            string
	AuthMethod           string
	Namespace            *string
	MountPath            string
	EncryptedAuthPayload string
	AuthPayloadIV        string
	AuthPayloadTag       string
	CACertificate        *string
	CacheTTLSeconds      int
	Enabled              bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type providerCreatePayload struct {
	Name            string  `json:"name"`
	ProviderType    string  `json:"providerType"`
	ServerURL       string  `json:"serverUrl"`
	AuthMethod      string  `json:"authMethod"`
	Namespace       *string `json:"namespace"`
	MountPath       *string `json:"mountPath"`
	AuthPayload     string  `json:"authPayload"`
	CACertificate   *string `json:"caCertificate"`
	CacheTTLSeconds *int    `json:"cacheTtlSeconds"`
}

type providerUpdatePayload struct {
	Name            *string `json:"name"`
	ProviderType    *string `json:"providerType"`
	ServerURL       *string `json:"serverUrl"`
	AuthMethod      *string `json:"authMethod"`
	Namespace       *string `json:"namespace"`
	MountPath       *string `json:"mountPath"`
	AuthPayload     *string `json:"authPayload"`
	CACertificate   *string `json:"caCertificate"`
	CacheTTLSeconds *int    `json:"cacheTtlSeconds"`
	Enabled         *bool   `json:"enabled"`
}

type normalizedPayload struct {
	Name            string
	ProviderType    string
	ServerURL       string
	AuthMethod      string
	Namespace       *string
	MountPath       string
	AuthPayload     string
	CACertificate   *string
	CacheTTLSeconds int
	Enabled         bool
}

type encryptedField struct {
	Ciphertext string
	IV         string
	Tag        string
}
