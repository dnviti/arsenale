package dbsessions

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStoreAndResolveOverrideCredentials(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	metadata := map[string]any{"username": "manual-user"}

	if err := storeOverridePasswordMetadata(metadata, "ManualPass123!", key); err != nil {
		t.Fatalf("storeOverridePasswordMetadata() error = %v", err)
	}

	raw, _ := json.Marshal(metadata)
	if strings.Contains(string(raw), "ManualPass123!") {
		t.Fatalf("metadata unexpectedly contains plaintext password: %s", string(raw))
	}

	username, password, err := resolveOverrideCredentials(metadata, key)
	if err != nil {
		t.Fatalf("resolveOverrideCredentials() error = %v", err)
	}
	if username != "manual-user" {
		t.Fatalf("username = %q, want %q", username, "manual-user")
	}
	if password != "ManualPass123!" {
		t.Fatalf("password = %q, want %q", password, "ManualPass123!")
	}
}

func TestShouldUseGoDatabaseSessionRuntimeAllowsOverrideCredentials(t *testing.T) {
	if !shouldUseGoDatabaseSessionRuntime("postgresql", true) {
		t.Fatal("shouldUseGoDatabaseSessionRuntime() = false, want true for PostgreSQL override credentials")
	}
}

func TestShouldUseGoDatabaseSessionRuntimeAllowsPostgresByDefault(t *testing.T) {
	t.Setenv("GO_QUERY_RUNNER_ENABLED", "")
	if !shouldUseGoDatabaseSessionRuntime("postgresql", false) {
		t.Fatal("shouldUseGoDatabaseSessionRuntime() = false, want true for PostgreSQL by default")
	}
}

func TestShouldUseGoDatabaseSessionRuntimeHonorsExplicitDisable(t *testing.T) {
	t.Setenv("GO_QUERY_RUNNER_ENABLED", "false")
	if shouldUseGoDatabaseSessionRuntime("postgresql", true) {
		t.Fatal("shouldUseGoDatabaseSessionRuntime() = true, want false when explicitly disabled")
	}
}

func TestWriteOwnedQueryErrorUnsupported(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeOwnedQueryError(recorder, ErrQueryRuntimeUnsupported)
	if recorder.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotImplemented)
	}
	if !strings.Contains(recorder.Body.String(), "unsupported") {
		t.Fatalf("body = %q, want unsupported error message", recorder.Body.String())
	}
}
