package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestBashCompletionIncludesCommandTree(t *testing.T) {
	var buf bytes.Buffer
	if err := rootCmd.GenBashCompletion(&buf); err != nil {
		t.Fatalf("GenBashCompletion failed: %v", err)
	}
	output := buf.String()
	for _, want := range []string{
		"connect",
		"connection",
		"gateway",
		"instances",
		"session",
		"vault",
		"rdp",
		"vnc",
		"--server",
		"--config",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("completion output missing %q", want)
		}
	}
}

func TestCompletionCommandCompletesShellNamesOnly(t *testing.T) {
	values, directive := completionCmd.ValidArgsFunction(completionCmd, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("directive = %v; want no file completion", directive)
	}
	for _, want := range []string{"bash", "zsh", "fish", "powershell"} {
		if !containsString(values, want) {
			t.Fatalf("completion shell names missing %q from %v", want, values)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
