package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func runGwTemplateList(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiGet("/api/gateways/templates", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, gatewayTemplateColumns)
}

func runGwTemplateCreate(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	data, err := readResourceFromFileOrStdin(gwTemplateFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/gateways/templates", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runGwTemplateUpdate(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	data, err := readResourceFromFileOrStdin(gwTemplateFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/gateways/templates/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, gatewayTemplateColumns)
}

func runGwTemplateDelete(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiDelete("/api/gateways/templates/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Template", args[0])
}

func runGwTemplateDeploy(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiPost(fmt.Sprintf("/api/gateways/templates/%s/deploy", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if quiet {
		if err := printer().PrintCreated(body, "id"); err != nil {
			fatal("%v", err)
		}
		return
	}
	if err := printer().PrintSingle(body, gatewayColumns); err != nil {
		fatal("%v", err)
	}
}
