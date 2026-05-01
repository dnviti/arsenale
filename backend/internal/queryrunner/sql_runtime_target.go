package queryrunner

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	go_ora "github.com/sijms/go-ora/v2"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

type sqlTargetConn struct {
	db       *sql.DB
	conn     *sql.Conn
	protocol string
}

type objectRef struct {
	Schema string
	Name   string
	Column string
}

func openSQLTargetConn(ctx context.Context, target *contracts.DatabaseTarget) (*sqlTargetConn, error) {
	protocol := targetProtocol(target)
	if !isSQLProtocol(protocol) {
		return nil, fmt.Errorf("unsupported database protocol %q", target.Protocol)
	}

	driverName, dsn, err := sqlTargetDSN(target)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s connection: %w", protocol, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)
	db.SetConnMaxLifetime(30 * time.Second)

	pingCtx, pingCancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping %s: %w", protocol, err)
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("open %s dedicated connection: %w", protocol, err)
	}

	if err := applySQLSessionConfig(ctx, conn, target, protocol); err != nil {
		conn.Close()
		db.Close()
		return nil, err
	}

	return &sqlTargetConn{
		db:       db,
		conn:     conn,
		protocol: protocol,
	}, nil
}

func (c *sqlTargetConn) Close() {
	if c == nil {
		return
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	if c.db != nil {
		_ = c.db.Close()
	}
}

func sqlTargetDSN(target *contracts.DatabaseTarget) (string, string, error) {
	if target == nil {
		return "", "", fmt.Errorf("target is required")
	}

	protocol := targetProtocol(target)
	switch protocol {
	case protocolPostgreSQL:
		dsn, err := buildPostgresDSN(target)
		return "pgx", dsn, err
	case protocolMySQL:
		dsn, err := buildMySQLDSN(target)
		return "mysql", dsn, err
	case protocolMSSQL:
		dsn, err := buildMSSQLDSN(target)
		return "sqlserver", dsn, err
	case protocolOracle:
		dsn, err := buildOracleDSN(target)
		return "oracle", dsn, err
	default:
		return "", "", fmt.Errorf("unsupported database protocol %q", target.Protocol)
	}
}

func buildPostgresDSN(target *contracts.DatabaseTarget) (string, error) {
	if err := validateNetworkTarget(target); err != nil {
		return "", err
	}

	database := effectiveTargetDatabase(target)
	if database == "" {
		database = "postgres"
	}

	u := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(target.Username, target.Password),
		Host:   net.JoinHostPort(target.Host, strconv.Itoa(target.Port)),
		Path:   database,
	}
	query := u.Query()
	query.Set("application_name", "arsenale-query-runner")
	query.Set("connect_timeout", "10")
	if sslMode := normalizePostgresSSLMode(target.SSLMode); sslMode != "" {
		query.Set("sslmode", sslMode)
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func buildMySQLDSN(target *contracts.DatabaseTarget) (string, error) {
	if err := validateNetworkTarget(target); err != nil {
		return "", err
	}

	cfg := mysqlDriver.NewConfig()
	cfg.User = target.Username
	cfg.Passwd = target.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(target.Host, strconv.Itoa(target.Port))
	cfg.DBName = effectiveTargetDatabase(target)
	cfg.ParseTime = true
	cfg.Timeout = 10 * time.Second
	cfg.ReadTimeout = defaultQueryTimeout
	cfg.WriteTimeout = defaultQueryTimeout
	cfg.Params = map[string]string{
		"charset": "utf8mb4",
	}

	switch tlsMode := normalizeMySQLTLSConfig(target.SSLMode); tlsMode {
	case "":
		cfg.TLSConfig = "false"
	default:
		cfg.TLSConfig = tlsMode
	}

	return cfg.FormatDSN(), nil
}

func buildMSSQLDSN(target *contracts.DatabaseTarget) (string, error) {
	if err := validateNetworkTarget(target); err != nil {
		return "", err
	}

	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(target.Username, target.Password),
		Host:   net.JoinHostPort(target.Host, strconv.Itoa(target.Port)),
	}
	query := u.Query()
	if database := effectiveTargetDatabase(target); database != "" {
		query.Set("database", database)
	}
	if instance := strings.TrimSpace(target.MSSQLInstanceName); instance != "" {
		query.Set("instance", instance)
	}
	query.Set("app name", "arsenale-query-runner")
	query.Set("connection timeout", "10")

	switch strings.ToLower(strings.TrimSpace(target.SSLMode)) {
	case "", "disable", "disabled", "false", "off":
		query.Set("encrypt", "disable")
	case "require", "required", "true", "on":
		query.Set("encrypt", "true")
		query.Set("TrustServerCertificate", "true")
	default:
		query.Set("encrypt", "true")
		query.Set("TrustServerCertificate", "true")
	}

	u.RawQuery = query.Encode()
	return u.String(), nil
}

func buildOracleDSN(target *contracts.DatabaseTarget) (string, error) {
	connectionType := strings.ToLower(strings.TrimSpace(target.OracleConnectionType))
	switch connectionType {
	case "custom":
		if strings.TrimSpace(target.OracleConnectString) == "" {
			return "", fmt.Errorf("oracle custom connection string is required")
		}
		return go_ora.BuildJDBC(target.Username, target.Password, target.OracleConnectString, nil), nil
	case "tns":
		if descriptor := strings.TrimSpace(target.OracleTNSDescriptor); descriptor != "" {
			return go_ora.BuildJDBC(target.Username, target.Password, descriptor, nil), nil
		}
		if alias := strings.TrimSpace(target.OracleTNSAlias); alias != "" {
			return go_ora.BuildJDBC(target.Username, target.Password, alias, nil), nil
		}
		return "", fmt.Errorf("oracle tns alias or descriptor is required")
	default:
		if err := validateNetworkTarget(target); err != nil {
			return "", err
		}
		service := strings.TrimSpace(target.OracleServiceName)
		if service == "" {
			service = effectiveTargetDatabase(target)
		}
		options := map[string]string{}
		if sid := strings.TrimSpace(target.OracleSID); sid != "" {
			options["SID"] = sid
		}
		if role := strings.TrimSpace(target.OracleRole); role != "" && !strings.EqualFold(role, "normal") {
			options["DBA PRIVILEGE"] = strings.ToUpper(role)
		}
		if strings.TrimSpace(target.SSLMode) != "" && !strings.EqualFold(target.SSLMode, "disable") {
			options["AUTH TYPE"] = "TCPS"
		}
		return go_ora.BuildUrl(target.Host, target.Port, service, target.Username, target.Password, options), nil
	}
}

func validateNetworkTarget(target *contracts.DatabaseTarget) error {
	if target == nil {
		return fmt.Errorf("target is required")
	}
	if strings.TrimSpace(target.Host) == "" {
		return fmt.Errorf("target.host is required")
	}
	if target.Port <= 0 || target.Port > 65535 {
		return fmt.Errorf("target.port must be between 1 and 65535")
	}
	if strings.TrimSpace(target.Username) == "" {
		return fmt.Errorf("target.username is required")
	}
	return nil
}

func effectiveTargetDatabase(target *contracts.DatabaseTarget) string {
	if target == nil {
		return ""
	}
	if target.SessionConfig != nil && strings.TrimSpace(target.SessionConfig.ActiveDatabase) != "" {
		return strings.TrimSpace(target.SessionConfig.ActiveDatabase)
	}
	return strings.TrimSpace(target.Database)
}

func applySQLSessionConfig(ctx context.Context, conn *sql.Conn, target *contracts.DatabaseTarget, protocol string) error {
	if conn == nil {
		return fmt.Errorf("database connection is unavailable")
	}

	statements := buildTargetSessionInitStatements(target)
	if len(statements) == 0 {
		return nil
	}

	queryCtx, cancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer cancel()

	for _, statement := range statements {
		if _, err := conn.ExecContext(queryCtx, statement); err != nil {
			return fmt.Errorf("apply session config statement %q: %w", statement, err)
		}
	}
	return nil
}
