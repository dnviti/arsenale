package cmd

import "testing"

func TestBuildSecretRotationStatusPayload(t *testing.T) {
	payload := buildSecretRotationStatusPayload("secret-1")
	if got := payload["secretId"]; got != "secret-1" {
		t.Fatalf("secretId = %q, want secret-1", got)
	}
}

func TestBuildSecretRotationHistoryPayload(t *testing.T) {
	payload := buildSecretRotationHistoryPayload("secret-1", 42)
	if got := payload["secretId"]; got != "secret-1" {
		t.Fatalf("secretId = %v, want secret-1", got)
	}
	if got := payload["limit"]; got != 42 {
		t.Fatalf("limit = %v, want 42", got)
	}
}
