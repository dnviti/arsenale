package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a target via Arsenale proxy",
}

var connectSSHCmd = &cobra.Command{
	Use:   "ssh <connection-name-or-id> [-- remote-command...]",
	Short: "Connect to an SSH target via Arsenale proxy",
	Args:  cobra.MinimumNArgs(1),
	Run:   runConnectSSH,
}

var connectRDPCmd = &cobra.Command{
	Use:   "rdp <connection-name-or-id>",
	Short: "Open an RDP target in the Arsenale desktop viewer",
	Args:  cobra.ExactArgs(1),
	Run:   runConnectRDP,
}

var connectVNCCmd = &cobra.Command{
	Use:   "vnc <connection-name-or-id>",
	Short: "Open a VNC target in the Arsenale desktop viewer",
	Args:  cobra.ExactArgs(1),
	Run:   runConnectVNC,
}

var connectNoOpen bool

type openSSHConfigOptions struct {
	ProxyHost string
	ProxyPort int
	Token     string
}

var connectDesktopLaunchColumns = []Column{
	{Header: "PROTOCOL", Field: "protocol"},
	{Header: "CONNECTION_ID", Field: "connectionId"},
	{Header: "LAUNCH_URL", Field: "launchUrl"},
	{Header: "EXPIRES_AT", Field: "expiresAt"},
	{Header: "EXPIRES_IN", Field: "expiresIn"},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.AddCommand(connectSSHCmd)
	connectCmd.AddCommand(connectRDPCmd)
	connectCmd.AddCommand(connectVNCCmd)
	connectRDPCmd.Flags().BoolVar(&connectNoOpen, "no-open", false, "Print the viewer launch URL without opening a browser")
	connectVNCCmd.Flags().BoolVar(&connectNoOpen, "no-open", false, "Print the viewer launch URL without opening a browser")
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

	if outputFormat == "json" || outputFormat == "yaml" {
		if err := printer().PrintSingle(respBody, nil); err != nil {
			fatal("%v", err)
		}
		return
	}

	tmpDir, err := os.MkdirTemp("", "arsenale-ssh-*")
	if err != nil {
		fatal("%v", err)
	}
	defer os.RemoveAll(tmpDir)

	sshConfigPath := filepath.Join(tmpDir, "ssh_config")
	proxyHost := tokenResp.ConnectionInstructions.Host
	proxyPort := tokenResp.ConnectionInstructions.Port

	sshConfig := buildOpenSSHConfig(openSSHConfigOptions{
		ProxyHost: proxyHost,
		ProxyPort: proxyPort,
		Token:     tokenResp.Token,
	})

	if err := os.WriteFile(sshConfigPath, []byte(sshConfig), 0600); err != nil {
		fatal("failed to write SSH config: %v", err)
	}

	fmt.Printf("Connecting to %s (%s:%d) via Arsenale SSH proxy...\n", name, conn.Host, conn.Port)
	fmt.Printf("Proxy: %s:%d (token expires in %ds)\n\n", proxyHost, proxyPort, tokenResp.ExpiresIn)

	sshArgs := buildOpenSSHArgs(sshConfigPath, args[1:])
	sshCmd := exec.Command("ssh", sshArgs...)
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

func buildOpenSSHArgs(sshConfigPath string, remoteArgs []string) []string {
	sshArgs := []string{"-F", sshConfigPath, "arsenale-target"}
	return append(sshArgs, remoteArgs...)
}

func buildOpenSSHConfig(opts openSSHConfigOptions) string {
	return fmt.Sprintf(`Host arsenale-target
    HostName %s
    Port %d
    User %s
    PreferredAuthentications none
    PubkeyAuthentication no
    PasswordAuthentication no
    KbdInteractiveAuthentication no
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    LogLevel ERROR
`, opts.ProxyHost, opts.ProxyPort, opts.Token)
}

func runConnectRDP(cmd *cobra.Command, args []string) {
	runConnectDesktop("RDP", args[0])
}

func runConnectVNC(cmd *cobra.Command, args []string) {
	runConnectDesktop("VNC", args[0])
}

func runConnectDesktop(protocol, connectionRef string) {
	cfg := getCfg()

	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	conn, err := findConnectionByName(connectionRef, cfg)
	if err != nil {
		fatal("%v", err)
	}
	if conn.Type != protocol {
		fatal("connection %q is type %s, not %s", conn.Name, conn.Type, protocol)
	}

	respBody, status, err := apiPost("/api/cli/connect/desktop/launch", map[string]string{
		"protocol":     protocol,
		"connectionId": conn.ID,
	}, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, respBody)

	var launch struct {
		LaunchURL string `json:"launchUrl"`
	}
	if err := json.Unmarshal(respBody, &launch); err != nil {
		fatal("failed to parse launch response: %v", err)
	}
	if launch.LaunchURL == "" {
		fatal("server response did not include launchUrl")
	}

	if connectNoOpen || outputFormat != "table" {
		if err := printer().PrintSingle(respBody, connectDesktopLaunchColumns); err != nil {
			fatal("%v", err)
		}
		return
	}

	fmt.Printf("Opening %s viewer for %s (%s:%d)...\n", protocol, conn.Name, conn.Host, conn.Port)
	fmt.Printf("Launch URL: %s\n", launch.LaunchURL)
	if err := openBrowser(launch.LaunchURL); err != nil {
		fatal("failed to open browser: %v", err)
	}
}
