package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for arsenale.

Examples:
  # Bash (add to ~/.bashrc)
  source <(arsenale completion bash)

  # Zsh (add to ~/.zshrc)
  source <(arsenale completion zsh)

  # Fish
  arsenale completion fish | source

  # PowerShell
  arsenale completion powershell | Out-String | Invoke-Expression`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Run:       runCompletion,
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

func runCompletion(cmd *cobra.Command, args []string) {
	switch args[0] {
	case "bash":
		rootCmd.GenBashCompletion(os.Stdout)
	case "zsh":
		rootCmd.GenZshCompletion(os.Stdout)
	case "fish":
		rootCmd.GenFishCompletion(os.Stdout, true)
	case "powershell":
		rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		fatal("unsupported shell: %s", args[0])
	}
}
