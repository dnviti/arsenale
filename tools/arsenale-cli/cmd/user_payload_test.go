package cmd

import (
	"encoding/json"
	"testing"
)

func TestBuildUserCreateBodyIncludesPasswordAndDefaultRole(t *testing.T) {
	body, err := buildUserCreateBody(" user@example.com ", "", "ArsenaleTemp91Qx2", "")
	if err != nil {
		t.Fatalf("buildUserCreateBody() error = %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["email"] != "user@example.com" {
		t.Fatalf("email = %v, want user@example.com", got["email"])
	}
	if got["password"] != "ArsenaleTemp91Qx2" {
		t.Fatalf("password = %v, want ArsenaleTemp91Qx2", got["password"])
	}
	if got["role"] != "MEMBER" {
		t.Fatalf("role = %v, want MEMBER", got["role"])
	}
}

func TestBuildUserCreateBodyRequiresPassword(t *testing.T) {
	if _, err := buildUserCreateBody("user@example.com", "MEMBER", "", ""); err == nil {
		t.Fatal("buildUserCreateBody() error = nil, want error")
	}
}

func TestBuildUserInviteBodyIncludesRoleAndExpiry(t *testing.T) {
	got, err := buildUserInviteBody(" user@example.com ", "operator", "2027-01-02T03:04:05Z")
	if err != nil {
		t.Fatalf("buildUserInviteBody() error = %v", err)
	}

	if got["email"] != "user@example.com" {
		t.Fatalf("email = %v, want user@example.com", got["email"])
	}
	if got["role"] != "OPERATOR" {
		t.Fatalf("role = %v, want OPERATOR", got["role"])
	}
	if got["expiresAt"] != "2027-01-02T03:04:05Z" {
		t.Fatalf("expiresAt = %v, want 2027-01-02T03:04:05Z", got["expiresAt"])
	}
}

func TestNormalizeUserPermissionsPayloadWrapsOverrides(t *testing.T) {
	body, err := normalizeUserPermissionsPayload([]byte(`{"canManageConnections":true}`))
	if err != nil {
		t.Fatalf("normalizeUserPermissionsPayload() error = %v", err)
	}

	var got map[string]map[string]bool
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !got["overrides"]["canManageConnections"] {
		t.Fatalf("overrides.canManageConnections = false, want true")
	}
}

func TestNormalizeUserPermissionsPayloadKeepsWrappedOverrides(t *testing.T) {
	body, err := normalizeUserPermissionsPayload([]byte(`{"overrides":{"canManageConnections":true}}`))
	if err != nil {
		t.Fatalf("normalizeUserPermissionsPayload() error = %v", err)
	}

	var got map[string]map[string]bool
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !got["overrides"]["canManageConnections"] {
		t.Fatalf("overrides.canManageConnections = false, want true")
	}
}
