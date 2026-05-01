package adminapi

import (
	"github.com/dnviti/arsenale/backend/internal/tenantauth"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	DB         *pgxpool.Pool
	TenantAuth tenantauth.Service
}

type requestError struct {
	status  int
	message string
}

type emailStatusResponse struct {
	Provider   string `json:"provider"`
	Configured bool   `json:"configured"`
	From       string `json:"from"`
	Host       string `json:"host,omitempty"`
	Port       int    `json:"port,omitempty"`
	Secure     bool   `json:"secure,omitempty"`
}

type appConfigResponse struct {
	SelfSignupEnabled   bool `json:"selfSignupEnabled"`
	SelfSignupEnvLocked bool `json:"selfSignupEnvLocked"`
}

type dbStatusResponse struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Database  string `json:"database"`
	Connected bool   `json:"connected"`
	Version   any    `json:"version"`
}

type authProviderDetail struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Enabled      bool   `json:"enabled"`
	ProviderName string `json:"providerName,omitempty"`
}

func (e *requestError) Error() string {
	return e.message
}
