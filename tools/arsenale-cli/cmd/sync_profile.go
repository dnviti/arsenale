package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var syncProfileColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
	{Header: "STATUS", Field: "status"},
	{Header: "LAST_SYNC", Field: "lastSync"},
}

var syncProfileCmd = &cobra.Command{
	Use:   "sync-profile",
	Short: "Manage sync profiles",
}

var syncProfileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all sync profiles",
	Run:   runSyncProfileList,
}

var syncProfileGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get sync profile details",
	Args:  cobra.ExactArgs(1),
	Run:   runSyncProfileGet,
}

var syncProfileCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new sync profile",
	Long:  `Create a sync profile from a JSON/YAML file: arsenale sync-profile create --from-file profile.yaml`,
	Run:   runSyncProfileCreate,
}

var syncProfileUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a sync profile",
	Args:  cobra.ExactArgs(1),
	Run:   runSyncProfileUpdate,
}

var syncProfileDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a sync profile",
	Args:  cobra.ExactArgs(1),
	Run:   runSyncProfileDelete,
}

var syncProfileTestCmd = &cobra.Command{
	Use:   "test <id>",
	Short: "Test a sync profile connection",
	Args:  cobra.ExactArgs(1),
	Run:   runSyncProfileTest,
}

var syncProfileSyncCmd = &cobra.Command{
	Use:   "sync <id>",
	Short: "Trigger sync for a profile",
	Args:  cobra.ExactArgs(1),
	Run:   runSyncProfileSync,
}

var syncProfileLogsCmd = &cobra.Command{
	Use:   "logs <id>",
	Short: "Get sync profile logs",
	Args:  cobra.ExactArgs(1),
	Run:   runSyncProfileLogs,
}

var syncProfileFromFile string

func init() {
	rootCmd.AddCommand(syncProfileCmd)

	syncProfileCmd.AddCommand(syncProfileListCmd)
	syncProfileCmd.AddCommand(syncProfileGetCmd)
	syncProfileCmd.AddCommand(syncProfileCreateCmd)
	syncProfileCmd.AddCommand(syncProfileUpdateCmd)
	syncProfileCmd.AddCommand(syncProfileDeleteCmd)
	syncProfileCmd.AddCommand(syncProfileTestCmd)
	syncProfileCmd.AddCommand(syncProfileSyncCmd)
	syncProfileCmd.AddCommand(syncProfileLogsCmd)

	syncProfileCreateCmd.Flags().StringVarP(&syncProfileFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	syncProfileCreateCmd.MarkFlagRequired("from-file")

	syncProfileUpdateCmd.Flags().StringVarP(&syncProfileFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	syncProfileUpdateCmd.MarkFlagRequired("from-file")
}

func runSyncProfileList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/sync-profiles", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, syncProfileColumns)
}

func runSyncProfileGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/sync-profiles/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, syncProfileColumns)
}

func runSyncProfileCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(syncProfileFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/sync-profiles", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runSyncProfileUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(syncProfileFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/sync-profiles/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, syncProfileColumns)
}

func runSyncProfileDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/sync-profiles/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Sync profile", args[0])
}

func runSyncProfileTest(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/sync-profiles/%s/test", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Println("Sync profile test successful")
	}
}

func runSyncProfileSync(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/sync-profiles/%s/sync", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Println("Sync triggered")
	}
}

func runSyncProfileLogs(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/sync-profiles/%s/logs", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "TIMESTAMP", Field: "timestamp"},
		{Header: "LEVEL", Field: "level"},
		{Header: "MESSAGE", Field: "message"},
	})
}
