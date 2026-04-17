package cmd

import (
	"github.com/spf13/cobra"
	"strings"
	"testing"
)

func TestFileSSHCmdDescriptionsUseManagedBrowserLanguage(t *testing.T) {
	// Verify the SSH file command uses sandbox language, not remote-browse wording.
	if !strings.Contains(strings.ToLower(fileSSHCmd.Short), "sandbox") {
		t.Errorf("fileSSHCmd.Short should mention sandbox, got: %s", fileSSHCmd.Short)
	}
	if strings.Contains(strings.ToLower(fileSSHCmd.Short), "sftp") {
		t.Errorf("fileSSHCmd.Short should not contain 'SFTP', got: %s", fileSSHCmd.Short)
	}
}

func TestFileSSHUploadDescriptionUsesManagedStorage(t *testing.T) {
	if !strings.Contains(strings.ToLower(fileSSHUploadCmd.Short), "sandbox") {
		t.Errorf("fileSSHUploadCmd.Short should mention sandbox, got: %s", fileSSHUploadCmd.Short)
	}
}

func TestFileSSHDownloadDescriptionUsesManagedStorage(t *testing.T) {
	if strings.Contains(fileSSHDownloadCmd.Short, "staged storage") {
		t.Errorf("fileSSHDownloadCmd.Short should not contain 'staged storage', got: %s", fileSSHDownloadCmd.Short)
	}
}

func TestFileCmdConnectionFlagDescription(t *testing.T) {
	// Check that connection flag descriptions don't say "shared drive" or remote browsing.
	for _, cmd := range []*cobra.Command{fileListCmd, fileUploadCmd, fileDownloadCmd, fileDeleteCmd, fileHistoryListCmd, fileHistoryDownloadCmd, fileHistoryRestoreCmd, fileHistoryDeleteCmd} {
		flag := cmd.Flags().Lookup("connection")
		if flag == nil {
			t.Errorf("command %s missing connection flag", cmd.Name())
			continue
		}
		if strings.Contains(flag.Usage, "shared drive") {
			t.Errorf("command %s connection flag should not mention 'shared drive', got: %s",
				cmd.Name(), flag.Usage)
		}
		if strings.Contains(strings.ToLower(flag.Usage), "remote") {
			t.Errorf("command %s connection flag should not mention remote browsing, got: %s",
				cmd.Name(), flag.Usage)
		}
	}
}

func TestFileSSHUploadFlagUsesSandboxDestination(t *testing.T) {
	flag := fileSSHUploadCmd.Flags().Lookup("to")
	if flag == nil {
		t.Fatal("missing --to flag")
	}
	if !strings.Contains(flag.Usage, "sandbox-relative") {
		t.Fatalf("--to usage = %q; want sandbox-relative wording", flag.Usage)
	}
	if !strings.Contains(flag.Usage, "workspace/current/") {
		t.Fatalf("--to usage = %q; want workspace/current/ wording", flag.Usage)
	}
	if hidden := fileSSHUploadCmd.Flags().Lookup("remote-path"); hidden != nil && !hidden.Hidden {
		t.Fatalf("remote-path alias should be hidden")
	}
}
