package oauthapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/app"
	"github.com/redis/go-redis/v9"
)

func availableProviders() map[string]bool {
	providers := map[string]bool{}
	if strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")) != "" {
		providers["google"] = true
	}
	if strings.TrimSpace(os.Getenv("MICROSOFT_CLIENT_ID")) != "" {
		providers["microsoft"] = true
	}
	if strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")) != "" {
		providers["github"] = true
	}
	if strings.TrimSpace(os.Getenv("OIDC_CLIENT_ID")) != "" {
		providers["oidc"] = true
	}
	if strings.TrimSpace(os.Getenv("SAML_ENTRY_POINT")) != "" {
		providers["saml"] = true
	}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("LDAP_ENABLED")), "true") && strings.TrimSpace(os.Getenv("LDAP_SERVER_URL")) != "" {
		providers["ldap"] = true
	}
	return providers
}

func (s Service) buildAuthURL(ctx context.Context, provider string, options providerAuthOptions) (string, error) {
	if strings.EqualFold(strings.TrimSpace(provider), "oidc") {
		return s.buildOIDCAuthURL(ctx, options)
	}
	return buildProviderAuthURL(provider, options)
}

func buildProviderAuthURL(provider string, options providerAuthOptions) (string, error) {
	cfg, err := providerConfig(provider)
	if err != nil {
		return "", err
	}
	if !cfg.Enabled || cfg.ClientID == "" {
		return "", &requestError{status: http.StatusBadRequest, message: "OAuth provider not available"}
	}

	values := url.Values{}
	values.Set("client_id", cfg.ClientID)
	values.Set("redirect_uri", cfg.CallbackURL)
	values.Set("response_type", "code")
	if len(cfg.Scopes) > 0 {
		values.Set("scope", strings.Join(cfg.Scopes, " "))
	}
	if strings.TrimSpace(options.State) != "" {
		values.Set("state", strings.TrimSpace(options.State))
	}
	for key, value := range cfg.Params {
		values.Set(key, value)
	}

	return cfg.AuthURL + "?" + values.Encode(), nil
}

func (s Service) buildOIDCAuthURL(ctx context.Context, options providerAuthOptions) (string, error) {
	cfg, err := providerConfig("oidc")
	if err != nil {
		return "", err
	}
	if !cfg.Enabled || cfg.ClientID == "" {
		return "", &requestError{status: http.StatusBadRequest, message: "OAuth provider not available"}
	}

	discovery, err := s.discoverOIDC(ctx)
	if err != nil {
		return "", err
	}

	state := strings.TrimSpace(options.State)
	if state == "" {
		state, err = randomCode()
		if err != nil {
			return "", err
		}
	}

	codeVerifier, err := randomCode()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(sum[:])

	if err := s.storeOIDCPKCE(ctx, state, codeVerifier); err != nil {
		return "", err
	}

	values := url.Values{}
	values.Set("client_id", cfg.ClientID)
	values.Set("redirect_uri", cfg.CallbackURL)
	values.Set("response_type", "code")
	values.Set("scope", strings.Join(cfg.Scopes, " "))
	values.Set("state", state)
	values.Set("code_challenge", codeChallenge)
	values.Set("code_challenge_method", "S256")

	return discovery.AuthorizationEndpoint + "?" + values.Encode(), nil
}

