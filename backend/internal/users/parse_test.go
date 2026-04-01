package users

import (
	"encoding/json"
	"testing"
)

func TestParseIdentityConfirmationAllowsCode(t *testing.T) {
	payload := map[string]json.RawMessage{
		"verificationId": json.RawMessage(`"11111111-1111-1111-1111-111111111111"`),
		"code":           json.RawMessage(`"123456"`),
	}

	result, err := parseIdentityConfirmation(payload)
	if err != nil {
		t.Fatalf("parseIdentityConfirmation returned error: %v", err)
	}
	if result.VerificationID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected verification id: %s", result.VerificationID)
	}
	if result.Code != "123456" {
		t.Fatalf("unexpected code: %s", result.Code)
	}
	if len(result.Credential) != 0 {
		t.Fatalf("expected code payload to stay native")
	}
}

func TestParseIdentityConfirmationCapturesCredential(t *testing.T) {
	payload := map[string]json.RawMessage{
		"verificationId": json.RawMessage(`"11111111-1111-1111-1111-111111111111"`),
		"credential":     json.RawMessage(`{"id":"cred-1"}`),
	}

	result, err := parseIdentityConfirmation(payload)
	if err != nil {
		t.Fatalf("parseIdentityConfirmation returned error: %v", err)
	}
	if string(result.Credential) != `{"id":"cred-1"}` {
		t.Fatalf("unexpected credential payload: %s", string(result.Credential))
	}
}

func TestParseEmailChangeConfirmationAcceptsDualOTP(t *testing.T) {
	payload := map[string]json.RawMessage{
		"codeOld": json.RawMessage(`"123456"`),
		"codeNew": json.RawMessage(`"654321"`),
	}

	result, err := parseEmailChangeConfirmation(payload)
	if err != nil {
		t.Fatalf("parseEmailChangeConfirmation returned error: %v", err)
	}
	if !result.UsesOTP {
		t.Fatalf("expected dual-otp flow")
	}
	if result.CodeOld != "123456" || result.CodeNew != "654321" {
		t.Fatalf("unexpected otp payload: %#v", result)
	}
}

func TestParseEmailChangeConfirmationRejectsPartialOTP(t *testing.T) {
	payload := map[string]json.RawMessage{
		"codeOld": json.RawMessage(`"123456"`),
	}

	if _, err := parseEmailChangeConfirmation(payload); err == nil {
		t.Fatalf("expected parseEmailChangeConfirmation to reject partial otp payload")
	}
}
