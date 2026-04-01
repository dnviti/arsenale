package passwordrotationapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseLoginPayload(t *testing.T) {
	t.Parallel()

	payload, err := parseLoginPayload(json.RawMessage(`{"type":"LOGIN","username":"alice","password":"old-pass","url":"https://example.test"}`))
	if err != nil {
		t.Fatalf("parseLoginPayload returned error: %v", err)
	}
	if payload.Username != "alice" {
		t.Fatalf("unexpected username: %q", payload.Username)
	}
	if payload.Password != "old-pass" {
		t.Fatalf("unexpected password: %q", payload.Password)
	}
	if payload.Data["url"] != "https://example.test" {
		t.Fatalf("expected payload to preserve url field, got %#v", payload.Data["url"])
	}
}

func TestParseLoginPayloadRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{
		`{"type":"SSH_KEY","username":"alice","password":"old-pass"}`,
		`{"type":"LOGIN","username":"","password":"old-pass"}`,
		`{"type":"LOGIN","username":"alice","password":""}`,
	} {
		if _, err := parseLoginPayload(json.RawMessage(raw)); err == nil {
			t.Fatalf("expected parseLoginPayload to fail for %s", raw)
		}
	}
}

func TestLoginPayloadWithPasswordPreservesFields(t *testing.T) {
	t.Parallel()

	payload, err := parseLoginPayload(json.RawMessage(`{"type":"LOGIN","username":"alice","password":"old-pass","notes":"keep-me"}`))
	if err != nil {
		t.Fatalf("parseLoginPayload returned error: %v", err)
	}

	updated, err := payload.withPassword("new-pass")
	if err != nil {
		t.Fatalf("withPassword returned error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(updated, &decoded); err != nil {
		t.Fatalf("unmarshal updated payload: %v", err)
	}
	if decoded["password"] != "new-pass" {
		t.Fatalf("expected password to be updated, got %#v", decoded["password"])
	}
	if decoded["notes"] != "keep-me" {
		t.Fatalf("expected notes to be preserved, got %#v", decoded["notes"])
	}
}

func TestGenerateStrongPassword(t *testing.T) {
	t.Parallel()

	password, err := generateStrongPassword(passwordLength)
	if err != nil {
		t.Fatalf("generateStrongPassword returned error: %v", err)
	}
	if len(password) != passwordLength {
		t.Fatalf("expected password length %d, got %d", passwordLength, len(password))
	}
	if !strings.ContainsAny(password, lowerCharset) {
		t.Fatalf("expected password to contain a lowercase character: %q", password)
	}
	if !strings.ContainsAny(password, upperCharset) {
		t.Fatalf("expected password to contain an uppercase character: %q", password)
	}
	if !strings.ContainsAny(password, digitCharset) {
		t.Fatalf("expected password to contain a digit: %q", password)
	}
	if !strings.ContainsAny(password, specialCharset) {
		t.Fatalf("expected password to contain a special character: %q", password)
	}
}
