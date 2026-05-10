package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

var userColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "EMAIL", Field: "email"},
	{Header: "NAME", Field: "name"},
	{Header: "ROLE", Field: "role"},
	{Header: "ENABLED", Field: "enabled"},
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users in current tenant",
	Run:   runUserList,
}

var userGetCmd = &cobra.Command{
	Use:   "get <userId>",
	Short: "Get user profile",
	Args:  cobra.ExactArgs(1),
	Run:   runUserGet,
}

var userCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	Long:  `Create a user from a JSON/YAML file or with flags: arsenale user create --email user@example.com --password '...' --role MEMBER`,
	Run:   runUserCreate,
}

var userInviteCmd = &cobra.Command{
	Use:   "invite",
	Short: "Invite a user by email",
	Long:  `Invite a user: arsenale user invite --email user@example.com --role MEMBER`,
	Run:   runUserInvite,
}

var userUpdateRoleCmd = &cobra.Command{
	Use:   "update-role <userId>",
	Short: "Update a user's role",
	Args:  cobra.ExactArgs(1),
	Run:   runUserUpdateRole,
}

var userRemoveCmd = &cobra.Command{
	Use:   "remove <userId>",
	Short: "Remove a user from the tenant",
	Args:  cobra.ExactArgs(1),
	Run:   runUserRemove,
}

var userEnableCmd = &cobra.Command{
	Use:   "enable <userId>",
	Short: "Enable a user account",
	Args:  cobra.ExactArgs(1),
	Run:   runUserEnable,
}

var userDisableCmd = &cobra.Command{
	Use:   "disable <userId>",
	Short: "Disable a user account",
	Args:  cobra.ExactArgs(1),
	Run:   runUserDisable,
}

var userSetExpiryCmd = &cobra.Command{
	Use:   "set-expiry <userId>",
	Short: "Set account expiry for a user",
	Args:  cobra.ExactArgs(1),
	Run:   runUserSetExpiry,
}

var userChangeEmailCmd = &cobra.Command{
	Use:   "change-email <userId>",
	Short: "Change a user's email",
	Args:  cobra.ExactArgs(1),
	Run:   runUserChangeEmail,
}

var userChangePasswordCmd = &cobra.Command{
	Use:   "change-password <userId>",
	Short: "Change a user's password",
	Args:  cobra.ExactArgs(1),
	Run:   runUserChangePassword,
}

var userPermissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Manage user permissions",
}

var userPermissionsGetCmd = &cobra.Command{
	Use:   "get <userId>",
	Short: "Get user permissions",
	Args:  cobra.ExactArgs(1),
	Run:   runUserPermissionsGet,
}

var userPermissionsSelfCmd = &cobra.Command{
	Use:   "self",
	Short: "Get current user's effective permissions",
	Args:  cobra.NoArgs,
	Run:   runUserPermissionsSelf,
}

var userPermissionsSetCmd = &cobra.Command{
	Use:   "set <userId>",
	Short: "Set user permissions",
	Args:  cobra.ExactArgs(1),
	Run:   runUserPermissionsSet,
}

var userSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search users",
	Long:  `Search users: arsenale user search --query "john" --scope tenant`,
	Run:   runUserSearch,
}

var (
	userFromFile      string
	userEmail         string
	userRole          string
	userExpiry        string
	userPassword      string
	userVerification  string
	userSearchQuery   string
	userSearchScope   string
	userSearchTeamID  string
	userPermsFromFile string
)

