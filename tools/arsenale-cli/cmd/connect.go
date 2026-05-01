package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a target via Arsenale proxy",
}

var connectSSHCmd = &cobra.Command{
	Use:   "ssh <connection-name>",
	Short: "Connect to an SSH target via Arsenale proxy",
	Args:  cobra.ExactArgs(1),
	Run:   runConnectSSH,
}

var connectRDPCmd = &cobra.Command{
	Use:   "rdp <connection-name>",
	Short: "Connect to an RDP target via RD Gateway",
	Args:  cobra.ExactArgs(1),
	Run:   runConnectRDP,
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.AddCommand(connectSSHCmd)
	connectCmd.AddCommand(connectRDPCmd)
}

func runConnectSSH(cmd *cobra.Command, args []string) {
	name := args[0]
	cfg := getCfg()

	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	conn, err := findConnectionByName(name, cfg)
	if err != nil {
		fatal("%v", err)
	}

	if conn.Type != "SSH" {
		fatal("connection '%s' is type %s, not SSH", name, conn.Type)
	}

	body := map[string]string{
		"connectionId": conn.ID,
	}

	respBody, status, err := apiPost("/api/sessions/ssh-proxy/token", body, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, respBody)

	var tokenResp struct {
		Token                  string `json:"token"`
		ExpiresIn              int    `json:"expiresIn"`
		ConnectionInstructions struct {
			Command string `json:"command"`
			Port    int    `json:"port"`
			Host    string `json:"host"`
			Note    string `json:"note"`
		} `json:"connectionInstructions"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		fatal("failed to parse token response: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "arsenale-ssh-*")
	if err != nil {
		fatal("%v", err)
	}
	defer os.RemoveAll(tmpDir)

	sshConfigPath := filepath.Join(tmpDir, "ssh_config")
	proxyHost := tokenResp.ConnectionInstructions.Host
	proxyPort := tokenResp.ConnectionInstructions.Port

	sshConfig := fmt.Sprintf(`Host arsenale-target
    HostName %s
    Port %d
    ProxyCommand echo '%s' | nc %s %d
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    LogLevel ERROR
`,
		conn.Host, conn.Port, tokenResp.Token, proxyHost, proxyPort,
	)

	if err := os.WriteFile(sshConfigPath, []byte(sshConfig), 0600); err != nil {
		fatal("failed to write SSH config: %v", err)
	}

	fmt.Printf("Connecting to %s (%s:%d) via Arsenale SSH proxy...\n", name, conn.Host, conn.Port)
	fmt.Printf("Proxy: %s:%d (token expires in %ds)\n\n", proxyHost, proxyPort, tokenResp.ExpiresIn)

	sshCmd := exec.Command("ssh", "-F", sshConfigPath, "arsenale-target")
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		if sshCmd.Process != nil {
			sshCmd.Process.Signal(syscall.SIGTERM)
		}
	}()

	if err := sshCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fatal("SSH client failed: %v", err)
	}
}

func runConnectRDP(cmd *cobra.Command, args []string) {
	name := args[0]
	cfg := getCfg()

	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	conn, err := findConnectionByName(name, cfg)
	if err != nil {
		fatal("%v", err)
	}

	if conn.Type != "RDP" {
		fatal("connection '%s' is type %s, not RDP", name, conn.Type)
	}

	respBody, status, err := apiGet(fmt.Sprintf("/api/rdgw/connections/%s/rdpfile", conn.ID), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, respBody)

	tmpDir, err := os.MkdirTemp("", "arsenale-rdp-*")
	if err != nil {
		fatal("%v", err)
	}
	defer os.RemoveAll(tmpDir)

	safeName := sanitizeFilename(name)
	rdpFilePath := filepath.Join(tmpDir, safeName+".rdp")

	if err := os.WriteFile(rdpFilePath, respBody, 0600); err != nil {
		fatal("failed to write .rdp file: %v", err)
	}

	fmt.Printf("Connecting to %s (%s:%d) via RD Gateway...\n", name, conn.Host, conn.Port)
	fmt.Printf("RDP file: %s\n\n", rdpFilePath)

	var rdpCmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		rdpCmd = exec.Command("mstsc.exe", rdpFilePath)
	case "darwin":
		rdpCmd = exec.Command("open", rdpFilePath)
	default:
		if _, err := exec.LookPath("xfreerdp"); err == nil {
			rdpCmd = exec.Command("xfreerdp", rdpFilePath)
		} else if _, err := exec.LookPath("rdesktop"); err == nil {
			rdpCmd = exec.Command("rdesktop", "-r", "rdpfile:"+rdpFilePath)
		} else {
			fmt.Println("RDP file saved to:", rdpFilePath)
			fmt.Println("No RDP client found. Install xfreerdp or rdesktop, or open the file manually.")
			return
		}
	}

	rdpCmd.Stdin = os.Stdin
	rdpCmd.Stdout = os.Stdout
	rdpCmd.Stderr = os.Stderr

	if err := rdpCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to launch RDP client: %v\n", err)
		fmt.Println("RDP file saved to:", rdpFilePath)
		os.Exit(1)
	}

	fmt.Println("RDP client launched. The connection file will be cleaned up when this process exits.")

	if err := rdpCmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
	}
}

func sanitizeFilename(name string) string {
	result := make([]byte, 0, len(name))
	for _, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}
