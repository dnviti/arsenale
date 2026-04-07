package systemsettingsapi

import (
	"github.com/dnviti/arsenale/backend/internal/tenantauth"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sensitiveMask = "••••••••"

type SettingType string

type SettingDef struct {
	Key             string      `json:"key"`
	EnvVar          string      `json:"envVar"`
	ConfigPath      string      `json:"configPath,omitempty"`
	Type            SettingType `json:"type"`
	Default         any         `json:"default"`
	Options         []string    `json:"options,omitempty"`
	Group           string      `json:"group"`
	Label           string      `json:"label"`
	Description     string      `json:"description"`
	MinEditRole     string      `json:"minEditRole"`
	RestartRequired bool        `json:"restartRequired,omitempty"`
	Sensitive       bool        `json:"sensitive,omitempty"`
}

type SettingValue struct {
	Key             string      `json:"key"`
	Value           any         `json:"value"`
	Source          string      `json:"source"`
	EnvLocked       bool        `json:"envLocked"`
	CanEdit         bool        `json:"canEdit"`
	Type            SettingType `json:"type"`
	Default         any         `json:"default"`
	Options         []string    `json:"options,omitempty"`
	Group           string      `json:"group"`
	Label           string      `json:"label"`
	Description     string      `json:"description"`
	RestartRequired bool        `json:"restartRequired"`
	Sensitive       bool        `json:"sensitive"`
}

type SettingGroup struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Order int    `json:"order"`
}

type updateResult struct {
	Key     string `json:"key"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type Service struct {
	DB         *pgxpool.Pool
	TenantAuth tenantauth.Service
}

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string {
	return e.message
}
