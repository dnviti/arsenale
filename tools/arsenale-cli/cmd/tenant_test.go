package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestTenantListTableUsesTenantID(t *testing.T) {
	var out bytes.Buffer
	p := &Printer{Writer: &out}

	body := []byte(`[{
		"tenantId":"tenant-1",
		"name":"Development Environment",
		"role":"OWNER",
		"status":"ACCEPTED",
		"isActive":true,
		"pending":false
	}]`)

	if err := p.Print(body, tenantListColumns); err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "tenant-1") {
		t.Fatalf("output = %q, want tenant id", got)
	}
	if strings.Contains(got, "MFA_REQ") {
		t.Fatalf("output = %q, did not expect stale MFA_REQ column", got)
	}
}

func TestTenantDetailTableUsesTenantFields(t *testing.T) {
	var out bytes.Buffer
	p := &Printer{Writer: &out}

	body := []byte(`{
		"id":"tenant-1",
		"name":"Development Environment",
		"slug":"development-environment",
		"mfaRequired":false,
		"userCount":29,
		"teamCount":1
	}`)

	if err := p.PrintSingle(body, tenantColumns); err != nil {
		t.Fatalf("PrintSingle() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{"ID:", "tenant-1", "SLUG:", "development-environment", "USERS:", "29"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output = %q, want %q", got, want)
		}
	}
	if strings.Contains(got, "ROLE:") {
		t.Fatalf("output = %q, did not expect stale ROLE column", got)
	}
}

func TestTenantCreateDisplayBodyUnwrapsTenant(t *testing.T) {
	body := []byte(`{"tenant":{"id":"tenant-1","name":"Created Tenant"},"accessToken":"token"}`)

	got := string(tenantCreateDisplayBody(body))

	if !strings.Contains(got, `"id":"tenant-1"`) {
		t.Fatalf("tenantCreateDisplayBody() = %q, want nested tenant body", got)
	}
}

func TestApplyTenantAuthResponseStoresReturnedAccessTokenAndTenant(t *testing.T) {
	cfg := &CLIConfig{AccessToken: "old", TenantID: "old-tenant"}
	body := []byte(`{"accessToken":"new-token","tenant":{"id":"new-tenant"}}`)

	tenantID := applyTenantAuthResponse(cfg, body, "fallback-tenant")

	if tenantID != "new-tenant" {
		t.Fatalf("tenantID = %q, want new-tenant", tenantID)
	}
	if cfg.AccessToken != "new-token" {
		t.Fatalf("AccessToken = %q, want new-token", cfg.AccessToken)
	}
	if cfg.TokenExpiry != persistentCLITokenExpiry {
		t.Fatalf("TokenExpiry = %q, want %q", cfg.TokenExpiry, persistentCLITokenExpiry)
	}
	if cfg.TenantID != "new-tenant" {
		t.Fatalf("TenantID = %q, want new-tenant", cfg.TenantID)
	}
}

func TestApplyTenantAuthResponseUsesFallbackTenantForSwitchResponse(t *testing.T) {
	cfg := &CLIConfig{}
	body := []byte(`{"accessToken":"new-token"}`)

	tenantID := applyTenantAuthResponse(cfg, body, "fallback-tenant")

	if tenantID != "fallback-tenant" {
		t.Fatalf("tenantID = %q, want fallback-tenant", tenantID)
	}
	if cfg.TenantID != "fallback-tenant" {
		t.Fatalf("TenantID = %q, want fallback-tenant", cfg.TenantID)
	}
}
