package importexportapi

import (
	"encoding/json"
	"time"

	"github.com/dnviti/arsenale/backend/internal/connections"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	DB                  *pgxpool.Pool
	Redis               *redis.Client
	ServerEncryptionKey []byte
	Connections         *connections.Service
}

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string {
	return e.message
}

type exportPayload struct {
	Format             string   `json:"format"`
	IncludeCredentials bool     `json:"includeCredentials"`
	ConnectionIDs      []string `json:"connectionIds"`
	FolderID           *string  `json:"folderId"`
}

type encryptedField struct {
	Ciphertext string `json:"ciphertext"`
	IV         string `json:"iv"`
	Tag        string `json:"tag"`
}

type exportConnection struct {
	ID                    string          `json:"id"`
	Name                  string          `json:"name"`
	Type                  string          `json:"type"`
	Host                  string          `json:"host"`
	Port                  int             `json:"port"`
	Description           *string         `json:"description"`
	IsFavorite            bool            `json:"isFavorite"`
	EnableDrive           bool            `json:"enableDrive"`
	FolderName            *string         `json:"folderName"`
	SSHTerminalConfig     json.RawMessage `json:"sshTerminalConfig,omitempty"`
	RDPSettings           json.RawMessage `json:"rdpSettings,omitempty"`
	VNCSettings           json.RawMessage `json:"vncSettings,omitempty"`
	DefaultCredentialMode *string         `json:"defaultCredentialMode"`
	CreatedAt             time.Time       `json:"createdAt"`
	UpdatedAt             time.Time       `json:"updatedAt"`
	Username              *string         `json:"username,omitempty"`
	Password              *string         `json:"password,omitempty"`
	Domain                *string         `json:"domain,omitempty"`
}

type importResult struct {
	Imported int                 `json:"imported"`
	Skipped  int                 `json:"skipped"`
	Failed   int                 `json:"failed"`
	Errors   []importResultError `json:"errors"`
}

type importResultError struct {
	Row      *int   `json:"row,omitempty"`
	Filename string `json:"filename"`
	Error    string `json:"error"`
}

type importRecord struct {
	Name        string
	Type        string
	Host        string
	Port        int
	Username    string
	Password    string
	Domain      *string
	FolderName  *string
	Description *string
}

type rawConnectionRow struct {
	exportConnection
	EncryptedUsername *encryptedField
	EncryptedPassword *encryptedField
	EncryptedDomain   *encryptedField
}

type scanRow interface {
	Scan(...any) error
}

func (r rawConnectionRow) toExportConnection() exportConnection {
	return exportConnection{
		ID:                    r.ID,
		Name:                  r.Name,
		Type:                  r.Type,
		Host:                  r.Host,
		Port:                  r.Port,
		Description:           r.Description,
		IsFavorite:            r.IsFavorite,
		EnableDrive:           r.EnableDrive,
		FolderName:            r.FolderName,
		SSHTerminalConfig:     r.SSHTerminalConfig,
		RDPSettings:           r.RDPSettings,
		VNCSettings:           r.VNCSettings,
		DefaultCredentialMode: r.DefaultCredentialMode,
		CreatedAt:             r.CreatedAt,
		UpdatedAt:             r.UpdatedAt,
		Username:              r.Username,
		Password:              r.Password,
		Domain:                r.Domain,
	}
}