func providerConfig(provider string) (providerAuthConfig, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "google":
		params := map[string]string{
			"access_type": "offline",
		}
		if hostedDomain := strings.TrimSpace(os.Getenv("GOOGLE_HD")); hostedDomain != "" {
			params["hd"] = hostedDomain
		}
		return providerAuthConfig{
			Enabled:      strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")) != "",
			ClientID:     strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET")),
			CallbackURL:  oauthEnv("GOOGLE_CALLBACK_URL", defaultPublicCallbackURL("/api/auth/oauth/google/callback")),
			Scopes:       []string{"profile", "email"},
			AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			Params:       params,
		}, nil
	case "microsoft":
		tenantID := oauthEnv("MICROSOFT_TENANT_ID", "common")
		return providerAuthConfig{
			Enabled:      strings.TrimSpace(os.Getenv("MICROSOFT_CLIENT_ID")) != "",
			ClientID:     strings.TrimSpace(os.Getenv("MICROSOFT_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(os.Getenv("MICROSOFT_CLIENT_SECRET")),
			CallbackURL:  oauthEnv("MICROSOFT_CALLBACK_URL", defaultPublicCallbackURL("/api/auth/oauth/microsoft/callback")),
			Scopes:       []string{"user.read"},
			AuthURL:      "https://login.microsoftonline.com/" + url.PathEscape(tenantID) + "/oauth2/v2.0/authorize",
		}, nil
	case "github":
		return providerAuthConfig{
			Enabled:      strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")) != "",
			ClientID:     strings.TrimSpace(os.Getenv("GITHUB_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(os.Getenv("GITHUB_CLIENT_SECRET")),
			CallbackURL:  oauthEnv("GITHUB_CALLBACK_URL", defaultPublicCallbackURL("/api/auth/oauth/github/callback")),
			Scopes:       []string{"user:email"},
			AuthURL:      "https://github.com/login/oauth/authorize",
		}, nil
	case "oidc":
		return providerAuthConfig{
			Enabled:      strings.TrimSpace(os.Getenv("OIDC_CLIENT_ID")) != "" && strings.TrimSpace(os.Getenv("OIDC_ISSUER_URL")) != "",
			ClientID:     strings.TrimSpace(os.Getenv("OIDC_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(os.Getenv("OIDC_CLIENT_SECRET")),
			CallbackURL:  oauthEnv("OIDC_CALLBACK_URL", defaultPublicCallbackURL("/api/auth/oauth/oidc/callback")),
			Scopes:       strings.Fields(oauthEnv("OIDC_SCOPES", "openid profile email")),
		}, nil
	default:
		return providerAuthConfig{}, &requestError{status: http.StatusBadRequest, message: "OAuth provider not available"}
	}
}

func defaultPublicCallbackURL(path string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(oauthEnv("CLIENT_URL", "https://localhost:3000")), "/")
	if baseURL == "" {
		baseURL = "https://localhost:3000"
	}
	return baseURL + path
}

func normalizeProvider(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "GOOGLE", "MICROSOFT", "GITHUB", "OIDC", "SAML", "LDAP":
		return strings.ToUpper(strings.TrimSpace(value)), nil
	default:
		return "", &requestError{status: http.StatusBadRequest, message: "OAuth provider not available"}
	}
}

func randomCode() (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("generate random code: %w", err)
	}
	return fmt.Sprintf("%x", buf), nil
}

func oauthEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func (s Service) discoverOIDC(ctx context.Context) (oidcDiscoveryDocument, error) {
	cfg, err := providerConfig("oidc")
	if err != nil {
		return oidcDiscoveryDocument{}, err
	}
	if !cfg.Enabled {
		return oidcDiscoveryDocument{}, &requestError{status: http.StatusBadRequest, message: "OAuth provider not available"}
	}

	issuer := strings.TrimRight(strings.TrimSpace(oauthEnv("OIDC_ISSUER_URL", "")), "/")
	if issuer == "" {
		return oidcDiscoveryDocument{}, &requestError{status: http.StatusBadRequest, message: "OAuth provider not available"}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, issuer+"/.well-known/openid-configuration", nil)
	if err != nil {
		return oidcDiscoveryDocument{}, fmt.Errorf("build oidc discovery request: %w", err)
	}
	req.Header.Set("accept", "application/json")

	resp, err := s.client().Do(req)
	if err != nil {
		return oidcDiscoveryDocument{}, fmt.Errorf("discover oidc endpoints: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return oidcDiscoveryDocument{}, &requestError{status: http.StatusBadGateway, message: "OAuth authentication failed"}
	}

	var doc oidcDiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return oidcDiscoveryDocument{}, fmt.Errorf("decode oidc discovery response: %w", err)
	}
	if strings.TrimSpace(doc.AuthorizationEndpoint) == "" || strings.TrimSpace(doc.TokenEndpoint) == "" || strings.TrimSpace(doc.UserInfoEndpoint) == "" {
		return oidcDiscoveryDocument{}, fmt.Errorf("oidc discovery response missing required endpoints")
	}
	return doc, nil
}

func (s Service) storeOIDCPKCE(ctx context.Context, state, codeVerifier string) error {
	state = strings.TrimSpace(state)
	codeVerifier = strings.TrimSpace(codeVerifier)
	if state == "" || codeVerifier == "" {
		return nil
	}

	if s.Redis != nil {
		if err := s.Redis.Set(ctx, "oidc:pkce:"+state, codeVerifier, oidcPKCETTL).Err(); err != nil {
			return fmt.Errorf("store oidc pkce: %w", err)
		}
		return nil
	}

	oidcPKCEMu.Lock()
	defer oidcPKCEMu.Unlock()
	cleanupExpiredOIDCPKCELocked(time.Now().UnixMilli())
	oidcPKCEStore[state] = linkCodeEntry{
		UserID:    codeVerifier,
		ExpiresAt: time.Now().Add(oidcPKCETTL).UnixMilli(),
	}
	return nil
}

func (s Service) ConsumeOIDCPKCE(ctx context.Context, state string) (string, error) {
	state = strings.TrimSpace(state)
	if state == "" {
		return "", nil
	}

	if s.Redis != nil {
		value, err := s.Redis.GetDel(ctx, "oidc:pkce:"+state).Result()
		if err == nil {
			return strings.TrimSpace(value), nil
		}
		if !errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("load oidc pkce: %w", err)
		}
		return "", nil
	}

	oidcPKCEMu.Lock()
	defer oidcPKCEMu.Unlock()
	cleanupExpiredOIDCPKCELocked(time.Now().UnixMilli())
	entry, ok := oidcPKCEStore[state]
	if !ok {
		return "", nil
	}
	delete(oidcPKCEStore, state)
	if entry.ExpiresAt <= time.Now().UnixMilli() {
		return "", nil
	}
	return strings.TrimSpace(entry.UserID), nil
}

func (s Service) client() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return http.DefaultClient
}

func (s Service) writeError(w http.ResponseWriter, err error) {
	var reqErr *requestError
	if errors.As(err, &reqErr) {
		app.ErrorJSON(w, reqErr.status, reqErr.message)
		return
	}
	app.ErrorJSON(w, http.StatusServiceUnavailable, err.Error())
}
