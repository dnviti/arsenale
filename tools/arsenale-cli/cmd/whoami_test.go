package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestWhoamiTableUsesProfileFields(t *testing.T) {
	var out bytes.Buffer
	p := &Printer{Writer: &out}

	body := []byte(`{
		"id":"user-1",
		"email":"admin@example.com",
		"username":"admin",
		"vaultSetupComplete":true,
		"hasPassword":true
	}`)

	if err := p.PrintSingle(body, whoamiColumns); err != nil {
		t.Fatalf("PrintSingle() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"ID:", "user-1", "USERNAME:", "admin", "VAULT_SETUP:", "true"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, want %q", got, want)
		}
	}
	for _, stale := range []string{"ROLE:", "TENANT:"} {
		if strings.Contains(got, stale) {
			t.Fatalf("output = %q, did not expect stale %s column", got, stale)
		}
	}
}
