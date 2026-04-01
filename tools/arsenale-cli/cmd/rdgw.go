package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Top-level: arsenale rdgw
// ---------------------------------------------------------------------------

var rdgwCmd = &cobra.Command{
	Use:   "rdgw",
	Short: "Manage RD Gateway configuration",
}

// ---------------------------------------------------------------------------
// Config subcommand group
// ---------------------------------------------------------------------------

var rdgwConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage RD Gateway configuration",
}

var rdgwConfigGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current RD Gateway configuration",
	Run:   runRdgwConfigGet,
}

var rdgwConfigSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set RD Gateway configuration",
	Long:  `Set RD Gateway configuration from a JSON/YAML file: arsenale rdgw config set --from-file rdgw.yaml`,
	Run:   runRdgwConfigSet,
}

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

var rdgwStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get RD Gateway status",
	Run:   runRdgwStatus,
}

// ---------------------------------------------------------------------------
// Flags
// ---------------------------------------------------------------------------

var rdgwConfigFromFile string

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(rdgwCmd)

	rdgwCmd.AddCommand(rdgwConfigCmd)
	rdgwConfigCmd.AddCommand(rdgwConfigGetCmd)
	rdgwConfigCmd.AddCommand(rdgwConfigSetCmd)

	rdgwCmd.AddCommand(rdgwStatusCmd)

	rdgwConfigSetCmd.Flags().StringVarP(&rdgwConfigFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	rdgwConfigSetCmd.MarkFlagRequired("from-file")
}

// ===========================================================================
// Run functions
// ===========================================================================

func runRdgwConfigGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/rdgw/config", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "ENABLED", Field: "enabled"},
		{Header: "HOSTNAME", Field: "hostname"},
		{Header: "PORT", Field: "port"},
	})
}

func runRdgwConfigSet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(rdgwConfigFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/rdgw/config", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "ENABLED", Field: "enabled"},
		{Header: "HOSTNAME", Field: "hostname"},
		{Header: "PORT", Field: "port"},
	})
}

func runRdgwStatus(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/rdgw/status", cfg)
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
