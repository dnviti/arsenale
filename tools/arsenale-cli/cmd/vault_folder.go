package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var vaultFolderColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
}

var vaultFolderCmd = &cobra.Command{
	Use:     "vault-folder",
	Aliases: []string{"vf"},
	Short:   "Manage vault folders",
}

var vaultFolderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all vault folders",
	Run:   runVaultFolderList,
}

var vaultFolderCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new vault folder",
	Long:  `Create a vault folder from a JSON/YAML file or with flags: arsenale vault-folder create --name "My Folder"`,
	Run:   runVaultFolderCreate,
}

var vaultFolderUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a vault folder",
	Args:  cobra.ExactArgs(1),
	Run:   runVaultFolderUpdate,
}

var vaultFolderDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a vault folder",
	Args:  cobra.ExactArgs(1),
	Run:   runVaultFolderDelete,
}

var (
	vaultFolderFromFile string
	vaultFolderName     string
)

func init() {
	rootCmd.AddCommand(vaultFolderCmd)

	vaultFolderCmd.AddCommand(vaultFolderListCmd)
	vaultFolderCmd.AddCommand(vaultFolderCreateCmd)
	vaultFolderCmd.AddCommand(vaultFolderUpdateCmd)
	vaultFolderCmd.AddCommand(vaultFolderDeleteCmd)

	vaultFolderCreateCmd.Flags().StringVarP(&vaultFolderFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	vaultFolderCreateCmd.Flags().StringVar(&vaultFolderName, "name", "", "Vault folder name")

	vaultFolderUpdateCmd.Flags().StringVarP(&vaultFolderFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	vaultFolderUpdateCmd.Flags().StringVar(&vaultFolderName, "name", "", "Vault folder name")
}

func runVaultFolderList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/vault-folders", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, vaultFolderColumns)
}

func runVaultFolderCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	var data []byte
	var err error

	if vaultFolderFromFile != "" {
		data, err = readResourceFromFileOrStdin(vaultFolderFromFile)
		if err != nil {
			fatal("%v", err)
		}
	} else {
		if vaultFolderName == "" {
			fatal("provide --from-file or --name")
		}
		data, err = buildJSONBody(map[string]interface{}{
			"name": vaultFolderName,
		})
		if err != nil {
			fatal("%v", err)
		}
	}

	body, status, err := apiPost("/api/vault-folders", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runVaultFolderUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	var data []byte
	var err error

	if vaultFolderFromFile != "" {
		data, err = readResourceFromFileOrStdin(vaultFolderFromFile)
		if err != nil {
			fatal("%v", err)
		}
	} else {
		if vaultFolderName == "" {
			fatal("provide --from-file or --name")
		}
		data, err = buildJSONBody(map[string]interface{}{
			"name": vaultFolderName,
		})
		if err != nil {
			fatal("%v", err)
		}
	}

	body, status, err := apiPut("/api/vault-folders/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, vaultFolderColumns)
}

func runVaultFolderDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/vault-folders/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Vault folder", args[0])
}
