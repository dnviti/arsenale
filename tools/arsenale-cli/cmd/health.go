package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check server health",
	Run:   runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)
}

func runHealth(cmd *cobra.Command, args []string) {
	cfg := getCfg()

	body, status, err := doRequest("GET", "/api/health", nil, cfg)
	if err != nil {
		fatal("cannot reach server: %v", err)
	}

	if status == 200 {
		if outputFormat == "table" {
			fmt.Printf("Server %s is healthy\n", cfg.ServerURL)
		} else {
			printer().Print(body, nil)
		}
	} else {
		fatal("server returned HTTP %d: %s", status, string(body))
	}
}
