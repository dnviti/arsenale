package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func runGwDeploy(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiPost(fmt.Sprintf("/api/gateways/%s/deploy", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("Gateway %q deployed\n", args[0])
	}
}

func runGwStatus(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	gateway, err := getGatewayByID(cfg, args[0])
	if err != nil {
		fatal("%v", err)
	}
	instances, err := getGatewayInstances(cfg, args[0])
	if err != nil {
		fatal("%v", err)
	}

	combined := map[string]any{
		"gateway":   gateway,
		"instances": instances,
	}
	combinedJSON, err := json.Marshal(combined)
	if err != nil {
		fatal("marshal gateway status: %v", err)
	}

	if outputFormat == "json" || outputFormat == "yaml" {
		if err := printer().PrintSingle(combinedJSON, nil); err != nil {
			fatal("%v", err)
		}
		return
	}

	fmt.Fprintln(os.Stdout, "Gateway")
	if err := printer().PrintSingle(mustMarshalJSON(gateway), gatewayStatusColumns); err != nil {
		fatal("%v", err)
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Instances")
	if len(instances) == 0 {
		fmt.Fprintln(os.Stdout, "(none)")
		return
	}
	if err := printer().Print(mustMarshalJSON(instances), gatewayInstanceColumns); err != nil {
		fatal("%v", err)
	}
}

func runGwUndeploy(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiDelete(fmt.Sprintf("/api/gateways/%s/deploy", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("Gateway %q undeployed\n", args[0])
	}
}

func runGwScale(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	payload := map[string]interface{}{
		"replicas": gwScaleReplicas,
	}

	body, status, err := apiPost(fmt.Sprintf("/api/gateways/%s/scale", args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("Gateway %q scaled to %d replicas\n", args[0], gwScaleReplicas)
	}
}

func runGwTest(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiPost(fmt.Sprintf("/api/gateways/%s/test", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "REACHABLE", Field: "reachable"},
		{Header: "LATENCY_MS", Field: "latencyMs"},
		{Header: "ERROR", Field: "error"},
	})
}

func runGwPushKey(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiPost(fmt.Sprintf("/api/gateways/%s/push-key", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("SSH key pushed to gateway %q\n", args[0])
	}
}
