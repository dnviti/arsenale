package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var keystrokePolicyColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "ENABLED", Field: "enabled"},
}

var keystrokePolicyCmd = &cobra.Command{
	Use:   "keystroke-policy",
	Short: "Manage keystroke policies",
}

var keystrokePolicyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all keystroke policies",
	Run:   runKeystrokePolicyList,
}

var keystrokePolicyGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get keystroke policy details",
	Args:  cobra.ExactArgs(1),
	Run:   runKeystrokePolicyGet,
}

var keystrokePolicyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new keystroke policy",
	Long:  `Create a keystroke policy from a JSON/YAML file: arsenale keystroke-policy create --from-file policy.yaml`,
	Run:   runKeystrokePolicyCreate,
}

var keystrokePolicyUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a keystroke policy",
	Args:  cobra.ExactArgs(1),
	Run:   runKeystrokePolicyUpdate,
}

var keystrokePolicyDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a keystroke policy",
	Args:  cobra.ExactArgs(1),
	Run:   runKeystrokePolicyDelete,
}

var keystrokePolicyFromFile string

func init() {
	rootCmd.AddCommand(keystrokePolicyCmd)

	keystrokePolicyCmd.AddCommand(keystrokePolicyListCmd)
	keystrokePolicyCmd.AddCommand(keystrokePolicyGetCmd)
	keystrokePolicyCmd.AddCommand(keystrokePolicyCreateCmd)
	keystrokePolicyCmd.AddCommand(keystrokePolicyUpdateCmd)
	keystrokePolicyCmd.AddCommand(keystrokePolicyDeleteCmd)

	keystrokePolicyCreateCmd.Flags().StringVarP(&keystrokePolicyFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	keystrokePolicyCreateCmd.MarkFlagRequired("from-file")

	keystrokePolicyUpdateCmd.Flags().StringVarP(&keystrokePolicyFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	keystrokePolicyUpdateCmd.MarkFlagRequired("from-file")
}

func runKeystrokePolicyList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/keystroke-policies", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, keystrokePolicyColumns)
}

func runKeystrokePolicyGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/keystroke-policies/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, keystrokePolicyColumns)
}

func runKeystrokePolicyCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(keystrokePolicyFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/keystroke-policies", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runKeystrokePolicyUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(keystrokePolicyFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/keystroke-policies/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, keystrokePolicyColumns)
}

func runKeystrokePolicyDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/keystroke-policies/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Keystroke policy", args[0])
}
