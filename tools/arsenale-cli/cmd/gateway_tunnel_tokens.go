package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type tunnelTokenBundle struct {
	Token               string `json:"token"`
	GatewayID           string `json:"gatewayId"`
	GatewayType         string `json:"gatewayType"`
	TunnelLocalHost     string `json:"tunnelLocalHost"`
	TunnelLocalPort     int    `json:"tunnelLocalPort"`
	TunnelClientCert    string `json:"tunnelClientCert"`
	TunnelClientKey     string `json:"tunnelClientKey"`
	TunnelClientCertExp string `json:"tunnelClientCertExp"`
}

func runGwTunnelTokenCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/gateways/%s/tunnel-token", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if gwTunnelTokenEnv || strings.TrimSpace(gwTunnelBundleDir) != "" {
		bundle, err := parseTunnelTokenBundle(body)
		if err != nil {
			fatal("%v", err)
		}
		certFile := "./certs/tunnel-client-cert.pem"
		keyFile := "./certs/tunnel-client-key.pem"
		envContent := tunnelTokenEnvContent(bundle, cfg.ServerURL, certFile, keyFile)
		if strings.TrimSpace(gwTunnelBundleDir) != "" {
			envPath, err := writeTunnelTokenBundle(gwTunnelBundleDir, bundle, cfg.ServerURL)
			if err != nil {
				fatal("%v", err)
			}
			if !quiet && !gwTunnelTokenEnv {
				fmt.Fprintf(os.Stdout, "Tunnel bundle written to %s\n", envPath)
			}
		}
		if gwTunnelTokenEnv {
			fmt.Fprint(os.Stdout, envContent)
		}
		return
	}
	printer().PrintCreated(body, "token")
}

func parseTunnelTokenBundle(body []byte) (tunnelTokenBundle, error) {
	var bundle tunnelTokenBundle
	if err := json.Unmarshal(body, &bundle); err != nil {
		return tunnelTokenBundle{}, fmt.Errorf("parse tunnel token response: %w", err)
	}
	if strings.TrimSpace(bundle.Token) == "" {
		return tunnelTokenBundle{}, fmt.Errorf("tunnel token response did not include token")
	}
	if strings.TrimSpace(bundle.GatewayID) == "" {
		return tunnelTokenBundle{}, fmt.Errorf("tunnel token response did not include gatewayId")
	}
	if strings.TrimSpace(bundle.TunnelClientCert) == "" || strings.TrimSpace(bundle.TunnelClientKey) == "" {
		return tunnelTokenBundle{}, fmt.Errorf("tunnel token response did not include client certificate material")
	}
	return bundle, nil
}

func tunnelTokenEnvContent(bundle tunnelTokenBundle, serverURL, certFile, keyFile string) string {
	localHost := strings.TrimSpace(bundle.TunnelLocalHost)
	if localHost == "" {
		localHost = "127.0.0.1"
	}
	localPort := bundle.TunnelLocalPort
	if localPort <= 0 {
		localPort = 4822
	}
	lines := []string{
		envLine("TUNNEL_SERVER_URL", strings.TrimRight(strings.TrimSpace(serverURL), "/")),
		envLine("TUNNEL_TOKEN", bundle.Token),
		envLine("TUNNEL_GATEWAY_ID", bundle.GatewayID),
		envLine("TUNNEL_LOCAL_HOST", localHost),
		envLine("TUNNEL_LOCAL_PORT", strconv.Itoa(localPort)),
		envLine("TUNNEL_CLIENT_CERT_FILE", certFile),
		envLine("TUNNEL_CLIENT_KEY_FILE", keyFile),
	}
	return strings.Join(lines, "\n") + "\n"
}

func writeTunnelTokenBundle(bundleDir string, bundle tunnelTokenBundle, serverURL string) (string, error) {
	bundleDir = strings.TrimSpace(bundleDir)
	if bundleDir == "" {
		return "", fmt.Errorf("bundle directory is required")
	}
	certsDir := filepath.Join(bundleDir, "certs")
	if err := os.MkdirAll(certsDir, 0700); err != nil {
		return "", fmt.Errorf("create certs directory: %w", err)
	}
	certPath := filepath.Join(certsDir, "tunnel-client-cert.pem")
	keyPath := filepath.Join(certsDir, "tunnel-client-key.pem")
	if err := os.WriteFile(certPath, []byte(strings.TrimSpace(bundle.TunnelClientCert)+"\n"), 0600); err != nil {
		return "", fmt.Errorf("write tunnel client cert: %w", err)
	}
	if err := os.WriteFile(keyPath, []byte(strings.TrimSpace(bundle.TunnelClientKey)+"\n"), 0600); err != nil {
		return "", fmt.Errorf("write tunnel client key: %w", err)
	}
	envPath := filepath.Join(bundleDir, "tunnel.env")
	envContent := tunnelTokenEnvContent(bundle, serverURL, "./certs/tunnel-client-cert.pem", "./certs/tunnel-client-key.pem")
	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		return "", fmt.Errorf("write tunnel env file: %w", err)
	}
	return envPath, nil
}

func envLine(key, value string) string {
	return key + "=" + strconv.Quote(strings.TrimSpace(value))
}

func runGwTunnelTokenRevoke(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete(fmt.Sprintf("/api/gateways/%s/tunnel-token", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("Tunnel token revoked for gateway %q\n", args[0])
	}
}

func runGwTunnelDisconnect(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/gateways/%s/tunnel-disconnect", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("Tunnel disconnected for gateway %q\n", args[0])
	}
}

func runGwTunnelEvents(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/gateways/%s/tunnel-events", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "TIMESTAMP", Field: "timestamp"},
		{Header: "EVENT", Field: "event"},
		{Header: "DETAILS", Field: "details"},
	})
}

func runGwTunnelMetrics(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/gateways/%s/tunnel-metrics", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "CONNECTED", Field: "connected"},
		{Header: "BYTES_IN", Field: "bytesIn"},
		{Header: "BYTES_OUT", Field: "bytesOut"},
		{Header: "UPTIME", Field: "uptime"},
	})
}

func runGwTunnelOverview(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/gateways/tunnel-overview", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "GATEWAY_ID", Field: "gatewayId"},
		{Header: "GATEWAY_NAME", Field: "gatewayName"},
		{Header: "CONNECTED", Field: "connected"},
		{Header: "STATUS", Field: "status"},
	})
}
