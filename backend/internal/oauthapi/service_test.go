package oauthapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBuildProviderAuthURLGoogle(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "google-client")
	t.Setenv("GOOGLE_CALLBACK_URL", "https://example.test/api/auth/oauth/google/callback")

	target, err := buildProviderAuthURL("google", providerAuthOptions{})
	if err != nil {
		t.Fatalf("buildProviderAuthURL returned error: %v", err)
	}

	parsed, err := url.Parse(target)
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	if parsed.Host != "accounts.google.com" {
		t.Fatalf("unexpected host: %s", parsed.Host)
	}

	query := parsed.Query()
	if query.Get("client_id") != "google-client" {
		t.Fatalf("unexpected client_id: %s", query.Get("client_id"))
	}
	if query.Get("redirect_uri") != "https://example.test/api/auth/oauth/google/callback" {
		t.Fatalf("unexpected redirect_uri: %s", query.Get("redirect_uri"))
	}
	if query.Get("scope") != "profile email" {
		t.Fatalf("unexpected scope: %s", query.Get("scope"))
	}
}

func TestBuildProviderAuthURLGoogleIncludesHostedDomainAndState(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "google-client")
	t.Setenv("GOOGLE_HD", "example.com")

	target, err := buildProviderAuthURL("google", providerAuthOptions{State: "relay-code"})
	if err != nil {
		t.Fatalf("buildProviderAuthURL returned error: %v", err)
	}

	parsed, err := url.Parse(target)
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	query := parsed.Query()
	if query.Get("hd") != "example.com" {
		t.Fatalf("unexpected hd: %s", query.Get("hd"))
	}
	if query.Get("state") != "relay-code" {
		t.Fatalf("unexpected state: %s", query.Get("state"))
	}
}

func TestBuildAuthURLOIDCIncludesPKCEAndState(t *testing.T) {
	discovery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			t.Fatalf("unexpected discovery path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer":                 discoveryURLWithoutScheme(r.Host),
			"authorization_endpoint": "https://issuer.example.test/oauth2/authorize",
			"token_endpoint":         "https://issuer.example.test/oauth2/token",
			"userinfo_endpoint":      "https://issuer.example.test/oauth2/userinfo",
		})
	}))
	defer discovery.Close()

	t.Setenv("OIDC_CLIENT_ID", "oidc-client")
	t.Setenv("OIDC_CLIENT_SECRET", "oidc-secret")
	t.Setenv("OIDC_ISSUER_URL", discovery.URL)
	t.Setenv("OIDC_CALLBACK_URL", "https://example.test/api/auth/oauth/oidc/callback")
	t.Setenv("OIDC_SCOPES", "openid profile email groups")

	svc := Service{HTTPClient: discovery.Client()}
	target, err := svc.buildAuthURL(context.Background(), "oidc", providerAuthOptions{State: "relay-code"})
	if err != nil {
		t.Fatalf("buildAuthURL returned error: %v", err)
	}

	parsed, err := url.Parse(target)
	if err != nil {
		t.Fatalf("parse target: %v", err)
	}
	query := parsed.Query()
	if parsed.Host != "issuer.example.test" {
		t.Fatalf("unexpected host: %s", parsed.Host)
	}
	if query.Get("client_id") != "oidc-client" {
		t.Fatalf("unexpected client_id: %s", query.Get("client_id"))
	}
	if query.Get("redirect_uri") != "https://example.test/api/auth/oauth/oidc/callback" {
		t.Fatalf("unexpected redirect_uri: %s", query.Get("redirect_uri"))
	}
	if query.Get("scope") != "openid profile email groups" {
		t.Fatalf("unexpected scope: %s", query.Get("scope"))
	}
	if query.Get("state") != "relay-code" {
		t.Fatalf("unexpected state: %s", query.Get("state"))
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Fatalf("unexpected code_challenge_method: %s", query.Get("code_challenge_method"))
	}
	if query.Get("code_challenge") == "" {
		t.Fatal("expected code_challenge to be set")
	}

	codeVerifier, err := svc.ConsumeOIDCPKCE(context.Background(), "relay-code")
	if err != nil {
		t.Fatalf("ConsumeOIDCPKCE returned error: %v", err)
	}
	if codeVerifier == "" {
		t.Fatal("expected oidc code verifier to be stored")
	}
}

