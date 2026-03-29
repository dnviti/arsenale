package terminalbroker

import (
	"testing"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestIssueAndValidateGrant(t *testing.T) {
	grant := contracts.TerminalSessionGrant{
		SessionID: "session-1",
		UserID:    "user-1",
		ExpiresAt: time.Now().UTC().Add(2 * time.Minute),
		Target: contracts.TerminalEndpoint{
			Host:     "terminal-target",
			Port:     2224,
			Username: "acceptance",
			Password: "acceptance",
		},
	}

	token, err := IssueGrant("secret", grant)
	if err != nil {
		t.Fatalf("IssueGrant() error = %v", err)
	}
	if token == "" {
		t.Fatal("IssueGrant() returned empty token")
	}

	validated, err := ValidateGrant("secret", token, time.Now().UTC())
	if err != nil {
		t.Fatalf("ValidateGrant() error = %v", err)
	}
	if validated.Target.Host != grant.Target.Host {
		t.Fatalf("validated target host = %q, want %q", validated.Target.Host, grant.Target.Host)
	}
	if validated.Target.Password != grant.Target.Password {
		t.Fatalf("validated target password mismatch")
	}
	if validated.Terminal.Term != "xterm-256color" || validated.Terminal.Cols != 80 || validated.Terminal.Rows != 24 {
		t.Fatalf("validated default terminal = %+v, want xterm-256color/80x24", validated.Terminal)
	}
}

func TestValidateGrantRejectsExpired(t *testing.T) {
	token, err := IssueGrant("secret", contracts.TerminalSessionGrant{
		ExpiresAt: time.Now().UTC().Add(-1 * time.Minute),
		Target: contracts.TerminalEndpoint{
			Host:     "terminal-target",
			Username: "acceptance",
			Password: "acceptance",
		},
	})
	if err != nil {
		t.Fatalf("IssueGrant() error = %v", err)
	}

	if _, err := ValidateGrant("secret", token, time.Now().UTC()); err == nil {
		t.Fatal("ValidateGrant() error = nil, want expired grant error")
	}
}
