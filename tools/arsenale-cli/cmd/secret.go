package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Column definitions
// ---------------------------------------------------------------------------

var secretColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
	{Header: "FOLDER", Field: "folder"},
	{Header: "UPDATED_AT", Field: "updatedAt"},
}

var secretVersionColumns = []Column{
	{Header: "VERSION", Field: "version"},
	{Header: "CREATED_AT", Field: "createdAt"},
	{Header: "CREATED_BY", Field: "createdBy"},
}

var secretShareColumns = []Column{
	{Header: "USER_ID", Field: "userId"},
	{Header: "EMAIL", Field: "email"},
	{Header: "PERMISSION", Field: "permission"},
}

// ---------------------------------------------------------------------------
// Parent command
// ---------------------------------------------------------------------------

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage secrets",
}

// ---------------------------------------------------------------------------
// CRUD subcommands
// ---------------------------------------------------------------------------

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets",
	Run:   runSecretList,
}

var secretGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get secret details",
	Args:  cobra.ExactArgs(1),
	Run:   runSecretGet,
}

var secretCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new secret",
	Long:  `Create a secret from a JSON/YAML file: arsenale secret create --from-file secret.yaml`,
	Run:   runSecretCreate,
}

var secretUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a secret",
	Args:  cobra.ExactArgs(1),
	Run:   runSecretUpdate,
}

var secretDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a secret",
	Args:  cobra.ExactArgs(1),
	Run:   runSecretDelete,
}

var secretCountsCmd = &cobra.Command{
	Use:   "counts",
	Short: "Get secret counts",
	Run:   runSecretCounts,
}

// ---------------------------------------------------------------------------
// Sharing subcommands
// ---------------------------------------------------------------------------

var secretShareCmd = &cobra.Command{
	Use:   "share <id>",
	Short: "Share a secret with a user",
	Long:  `Share a secret: arsenale secret share <id> --user-id <userId> --permission <read|write>`,
	Args:  cobra.ExactArgs(1),
	Run:   runSecretShare,
}

var secretUnshareCmd = &cobra.Command{
	Use:   "unshare <id> <userId>",
	Short: "Remove sharing for a user",
	Args:  cobra.ExactArgs(2),
	Run:   runSecretUnshare,
}

var secretUpdateShareCmd = &cobra.Command{
	Use:   "update-share <id> <userId>",
	Short: "Update share permission for a user",
	Args:  cobra.ExactArgs(2),
	Run:   runSecretUpdateShare,
}

var secretSharesCmd = &cobra.Command{
	Use:   "shares <id>",
	Short: "List users with access to a secret",
	Args:  cobra.ExactArgs(1),
	Run:   runSecretShares,
}

// ---------------------------------------------------------------------------
// External sharing subcommands
// ---------------------------------------------------------------------------

var secretShareExternalCmd = &cobra.Command{
	Use:   "share-external <id>",
	Short: "Create an external share for a secret",
	Long:  `Create an external share: arsenale secret share-external <id> --from-file share.yaml`,
	Args:  cobra.ExactArgs(1),
	Run:   runSecretShareExternal,
}

var secretExternalSharesCmd = &cobra.Command{
	Use:   "external-shares <id>",
	Short: "List external shares for a secret",
	Args:  cobra.ExactArgs(1),
	Run:   runSecretExternalShares,
}

var secretRevokeExternalShareCmd = &cobra.Command{
	Use:   "revoke-external-share <shareId>",
	Short: "Revoke an external share",
	Args:  cobra.ExactArgs(1),
	Run:   runSecretRevokeExternalShare,
}

// ---------------------------------------------------------------------------
// Version subcommands
// ---------------------------------------------------------------------------

var secretVersionsCmd = &cobra.Command{
	Use:   "versions <id>",
	Short: "List versions of a secret",
	Args:  cobra.ExactArgs(1),
	Run:   runSecretVersions,
}

var secretVersionDataCmd = &cobra.Command{
	Use:   "version-data <id> <version>",
	Short: "Get data for a specific secret version",
	Args:  cobra.ExactArgs(2),
	Run:   runSecretVersionData,
}

var secretRestoreVersionCmd = &cobra.Command{
	Use:   "restore-version <id> <version>",
	Short: "Restore a secret to a specific version",
	Args:  cobra.ExactArgs(2),
	Run:   runSecretRestoreVersion,
}

// ---------------------------------------------------------------------------
// Breach check subcommands
// ---------------------------------------------------------------------------

var secretBreachCheckAllCmd = &cobra.Command{
	Use:   "breach-check-all",
	Short: "Run breach check on all secrets",
	Run:   runSecretBreachCheckAll,
}