func TestProviderConfigMarksDisabledProviders(t *testing.T) {
	t.Setenv("GITHUB_CLIENT_ID", "")

	cfg, err := providerConfig("github")
	if err != nil {
		t.Fatalf("providerConfig returned error: %v", err)
	}
	if cfg.Enabled {
		t.Fatal("expected github to be disabled")
	}
}

func TestBuildProviderAuthURLRejectsUnknownProvider(t *testing.T) {
	if _, err := buildProviderAuthURL("unknown", providerAuthOptions{}); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestHandleInitiateProviderPathValue(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "google-client")
	t.Setenv("GOOGLE_CALLBACK_URL", "https://example.test/api/auth/oauth/google/callback")

	req := httptest.NewRequest(http.MethodGet, "https://example.test/api/auth/oauth/google", nil)
	req.SetPathValue("provider", "google")
	rec := httptest.NewRecorder()

	Service{}.HandleInitiateProviderPathValue(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("expected redirect location")
	}
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	if parsed.Host != "accounts.google.com" {
		t.Fatalf("unexpected redirect host: %s", parsed.Host)
	}
}

func TestHandleInitiateLinkPathValue(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "google-client")
	t.Setenv("GOOGLE_CALLBACK_URL", "https://example.test/api/auth/oauth/google/callback")

	svc := Service{}
	linkCode, err := svc.GenerateLinkCode(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("GenerateLinkCode returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "https://example.test/api/auth/oauth/link/google?code="+url.QueryEscape(linkCode), nil)
	req.SetPathValue("provider", "google")
	rec := httptest.NewRecorder()

	svc.HandleInitiateLinkPathValue(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	location := rec.Header().Get("Location")
	if location == "" {
		t.Fatal("expected redirect location")
	}
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	if parsed.Host != "accounts.google.com" {
		t.Fatalf("unexpected redirect host: %s", parsed.Host)
	}
	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatal("expected relay state to be set")
	}
	userID, err := svc.ConsumeRelayCode(context.Background(), state)
	if err != nil {
		t.Fatalf("ConsumeRelayCode returned error: %v", err)
	}
	if userID != "user-123" {
		t.Fatalf("relay user = %q, want %q", userID, "user-123")
	}
}

func TestHandleCallbackPathValueWithProviderError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://example.test/api/auth/oauth/google/callback?error=access_denied", nil)
	req.SetPathValue("provider", "google")
	rec := httptest.NewRecorder()

	Service{ClientURL: "https://client.example.test"}.HandleCallbackPathValue(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got := rec.Header().Get("Location"); got != "https://client.example.test/login?error=authentication_failed" {
		t.Fatalf("redirect = %q, want %q", got, "https://client.example.test/login?error=authentication_failed")
	}
}

func TestPickGitHubEmailPrefersPrimaryVerified(t *testing.T) {
	emails := []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}{
		{Email: "secondary@example.com", Primary: false, Verified: true},
		{Email: "primary@example.com", Primary: true, Verified: true},
	}

	if got := pickGitHubEmail(emails); got != "primary@example.com" {
		t.Fatalf("pickGitHubEmail() = %q, want %q", got, "primary@example.com")
	}
}

func TestPickGitHubEmailFallsBackToVerified(t *testing.T) {
	emails := []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}{
		{Email: "unverified@example.com", Primary: true, Verified: false},
		{Email: "verified@example.com", Primary: false, Verified: true},
	}

	if got := pickGitHubEmail(emails); got != "verified@example.com" {
		t.Fatalf("pickGitHubEmail() = %q, want %q", got, "verified@example.com")
	}
}

func discoveryURLWithoutScheme(host string) string {
	return "https://" + host
}
