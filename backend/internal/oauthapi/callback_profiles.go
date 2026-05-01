package oauthapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func (s Service) fetchProviderProfile(ctx context.Context, provider, accessToken string) (oauthProfile, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "google":
		return s.fetchGoogleProfile(ctx, accessToken)
	case "microsoft":
		return s.fetchMicrosoftProfile(ctx, accessToken)
	case "github":
		return s.fetchGitHubProfile(ctx, accessToken)
	case "oidc":
		return s.fetchOIDCProfile(ctx, accessToken)
	default:
		return oauthProfile{}, &requestError{status: http.StatusBadRequest, message: "OAuth provider not available"}
	}
}

func (s Service) fetchGoogleProfile(ctx context.Context, accessToken string) (oauthProfile, error) {
	var payload struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := s.getProviderJSON(ctx, "https://www.googleapis.com/oauth2/v2/userinfo", accessToken, &payload, nil); err != nil {
		return oauthProfile{}, err
	}
	if strings.TrimSpace(payload.Email) == "" || strings.TrimSpace(payload.ID) == "" {
		return oauthProfile{}, &requestError{status: http.StatusUnauthorized, message: "OAuth authentication failed"}
	}
	return oauthProfile{
		Provider:       "GOOGLE",
		ProviderUserID: strings.TrimSpace(payload.ID),
		Email:          strings.ToLower(strings.TrimSpace(payload.Email)),
		DisplayName:    strings.TrimSpace(payload.Name),
	}, nil
}

func (s Service) fetchMicrosoftProfile(ctx context.Context, accessToken string) (oauthProfile, error) {
	var payload struct {
		ID                string `json:"id"`
		DisplayName       string `json:"displayName"`
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
	}
	if err := s.getProviderJSON(ctx, "https://graph.microsoft.com/v1.0/me", accessToken, &payload, nil); err != nil {
		return oauthProfile{}, err
	}
	email := strings.ToLower(strings.TrimSpace(payload.Mail))
	if email == "" {
		email = strings.ToLower(strings.TrimSpace(payload.UserPrincipalName))
	}
	if email == "" || strings.TrimSpace(payload.ID) == "" {
		return oauthProfile{}, &requestError{status: http.StatusUnauthorized, message: "OAuth authentication failed"}
	}
	return oauthProfile{
		Provider:       "MICROSOFT",
		ProviderUserID: strings.TrimSpace(payload.ID),
		Email:          email,
		DisplayName:    strings.TrimSpace(payload.DisplayName),
	}, nil
}

func (s Service) fetchGitHubProfile(ctx context.Context, accessToken string) (oauthProfile, error) {
	var payload struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	headers := map[string]string{
		"user-agent": "arsenale-go",
	}
	if err := s.getProviderJSON(ctx, "https://api.github.com/user", accessToken, &payload, headers); err != nil {
		return oauthProfile{}, err
	}
	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if email == "" {
		resolved, err := s.fetchGitHubEmail(ctx, accessToken)
		if err != nil {
			return oauthProfile{}, err
		}
		email = resolved
	}
	if email == "" || payload.ID == 0 {
		return oauthProfile{}, &requestError{status: http.StatusUnauthorized, message: "OAuth authentication failed"}
	}
	displayName := strings.TrimSpace(payload.Name)
	if displayName == "" {
		displayName = strings.TrimSpace(payload.Login)
	}
	return oauthProfile{
		Provider:       "GITHUB",
		ProviderUserID: fmt.Sprintf("%d", payload.ID),
		Email:          email,
		DisplayName:    displayName,
	}, nil
}

func (s Service) fetchOIDCProfile(ctx context.Context, accessToken string) (oauthProfile, error) {
	discovery, err := s.discoverOIDC(ctx)
	if err != nil {
		return oauthProfile{}, err
	}

	var payload struct {
		Sub               string `json:"sub"`
		Email             string `json:"email"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	if err := s.getProviderJSON(ctx, discovery.UserInfoEndpoint, accessToken, &payload, nil); err != nil {
		return oauthProfile{}, err
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if email == "" || strings.TrimSpace(payload.Sub) == "" {
		return oauthProfile{}, &requestError{status: http.StatusUnauthorized, message: "OAuth authentication failed"}
	}
	displayName := strings.TrimSpace(payload.Name)
	if displayName == "" {
		displayName = strings.TrimSpace(payload.PreferredUsername)
	}
	return oauthProfile{
		Provider:       "OIDC",
		ProviderUserID: strings.TrimSpace(payload.Sub),
		Email:          email,
		DisplayName:    displayName,
	}, nil
}

func (s Service) fetchGitHubEmail(ctx context.Context, accessToken string) (string, error) {
	var payload []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	headers := map[string]string{
		"user-agent": "arsenale-go",
	}
	if err := s.getProviderJSON(ctx, "https://api.github.com/user/emails", accessToken, &payload, headers); err != nil {
		return "", err
	}
	return pickGitHubEmail(payload), nil
}

func pickGitHubEmail(emails []struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}) string {
	for _, item := range emails {
		email := strings.ToLower(strings.TrimSpace(item.Email))
		if item.Primary && item.Verified && email != "" {
			return email
		}
	}
	for _, item := range emails {
		email := strings.ToLower(strings.TrimSpace(item.Email))
		if item.Verified && email != "" {
			return email
		}
	}
	for _, item := range emails {
		email := strings.ToLower(strings.TrimSpace(item.Email))
		if email != "" {
			return email
		}
	}
	return ""
}

func (s Service) getProviderJSON(ctx context.Context, endpoint, accessToken string, target any, headers map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build provider profile request: %w", err)
	}
	req.Header.Set("authorization", "Bearer "+strings.TrimSpace(accessToken))
	req.Header.Set("accept", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := s.client().Do(req)
	if err != nil {
		return fmt.Errorf("fetch oauth profile: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return &requestError{status: http.StatusUnauthorized, message: "OAuth authentication failed"}
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode oauth profile response: %w", err)
	}
	return nil
}