var secretBreachCheckCmd = &cobra.Command{
	Use:   "breach-check <id>",
	Short: "Run breach check on a specific secret",
	Args:  cobra.ExactArgs(1),
	Run:   runSecretBreachCheck,
}

// ---------------------------------------------------------------------------
// Rotation subcommand group
// ---------------------------------------------------------------------------

var secretRotationCmd = &cobra.Command{
	Use:   "rotation",
	Short: "Manage secret rotation",
}

var secretRotationEnableCmd = &cobra.Command{
	Use:   "enable <id>",
	Short: "Enable rotation for a secret",
	Long:  `Enable rotation: arsenale secret rotation enable <id> --from-file config.yaml`,
	Args:  cobra.ExactArgs(1),
	Run:   runSecretRotationEnable,
}

var secretRotationDisableCmd = &cobra.Command{
	Use:   "disable <id>",
	Short: "Disable rotation for a secret",
	Args:  cobra.ExactArgs(1),
	Run:   runSecretRotationDisable,
}

var secretRotationTriggerCmd = &cobra.Command{
	Use:   "trigger <id>",
	Short: "Trigger rotation for a secret",
	Args:  cobra.ExactArgs(1),
	Run:   runSecretRotationTrigger,
}

var secretRotationStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get rotation status for all secrets",
	Run:   runSecretRotationStatus,
}

var secretRotationHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Get rotation history",
	Run:   runSecretRotationHistory,
}

// ---------------------------------------------------------------------------
// Flags
// ---------------------------------------------------------------------------

var (
	secretFromFile          string
	secretShareUserID       string
	secretSharePerm         string
	secretUpdateSharePerm   string
	secretExtShareFromFile  string
	secretRotationFromFile  string
)

// ---------------------------------------------------------------------------
// init — register all commands
// ---------------------------------------------------------------------------

func init() {
	rootCmd.AddCommand(secretCmd)

	// CRUD
	secretCmd.AddCommand(secretListCmd)
	secretCmd.AddCommand(secretGetCmd)
	secretCmd.AddCommand(secretCreateCmd)
	secretCmd.AddCommand(secretUpdateCmd)
	secretCmd.AddCommand(secretDeleteCmd)
	secretCmd.AddCommand(secretCountsCmd)

	// Sharing
	secretCmd.AddCommand(secretShareCmd)
	secretCmd.AddCommand(secretUnshareCmd)
	secretCmd.AddCommand(secretUpdateShareCmd)
	secretCmd.AddCommand(secretSharesCmd)

	// External sharing
	secretCmd.AddCommand(secretShareExternalCmd)
	secretCmd.AddCommand(secretExternalSharesCmd)
	secretCmd.AddCommand(secretRevokeExternalShareCmd)

	// Versions
	secretCmd.AddCommand(secretVersionsCmd)
	secretCmd.AddCommand(secretVersionDataCmd)
	secretCmd.AddCommand(secretRestoreVersionCmd)

	// Breach check
	secretCmd.AddCommand(secretBreachCheckAllCmd)
	secretCmd.AddCommand(secretBreachCheckCmd)

	// Rotation group
	secretCmd.AddCommand(secretRotationCmd)
	secretRotationCmd.AddCommand(secretRotationEnableCmd)
	secretRotationCmd.AddCommand(secretRotationDisableCmd)
	secretRotationCmd.AddCommand(secretRotationTriggerCmd)
	secretRotationCmd.AddCommand(secretRotationStatusCmd)
	secretRotationCmd.AddCommand(secretRotationHistoryCmd)

	// Flags: CRUD
	secretCreateCmd.Flags().StringVarP(&secretFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	secretCreateCmd.MarkFlagRequired("from-file")
	secretUpdateCmd.Flags().StringVarP(&secretFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	secretUpdateCmd.MarkFlagRequired("from-file")

	// Flags: sharing
	secretShareCmd.Flags().StringVar(&secretShareUserID, "user-id", "", "User ID to share with")
	secretShareCmd.Flags().StringVar(&secretSharePerm, "permission", "read", "Permission level")
	secretShareCmd.MarkFlagRequired("user-id")

	secretUpdateShareCmd.Flags().StringVar(&secretUpdateSharePerm, "permission", "", "New permission level")
	secretUpdateShareCmd.MarkFlagRequired("permission")

	// Flags: external sharing
	secretShareExternalCmd.Flags().StringVarP(&secretExtShareFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")

	// Flags: rotation
	secretRotationEnableCmd.Flags().StringVarP(&secretRotationFromFile, "from-file", "f", "", "JSON/YAML rotation config file (- for stdin)")
	secretRotationEnableCmd.MarkFlagRequired("from-file")
}

// ---------------------------------------------------------------------------
// CRUD runners
// ---------------------------------------------------------------------------

func runSecretList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/secrets", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, secretColumns)
}

func runSecretGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/secrets/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, secretColumns)
}

func runSecretCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(secretFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/secrets", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runSecretUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(secretFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/secrets/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, secretColumns)
}

func runSecretDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/secrets/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Secret", args[0])
}

func runSecretCounts(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/secrets/counts", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "TOTAL", Field: "total"},
		{Header: "BY_TYPE", Field: "byType"},
	})
}

// ---------------------------------------------------------------------------
// Sharing runners
// ---------------------------------------------------------------------------

func runSecretShare(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	payload := map[string]string{
		"userId":     secretShareUserID,
		"permission": secretSharePerm,
	}

	body, status, err := apiPost(fmt.Sprintf("/api/secrets/%s/share", args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	fmt.Println("Secret shared successfully")
}

func runSecretUnshare(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete(fmt.Sprintf("/api/secrets/%s/share/%s", args[0], args[1]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	fmt.Println("Share removed")
}

func runSecretUpdateShare(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	payload := map[string]string{
		"permission": secretUpdateSharePerm,
	}

	body, status, err := apiPut(fmt.Sprintf("/api/secrets/%s/share/%s", args[0], args[1]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	fmt.Println("Share permission updated")
}

func runSecretShares(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/secrets/%s/shares", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, secretShareColumns)
}

// ---------------------------------------------------------------------------
// External sharing runners
// ---------------------------------------------------------------------------

func runSecretShareExternal(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	var payload interface{}
	if secretExtShareFromFile != "" {
		data, err := readResourceFromFileOrStdin(secretExtShareFromFile)
		if err != nil {
			fatal("%v", err)
		}
		payload = json.RawMessage(data)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/secrets/%s/external-shares", args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runSecretExternalShares(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/secrets/%s/external-shares", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "ID", Field: "id"},
		{Header: "EXPIRES_AT", Field: "expiresAt"},
		{Header: "CREATED_AT", Field: "createdAt"},
	})
}

func runSecretRevokeExternalShare(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/secrets/external-shares/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("External share", args[0])
}

// ---------------------------------------------------------------------------
// Version runners
// ---------------------------------------------------------------------------

func runSecretVersions(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/secrets/%s/versions", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, secretVersionColumns)
}

func runSecretVersionData(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/secrets/%s/versions/%s/data", args[0], args[1]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "VERSION", Field: "version"},
		{Header: "DATA", Field: "data"},
	})
}

func runSecretRestoreVersion(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/secrets/%s/versions/%s/restore", args[0], args[1]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	fmt.Printf("Secret %q restored to version %s\n", args[0], args[1])
}

// ---------------------------------------------------------------------------
// Breach check runners
// ---------------------------------------------------------------------------

func runSecretBreachCheckAll(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/secrets/breach-check", nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "STATUS", Field: "status"},
		{Header: "BREACHED", Field: "breached"},
		{Header: "CHECKED", Field: "checked"},
	})
}

func runSecretBreachCheck(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/secrets/%s/breach-check", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "STATUS", Field: "status"},
		{Header: "BREACHED", Field: "breached"},
	})
}

// ---------------------------------------------------------------------------
// Rotation runners
// ---------------------------------------------------------------------------

func runSecretRotationEnable(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(secretRotationFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/secrets/%s/rotation/enable", args[0]), json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	fmt.Println("Rotation enabled")
}

func runSecretRotationDisable(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/secrets/%s/rotation/disable", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	fmt.Println("Rotation disabled")
}

func runSecretRotationTrigger(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/secrets/%s/rotation/trigger", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	fmt.Println("Rotation triggered")
}

func runSecretRotationStatus(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/secrets/rotation/status", nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "SECRET_ID", Field: "secretId"},
		{Header: "ENABLED", Field: "enabled"},
		{Header: "LAST_ROTATED", Field: "lastRotated"},
		{Header: "NEXT_ROTATION", Field: "nextRotation"},
	})
}

func runSecretRotationHistory(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/secrets/rotation/history", nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "SECRET_ID", Field: "secretId"},
		{Header: "VERSION", Field: "version"},
		{Header: "ROTATED_AT", Field: "rotatedAt"},
		{Header: "STATUS", Field: "status"},
	})
}
