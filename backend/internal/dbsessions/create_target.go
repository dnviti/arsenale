package dbsessions

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/sshsessions"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func parseDatabaseSettings(raw json.RawMessage) databaseSettings {
	settings := databaseSettings{Protocol: "postgresql"}
	if len(raw) == 0 {
		return settings
	}
	if err := json.Unmarshal(raw, &settings); err != nil {
		return settings
	}
	settings.Protocol = normalizeDatabaseProtocol(settings.Protocol)
	return settings
}

func normalizeDatabaseProtocol(protocol string) string {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	switch protocol {
	case "", "postgres", "postgresql":
		return "postgresql"
	case "mariadb":
		return "mysql"
	case "sqlserver":
		return "mssql"
	case "mongo":
		return "mongodb"
	default:
		return protocol
	}
}

func hasOverrideCredentials(username, password string) bool {
	return strings.TrimSpace(username) != "" && strings.TrimSpace(password) != ""
}

func shouldUseOwnedDatabaseSessionRuntime(dbProtocol string, usesOverrideCredentials bool) bool {
	_ = usesOverrideCredentials
	if strings.EqualFold(strings.TrimSpace(os.Getenv("DB_PROXY_QUERY_RUNTIME_ENABLED")), "false") {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("GO_QUERY_RUNNER_ENABLED")), "false") {
		return false
	}
	switch normalizeDatabaseProtocol(dbProtocol) {
	case "postgresql", "mysql", "mssql", "oracle", "mongodb":
		return true
	default:
		return false
	}
}

func buildSessionMetadata(connectionHost string, connectionPort int, resolvedHost string, resolvedPort int, dbProtocol string, databaseName string, username string, settings databaseSettings, sessionConfig *contracts.DatabaseSessionConfig, usesOverrideCredentials bool) map[string]any {
	metadata := map[string]any{
		"host":                    strings.TrimSpace(connectionHost),
		"port":                    connectionPort,
		"dbProtocol":              normalizeDatabaseProtocol(dbProtocol),
		"databaseName":            strings.TrimSpace(databaseName),
		"username":                strings.TrimSpace(username),
		"resolvedHost":            strings.TrimSpace(resolvedHost),
		"resolvedPort":            resolvedPort,
		"usesOverrideCredentials": usesOverrideCredentials,
	}

	addMetadataString(metadata, "sslMode", settings.SSLMode)
	addMetadataString(metadata, "oracleConnectionType", settings.OracleConnectionType)
	addMetadataString(metadata, "oracleSid", settings.OracleSID)
	addMetadataString(metadata, "oracleServiceName", settings.OracleServiceName)
	addMetadataString(metadata, "oracleRole", settings.OracleRole)
	addMetadataString(metadata, "oracleTnsAlias", settings.OracleTNSAlias)
	addMetadataString(metadata, "oracleTnsDescriptor", settings.OracleTNSDescriptor)
	addMetadataString(metadata, "oracleConnectString", settings.OracleConnectString)
	addMetadataString(metadata, "mssqlInstanceName", settings.MSSQLInstanceName)
	addMetadataString(metadata, "mssqlAuthMode", settings.MSSQLAuthMode)
	addMetadataString(metadata, "db2DatabaseAlias", settings.DB2DatabaseAlias)

	if sessionConfig != nil {
		metadata["sessionConfig"] = normalizeSessionConfig(*sessionConfig)
	}

	return metadata
}

func addMetadataString(metadata map[string]any, key, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		metadata[key] = value
	}
}

func buildDatabaseTarget(host string, port int, dbProtocol string, databaseName string, credentials sshsessions.ResolvedCredentials, settings databaseSettings, sessionConfig *contracts.DatabaseSessionConfig) *contracts.DatabaseTarget {
	if port <= 0 {
		return nil
	}
	target := &contracts.DatabaseTarget{
		Protocol:             normalizeDatabaseProtocol(dbProtocol),
		Host:                 strings.TrimSpace(host),
		Port:                 port,
		Database:             strings.TrimSpace(databaseName),
		SSLMode:              strings.TrimSpace(settings.SSLMode),
		Username:             strings.TrimSpace(credentials.Username),
		Password:             credentials.Password,
		OracleConnectionType: strings.TrimSpace(settings.OracleConnectionType),
		OracleSID:            strings.TrimSpace(settings.OracleSID),
		OracleServiceName:    strings.TrimSpace(settings.OracleServiceName),
		OracleRole:           strings.TrimSpace(settings.OracleRole),
		OracleTNSAlias:       strings.TrimSpace(settings.OracleTNSAlias),
		OracleTNSDescriptor:  strings.TrimSpace(settings.OracleTNSDescriptor),
		OracleConnectString:  strings.TrimSpace(settings.OracleConnectString),
		MSSQLInstanceName:    strings.TrimSpace(settings.MSSQLInstanceName),
		MSSQLAuthMode:        strings.TrimSpace(settings.MSSQLAuthMode),
		SessionConfig:        sessionConfig,
	}
	if sessionConfig != nil && strings.TrimSpace(sessionConfig.ActiveDatabase) != "" {
		target.Database = strings.TrimSpace(sessionConfig.ActiveDatabase)
	}
	return target
}
