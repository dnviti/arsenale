package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var teamColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "MEMBER_COUNT", Field: "memberCount"},
}

var teamMemberColumns = []Column{
	{Header: "USER_ID", Field: "userId"},
	{Header: "EMAIL", Field: "email"},
	{Header: "ROLE", Field: "role"},
}

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage teams",
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all teams",
	Run:   runTeamList,
}

var teamGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get team details",
	Args:  cobra.ExactArgs(1),
	Run:   runTeamGet,
}

var teamCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new team",
	Long:  `Create a team from a JSON/YAML file or with flags: arsenale team create --name "My Team"`,
	Run:   runTeamCreate,
}

var teamUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a team",
	Args:  cobra.ExactArgs(1),
	Run:   runTeamUpdate,
}

var teamDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a team",
	Args:  cobra.ExactArgs(1),
	Run:   runTeamDelete,
}

var teamMembersCmd = &cobra.Command{
	Use:   "members <id>",
	Short: "List team members",
	Args:  cobra.ExactArgs(1),
	Run:   runTeamMembers,
}

var teamAddMemberCmd = &cobra.Command{
	Use:   "add-member <id>",
	Short: "Add a member to a team",
	Long:  `Add a member: arsenale team add-member <teamId> --user-id <userId> --role member`,
	Args:  cobra.ExactArgs(1),
	Run:   runTeamAddMember,
}

var teamUpdateMemberCmd = &cobra.Command{
	Use:   "update-member <id> <userId>",
	Short: "Update a team member's role",
	Args:  cobra.ExactArgs(2),
	Run:   runTeamUpdateMember,
}

var teamRemoveMemberCmd = &cobra.Command{
	Use:   "remove-member <id> <userId>",
	Short: "Remove a member from a team",
	Args:  cobra.ExactArgs(2),
	Run:   runTeamRemoveMember,
}

var teamSetMemberExpiryCmd = &cobra.Command{
	Use:   "set-member-expiry <id> <userId>",
	Short: "Set expiry for a team member",
	Args:  cobra.ExactArgs(2),
	Run:   runTeamSetMemberExpiry,
}

var (
	teamFromFile     string
	teamName         string
	teamMemberUserID string
	teamMemberRole   string
	teamMemberExpiry string
)

func init() {
	rootCmd.AddCommand(teamCmd)

	teamCmd.AddCommand(teamListCmd)
	teamCmd.AddCommand(teamGetCmd)
	teamCmd.AddCommand(teamCreateCmd)
	teamCmd.AddCommand(teamUpdateCmd)
	teamCmd.AddCommand(teamDeleteCmd)
	teamCmd.AddCommand(teamMembersCmd)
	teamCmd.AddCommand(teamAddMemberCmd)
	teamCmd.AddCommand(teamUpdateMemberCmd)
	teamCmd.AddCommand(teamRemoveMemberCmd)
	teamCmd.AddCommand(teamSetMemberExpiryCmd)

	teamCreateCmd.Flags().StringVarP(&teamFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	teamCreateCmd.Flags().StringVar(&teamName, "name", "", "Team name")

	teamUpdateCmd.Flags().StringVarP(&teamFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	teamUpdateCmd.Flags().StringVar(&teamName, "name", "", "Team name")

	teamAddMemberCmd.Flags().StringVar(&teamMemberUserID, "user-id", "", "User ID to add")
	teamAddMemberCmd.Flags().StringVar(&teamMemberRole, "role", "", "Member role")
	teamAddMemberCmd.MarkFlagRequired("user-id")

	teamUpdateMemberCmd.Flags().StringVar(&teamMemberRole, "role", "", "New role")
	teamUpdateMemberCmd.MarkFlagRequired("role")

	teamSetMemberExpiryCmd.Flags().StringVar(&teamMemberExpiry, "expiry", "", "Expiry date (RFC3339)")
	teamSetMemberExpiryCmd.MarkFlagRequired("expiry")
}

func runTeamList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/teams", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, teamColumns)
}

func runTeamGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/teams/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, teamColumns)
}

func runTeamCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	var data []byte
	var err error

	if teamFromFile != "" {
		data, err = readResourceFromFileOrStdin(teamFromFile)
		if err != nil {
			fatal("%v", err)
		}
	} else {
		if teamName == "" {
			fatal("provide --from-file or --name")
		}
		data, err = buildJSONBody(map[string]interface{}{
			"name": teamName,
		})
		if err != nil {
			fatal("%v", err)
		}
	}

	body, status, err := apiPost("/api/teams", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runTeamUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	var data []byte
	var err error

	if teamFromFile != "" {
		data, err = readResourceFromFileOrStdin(teamFromFile)
		if err != nil {
			fatal("%v", err)
		}
	} else {
		if teamName == "" {
			fatal("provide --from-file or --name")
		}
		data, err = buildJSONBody(map[string]interface{}{
			"name": teamName,
		})
		if err != nil {
			fatal("%v", err)
		}
	}

	body, status, err := apiPut("/api/teams/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, teamColumns)
}

func runTeamDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/teams/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Team", args[0])
}

func runTeamMembers(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/teams/%s/members", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, teamMemberColumns)
}

func runTeamAddMember(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	payload := map[string]string{
		"userId": teamMemberUserID,
		"role":   teamMemberRole,
	}

	body, status, err := apiPost(fmt.Sprintf("/api/teams/%s/members", args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("Member %q added to team %q\n", teamMemberUserID, args[0])
	}
}

func runTeamUpdateMember(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	payload := map[string]string{
		"role": teamMemberRole,
	}

	body, status, err := apiPut(fmt.Sprintf("/api/teams/%s/members/%s", args[0], args[1]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("Member %q role updated to %q\n", args[1], teamMemberRole)
	}
}

func runTeamRemoveMember(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete(fmt.Sprintf("/api/teams/%s/members/%s", args[0], args[1]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Member", args[1])
}

func runTeamSetMemberExpiry(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	payload := map[string]string{
		"expiry": teamMemberExpiry,
	}

	body, status, err := apiPatch(fmt.Sprintf("/api/teams/%s/members/%s/expiry", args[0], args[1]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("Member %q expiry set to %s\n", args[1], teamMemberExpiry)
	}
}
