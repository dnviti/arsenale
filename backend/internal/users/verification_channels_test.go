package users

import "testing"

func TestTimingSafeHexEqual(t *testing.T) {
	valid := hashOTP("123456")
	if !timingSafeHexEqual(valid, valid) {
		t.Fatalf("expected equal hashes to match")
	}
	if timingSafeHexEqual(valid, hashOTP("654321")) {
		t.Fatalf("expected different hashes to fail")
	}
}

func TestMaskEmail(t *testing.T) {
	if got := maskEmail("alice@example.com"); got != "al***@example.com" {
		t.Fatalf("unexpected masked email: %s", got)
	}
}

func TestMaskPhone(t *testing.T) {
	if got := maskPhone("+15551234567"); got != "+*******4567" {
		t.Fatalf("unexpected masked phone: %s", got)
	}
}
