package main

import (
	"os"
	"path/filepath"
	"strings"
)

type gatewayRuntimeConfig struct {
	GRPCTLSCA             string
	GRPCTLSCert           string
	GRPCTLSKey            string
	GRPCClientCA          string
	GRPCServerCert        string
	GRPCServerKey         string
	GuacdTLSCert          string
	GuacdTLSKey           string
	OrchestratorResolv    string
	OrchestratorEgressNet string
}

func loadGatewayRuntimeConfig() gatewayRuntimeConfig {
	cfg := gatewayRuntimeConfig{
		GRPCTLSCA:             strings.TrimSpace(os.Getenv("GATEWAY_GRPC_TLS_CA")),
		GRPCTLSCert:           strings.TrimSpace(os.Getenv("GATEWAY_GRPC_TLS_CERT")),
		GRPCTLSKey:            strings.TrimSpace(os.Getenv("GATEWAY_GRPC_TLS_KEY")),
		GRPCClientCA:          strings.TrimSpace(os.Getenv("GATEWAY_GRPC_CLIENT_CA")),
		GRPCServerCert:        strings.TrimSpace(os.Getenv("GATEWAY_GRPC_SERVER_CERT")),
		GRPCServerKey:         strings.TrimSpace(os.Getenv("GATEWAY_GRPC_SERVER_KEY")),
		GuacdTLSCert:          strings.TrimSpace(os.Getenv("ORCHESTRATOR_GUACD_TLS_CERT")),
		GuacdTLSKey:           strings.TrimSpace(os.Getenv("ORCHESTRATOR_GUACD_TLS_KEY")),
		OrchestratorResolv:    strings.TrimSpace(os.Getenv("ORCHESTRATOR_RESOLV_CONF_PATH")),
		OrchestratorEgressNet: strings.TrimSpace(os.Getenv("ORCHESTRATOR_EGRESS_NETWORK")),
	}

	if cfg.GRPCTLSCert != "" {
		cfg.GRPCClientCA = existingPathOrValue(cfg.GRPCClientCA, filepath.Join(filepath.Dir(cfg.GRPCTLSCert), "client-ca.pem"))
		cfg.GRPCServerCert = existingPathOrValue(cfg.GRPCServerCert, filepath.Join(filepath.Dir(cfg.GRPCTLSCert), "server-cert.pem"))
	}
	if cfg.GRPCTLSKey != "" {
		cfg.GRPCServerKey = existingPathOrValue(cfg.GRPCServerKey, filepath.Join(filepath.Dir(cfg.GRPCTLSKey), "server-key.pem"))
	}
	cfg.GuacdTLSCert = existingPathOrValue(cfg.GuacdTLSCert, "/certs/guacd/server-cert.pem")
	cfg.GuacdTLSKey = existingPathOrValue(cfg.GuacdTLSKey, "/certs/guacd/server-key.pem")
	return cfg
}

func existingPathOrValue(value, candidate string) string {
	if value != "" {
		return value
	}
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

func closeRuntimeResources(closeFns []func()) {
	for i := len(closeFns) - 1; i >= 0; i-- {
		if closeFns[i] != nil {
			closeFns[i]()
		}
	}
}
