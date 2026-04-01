package secretsmeta

import (
	"encoding/json"
	"testing"
)

func TestParseCreateSecretInput(t *testing.T) {
	t.Parallel()

	input, err := parseCreateSecretInput([]byte(`{
		"name":"Admin Login",
		"type":"LOGIN",
		"scope":"PERSONAL",
		"data":{"type":"LOGIN","username":"root","password":"hunter2","domain":"corp","unknown":"drop-me"},
		"tags":["prod"]
	}`))
	if err != nil {
		t.Fatalf("parseCreateSecretInput returned error: %v", err)
	}

	if input.Name != "Admin Login" {
		t.Fatalf("expected name to be preserved, got %q", input.Name)
	}

	var payload map[string]any
	if err := json.Unmarshal(input.Data, &payload); err != nil {
		t.Fatalf("unmarshal sanitized payload: %v", err)
	}
	if payload["domain"] != "corp" {
		t.Fatalf("expected domain to survive sanitization, got %#v", payload["domain"])
	}
	if _, ok := payload["unknown"]; ok {
		t.Fatalf("expected unknown payload field to be stripped")
	}
}

func TestParseCreateSecretInputRequiresMatchingType(t *testing.T) {
	t.Parallel()

	_, err := parseCreateSecretInput([]byte(`{
		"name":"Mismatch",
		"type":"LOGIN",
		"scope":"PERSONAL",
		"data":{"type":"API_KEY","apiKey":"secret"}
	}`))
	if err == nil || err.Error() != "Secret data type does not match declared type" {
		t.Fatalf("expected type mismatch error, got %v", err)
	}
}

func TestParseUpdateSecretInputSupportsNullables(t *testing.T) {
	t.Parallel()

	input, err := parseUpdateSecretInput([]byte(`{
		"description": null,
		"folderId": null,
		"expiresAt": null,
		"isFavorite": true,
		"data": {"type":"SECURE_NOTE","content":"rotated"}
	}`))
	if err != nil {
		t.Fatalf("parseUpdateSecretInput returned error: %v", err)
	}

	if !input.DescriptionSet || input.Description != nil {
		t.Fatalf("expected description to be explicitly cleared")
	}
	if !input.FolderIDSet || input.FolderID != nil {
		t.Fatalf("expected folderId to be explicitly cleared")
	}
	if !input.ExpiresAtSet || input.ExpiresAt != nil {
		t.Fatalf("expected expiresAt to be explicitly cleared")
	}
	if input.IsFavorite == nil || !*input.IsFavorite {
		t.Fatalf("expected isFavorite to be true")
	}
	if !input.DataSet {
		t.Fatalf("expected data update to be present")
	}
}

func TestParseUpdateSecretInputRejectsEmptyPatch(t *testing.T) {
	t.Parallel()

	_, err := parseUpdateSecretInput([]byte(`{}`))
	if err == nil || err.Error() != "No fields to update" {
		t.Fatalf("expected empty patch error, got %v", err)
	}
}

func TestParseCreateExternalShareInput(t *testing.T) {
	t.Parallel()

	input, err := parseCreateExternalShareInput([]byte(`{
		"expiresInMinutes": 15,
		"maxAccessCount": 3,
		"pin": "1234"
	}`))
	if err != nil {
		t.Fatalf("parseCreateExternalShareInput returned error: %v", err)
	}

	if input.ExpiresInMinutes != 15 {
		t.Fatalf("expected expiresInMinutes to be preserved, got %d", input.ExpiresInMinutes)
	}
	if input.MaxAccessCount == nil || *input.MaxAccessCount != 3 {
		t.Fatalf("expected maxAccessCount to be 3, got %#v", input.MaxAccessCount)
	}
	if input.Pin == nil || *input.Pin != "1234" {
		t.Fatalf("expected pin to be preserved, got %#v", input.Pin)
	}
}

func TestParseCreateExternalShareInputRejectsInvalidPin(t *testing.T) {
	t.Parallel()

	_, err := parseCreateExternalShareInput([]byte(`{
		"expiresInMinutes": 15,
		"pin": "12ab"
	}`))
	if err == nil || err.Error() != "PIN must be 4-8 digits" {
		t.Fatalf("expected pin validation error, got %v", err)
	}
}

func TestParseShareSecretInput(t *testing.T) {
	t.Parallel()

	input, err := parseShareSecretInput([]byte(`{
		"email":"user@example.com",
		"permission":"FULL_ACCESS"
	}`))
	if err != nil {
		t.Fatalf("parseShareSecretInput returned error: %v", err)
	}

	if input.Email == nil || *input.Email != "user@example.com" {
		t.Fatalf("expected email to be preserved, got %#v", input.Email)
	}
	if input.Permission != "FULL_ACCESS" {
		t.Fatalf("expected permission to be FULL_ACCESS, got %q", input.Permission)
	}
}

func TestParseShareSecretInputRequiresTarget(t *testing.T) {
	t.Parallel()

	_, err := parseShareSecretInput([]byte(`{"permission":"READ_ONLY"}`))
	if err == nil || err.Error() != "Either email or userId is required" {
		t.Fatalf("expected missing target error, got %v", err)
	}
}

func TestParseUpdateSecretSharePermissionInput(t *testing.T) {
	t.Parallel()

	input, err := parseUpdateSecretSharePermissionInput([]byte(`{"permission":"READ_ONLY"}`))
	if err != nil {
		t.Fatalf("parseUpdateSecretSharePermissionInput returned error: %v", err)
	}
	if input.Permission != "READ_ONLY" {
		t.Fatalf("expected READ_ONLY permission, got %q", input.Permission)
	}
}
