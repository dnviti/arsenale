package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConnectCommandsExposeDesktopLaunches(t *testing.T) {
	if connectRDPCmd == nil || connectVNCCmd == nil {
		t.Fatal("connect rdp/vnc commands must be registered")
	}
	if connectRDPCmd.Flags().Lookup("no-open") == nil {
		t.Fatal("connect rdp missing --no-open")
	}
	if connectVNCCmd.Flags().Lookup("no-open") == nil {
		t.Fatal("connect vnc missing --no-open")
	}
	for _, command := range rootCmd.Commands() {
		if command.Name() == "rdgw" {
			t.Fatal("rdgw command should not be registered")
		}
	}
}

func TestBuildOpenSSHArgsPreservesRemoteCommand(t *testing.T) {
	got := buildOpenSSHArgs("/tmp/ssh_config", []string{"true", "--flag"})
	want := []string{"-F", "/tmp/ssh_config", "arsenale-target", "true", "--flag"}
	if len(got) != len(want) {
		t.Fatalf("args len = %d; want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestBuildOpenSSHConfigConnectsDirectlyToProxy(t *testing.T) {
	got := buildOpenSSHConfig(openSSHConfigOptions{
		ProxyHost: "arsenale.example.test",
		ProxyPort: 2222,
		Token:     "grant.secret",
	})

	for _, want := range []string{
		"HostName arsenale.example.test",
		"Port 2222",
		"User grant.secret",
		"PreferredAuthentications none",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("config missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "ProxyCommand") || strings.Contains(got, " nc ") {
		t.Fatalf("config must not depend on netcat proxy command:\n%s", got)
	}
}

func TestFindConnectionByNameRejectsDuplicateNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/cli/connections" {
			t.Fatalf("path = %q; want /api/cli/connections", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"conn-1","name":"prod","type":"SSH"},
			{"id":"conn-2","name":"prod","type":"RDP"}
		]`))
	}))
	defer server.Close()

	_, err := findConnectionByName("prod", &CLIConfig{ServerURL: server.URL})
	if err == nil || err.Error() != "connection name 'prod' is ambiguous (2 matches). Use the connection ID instead" {
		t.Fatalf("error = %v", err)
	}

	conn, err := findConnectionByName("conn-2", &CLIConfig{ServerURL: server.URL})
	if err != nil {
		t.Fatalf("find by id failed: %v", err)
	}
	if conn.ID != "conn-2" {
		t.Fatalf("id = %q; want conn-2", conn.ID)
	}
}
