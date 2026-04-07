package oauthapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/authservice"
	"github.com/dnviti/arsenale/backend/internal/tenantvaultapi"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	DB                 *pgxpool.Pool
	Redis              *redis.Client
	ServerKey          []byte
	VaultTTL           time.Duration
	ClientURL          string
	HTTPClient         *http.Client
	Auth               *authservice.Service
	Authenticator      *authn.Authenticator
	TenantVaultService *tenantvaultapi.Service
}

type requestError struct {
	status  int
	message string
}

type linkedAccount struct {
	ID            string    `json:"id"`
	Provider      string    `json:"provider"`
	ProviderEmail *string   `json:"providerEmail"`
	CreatedAt     time.Time `json:"createdAt"`
}

type linkCodeEntry struct {
	UserID    string `json:"userId"`
	ExpiresAt int64  `json:"expiresAt"`
}

type authCodeEntry struct {
	AccessToken     string `json:"accessToken"`
	CSRFToken       string `json:"csrfToken"`
	NeedsVaultSetup bool   `json:"needsVaultSetup"`
	UserID          string `json:"userId"`
	Email           string `json:"email"`
	Username        string `json:"username"`
	AvatarData      string `json:"avatarData"`
	TenantID        string `json:"tenantId"`
	TenantRole      string `json:"tenantRole"`
	ExpiresAt       int64  `json:"expiresAt"`
}

type encryptedField struct {
	Ciphertext string `json:"ciphertext"`
	IV         string `json:"iv"`
	Tag        string `json:"tag"`
}

type oidcDiscoveryDocument struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
}

type providerAuthConfig struct {
	Enabled      bool
	ClientID     string
	ClientSecret string
	CallbackURL  string
	Scopes       []string
	AuthURL      string
	Params       map[string]string
}

type providerAuthOptions struct {
	State string
}

var (
	linkCodeMu     sync.Mutex
	linkCodeStore  = map[string]linkCodeEntry{}
	relayCodeMu    sync.Mutex
	relayCodeStore = map[string]linkCodeEntry{}
	oidcPKCEMu     sync.Mutex
	oidcPKCEStore  = map[string]linkCodeEntry{}
	authCodeMu     sync.Mutex
	authCodeStore  = map[string]authCodeEntry{}
)

const (
	linkCodeTTL  = 60 * time.Second
	relayCodeTTL = 5 * time.Minute
	oidcPKCETTL  = 10 * time.Minute
	authCodeTTL  = 60 * time.Second
)

func (e *requestError) Error() string {
	return e.message
}
