package cmd

import (
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current authenticated user and tenant",
	Run:   runWhoami,
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}

func runWhoami(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/user/profile", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	printer().PrintSingle(body, []Column{
		{Header: "ID", Field: "id"},
		{Header: "EMAIL", Field: "email"},
		{Header: "NAME", Field: "name"},
		{Header: "ROLE", Field: "role"},
		{Header: "TENANT", Field: "tenantId"},
	})
}
