package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func runGwTunnelTokenCreate(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiPost(fmt.Sprintf("/api/gateways/%s/tunnel-token", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "token")
}

func runGwTunnelTokenRevoke(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

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
	cfg := authenticatedGatewayConfig()

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
	cfg := authenticatedGatewayConfig()

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
	cfg := authenticatedGatewayConfig()

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
	cfg := authenticatedGatewayConfig()

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
