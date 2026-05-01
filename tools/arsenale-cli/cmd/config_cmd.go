package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or manage CLI configuration",
	Run:   runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Valid keys:
  server_url    - Arsenale server URL
  tenant_id     - Default tenant ID
  cache_ttl     - Cache TTL duration`,
	Args: cobra.ExactArgs(2),
	Run:  runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	Run:   runConfigGet,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) {
	cfg := loadConfig()

	fmt.Println("Arsenale CLI Configuration")
	fmt.Println("==========================")
	fmt.Printf("Config file:   %s\n", configPath())
	fmt.Printf("Server URL:    %s\n", cfg.ServerURL)
	fmt.Printf("Tenant ID:     %s\n", valueOrDash(cfg.TenantID))
	fmt.Printf("Cache TTL:     %s\n", cfg.CacheTTL)

	if cfg.AccessToken != "" {
		if cfg.isTokenValid() {
			fmt.Println("Auth status:   authenticated")
		} else {
			fmt.Println("Auth status:   token expired (run 'arsenale login')")
		}
	} else {
		fmt.Println("Auth status:   not authenticated (run 'arsenale login')")
	}
}

func runConfigSet(cmd *cobra.Command, args []string) {
	cfg := loadConfig()
	key, value := args[0], args[1]

	switch key {
	case "server_url":
		cfg.ServerURL = value
	case "tenant_id":
		cfg.TenantID = value
	case "cache_ttl":
		cfg.CacheTTL = value
	default:
		fatal("unknown config key: %s", key)
	}

	if err := saveConfig(cfg); err != nil {
		fatal("save config: %v", err)
	}
	fmt.Printf("%s = %s\n", key, value)
}

func runConfigGet(cmd *cobra.Command, args []string) {
	cfg := loadConfig()

	switch args[0] {
	case "server_url":
		fmt.Println(cfg.ServerURL)
	case "tenant_id":
		fmt.Println(valueOrDash(cfg.TenantID))
	case "cache_ttl":
		fmt.Println(cfg.CacheTTL)
	case "access_token":
		fmt.Println(valueOrDash(cfg.AccessToken))
	default:
		fatal("unknown config key: %s", args[0])
	}
}

func valueOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
