package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

func runGwList(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiGet("/api/gateways", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, gatewayColumns)
}

func runGwCreate(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	data, err := readResourceFromFileOrStdin(gwFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/gateways", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runGwUpdate(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	data, err := readResourceFromFileOrStdin(gwFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/gateways/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, gatewayColumns)
}

func runGwDelete(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiDelete("/api/gateways/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Gateway", args[0])
}
