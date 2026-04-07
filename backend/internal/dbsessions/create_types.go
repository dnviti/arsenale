package dbsessions

import (
	"time"

	"github.com/dnviti/arsenale/backend/internal/sessions"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

type requestError struct {
	status  int
	message string
}

func (e *requestError) Error() string {
	return e.message
}

type createRequest struct {
	ConnectionID  string                           `json:"connectionId"`
	Username      string                           `json:"username,omitempty"`
	Password      string                           `json:"password,omitempty"`
	SessionConfig *contracts.DatabaseSessionConfig `json:"sessionConfig,omitempty"`
}

type databaseSettings struct {
	Protocol             string `json:"protocol"`
	DatabaseName         string `json:"databaseName"`
	PersistExecutionPlan bool   `json:"persistExecutionPlan"`
	SSLMode              string `json:"sslMode"`
	OracleConnectionType string `json:"oracleConnectionType"`
	OracleSID            string `json:"oracleSid"`
	OracleServiceName    string `json:"oracleServiceName"`
	OracleRole           string `json:"oracleRole"`
	OracleTNSAlias       string `json:"oracleTnsAlias"`
	OracleTNSDescriptor  string `json:"oracleTnsDescriptor"`
	OracleConnectString  string `json:"oracleConnectString"`
	MSSQLInstanceName    string `json:"mssqlInstanceName"`
	MSSQLAuthMode        string `json:"mssqlAuthMode"`
	DB2DatabaseAlias     string `json:"db2DatabaseAlias"`
}

type gatewaySnapshot struct {
	ID             string
	Type           string
	Host           string
	Port           int
	IsManaged      bool
	DeploymentMode string
	TunnelEnabled  bool
	LBStrategy     string
}

type managedGatewayInstance struct {
	ID             string
	Host           string
	Port           int
	ActiveSessions int
	CreatedAt      time.Time
}

type databaseRoute struct {
	GatewayID       string
	InstanceID      string
	ProxyHost       string
	ProxyPort       int
	RoutingDecision *sessions.RoutingDecision
}
