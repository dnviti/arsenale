package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var dbAuditLogColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "USER", Field: "user"},
	{Header: "QUERY", Field: "query"},
	{Header: "CONNECTION", Field: "connection"},
	{Header: "TIMESTAMP", Field: "timestamp"},
}

var dbAuditFirewallColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "ACTION", Field: "action"},
	{Header: "ENABLED", Field: "enabled"},
}

var dbAuditMaskingColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TABLE", Field: "table"},
	{Header: "COLUMN", Field: "column"},
}

var dbAuditRateLimitColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "LIMIT", Field: "limit"},
	{Header: "WINDOW", Field: "window"},
}

// Top-level command
var dbAuditCmd = &cobra.Command{
	Use:   "db-audit",
	Short: "Manage database audit logs, firewall rules, masking policies, and rate limits",
}

// --- Logs subcommands ---

var dbAuditLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "List database audit logs",
	Run:   runDbAuditLogs,
}

var dbAuditConnectionsCmd = &cobra.Command{
	Use:   "connections",
	Short: "List database audit log connections",
	Run:   runDbAuditConnections,
}

var dbAuditUsersCmd = &cobra.Command{
	Use:   "users",
	Short: "List database audit log users",
	Run:   runDbAuditUsers,
}

// --- Firewall rule subcommands ---

var dbAuditFirewallRuleCmd = &cobra.Command{
	Use:   "firewall-rule",
	Short: "Manage database firewall rules",
}

var dbAuditFirewallRuleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all firewall rules",
	Run:   runDbAuditFirewallRuleList,
}

var dbAuditFirewallRuleGetCmd = &cobra.Command{
	Use:   "get <ruleId>",
	Short: "Get firewall rule details",
	Args:  cobra.ExactArgs(1),
	Run:   runDbAuditFirewallRuleGet,
}

var dbAuditFirewallRuleCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new firewall rule",
	Long:  `Create a firewall rule from a JSON/YAML file: arsenale db-audit firewall-rule create --from-file rule.yaml`,
	Run:   runDbAuditFirewallRuleCreate,
}

var dbAuditFirewallRuleUpdateCmd = &cobra.Command{
	Use:   "update <ruleId>",
	Short: "Update a firewall rule",
	Args:  cobra.ExactArgs(1),
	Run:   runDbAuditFirewallRuleUpdate,
}

var dbAuditFirewallRuleDeleteCmd = &cobra.Command{
	Use:   "delete <ruleId>",
	Short: "Delete a firewall rule",
	Args:  cobra.ExactArgs(1),
	Run:   runDbAuditFirewallRuleDelete,
}

// --- Masking policy subcommands ---

var dbAuditMaskingPolicyCmd = &cobra.Command{
	Use:   "masking-policy",
	Short: "Manage database masking policies",
}

var dbAuditMaskingPolicyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all masking policies",
	Run:   runDbAuditMaskingPolicyList,
}

var dbAuditMaskingPolicyGetCmd = &cobra.Command{
	Use:   "get <policyId>",
	Short: "Get masking policy details",
	Args:  cobra.ExactArgs(1),
	Run:   runDbAuditMaskingPolicyGet,
}

var dbAuditMaskingPolicyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new masking policy",
	Long:  `Create a masking policy from a JSON/YAML file: arsenale db-audit masking-policy create --from-file policy.yaml`,
	Run:   runDbAuditMaskingPolicyCreate,
}

var dbAuditMaskingPolicyUpdateCmd = &cobra.Command{
	Use:   "update <policyId>",
	Short: "Update a masking policy",
	Args:  cobra.ExactArgs(1),
	Run:   runDbAuditMaskingPolicyUpdate,
}

var dbAuditMaskingPolicyDeleteCmd = &cobra.Command{
	Use:   "delete <policyId>",
	Short: "Delete a masking policy",
	Args:  cobra.ExactArgs(1),
	Run:   runDbAuditMaskingPolicyDelete,
}

// --- Rate limit subcommands ---

var dbAuditRateLimitCmd = &cobra.Command{
	Use:   "rate-limit",
	Short: "Manage database rate limit policies",
}

var dbAuditRateLimitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all rate limit policies",
	Run:   runDbAuditRateLimitList,
}

var dbAuditRateLimitGetCmd = &cobra.Command{
	Use:   "get <policyId>",
	Short: "Get rate limit policy details",
	Args:  cobra.ExactArgs(1),
	Run:   runDbAuditRateLimitGet,
}

var dbAuditRateLimitCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new rate limit policy",
	Long:  `Create a rate limit policy from a JSON/YAML file: arsenale db-audit rate-limit create --from-file policy.yaml`,
	Run:   runDbAuditRateLimitCreate,
}

var dbAuditRateLimitUpdateCmd = &cobra.Command{
	Use:   "update <policyId>",
	Short: "Update a rate limit policy",
	Args:  cobra.ExactArgs(1),
	Run:   runDbAuditRateLimitUpdate,
}

var dbAuditRateLimitDeleteCmd = &cobra.Command{
	Use:   "delete <policyId>",
	Short: "Delete a rate limit policy",
	Args:  cobra.ExactArgs(1),
	Run:   runDbAuditRateLimitDelete,
}

var (
	dbAuditFirewallFromFile string
	dbAuditMaskingFromFile  string
	dbAuditRateLimitFromFile string
)

