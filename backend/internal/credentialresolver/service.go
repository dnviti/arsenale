package credentialresolver

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Resolver struct {
	DB        *pgxpool.Pool
	Redis     *redis.Client
	ServerKey []byte
	VaultTTL  time.Duration
}

type RequestError struct {
	Status  int
	Message string
}

func (e *RequestError) Error() string {
	return e.Message
}

type SecretCredentials struct {
	Type         string
	Username     string
	Password     string
	Domain       string
	PrivateKey   string
	Passphrase   string
	SecretAccess string
}

type SecretSummary struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    *string        `json:"description"`
	Type           string         `json:"type"`
	Scope          string         `json:"scope"`
	TeamID         *string        `json:"teamId"`
	TenantID       *string        `json:"tenantId"`
	FolderID       *string        `json:"folderId"`
	Metadata       map[string]any `json:"metadata"`
	Tags           []string       `json:"tags"`
	IsFavorite     bool           `json:"isFavorite"`
	PwnedCount     int            `json:"pwnedCount"`
	ExpiresAt      *time.Time     `json:"expiresAt"`
	CurrentVersion int            `json:"currentVersion"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type SecretDetail struct {
	SecretSummary
	Data       json.RawMessage `json:"data"`
	Shared     bool            `json:"shared,omitempty"`
	Permission string          `json:"permission,omitempty"`
}

type SecretVersionUser struct {
	Email    string  `json:"email"`
	Username *string `json:"username"`
}

type SecretVersion struct {
	ID         string             `json:"id"`
	Version    int                `json:"version"`
	ChangedBy  string             `json:"changedBy"`
	ChangeNote *string            `json:"changeNote"`
	CreatedAt  time.Time          `json:"createdAt"`
	Changer    *SecretVersionUser `json:"changer,omitempty"`
}

type SecretManageAccess struct {
	ID       string
	Scope    string
	TeamID   *string
	TenantID *string
	TeamRole string
}

type secretRecord struct {
	ID            string
	Type          string
	Scope         string
	UserID        string
	TeamID        *string
	TenantID      *string
	TeamTenantID  *string
	EncryptedData string
	DataIV        string
	DataTag       string
}

type sharedSecretRecord struct {
	EncryptedData string
	DataIV        string
	DataTag       string
	Permission    string
}

type secretPayload struct {
	Type       string `json:"type"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Domain     string `json:"domain"`
	PrivateKey string `json:"privateKey"`
	Passphrase string `json:"passphrase"`
}

type encryptedField struct {
	Ciphertext string `json:"ciphertext"`
	IV         string `json:"iv"`
	Tag        string `json:"tag"`
}
