package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// Connection represents a connection resource.
type Connection struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

var connectionColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
	{Header: "HOST", Field: "host"},
	{Header: "PORT", Field: "port"},
}

var connectionCmd = &cobra.Command{
	Use:     "connection",
	Aliases: []string{"conn"},
	Short:   "Manage connections",
}

var connListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all connections",
	Run:   runConnList,
}

var connGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get connection details",
	Args:  cobra.ExactArgs(1),
	Run:   runConnGet,
}

var connCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new connection",
	Long:  `Create a connection from a JSON/YAML file: arsenale connection create --from-file conn.yaml`,
	Run:   runConnCreate,
}

var connUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a connection",
	Args:  cobra.ExactArgs(1),
	Run:   runConnUpdate,
}

var connDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a connection",
	Args:  cobra.ExactArgs(1),
	Run:   runConnDelete,
}

var connShareCmd = &cobra.Command{
	Use:   "share <id>",
	Short: "Share a connection with a user",
	Long:  `Share a connection: arsenale connection share <id> --user-id <userId> --permission READ_ONLY`,
	Args:  cobra.ExactArgs(1),
	Run:   runConnShare,
}

var connUnshareCmd = &cobra.Command{
	Use:   "unshare <id> <userId>",
	Short: "Remove sharing for a user",
	Args:  cobra.ExactArgs(2),
	Run:   runConnUnshare,
}

var connSharesCmd = &cobra.Command{
	Use:   "shares <id>",
	Short: "List users with access to a connection",
	Args:  cobra.ExactArgs(1),
	Run:   runConnShares,
}

var connBatchShareCmd = &cobra.Command{
	Use:   "batch-share",
	Short: "Batch share multiple connections",
	Long:  `Batch share from a JSON/YAML file: arsenale connection batch-share --from-file shares.yaml`,
	Run:   runConnBatchShare,
}

var connFavoriteCmd = &cobra.Command{
	Use:   "favorite <id>",
	Short: "Toggle favorite status",
	Args:  cobra.ExactArgs(1),
	Run:   runConnFavorite,
}

var connExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export connections",
	Run:   runConnExport,
}

var connImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import connections from file",
	Run:   runConnImport,
}

// Backwards-compat: `arsenale list` -> `arsenale connection list`
var listAliasCmd = &cobra.Command{
	Use:    "list",
	Short:  "List connections (alias for 'connection list')",
	Hidden: true,
	Run:    runConnList,
}

var (
	connFromFile                 string
	connShareUserID              string
	connShareEmail               string
	connSharePerm                string
	connImportFile               string
	connImportFormat             string
	connImportDuplicateStrategy  string
	connImportColumnMapping      string
	connExportIDs                []string
	connExportFormat             string
	connExportIncludeCredentials bool
	connExportFolderID           string
	connSearch                   string
)

func init() {
	rootCmd.AddCommand(connectionCmd)
	rootCmd.AddCommand(listAliasCmd)

	connectionCmd.AddCommand(connListCmd)
	connectionCmd.AddCommand(connGetCmd)
	connectionCmd.AddCommand(connCreateCmd)
	connectionCmd.AddCommand(connUpdateCmd)
	connectionCmd.AddCommand(connDeleteCmd)
	connectionCmd.AddCommand(connShareCmd)
	connectionCmd.AddCommand(connUnshareCmd)
	connectionCmd.AddCommand(connSharesCmd)
	connectionCmd.AddCommand(connBatchShareCmd)
	connectionCmd.AddCommand(connFavoriteCmd)
	connectionCmd.AddCommand(connExportCmd)
	connectionCmd.AddCommand(connImportCmd)

	connCreateCmd.Flags().StringVarP(&connFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	connCreateCmd.MarkFlagRequired("from-file")
	connUpdateCmd.Flags().StringVarP(&connFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	connUpdateCmd.MarkFlagRequired("from-file")

	connShareCmd.Flags().StringVar(&connShareUserID, "user-id", "", "User ID to share with")
	connShareCmd.Flags().StringVar(&connShareEmail, "email", "", "Email address to share with")
	connShareCmd.Flags().StringVar(&connSharePerm, "permission", "READ_ONLY", "Permission level: READ_ONLY|FULL_ACCESS")

	connBatchShareCmd.Flags().StringVarP(&connFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	connBatchShareCmd.MarkFlagRequired("from-file")

	connImportCmd.Flags().StringVar(&connImportFile, "file", "", "File to import")
	connImportCmd.Flags().StringVar(&connImportFormat, "format", "", "Import format: CSV|JSON|MREMOTENG|RDP (defaults from file extension)")
	connImportCmd.Flags().StringVar(&connImportDuplicateStrategy, "duplicate-strategy", "SKIP", "Duplicate handling: SKIP|OVERWRITE|RENAME")
	connImportCmd.Flags().StringVar(&connImportColumnMapping, "column-mapping", "", "JSON object mapping CSV column names")
	connImportCmd.MarkFlagRequired("file")

	connExportCmd.Flags().StringSliceVar(&connExportIDs, "ids", nil, "Connection IDs to export")
	connExportCmd.Flags().StringVar(&connExportFormat, "format", "JSON", "Export format: JSON|CSV")
	connExportCmd.Flags().BoolVar(&connExportIncludeCredentials, "include-credentials", false, "Include decrypted credentials in the export")
	connExportCmd.Flags().StringVar(&connExportFolderID, "folder-id", "", "Export connections from a folder")

	connListCmd.Flags().StringVar(&connSearch, "search", "", "Search filter")
}

// findConnectionByName looks up a connection by name.
func findConnectionByName(name string, cfg *CLIConfig) (*Connection, error) {
	respBody, status, err := apiGet("/api/cli/connections", cfg)
	if err != nil {
		return nil, fmt.Errorf("fetch connections: %w", err)
	}
	if status != 200 {
		return nil, fmt.Errorf("server returned HTTP %d: %s", status, string(respBody))
	}

	var connections []Connection
	if err := json.Unmarshal(respBody, &connections); err != nil {
		return nil, fmt.Errorf("parse connections: %w", err)
	}

	var nameMatches []Connection
	for _, c := range connections {
		if c.ID == name {
			return &c, nil
		}
		if c.Name == name {
			nameMatches = append(nameMatches, c)
		}
	}
	if len(nameMatches) == 1 {
		return &nameMatches[0], nil
	}
	if len(nameMatches) > 1 {
		return nil, fmt.Errorf("connection name '%s' is ambiguous (%d matches). Use the connection ID instead", name, len(nameMatches))
	}
	return nil, fmt.Errorf("connection '%s' not found. Run 'arsenale connection list' to see available connections", name)
}
