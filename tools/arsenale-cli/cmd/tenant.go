package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var tenantColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "SLUG", Field: "slug"},
	{Header: "MFA_REQ", Field: "mfaRequired"},
	{Header: "USERS", Field: "userCount"},
	{Header: "TEAMS", Field: "teamCount"},
}

var tenantCreateColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "SLUG", Field: "slug"},
	{Header: "USERS", Field: "userCount"},
	{Header: "TEAMS", Field: "teamCount"},
}

var tenantListColumns = []Column{
	{Header: "ID", Field: "tenantId"},
	{Header: "NAME", Field: "name"},
	{Header: "ROLE", Field: "role"},
	{Header: "STATUS", Field: "status"},
	{Header: "ACTIVE", Field: "isActive"},
	{Header: "PENDING", Field: "pending"},
}

var tenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "Manage tenants",
}

var tenantGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get current tenant details",
	Run:   runTenantGet,
}

var tenantListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tenants for current user",
	Run:   runTenantList,
}

var tenantCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new tenant",
	Long:  `Create a tenant from a JSON/YAML file or with flags: arsenale tenant create --name "My Tenant"`,
	Run:   runTenantCreate,
}

var tenantUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a tenant",
	Args:  cobra.ExactArgs(1),
	Run:   runTenantUpdate,
}

var tenantDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a tenant",
	Args:  cobra.ExactArgs(1),
	Run:   runTenantDelete,
}

var tenantSwitchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch to a different tenant",
	Long:  `Switch tenant: arsenale tenant switch --id <tenantId>`,
	Run:   runTenantSwitch,
}

var tenantIPAllowlistCmd = &cobra.Command{
	Use:   "ip-allowlist",
	Short: "Manage tenant IP allowlist",
}

var tenantIPAllowlistGetCmd = &cobra.Command{
	Use:   "get <tenantId>",
	Short: "Get IP allowlist for a tenant",
	Args:  cobra.ExactArgs(1),
	Run:   runTenantIPAllowlistGet,
}

var tenantIPAllowlistSetCmd = &cobra.Command{
	Use:   "set <tenantId>",
	Short: "Set IP allowlist for a tenant",
	Args:  cobra.ExactArgs(1),
	Run:   runTenantIPAllowlistSet,
}

var tenantMFAStatsCmd = &cobra.Command{
	Use:   "mfa-stats <tenantId>",
	Short: "Get MFA statistics for a tenant",
	Args:  cobra.ExactArgs(1),
	Run:   runTenantMFAStats,
}

var (
	tenantFromFile string
	tenantName     string
	tenantSwitchID string
	tenantIPFile   string
)

