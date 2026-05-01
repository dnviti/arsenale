package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
)

var folderColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
}

var folderCmd = &cobra.Command{
	Use:   "folder",
	Short: "Manage folders",
}

var folderListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all folders",
	Run:   runFolderList,
}

var folderCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new folder",
	Long:  `Create a folder from a JSON/YAML file or with flags: arsenale folder create --name "My Folder"`,
	Run:   runFolderCreate,
}

var folderUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a folder",
	Args:  cobra.ExactArgs(1),
	Run:   runFolderUpdate,
}

var folderDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a folder",
	Args:  cobra.ExactArgs(1),
	Run:   runFolderDelete,
}

var (
	folderFromFile string
	folderName     string
)

func init() {
	rootCmd.AddCommand(folderCmd)

	folderCmd.AddCommand(folderListCmd)
	folderCmd.AddCommand(folderCreateCmd)
	folderCmd.AddCommand(folderUpdateCmd)
	folderCmd.AddCommand(folderDeleteCmd)

	folderCreateCmd.Flags().StringVarP(&folderFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	folderCreateCmd.Flags().StringVar(&folderName, "name", "", "Folder name")

	folderUpdateCmd.Flags().StringVarP(&folderFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	folderUpdateCmd.Flags().StringVar(&folderName, "name", "", "Folder name")
}

func runFolderList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/folders", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, folderColumns)
}

func runFolderCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	var data []byte
	var err error

	if folderFromFile != "" {
		data, err = readResourceFromFileOrStdin(folderFromFile)
		if err != nil {
			fatal("%v", err)
		}
	} else {
		if folderName == "" {
			fatal("provide --from-file or --name")
		}
		data, err = buildJSONBody(map[string]interface{}{
			"name": folderName,
		})
		if err != nil {
			fatal("%v", err)
		}
	}

	body, status, err := apiPost("/api/folders", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runFolderUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	var data []byte
	var err error

	if folderFromFile != "" {
		data, err = readResourceFromFileOrStdin(folderFromFile)
		if err != nil {
			fatal("%v", err)
		}
	} else {
		if folderName == "" {
			fatal("provide --from-file or --name")
		}
		data, err = buildJSONBody(map[string]interface{}{
			"name": folderName,
		})
		if err != nil {
			fatal("%v", err)
		}
	}

	body, status, err := apiPut("/api/folders/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, folderColumns)
}

func runFolderDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/folders/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Folder", args[0])
}
