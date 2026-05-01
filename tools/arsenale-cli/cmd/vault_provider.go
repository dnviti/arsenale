package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var vaultProviderColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
	{Header: "STATUS", Field: "status"},
}

var vaultProviderCmd = &cobra.Command{
	Use:     "vault-provider",
	Aliases: []string{"vp"},
	Short:   "Manage vault providers",
}

var vaultProviderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all vault providers",
	Run:   runVaultProviderList,
}

var vaultProviderGetCmd = &cobra.Command{
	Use:   "get <providerId>",
	Short: "Get vault provider details",
	Args:  cobra.ExactArgs(1),
	Run:   runVaultProviderGet,
}

var vaultProviderCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new vault provider",
	Long:  `Create a vault provider from a JSON/YAML file: arsenale vault-provider create --from-file provider.yaml`,
	Run:   runVaultProviderCreate,
}

var vaultProviderUpdateCmd = &cobra.Command{
	Use:   "update <providerId>",
	Short: "Update a vault provider",
	Args:  cobra.ExactArgs(1),
	Run:   runVaultProviderUpdate,
}

var vaultProviderDeleteCmd = &cobra.Command{
	Use:   "delete <providerId>",
	Short: "Delete a vault provider",
	Args:  cobra.ExactArgs(1),
	Run:   runVaultProviderDelete,
}

var vaultProviderTestCmd = &cobra.Command{
	Use:   "test <providerId>",
	Short: "Test a vault provider connection",
	Args:  cobra.ExactArgs(1),
	Run:   runVaultProviderTest,
}

var (
	vaultProviderFromFile string
)

func init() {
	rootCmd.AddCommand(vaultProviderCmd)

	vaultProviderCmd.AddCommand(vaultProviderListCmd)
	vaultProviderCmd.AddCommand(vaultProviderGetCmd)
	vaultProviderCmd.AddCommand(vaultProviderCreateCmd)
	vaultProviderCmd.AddCommand(vaultProviderUpdateCmd)
	vaultProviderCmd.AddCommand(vaultProviderDeleteCmd)
	vaultProviderCmd.AddCommand(vaultProviderTestCmd)

	vaultProviderCreateCmd.Flags().StringVarP(&vaultProviderFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	vaultProviderCreateCmd.MarkFlagRequired("from-file")

	vaultProviderUpdateCmd.Flags().StringVarP(&vaultProviderFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	vaultProviderUpdateCmd.MarkFlagRequired("from-file")
}

func runVaultProviderList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/vault-providers", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, vaultProviderColumns)
}

func runVaultProviderGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/vault-providers/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, vaultProviderColumns)
}

func runVaultProviderCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(vaultProviderFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/vault-providers", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runVaultProviderUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(vaultProviderFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/vault-providers/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, vaultProviderColumns)
}

func runVaultProviderDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/vault-providers/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Vault provider", args[0])
}

func runVaultProviderTest(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/vault-providers/"+args[0]+"/test", nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	fmt.Println("Vault provider test successful")
}
