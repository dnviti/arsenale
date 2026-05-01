package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear local authentication tokens",
	Run:   runLogout,
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}

func runLogout(cmd *cobra.Command, args []string) {
	cfg := loadConfig()
	cfg.AccessToken = ""
	cfg.RefreshToken = ""
	cfg.TokenExpiry = ""

	if err := saveConfig(cfg); err != nil {
		fatal("failed to save config: %v", err)
	}

	fmt.Println("Logged out. Tokens cleared.")
}
