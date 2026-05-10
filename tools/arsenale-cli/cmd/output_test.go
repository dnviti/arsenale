package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintCreatedQuietSupportsNestedIDField(t *testing.T) {
	var out bytes.Buffer
	p := &Printer{Quiet: true, Writer: &out}

	if err := p.PrintCreated([]byte(`{"user":{"id":"user-1"}}`), "user.id"); err != nil {
		t.Fatalf("PrintCreated() error = %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "user-1" {
		t.Fatalf("output = %q, want user-1", got)
	}
}

func TestPrintCreatedQuietSupportsArrayResponse(t *testing.T) {
	var out bytes.Buffer
	p := &Printer{Quiet: true, Writer: &out}

	if err := p.PrintCreated([]byte(`[{"name":"upload.txt"}]`), "name"); err != nil {
		t.Fatalf("PrintCreated() error = %v", err)
	}

	if got := strings.TrimSpace(out.String()); got != "upload.txt" {
		t.Fatalf("output = %q, want upload.txt", got)
	}
}
