package cmd

import "testing"

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
