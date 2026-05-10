package cmd

import (
	"encoding/json"
	"testing"
)

func TestNormalizeProfileUpdatePayloadMapsNameToUsername(t *testing.T) {
	body, err := normalizeProfileUpdatePayload([]byte(`{"name":"admin"}`))
	if err != nil {
		t.Fatalf("normalizeProfileUpdatePayload() error = %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["username"] != "admin" {
		t.Fatalf("username = %q, want admin", got["username"])
	}
	if _, ok := got["name"]; ok {
		t.Fatal("name alias was not removed")
	}
}

func TestNormalizeDomainProfilePayloadMapsAliases(t *testing.T) {
	body, err := normalizeDomainProfilePayload([]byte(`{"domain":"CLI","username":"admin","password":"secret"}`))
	if err != nil {
		t.Fatalf("normalizeDomainProfilePayload() error = %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["domainName"] != "CLI" || got["domainUsername"] != "admin" || got["domainPassword"] != "secret" {
		t.Fatalf("normalized payload = %#v", got)
	}
}
