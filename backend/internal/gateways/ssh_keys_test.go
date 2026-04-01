package gateways

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateSSHKeyMaterial(t *testing.T) {
	privatePEM, publicKey, fingerprint, err := generateSSHKeyMaterial()
	if err != nil {
		t.Fatalf("generateSSHKeyMaterial returned error: %v", err)
	}
	if !strings.Contains(privatePEM, "BEGIN PRIVATE KEY") {
		t.Fatalf("expected PKCS8 private key PEM, got %q", privatePEM)
	}
	if !strings.HasPrefix(publicKey, "ssh-ed25519 ") {
		t.Fatalf("expected ed25519 authorized key, got %q", publicKey)
	}
	if !strings.HasPrefix(fingerprint, "SHA256:") {
		t.Fatalf("expected SHA256 fingerprint, got %q", fingerprint)
	}
}

func TestComputeSSHKeyRotationStatus(t *testing.T) {
	now := time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	lastAutoRotatedAt := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

	status := computeSSHKeyRotationStatus(true, 90, &expiresAt, &lastAutoRotatedAt, now, 7)
	if !status.KeyExists {
		t.Fatal("expected keyExists=true")
	}
	if status.NextRotationDate == nil {
		t.Fatal("expected nextRotationDate to be set")
	}
	expectedNextRotation := time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)
	if !status.NextRotationDate.Equal(expectedNextRotation) {
		t.Fatalf("expected nextRotationDate %s, got %s", expectedNextRotation, status.NextRotationDate)
	}
	if status.DaysUntilRotation == nil || *status.DaysUntilRotation != 3 {
		t.Fatalf("expected daysUntilRotation=3, got %#v", status.DaysUntilRotation)
	}
}

func TestComputeSSHKeyRotationStatusNoSchedule(t *testing.T) {
	now := time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC)

	status := computeSSHKeyRotationStatus(false, 90, nil, nil, now, 7)
	if !status.KeyExists {
		t.Fatal("expected keyExists=true")
	}
	if status.NextRotationDate != nil {
		t.Fatalf("expected nextRotationDate to be nil, got %s", status.NextRotationDate)
	}
	if status.DaysUntilRotation != nil {
		t.Fatalf("expected daysUntilRotation to be nil, got %d", *status.DaysUntilRotation)
	}
}

func TestOptionalTimeUnmarshal(t *testing.T) {
	var value optionalTime
	if err := value.UnmarshalJSON([]byte(`"2026-04-01T10:00:00Z"`)); err != nil {
		t.Fatalf("UnmarshalJSON returned error: %v", err)
	}
	if !value.Present {
		t.Fatal("expected Present=true")
	}
	if value.Value == nil || value.Value.Format(time.RFC3339) != "2026-04-01T10:00:00Z" {
		t.Fatalf("unexpected parsed time: %#v", value.Value)
	}

	var cleared optionalTime
	if err := cleared.UnmarshalJSON([]byte(`null`)); err != nil {
		t.Fatalf("UnmarshalJSON(null) returned error: %v", err)
	}
	if !cleared.Present {
		t.Fatal("expected Present=true for null")
	}
	if cleared.Value != nil {
		t.Fatalf("expected nil value for null, got %#v", cleared.Value)
	}
}
