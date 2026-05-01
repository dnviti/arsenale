package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func runGwInstances(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiGet(fmt.Sprintf("/api/gateways/%s/instances", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, gatewayInstanceColumns)
}

func runGwInstanceRestart(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiPost(fmt.Sprintf("/api/gateways/%s/instances/%s/restart", args[0], args[1]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("Instance %q restarted on gateway %q\n", args[1], args[0])
	}
}

func runGwInstanceLogs(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiGet(buildGatewayLogsPath(args[0], args[1]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printGatewayLogs(body)
}

func runGwLogs(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	instances, err := getGatewayInstances(cfg, args[0])
	if err != nil {
		fatal("%v", err)
	}
	instance, err := selectGatewayInstance(instances, gwLogInstanceID)
	if err != nil {
		fatal("%v", err)
	}

	if !quiet && gwLogInstanceID == "" {
		fmt.Fprintf(os.Stderr, "Using instance %s (%s)\n", instance.ID, instance.ContainerName)
	}
	body, status, err := apiGet(buildGatewayLogsPath(args[0], instance.ID), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printGatewayLogs(body)
}

func buildGatewayLogsPath(gatewayID, instanceID string) string {
	path := fmt.Sprintf("/api/gateways/%s/instances/%s/logs", gatewayID, instanceID)
	if gwLogTailLines > 0 {
		path = fmt.Sprintf("%s?tail=%d", path, gwLogTailLines)
	}
	return path
}

func printGatewayLogs(body []byte) {
	if outputFormat == "json" || outputFormat == "yaml" {
		if err := printer().PrintSingle(body, nil); err != nil {
			fatal("%v", err)
		}
		return
	}

	var payload struct {
		Logs string `json:"logs"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Logs == "" {
		fmt.Println(string(body))
		return
	}
	fmt.Print(payload.Logs)
}
