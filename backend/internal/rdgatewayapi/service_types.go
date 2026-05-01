package rdgatewayapi

import (
	"regexp"

	"github.com/dnviti/arsenale/backend/internal/connections"
	"github.com/jackc/pgx/v5/pgxpool"
)

var hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)

type Config struct {
	Enabled            bool   `json:"enabled"`
	ExternalHostname   string `json:"externalHostname"`
	Port               int    `json:"port"`
	IdleTimeoutSeconds int    `json:"idleTimeoutSeconds"`
}

type Status struct {
	ActiveTunnels  int `json:"activeTunnels"`
	ActiveChannels int `json:"activeChannels"`
}

type updateConfigRequest struct {
	Enabled            *bool   `json:"enabled"`
	ExternalHostname   *string `json:"externalHostname"`
	Port               *int    `json:"port"`
	IdleTimeoutSeconds *int    `json:"idleTimeoutSeconds"`
}

type Service struct {
	DB          *pgxpool.Pool
	Connections connections.Service
}

type rdpFileParams struct {
	ConnectionName  string
	TargetHost      string
	TargetPort      int
	GatewayHostname string
	GatewayPort     int
	ScreenMode      int
	DesktopWidth    int
	DesktopHeight   int
	Username        string
	Domain          string
}