func init() {
	rootCmd.AddCommand(dbAuditCmd)

	// Logs
	dbAuditCmd.AddCommand(dbAuditLogsCmd)
	dbAuditCmd.AddCommand(dbAuditConnectionsCmd)
	dbAuditCmd.AddCommand(dbAuditUsersCmd)

	// Firewall rules
	dbAuditCmd.AddCommand(dbAuditFirewallRuleCmd)
	dbAuditFirewallRuleCmd.AddCommand(dbAuditFirewallRuleListCmd)
	dbAuditFirewallRuleCmd.AddCommand(dbAuditFirewallRuleGetCmd)
	dbAuditFirewallRuleCmd.AddCommand(dbAuditFirewallRuleCreateCmd)
	dbAuditFirewallRuleCmd.AddCommand(dbAuditFirewallRuleUpdateCmd)
	dbAuditFirewallRuleCmd.AddCommand(dbAuditFirewallRuleDeleteCmd)

	dbAuditFirewallRuleCreateCmd.Flags().StringVarP(&dbAuditFirewallFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	dbAuditFirewallRuleCreateCmd.MarkFlagRequired("from-file")
	dbAuditFirewallRuleUpdateCmd.Flags().StringVarP(&dbAuditFirewallFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	dbAuditFirewallRuleUpdateCmd.MarkFlagRequired("from-file")

	// Masking policies
	dbAuditCmd.AddCommand(dbAuditMaskingPolicyCmd)
	dbAuditMaskingPolicyCmd.AddCommand(dbAuditMaskingPolicyListCmd)
	dbAuditMaskingPolicyCmd.AddCommand(dbAuditMaskingPolicyGetCmd)
	dbAuditMaskingPolicyCmd.AddCommand(dbAuditMaskingPolicyCreateCmd)
	dbAuditMaskingPolicyCmd.AddCommand(dbAuditMaskingPolicyUpdateCmd)
	dbAuditMaskingPolicyCmd.AddCommand(dbAuditMaskingPolicyDeleteCmd)

	dbAuditMaskingPolicyCreateCmd.Flags().StringVarP(&dbAuditMaskingFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	dbAuditMaskingPolicyCreateCmd.MarkFlagRequired("from-file")
	dbAuditMaskingPolicyUpdateCmd.Flags().StringVarP(&dbAuditMaskingFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	dbAuditMaskingPolicyUpdateCmd.MarkFlagRequired("from-file")

	// Rate limit policies
	dbAuditCmd.AddCommand(dbAuditRateLimitCmd)
	dbAuditRateLimitCmd.AddCommand(dbAuditRateLimitListCmd)
	dbAuditRateLimitCmd.AddCommand(dbAuditRateLimitGetCmd)
	dbAuditRateLimitCmd.AddCommand(dbAuditRateLimitCreateCmd)
	dbAuditRateLimitCmd.AddCommand(dbAuditRateLimitUpdateCmd)
	dbAuditRateLimitCmd.AddCommand(dbAuditRateLimitDeleteCmd)

	dbAuditRateLimitCreateCmd.Flags().StringVarP(&dbAuditRateLimitFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	dbAuditRateLimitCreateCmd.MarkFlagRequired("from-file")
	dbAuditRateLimitUpdateCmd.Flags().StringVarP(&dbAuditRateLimitFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	dbAuditRateLimitUpdateCmd.MarkFlagRequired("from-file")
}

// --- Logs run functions ---

func runDbAuditLogs(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/db-audit/logs", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, dbAuditLogColumns)
}

func runDbAuditConnections(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/db-audit/logs/connections", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "ID", Field: "id"},
		{Header: "NAME", Field: "name"},
		{Header: "TYPE", Field: "type"},
	})
}

func runDbAuditUsers(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/db-audit/logs/users", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, []Column{
		{Header: "ID", Field: "id"},
		{Header: "EMAIL", Field: "email"},
		{Header: "NAME", Field: "name"},
	})
}

// --- Firewall rule run functions ---

func runDbAuditFirewallRuleList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/db-audit/firewall-rules", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, dbAuditFirewallColumns)
}

func runDbAuditFirewallRuleGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/db-audit/firewall-rules/%s", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, dbAuditFirewallColumns)
}

func runDbAuditFirewallRuleCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(dbAuditFirewallFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/db-audit/firewall-rules", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runDbAuditFirewallRuleUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(dbAuditFirewallFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut(fmt.Sprintf("/api/db-audit/firewall-rules/%s", args[0]), json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, dbAuditFirewallColumns)
}

func runDbAuditFirewallRuleDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete(fmt.Sprintf("/api/db-audit/firewall-rules/%s", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Firewall rule", args[0])
}

// --- Masking policy run functions ---

func runDbAuditMaskingPolicyList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/db-audit/masking-policies", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, dbAuditMaskingColumns)
}

func runDbAuditMaskingPolicyGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/db-audit/masking-policies/%s", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, dbAuditMaskingColumns)
}

func runDbAuditMaskingPolicyCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(dbAuditMaskingFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/db-audit/masking-policies", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runDbAuditMaskingPolicyUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(dbAuditMaskingFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut(fmt.Sprintf("/api/db-audit/masking-policies/%s", args[0]), json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, dbAuditMaskingColumns)
}

func runDbAuditMaskingPolicyDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete(fmt.Sprintf("/api/db-audit/masking-policies/%s", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Masking policy", args[0])
}

// --- Rate limit run functions ---

func runDbAuditRateLimitList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/db-audit/rate-limit-policies", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, dbAuditRateLimitColumns)
}

func runDbAuditRateLimitGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/db-audit/rate-limit-policies/%s", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, dbAuditRateLimitColumns)
}

func runDbAuditRateLimitCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(dbAuditRateLimitFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/db-audit/rate-limit-policies", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runDbAuditRateLimitUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(dbAuditRateLimitFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut(fmt.Sprintf("/api/db-audit/rate-limit-policies/%s", args[0]), json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, dbAuditRateLimitColumns)
}

func runDbAuditRateLimitDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete(fmt.Sprintf("/api/db-audit/rate-limit-policies/%s", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintDeleted("Rate limit policy", args[0])
}
