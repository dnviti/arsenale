package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var accessPolicyColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
	{Header: "ENABLED", Field: "enabled"},
}

var accessPolicyCmd = &cobra.Command{
	Use:   "access-policy",
	Short: "Manage access policies",
}

var accessPolicyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all access policies",
	Run:   runAccessPolicyList,
}

var accessPolicyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new access policy",
	Long:  `Create an access policy from a JSON/YAML file: arsenale access-policy create --from-file policy.yaml`,
	Run:   runAccessPolicyCreate,
}

var accessPolicyUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an access policy",
	Args:  cobra.ExactArgs(1),
	Run:   runAccessPolicyUpdate,
}

var accessPolicyDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an access policy",
	Args:  cobra.ExactArgs(1),
	Run:   runAccessPolicyDelete,
}

var accessPolicyFromFile string

func init() {
	rootCmd.AddCommand(accessPolicyCmd)

	accessPolicyCmd.AddCommand(accessPolicyListCmd)
	accessPolicyCmd.AddCommand(accessPolicyCreateCmd)
	accessPolicyCmd.AddCommand(accessPolicyUpdateCmd)
	accessPolicyCmd.AddCommand(accessPolicyDeleteCmd)

	accessPolicyCreateCmd.Flags().StringVarP(&accessPolicyFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	accessPolicyCreateCmd.MarkFlagRequired("from-file")

	accessPolicyUpdateCmd.Flags().StringVarP(&accessPolicyFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	accessPolicyUpdateCmd.MarkFlagRequired("from-file")
}

func runAccessPolicyList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/access-policies", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, accessPolicyColumns)
}

func runAccessPolicyCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(accessPolicyFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/access-policies", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runAccessPolicyUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(accessPolicyFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/access-policies/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, accessPolicyColumns)
}

func runAccessPolicyDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/access-policies/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Access policy", args[0])
}
