package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var sessionColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "USER", Field: "user"},
	{Header: "CONNECTION", Field: "connection"},
	{Header: "TYPE", Field: "type"},
	{Header: "STARTED_AT", Field: "startedAt"},
	{Header: "GATEWAY", Field: "gateway"},
}

// ---------------------------------------------------------------------------
// Top-level: arsenale session
// ---------------------------------------------------------------------------

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage sessions",
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active sessions",
	Run:   runSessionList,
}

var sessionCountCmd = &cobra.Command{
	Use:   "count",
	Short: "Get total active session count",
	Run:   runSessionCount,
}

var sessionCountByGatewayCmd = &cobra.Command{
	Use:   "count-by-gateway",
	Short: "Get active session count grouped by gateway",
	Run:   runSessionCountByGateway,
}

var sessionTerminateCmd = &cobra.Command{
	Use:   "terminate <sessionId>",
	Short: "Terminate an active session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionTerminate,
}

var sessionSSHProxyStatusCmd = &cobra.Command{
	Use:   "ssh-proxy-status",
	Short: "Get SSH proxy status",
	Run:   runSessionSSHProxyStatus,
}

// ---------------------------------------------------------------------------
// DB Tunnel subcommand group
// ---------------------------------------------------------------------------

var sessionDBTunnelCmd = &cobra.Command{
	Use:   "db-tunnel",
	Short: "Manage database tunnels",
}

var sessionDBTunnelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active database tunnels",
	Run:   runSessionDBTunnelList,
}

var sessionDBTunnelCloseCmd = &cobra.Command{
	Use:   "close <tunnelId>",
	Short: "Close a database tunnel",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionDBTunnelClose,
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(sessionCmd)

	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionCountCmd)
	sessionCmd.AddCommand(sessionCountByGatewayCmd)
	sessionCmd.AddCommand(sessionTerminateCmd)
	sessionCmd.AddCommand(sessionSSHProxyStatusCmd)

	sessionCmd.AddCommand(sessionDBTunnelCmd)
	sessionDBTunnelCmd.AddCommand(sessionDBTunnelListCmd)
	sessionDBTunnelCmd.AddCommand(sessionDBTunnelCloseCmd)
}

// ===========================================================================
// Run functions
// ===========================================================================

func runSessionList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/sessions/active", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, sessionColumns)
}

func runSessionCount(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/sessions/count", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "COUNT", Field: "count"},
	})
}

func runSessionCountByGateway(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/sessions/count/gateway", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "GATEWAY_ID", Field: "gatewayId"},
		{Header: "GATEWAY_NAME", Field: "gatewayName"},
		{Header: "COUNT", Field: "count"},
	})
}

func runSessionTerminate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/sessions/%s/terminate", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("Session %q terminated\n", args[0])
	}
}

func runSessionSSHProxyStatus(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/sessions/ssh-proxy/status", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "STATUS", Field: "status"},
		{Header: "CONNECTIONS", Field: "connections"},
		{Header: "UPTIME", Field: "uptime"},
	})
}

func runSessionDBTunnelList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/sessions/db-tunnel", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "ID", Field: "id"},
		{Header: "CONNECTION", Field: "connection"},
		{Header: "LOCAL_PORT", Field: "localPort"},
		{Header: "STATUS", Field: "status"},
	})
}

func runSessionDBTunnelClose(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/sessions/db-tunnel/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("DB tunnel", args[0])
}