func init() {
	rootCmd.AddCommand(tenantCmd)

	tenantCmd.AddCommand(tenantGetCmd)
	tenantCmd.AddCommand(tenantListCmd)
	tenantCmd.AddCommand(tenantCreateCmd)
	tenantCmd.AddCommand(tenantUpdateCmd)
	tenantCmd.AddCommand(tenantDeleteCmd)
	tenantCmd.AddCommand(tenantSwitchCmd)
	tenantCmd.AddCommand(tenantIPAllowlistCmd)
	tenantCmd.AddCommand(tenantMFAStatsCmd)

	tenantIPAllowlistCmd.AddCommand(tenantIPAllowlistGetCmd)
	tenantIPAllowlistCmd.AddCommand(tenantIPAllowlistSetCmd)

	tenantCreateCmd.Flags().StringVarP(&tenantFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	tenantCreateCmd.Flags().StringVar(&tenantName, "name", "", "Tenant name")

	tenantUpdateCmd.Flags().StringVarP(&tenantFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	tenantUpdateCmd.MarkFlagRequired("from-file")

	tenantSwitchCmd.Flags().StringVar(&tenantSwitchID, "id", "", "Tenant ID to switch to")
	tenantSwitchCmd.MarkFlagRequired("id")

	tenantIPAllowlistSetCmd.Flags().StringVarP(&tenantIPFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	tenantIPAllowlistSetCmd.MarkFlagRequired("from-file")
}

func runTenantGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/tenants/mine", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, tenantColumns)
}

func runTenantList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/tenants/mine/all", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, tenantListColumns)
}

func runTenantCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}
	if err := ensureMultiTenancyEnabled(cfg); err != nil {
		fatal("%v", err)
	}

	var data []byte
	var err error

	if tenantFromFile != "" {
		data, err = readResourceFromFileOrStdin(tenantFromFile)
		if err != nil {
			fatal("%v", err)
		}
	} else {
		if tenantName == "" {
			fatal("provide --from-file or --name")
		}
		data, err = buildJSONBody(map[string]interface{}{
			"name": tenantName,
		})
		if err != nil {
			fatal("%v", err)
		}
	}

	body, status, err := apiPost("/api/tenants", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	tenantID := applyTenantAuthResponse(cfg, body, "")
	if err := saveConfig(cfg); err != nil {
		fatal("save config: %v", err)
	}
	if quiet {
		fmt.Fprintln(os.Stdout, tenantID)
		return
	}
	if outputFormat == "json" || outputFormat == "yaml" {
		if err := printer().PrintSingle(body, tenantColumns); err != nil {
			fatal("%v", err)
		}
		return
	}
	if err := printer().PrintSingle(tenantCreateDisplayBody(body), tenantCreateColumns); err != nil {
		fatal("%v", err)
	}
}

func runTenantUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(tenantFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/tenants/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, tenantColumns)
}

func runTenantDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/tenants/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Tenant", args[0])
}

func runTenantSwitch(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}
	if err := ensureMultiTenancyEnabled(cfg); err != nil {
		fatal("%v", err)
	}

	payload := map[string]string{
		"tenantId": tenantSwitchID,
	}

	body, status, err := apiPost("/api/auth/switch-tenant", payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	applyTenantAuthResponse(cfg, body, tenantSwitchID)
	if err := saveConfig(cfg); err != nil {
		fatal("save config: %v", err)
	}

	if outputFormat == "json" || outputFormat == "yaml" {
		if err := printer().PrintSingle(body, tenantColumns); err != nil {
			fatal("%v", err)
		}
		return
	}
	if !quiet {
		fmt.Printf("Switched to tenant %q\n", tenantSwitchID)
	}
}

func runTenantIPAllowlistGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/tenants/%s/ip-allowlist", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "CIDR", Field: "cidr"},
		{Header: "DESCRIPTION", Field: "description"},
	})
}

func runTenantIPAllowlistSet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(tenantIPFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut(fmt.Sprintf("/api/tenants/%s/ip-allowlist", args[0]), json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if outputFormat == "json" || outputFormat == "yaml" {
		if err := printer().PrintSingle(body, nil); err != nil {
			fatal("%v", err)
		}
		return
	}
	if !quiet {
		fmt.Println("IP allowlist updated")
	}
}

func runTenantMFAStats(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/tenants/%s/mfa-stats", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "TOTAL_USERS", Field: "totalUsers"},
		{Header: "MFA_ENABLED", Field: "mfaEnabled"},
		{Header: "MFA_DISABLED", Field: "mfaDisabled"},
	})
}

func applyTenantAuthResponse(cfg *CLIConfig, body []byte, fallbackTenantID string) string {
	var resp struct {
		AccessToken string `json:"accessToken"`
		Tenant      struct {
			ID string `json:"id"`
		} `json:"tenant"`
	}
	_ = json.Unmarshal(body, &resp)
	if resp.AccessToken != "" {
		cfg.AccessToken = resp.AccessToken
		cfg.TokenExpiry = persistentCLITokenExpiry
	}
	tenantID := fallbackTenantID
	if resp.Tenant.ID != "" {
		tenantID = resp.Tenant.ID
	}
	cfg.TenantID = tenantID
	return tenantID
}

func tenantCreateDisplayBody(body []byte) []byte {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return body
	}
	if tenant, ok := payload["tenant"]; ok {
		return tenant
	}
	return body
}
