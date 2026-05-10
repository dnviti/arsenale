package cmd

import "testing"

func TestNormalizeSecretSharePermission(t *testing.T) {
	tests := map[string]string{
		"read":        "READ_ONLY",
		"READ_ONLY":   "READ_ONLY",
		"write":       "FULL_ACCESS",
		"full":        "FULL_ACCESS",
		"FULL_ACCESS": "FULL_ACCESS",
	}

	for input, want := range tests {
		if got := normalizeSecretSharePermission(input); got != want {
			t.Fatalf("normalizeSecretSharePermission(%q) = %q, want %q", input, got, want)
		}
	}
}
