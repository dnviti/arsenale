package oauthapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (s Service) exchangeProviderCode(ctx context.Context, provider, code, state string) (oauthProviderTokens, error) {
	cfg, err := providerConfig(provider)
	if err != nil {
		return oauthProviderTokens{}, err
	}
	if !cfg.Enabled || strings.TrimSpace(cfg.ClientID) == "" {
		return oauthProviderTokens{}, &requestError{status: http.StatusBadRequest, message: "OAuth provider not available"}
	}

	tokenURL, err := s.providerTokenURL(ctx, provider)
	if err != nil {
		return oauthProviderTokens{}, err
	}

	values := url.Values{}
	values.Set("client_id", cfg.ClientID)
	values.Set("client_secret", cfg.ClientSecret)
	values.Set("code", code)
	values.Set("redirect_uri", cfg.CallbackURL)
	values.Set("grant_type", "authorization_code")
	if strings.EqualFold(strings.TrimSpace(provider), "oidc") {
		codeVerifier, err := s.ConsumeOIDCPKCE(ctx, state)
		if err != nil {
			return oauthProviderTokens{}, err
		}
		if codeVerifier != "" {
			values.Set("code_verifier", codeVerifier)
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return oauthProviderTokens{}, fmt.Errorf("build oauth token request: %w", err)
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	resp, err := s.client().Do(req)
	if err != nil {
		return oauthProviderTokens{}, fmt.Errorf("exchange oauth code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauthProviderTokens{}, fmt.Errorf("read oauth token response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return oauthProviderTokens{}, &requestError{status: http.StatusUnauthorized, message: "OAuth authentication failed"}
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return oauthProviderTokens{}, fmt.Errorf("decode oauth token response: %w", err)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return oauthProviderTokens{}, &requestError{status: http.StatusUnauthorized, message: "OAuth authentication failed"}
	}
	return oauthProviderTokens{
		AccessToken:  strings.TrimSpace(payload.AccessToken),
		RefreshToken: strings.TrimSpace(payload.RefreshToken),
	}, nil
}

func (s Service) providerTokenURL(ctx context.Context, provider string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "google":
		return "https://oauth2.googleapis.com/token", nil
	case "microsoft":
		tenantID := oauthEnv("MICROSOFT_TENANT_ID", "common")
		return "https://login.microsoftonline.com/" + url.PathEscape(tenantID) + "/oauth2/v2.0/token", nil
	case "github":
		return "https://github.com/login/oauth/access_token", nil
	case "oidc":
		discovery, err := s.discoverOIDC(ctx)
		if err != nil {
			return "", err
		}
		return discovery.TokenEndpoint, nil
	default:
		return "", &requestError{status: http.StatusBadRequest, message: "OAuth provider not available"}
	}
}
