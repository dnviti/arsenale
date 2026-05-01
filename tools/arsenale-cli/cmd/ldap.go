package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var ldapCmd = &cobra.Command{
	Use:   "ldap",
	Short: "Manage LDAP integration",
}

var ldapStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get LDAP status",
	Run:   runLdapStatus,
}

var ldapTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test LDAP connection",
	Run:   runLdapTest,
}

var ldapSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Trigger LDAP sync",
	Run:   runLdapSync,
}

func init() {
	rootCmd.AddCommand(ldapCmd)

	ldapCmd.AddCommand(ldapStatusCmd)
	ldapCmd.AddCommand(ldapTestCmd)
	ldapCmd.AddCommand(ldapSyncCmd)
}

func runLdapStatus(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/ldap/status", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "STATUS", Field: "status"},
		{Header: "CONNECTED", Field: "connected"},
		{Header: "LAST_SYNC", Field: "lastSync"},
	})
}

func runLdapTest(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/ldap/test", nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Println("LDAP connection test successful")
	}
}

func runLdapSync(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/ldap/sync", nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if !quiet {
		fmt.Println("LDAP sync triggered")
	}
}
