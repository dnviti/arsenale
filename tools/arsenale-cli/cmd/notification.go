package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var notificationColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "TYPE", Field: "type"},
	{Header: "MESSAGE", Field: "message"},
	{Header: "READ", Field: "read"},
	{Header: "CREATED_AT", Field: "createdAt"},
}

var notificationCmd = &cobra.Command{
	Use:   "notification",
	Short: "Manage notifications",
}

var notificationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notifications",
	Run:   runNotificationList,
}

var notificationMarkReadCmd = &cobra.Command{
	Use:   "mark-read <id>",
	Short: "Mark a notification as read",
	Args:  cobra.ExactArgs(1),
	Run:   runNotificationMarkRead,
}

var notificationMarkAllReadCmd = &cobra.Command{
	Use:   "mark-all-read",
	Short: "Mark all notifications as read",
	Run:   runNotificationMarkAllRead,
}

var notificationDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a notification",
	Args:  cobra.ExactArgs(1),
	Run:   runNotificationDelete,
}

// --- Preferences subcommands ---

var notificationPreferencesCmd = &cobra.Command{
	Use:   "preferences",
	Short: "Manage notification preferences",
}

var notificationPreferencesGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get notification preferences",
	Run:   runNotificationPreferencesGet,
}

var notificationPreferencesSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set notification preferences",
	Long:  `Set notification preferences from a JSON/YAML file: arsenale notification preferences set --from-file prefs.yaml`,
	Run:   runNotificationPreferencesSet,
}

var notificationPreferencesSetTypeCmd = &cobra.Command{
	Use:   "set-type <type>",
	Short: "Set preferences for a notification type",
	Args:  cobra.ExactArgs(1),
	Run:   runNotificationPreferencesSetType,
}

var notificationPrefsFromFile string

func init() {
	rootCmd.AddCommand(notificationCmd)

	notificationCmd.AddCommand(notificationListCmd)
	notificationCmd.AddCommand(notificationMarkReadCmd)
	notificationCmd.AddCommand(notificationMarkAllReadCmd)
	notificationCmd.AddCommand(notificationDeleteCmd)
	notificationCmd.AddCommand(notificationPreferencesCmd)

	notificationPreferencesCmd.AddCommand(notificationPreferencesGetCmd)
	notificationPreferencesCmd.AddCommand(notificationPreferencesSetCmd)
	notificationPreferencesCmd.AddCommand(notificationPreferencesSetTypeCmd)

	notificationPreferencesSetCmd.Flags().StringVarP(&notificationPrefsFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	notificationPreferencesSetCmd.MarkFlagRequired("from-file")

	notificationPreferencesSetTypeCmd.Flags().StringVarP(&notificationPrefsFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	notificationPreferencesSetTypeCmd.MarkFlagRequired("from-file")
}

func runNotificationList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/notifications", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, notificationColumns)
}

func runNotificationMarkRead(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut(fmt.Sprintf("/api/notifications/%s/read", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("Notification %q marked as read\n", args[0])
	}
}

func runNotificationMarkAllRead(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/notifications/read-all", nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Println("All notifications marked as read")
	}
}

func runNotificationDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/notifications/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Notification", args[0])
}

// --- Preferences run functions ---

func runNotificationPreferencesGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/notifications/preferences", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "EMAIL", Field: "email"},
		{Header: "PUSH", Field: "push"},
		{Header: "IN_APP", Field: "inApp"},
	})
}

func runNotificationPreferencesSet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(notificationPrefsFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/notifications/preferences", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Println("Notification preferences updated")
	}
}

func runNotificationPreferencesSetType(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(notificationPrefsFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut(fmt.Sprintf("/api/notifications/preferences/%s", args[0]), json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("Notification preferences for type %q updated\n", args[0])
	}
}