func init() {
	rootCmd.AddCommand(userCmd)

	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userGetCmd)
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userInviteCmd)
	userCmd.AddCommand(userUpdateRoleCmd)
	userCmd.AddCommand(userRemoveCmd)
	userCmd.AddCommand(userEnableCmd)
	userCmd.AddCommand(userDisableCmd)
	userCmd.AddCommand(userSetExpiryCmd)
	userCmd.AddCommand(userChangeEmailCmd)
	userCmd.AddCommand(userChangePasswordCmd)
	userCmd.AddCommand(userPermissionsCmd)
	userCmd.AddCommand(userSearchCmd)

	userPermissionsCmd.AddCommand(userPermissionsGetCmd)
	userPermissionsCmd.AddCommand(userPermissionsSelfCmd)
	userPermissionsCmd.AddCommand(userPermissionsSetCmd)

	userCreateCmd.Flags().StringVarP(&userFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	userCreateCmd.Flags().StringVar(&userEmail, "email", "", "User email")
	userCreateCmd.Flags().StringVar(&userRole, "role", "", "User role")
	userCreateCmd.Flags().StringVar(&userPassword, "password", "", "Initial password")
	userCreateCmd.Flags().StringVar(&userExpiry, "expiry", "", "Expiry date (RFC3339)")

	userInviteCmd.Flags().StringVar(&userEmail, "email", "", "Email to invite")
	userInviteCmd.Flags().StringVar(&userRole, "role", "MEMBER", "Tenant role")
	userInviteCmd.Flags().StringVar(&userExpiry, "expiry", "", "Expiry date (RFC3339)")
	userInviteCmd.MarkFlagRequired("email")

	userUpdateRoleCmd.Flags().StringVar(&userRole, "role", "", "New role")
	userUpdateRoleCmd.MarkFlagRequired("role")

	userSetExpiryCmd.Flags().StringVar(&userExpiry, "expiry", "", "Expiry date (RFC3339)")
	userSetExpiryCmd.MarkFlagRequired("expiry")

	userChangeEmailCmd.Flags().StringVar(&userEmail, "email", "", "New email address")
	userChangeEmailCmd.Flags().StringVar(&userVerification, "verification-id", "", "Admin action verification ID")
	userChangeEmailCmd.MarkFlagRequired("email")

	userChangePasswordCmd.Flags().StringVar(&userPassword, "password", "", "New password")
	userChangePasswordCmd.Flags().StringVar(&userVerification, "verification-id", "", "Admin action verification ID")
	userChangePasswordCmd.MarkFlagRequired("password")

	userPermissionsSetCmd.Flags().StringVarP(&userPermsFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	userPermissionsSetCmd.MarkFlagRequired("from-file")

	userSearchCmd.Flags().StringVar(&userSearchQuery, "query", "", "Search query")
	userSearchCmd.Flags().StringVar(&userSearchScope, "scope", "tenant", "Search scope (tenant or team)")
	userSearchCmd.Flags().StringVar(&userSearchTeamID, "team-id", "", "Team ID when --scope team")
	userSearchCmd.MarkFlagRequired("query")
}

func requireTenantID(cfg *CLIConfig) string {
	tid := cfg.resolveTenantID()
	if tid == "" {
		fatal("tenant ID required. Set it with --tenant flag or 'arsenale tenant switch'")
	}
	return tid
}

func runUserList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	body, status, err := apiGet(fmt.Sprintf("/api/tenants/%s/users", tid), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, userColumns)
}

func runUserGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	body, status, err := apiGet(fmt.Sprintf("/api/tenants/%s/users/%s/profile", tid, args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, userColumns)
}

func runUserCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)

	var data []byte
	var err error

	if userFromFile != "" {
		data, err = readResourceFromFileOrStdin(userFromFile)
		if err != nil {
			fatal("%v", err)
		}
	} else {
		data, err = buildUserCreateBody(userEmail, userRole, userPassword, userExpiry)
		if err != nil {
			fatal("%v", err)
		}
	}

	body, status, err := apiPost(fmt.Sprintf("/api/tenants/%s/users", tid), json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "user.id")
}

func runUserInvite(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	payload, err := buildUserInviteBody(userEmail, userRole, userExpiry)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost(fmt.Sprintf("/api/tenants/%s/invite", tid), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "userId")
}

