package cmd

import (
	"fmt"
	"net/url"
	"strings"

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

var sessionConsoleColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "USER", Field: "username"},
	{Header: "CONNECTION", Field: "connectionName"},
	{Header: "PROTOCOL", Field: "protocol"},
	{Header: "STATUS", Field: "status"},
	{Header: "GATEWAY", Field: "gatewayName"},
	{Header: "RECORDING", Field: "recording.exists"},
	{Header: "STARTED_AT", Field: "startedAt"},
	{Header: "ENDED_AT", Field: "endedAt"},
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

var sessionConsoleCmd = &cobra.Command{
	Use:   "console",
	Short: "List console sessions",
	Run:   runSessionConsole,
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

var sessionPauseCmd = &cobra.Command{
	Use:   "pause <sessionId>",
	Short: "Pause an active session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionPause,
}

var sessionResumeCmd = &cobra.Command{
	Use:   "resume <sessionId>",
	Short: "Resume a paused session",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionResume,
}

var sessionObserveCmd = &cobra.Command{
	Use:   "observe",
	Short: "Issue read-only observer grants",
}

var sessionObserveSSHCmd = &cobra.Command{
	Use:   "ssh <sessionId>",
	Short: "Issue a read-only SSH observer grant",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionObserveSSH,
}

var sessionObserveRDPCmd = &cobra.Command{
	Use:   "rdp <sessionId>",
	Short: "Issue a read-only RDP observer grant",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionObserveRDP,
}

var sessionObserveVNCCmd = &cobra.Command{
	Use:   "vnc <sessionId>",
	Short: "Issue a read-only VNC observer grant",
	Args:  cobra.ExactArgs(1),
	Run:   runSessionObserveVNC,
}

var sessionSSHProxyStatusCmd = &cobra.Command{
	Use:   "ssh-proxy-status",
	Short: "Get SSH proxy status",
	Run:   runSessionSSHProxyStatus,
}

var (
	sessionConsoleStatuses  []string
	sessionConsoleProtocol  string
	sessionConsoleGatewayID string
	sessionConsoleLimit     int
	sessionConsoleOffset    int
)

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
	sessionCmd.AddCommand(sessionConsoleCmd)
	sessionCmd.AddCommand(sessionCountCmd)
	sessionCmd.AddCommand(sessionCountByGatewayCmd)
	sessionCmd.AddCommand(sessionTerminateCmd)
	sessionCmd.AddCommand(sessionPauseCmd)
	sessionCmd.AddCommand(sessionResumeCmd)
	sessionCmd.AddCommand(sessionObserveCmd)
	sessionCmd.AddCommand(sessionSSHProxyStatusCmd)
	sessionObserveCmd.AddCommand(sessionObserveSSHCmd)
	sessionObserveCmd.AddCommand(sessionObserveRDPCmd)
	sessionObserveCmd.AddCommand(sessionObserveVNCCmd)

	sessionConsoleCmd.Flags().StringSliceVar(&sessionConsoleStatuses, "status", []string{"ACTIVE"}, "Session status filter (comma-separated or repeatable)")
	sessionConsoleCmd.Flags().StringVar(&sessionConsoleProtocol, "protocol", "", "Protocol filter")
	sessionConsoleCmd.Flags().StringVar(&sessionConsoleGatewayID, "gateway-id", "", "Gateway ID filter")
	sessionConsoleCmd.Flags().IntVar(&sessionConsoleLimit, "limit", 50, "Maximum sessions to return")
	sessionConsoleCmd.Flags().IntVar(&sessionConsoleOffset, "offset", 0, "Result offset")

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

func sessionConsoleQueryValues() url.Values {
	params := url.Values{}
	statuses := sessionConsoleStatuses
	if len(statuses) == 0 {
		statuses = []string{"ACTIVE"}
	}
	params.Set("status", strings.Join(statuses, ","))
	if value := strings.TrimSpace(sessionConsoleProtocol); value != "" {
		params.Set("protocol", value)
	}
	if value := strings.TrimSpace(sessionConsoleGatewayID); value != "" {
		params.Set("gatewayId", value)
	}
	params.Set("limit", fmt.Sprintf("%d", sessionConsoleLimit))
	params.Set("offset", fmt.Sprintf("%d", sessionConsoleOffset))
	return params
}

func runSessionConsole(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiRequestWithParams("GET", "/api/sessions/console", sessionConsoleQueryValues(), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if outputFormat == "json" || outputFormat == "yaml" {
		if err := printer().Print(body, nil); err != nil {
			fatal("%v", err)
		}
		return
	}

	if err := printer().Print(extractWrappedJSONField(body, "sessions"), sessionConsoleColumns); err != nil {
		fatal("%v", err)
	}
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

func runSessionPause(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/sessions/%s/pause", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if err := printer().PrintSingle(body, []Column{
		{Header: "SESSION_ID", Field: "sessionId"},
		{Header: "PROTOCOL", Field: "protocol"},
		{Header: "STATUS", Field: "status"},
		{Header: "PAUSED", Field: "paused"},
	}); err != nil {
		fatal("%v", err)
	}
}

func runSessionResume(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/sessions/%s/resume", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if err := printer().PrintSingle(body, []Column{
		{Header: "SESSION_ID", Field: "sessionId"},
		{Header: "PROTOCOL", Field: "protocol"},
		{Header: "STATUS", Field: "status"},
		{Header: "PAUSED", Field: "paused"},
	}); err != nil {
		fatal("%v", err)
	}
}

func runSessionObserveSSH(cmd *cobra.Command, args []string) {
	runSessionObserve(args[0], fmt.Sprintf("/api/sessions/ssh/%s/observe", args[0]))
}

func runSessionObserveRDP(cmd *cobra.Command, args []string) {
	runSessionObserve(args[0], fmt.Sprintf("/api/sessions/rdp/%s/observe", args[0]))
}

func runSessionObserveVNC(cmd *cobra.Command, args []string) {
	runSessionObserve(args[0], fmt.Sprintf("/api/sessions/vnc/%s/observe", args[0]))
}

func runSessionObserve(sessionID, path string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(path, nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if err := printer().PrintSingle(body, []Column{
		{Header: "SESSION_ID", Field: "sessionId"},
		{Header: "PROTOCOL", Field: "protocol"},
		{Header: "MODE", Field: "mode"},
		{Header: "READ_ONLY", Field: "readOnly"},
		{Header: "EXPIRES_AT", Field: "expiresAt"},
		{Header: "WS_PATH", Field: "webSocketPath"},
	}); err != nil {
		fatal("%v", err)
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
