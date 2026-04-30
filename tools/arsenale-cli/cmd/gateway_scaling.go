package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func runGwScalingGet(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	body, status, err := apiGet(fmt.Sprintf("/api/gateways/%s/scaling", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "MIN_REPLICAS", Field: "minReplicas"},
		{Header: "MAX_REPLICAS", Field: "maxReplicas"},
		{Header: "CURRENT", Field: "currentReplicas"},
	})
}

func runGwScalingSet(cmd *cobra.Command, args []string) {
	cfg := authenticatedGatewayConfig()

	data, err := readResourceFromFileOrStdin(gwScalingFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut(fmt.Sprintf("/api/gateways/%s/scaling", args[0]), json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if !quiet {
		fmt.Printf("Scaling configuration updated for gateway %q\n", args[0])
	}
}
