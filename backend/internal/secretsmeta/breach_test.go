package secretsmeta

import (
	"encoding/json"
	"testing"
)

func TestExtractPasswordFromPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name:    "login password",
			payload: `{"type":"LOGIN","password":"secret"}`,
			want:    "secret",
		},
		{
			name:    "ssh key passphrase",
			payload: `{"type":"SSH_KEY","passphrase":"phrase"}`,
			want:    "phrase",
		},
		{
			name:    "certificate passphrase",
			payload: `{"type":"CERTIFICATE","passphrase":"cert-pass"}`,
			want:    "cert-pass",
		},
		{
			name:    "api key has no password",
			payload: `{"type":"API_KEY","apiKey":"abc"}`,
			want:    "",
		},
		{
			name:    "invalid payload",
			payload: `not-json`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractPasswordFromPayload(json.RawMessage(tt.payload))
			if got != tt.want {
				t.Fatalf("extractPasswordFromPayload() = %q, want %q", got, tt.want)
			}
		})
	}
}
