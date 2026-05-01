package publicshareapi

import (
	"database/sql"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var pinPattern = regexp.MustCompile(`^\d{4,8}$`)

type Service struct {
	DB *pgxpool.Pool
}

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string {
	return e.message
}

type shareInfoResponse struct {
	ID          string `json:"id"`
	SecretName  string `json:"secretName"`
	SecretType  string `json:"secretType"`
	HasPin      bool   `json:"hasPin"`
	ExpiresAt   string `json:"expiresAt"`
	IsExpired   bool   `json:"isExpired"`
	IsExhausted bool   `json:"isExhausted"`
	IsRevoked   bool   `json:"isRevoked"`
}

type shareAccessResponse struct {
	SecretName string         `json:"secretName"`
	SecretType string         `json:"secretType"`
	Data       map[string]any `json:"data"`
}

type shareRecord struct {
	ID             string
	SecretID       string
	SecretName     string
	SecretType     string
	EncryptedData  string
	DataIV         string
	DataTag        string
	HasPin         bool
	PinSalt        sql.NullString
	TokenSalt      sql.NullString
	ExpiresAt      time.Time
	MaxAccessCount sql.NullInt32
	AccessCount    int
	IsRevoked      bool
}