func runUserUpdateRole(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	payload := map[string]string{
		"role": userRole,
	}

	body, status, err := apiPut(fmt.Sprintf("/api/tenants/%s/users/%s", tid, args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("User %q role updated to %q\n", args[0], userRole)
	}
}

func runUserRemove(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	body, status, err := apiDelete(fmt.Sprintf("/api/tenants/%s/users/%s", tid, args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("User", args[0])
}

func runUserEnable(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	payload := map[string]bool{
		"enabled": true,
	}

	body, status, err := apiPatch(fmt.Sprintf("/api/tenants/%s/users/%s/enabled", tid, args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("User %q enabled\n", args[0])
	}
}

func runUserDisable(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	payload := map[string]bool{
		"enabled": false,
	}

	body, status, err := apiPatch(fmt.Sprintf("/api/tenants/%s/users/%s/enabled", tid, args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("User %q disabled\n", args[0])
	}
}

func runUserSetExpiry(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	payload := map[string]string{
		"expiresAt": userExpiry,
	}

	body, status, err := apiPatch(fmt.Sprintf("/api/tenants/%s/users/%s/expiry", tid, args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("User %q expiry set to %s\n", args[0], userExpiry)
	}
}

func runUserChangeEmail(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	payload := map[string]string{
		"newEmail":       userEmail,
		"verificationId": userVerification,
	}

	body, status, err := apiPut(fmt.Sprintf("/api/tenants/%s/users/%s/email", tid, args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("User %q email changed to %q\n", args[0], userEmail)
	}
}

func runUserChangePassword(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	payload := map[string]string{
		"newPassword":    userPassword,
		"verificationId": userVerification,
	}

	body, status, err := apiPut(fmt.Sprintf("/api/tenants/%s/users/%s/password", tid, args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("User %q password changed\n", args[0])
	}
}

func runUserPermissionsGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	body, status, err := apiGet(fmt.Sprintf("/api/tenants/%s/users/%s/permissions", tid, args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "USER_ID", Field: "userId"},
		{Header: "PERMISSIONS", Field: "permissions"},
	})
}

func runUserPermissionsSelf(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/user/permissions", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "TENANT", Field: "tenantId"},
		{Header: "ROLE", Field: "role"},
		{Header: "PERMISSIONS", Field: "permissions"},
	})
}

func runUserPermissionsSet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	tid := requireTenantID(cfg)
	data, err := readResourceFromFileOrStdin(userPermsFromFile)
	if err != nil {
		fatal("%v", err)
	}
	data, err = normalizeUserPermissionsPayload(data)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut(fmt.Sprintf("/api/tenants/%s/users/%s/permissions", tid, args[0]), json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Printf("Permissions updated for user %q\n", args[0])
	}
}

func runUserSearch(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	params, err := buildUserSearchParams(userSearchQuery, userSearchScope, userSearchTeamID)
	if err != nil {
		fatal("%v", err)
	}
	body, status, err := apiRequestWithParams("GET", "/api/user/search", params, nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, userColumns)
}

func buildUserSearchParams(query, scope, teamID string) (url.Values, error) {
	if scope == "" {
		scope = "tenant"
	}
	if scope != "tenant" && scope != "team" {
		return nil, fmt.Errorf("scope must be one of tenant, team")
	}
	if scope == "team" && teamID == "" {
		return nil, fmt.Errorf("team-id is required when scope is team")
	}

	params := url.Values{"q": {query}, "scope": {scope}}
	if teamID != "" {
		params.Set("teamId", teamID)
	}
	return params, nil
}

func buildUserCreateBody(email, role, password, expiry string) ([]byte, error) {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil, fmt.Errorf("provide --from-file or --email and --password")
	}
	payload := map[string]interface{}{
		"email":    email,
		"password": password,
		"role":     normalizeTenantRole(role),
	}
	if value := strings.TrimSpace(expiry); value != "" {
		payload["expiresAt"] = value
	}
	return buildJSONBody(payload)
}

func buildUserInviteBody(email, role, expiry string) (map[string]interface{}, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	payload := map[string]interface{}{
		"email": email,
		"role":  normalizeTenantRole(role),
	}
	if value := strings.TrimSpace(expiry); value != "" {
		payload["expiresAt"] = value
	}
	return payload, nil
}

func normalizeTenantRole(role string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return "MEMBER"
	}
	return strings.ToUpper(role)
}

func normalizeUserPermissionsPayload(data []byte) ([]byte, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("permissions payload must be a JSON/YAML object: %w", err)
	}
	if _, ok := payload["overrides"]; ok {
		return data, nil
	}
	return json.Marshal(map[string]json.RawMessage{
		"overrides": json.RawMessage(data),
	})
}
