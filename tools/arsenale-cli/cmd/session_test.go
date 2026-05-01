package cmd

import (
	"testing"
)

func TestSessionObserveCommandsAreRegistered(t *testing.T) {
	if sessionObserveCmd == nil {
		t.Fatal("sessionObserveCmd = nil")
	}
	if sessionObserveSSHCmd == nil || sessionObserveRDPCmd == nil || sessionObserveVNCCmd == nil {
		t.Fatal("observe subcommands must be registered")
	}
	if sessionPauseCmd == nil || sessionResumeCmd == nil {
		t.Fatal("pause/resume commands must be registered")
	}
}

func TestSessionConsoleCommandWiring(t *testing.T) {
	if sessionConsoleCmd == nil {
		t.Fatal("sessionConsoleCmd = nil")
	}
	if sessionConsoleCmd.Parent() != sessionCmd {
		t.Fatalf("sessionConsoleCmd parent = %v; want sessionCmd", sessionConsoleCmd.Parent())
	}
	if got := sessionConsoleCmd.Flags().Lookup("status"); got == nil {
		t.Fatal("sessionConsoleCmd missing --status flag")
	} else if got.DefValue != "[ACTIVE]" {
		t.Fatalf("sessionConsoleCmd --status default = %q; want [ACTIVE]", got.DefValue)
	}
	for _, name := range []string{"protocol", "gateway-id", "limit", "offset"} {
		if sessionConsoleCmd.Flags().Lookup(name) == nil {
			t.Fatalf("sessionConsoleCmd missing --%s flag", name)
		}
	}
}

func TestSessionConsoleColumnsExposeOperatorView(t *testing.T) {
	checks := map[int]struct {
		header string
		field  string
	}{
		0: {header: "ID", field: "id"},
		1: {header: "USER", field: "username"},
		2: {header: "CONNECTION", field: "connectionName"},
		5: {header: "GATEWAY", field: "gatewayName"},
		6: {header: "RECORDING", field: "recording.exists"},
		7: {header: "STARTED_AT", field: "startedAt"},
		8: {header: "ENDED_AT", field: "endedAt"},
	}

	for idx, want := range checks {
		if got := sessionConsoleColumns[idx].Header; got != want.header {
			t.Fatalf("sessionConsoleColumns[%d].Header = %q; want %q", idx, got, want.header)
		}
		if got := sessionConsoleColumns[idx].Field; got != want.field {
			t.Fatalf("sessionConsoleColumns[%d].Field = %q; want %q", idx, got, want.field)
		}
	}
}

func TestSessionConsoleQueryValuesJoinFilters(t *testing.T) {
	oldStatuses := sessionConsoleStatuses
	oldProtocol := sessionConsoleProtocol
	oldGatewayID := sessionConsoleGatewayID
	oldLimit := sessionConsoleLimit
	oldOffset := sessionConsoleOffset
	t.Cleanup(func() {
		sessionConsoleStatuses = oldStatuses
		sessionConsoleProtocol = oldProtocol
		sessionConsoleGatewayID = oldGatewayID
		sessionConsoleLimit = oldLimit
		sessionConsoleOffset = oldOffset
	})

	sessionConsoleStatuses = []string{"ACTIVE", "PAUSED"}
	sessionConsoleProtocol = "ssh"
	sessionConsoleGatewayID = "gw-1"
	sessionConsoleLimit = 25
	sessionConsoleOffset = 10

	if got := sessionConsoleQueryValues().Encode(); got != "gatewayId=gw-1&limit=25&offset=10&protocol=ssh&status=ACTIVE%2CPAUSED" {
		t.Fatalf("sessionConsoleQueryValues().Encode() = %q", got)
	}
}

func TestSessionConsoleQueryValuesDefaultsToActiveOnly(t *testing.T) {
	oldStatuses := sessionConsoleStatuses
	oldProtocol := sessionConsoleProtocol
	oldGatewayID := sessionConsoleGatewayID
	oldLimit := sessionConsoleLimit
	oldOffset := sessionConsoleOffset
	t.Cleanup(func() {
		sessionConsoleStatuses = oldStatuses
		sessionConsoleProtocol = oldProtocol
		sessionConsoleGatewayID = oldGatewayID
		sessionConsoleLimit = oldLimit
		sessionConsoleOffset = oldOffset
	})

	sessionConsoleStatuses = nil
	sessionConsoleProtocol = ""
	sessionConsoleGatewayID = ""
	sessionConsoleLimit = 50
	sessionConsoleOffset = 0

	if got := sessionConsoleQueryValues().Encode(); got != "limit=50&offset=0&status=ACTIVE" {
		t.Fatalf("sessionConsoleQueryValues().Encode() = %q", got)
	}
}

func TestRunSessionObserveColumnsExposeObserverGrantShape(t *testing.T) {
	columns := []Column{
		{Header: "SESSION_ID", Field: "sessionId"},
		{Header: "PROTOCOL", Field: "protocol"},
		{Header: "MODE", Field: "mode"},
		{Header: "READ_ONLY", Field: "readOnly"},
		{Header: "EXPIRES_AT", Field: "expiresAt"},
		{Header: "WS_PATH", Field: "webSocketPath"},
		{Header: "WS_URL", Field: "webSocketUrl"},
	}

	if got := columns[2].Field; got != "mode" {
		t.Fatalf("columns[2].Field = %q; want mode", got)
	}
	if got := columns[3].Field; got != "readOnly" {
		t.Fatalf("columns[3].Field = %q; want readOnly", got)
	}
	if got := columns[5].Field; got != "webSocketPath" {
		t.Fatalf("columns[5].Field = %q; want webSocketPath", got)
	}
	if got := columns[6].Field; got != "webSocketUrl" {
		t.Fatalf("columns[6].Field = %q; want webSocketUrl", got)
	}
}
