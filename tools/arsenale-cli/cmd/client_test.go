package cmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestAPIClientRetriesOnceAfterRefresh(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.Header.Get("Authorization") {
		case "Bearer old-token":
			http.Error(w, `{"error":"expired"}`, http.StatusUnauthorized)
		case "Bearer new-token":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
		}
	}))
	defer server.Close()

	cfg := &CLIConfig{
		ServerURL:    server.URL,
		AccessToken:  "old-token",
		RefreshToken: "refresh-token",
	}
	client := APIClient{
		Config: cfg,
		Refresh: func(cfg *CLIConfig) error {
			cfg.AccessToken = "new-token"
			return nil
		},
	}

	body, status, err := client.Request(http.MethodGet, "/profile", nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", status, http.StatusOK, string(body))
	}
	if string(body) != `{"ok":true}` {
		t.Fatalf("body = %s", string(body))
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
}

func TestRefreshAccessTokenKeepsCLITokenExpiryPersistent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth/refresh" {
			t.Fatalf("path = %q, want /api/auth/refresh", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accessToken":"new-token","refreshToken":"new-refresh"}`))
	}))
	defer server.Close()

	cfg := &CLIConfig{
		ServerURL:    server.URL,
		AccessToken:  "old-token",
		RefreshToken: "old-refresh",
		TokenExpiry:  "2026-05-10T00:00:00Z",
	}

	if err := refreshAccessToken(cfg); err != nil {
		t.Fatalf("refreshAccessToken() error = %v", err)
	}

	if cfg.AccessToken != "new-token" {
		t.Fatalf("AccessToken = %q, want new-token", cfg.AccessToken)
	}
	if cfg.RefreshToken != "new-refresh" {
		t.Fatalf("RefreshToken = %q, want new-refresh", cfg.RefreshToken)
	}
	if cfg.TokenExpiry != persistentCLITokenExpiry {
		t.Fatalf("TokenExpiry = %q, want %q", cfg.TokenExpiry, persistentCLITokenExpiry)
	}
}

func TestAPIDownloadRetriesAfterRefresh(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var downloads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/refresh":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accessToken":"new-token","refreshToken":"new-refresh"}`))
		case "/artifact":
			downloads++
			if r.Header.Get("Authorization") == "Bearer old-token" {
				http.Error(w, `{"error":"expired"}`, http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Authorization") != "Bearer new-token" {
				t.Fatalf("Authorization = %q, want Bearer new-token", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte("artifact-body"))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	cfg := &CLIConfig{ServerURL: server.URL, AccessToken: "old-token", RefreshToken: "refresh-token"}
	dest := filepath.Join(t.TempDir(), "artifact.txt")

	status, err := apiDownload("/artifact", dest, cfg)
	if err != nil {
		t.Fatalf("apiDownload() error = %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if downloads != 2 {
		t.Fatalf("downloads = %d, want 2", downloads)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "artifact-body" {
		t.Fatalf("downloaded body = %q, want artifact-body", string(data))
	}
}

func TestAPIUploadRetriesAfterRefresh(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var uploads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/refresh":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accessToken":"new-token","refreshToken":"new-refresh"}`))
		case "/upload":
			uploads++
			if r.Header.Get("Authorization") == "Bearer old-token" {
				http.Error(w, `{"error":"expired"}`, http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Authorization") != "Bearer new-token" {
				t.Fatalf("Authorization = %q, want Bearer new-token", r.Header.Get("Authorization"))
			}
			if err := r.ParseMultipartForm(1024); err != nil {
				t.Fatalf("ParseMultipartForm() error = %v", err)
			}
			if got := r.FormValue("connectionId"); got != "conn-1" {
				t.Fatalf("connectionId = %q, want conn-1", got)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	source := filepath.Join(t.TempDir(), "upload.txt")
	if err := os.WriteFile(source, []byte("payload"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	cfg := &CLIConfig{ServerURL: server.URL, AccessToken: "old-token", RefreshToken: "refresh-token"}

	body, status, err := apiUploadWithFields("/upload", source, map[string]string{"connectionId": "conn-1"}, cfg)
	if err != nil {
		t.Fatalf("apiUploadWithFields() error = %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", status, http.StatusOK, string(body))
	}
	if uploads != 2 {
		t.Fatalf("uploads = %d, want 2", uploads)
	}
}

func TestAPIClientRequestWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("filter"); got != "active sessions" {
			t.Fatalf("filter = %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	params := url.Values{}
	params.Set("filter", "active sessions")
	_, status, err := APIClient{Config: &CLIConfig{ServerURL: server.URL}}.
		RequestWithParams(http.MethodGet, "/sessions", params, nil)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
}
