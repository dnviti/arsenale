package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	outputFormat string
	noHeaders    bool
	quiet        bool
	verbose      bool
	serverFlag   string
	tenantFlag   string
)

var rootCmd = &cobra.Command{
	Use:   "arsenale",
	Short: "Arsenale Connect CLI — manage your Arsenale platform",
	Long: `Arsenale Connect CLI is a command-line tool for managing the Arsenale
platform. It provides full administrative control over connections,
users, gateways, secrets, policies, and more.

Authenticate with 'arsenale login' to get started.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var versionCmd = &cobra.Command{
	Use:    "version",
	Short:  "Show version",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("arsenale-cli v%s\n", rootCmd.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table|json|yaml")
	rootCmd.PersistentFlags().BoolVar(&noHeaders, "no-headers", false, "Omit table headers (table format only)")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Minimal output (IDs only)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose error output")
	rootCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "Override server URL")
	rootCmd.PersistentFlags().StringVar(&tenantFlag, "tenant", "", "Override tenant ID")
}

// getCfg loads the CLI config and applies any flag overrides.
func getCfg() *CLIConfig {
	cfg := loadConfig()
	if serverFlag != "" {
		cfg.ServerURL = serverFlag
	}
	if tenantFlag != "" {
		cfg.TenantID = tenantFlag
	}
	return cfg
}

// printer returns a Printer configured from global flags.
func printer() *Printer {
	return &Printer{
		Format:    outputFormat,
		NoHeaders: noHeaders,
		Quiet:     quiet,
	}
}
