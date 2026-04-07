package oauthapi

import (
	"net/http"
	"net/url"
	"strings"
)

func (s Service) HandleInitiateLinkPathValue(w http.ResponseWriter, r *http.Request) {
	s.HandleInitiateLink(w, r, r.PathValue("provider"))
}

func (s Service) HandleInitiateLink(w http.ResponseWriter, r *http.Request, provider string) {
	userID, err := s.resolveLinkUserID(r)
	if err != nil {
		s.writeError(w, err)
		return
	}

	relayCode, err := s.GenerateRelayCode(r.Context(), userID)
	if err != nil {
		s.writeError(w, err)
		return
	}

	target, err := s.buildAuthURL(r.Context(), provider, providerAuthOptions{State: relayCode})
	if err != nil {
		s.writeError(w, err)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (s Service) HandleCallbackPathValue(w http.ResponseWriter, r *http.Request) {
	s.HandleCallback(w, r, r.PathValue("provider"))
}

func (s Service) HandleCallback(w http.ResponseWriter, r *http.Request, provider string) {
	clientURL := strings.TrimRight(strings.TrimSpace(s.ClientURL), "/")
	if clientURL == "" {
		clientURL = strings.TrimRight(oauthEnv("CLIENT_URL", "https://localhost:3000"), "/")
	}

	if strings.TrimSpace(r.URL.Query().Get("error")) != "" {
		http.Redirect(w, r, clientURL+"/login?error=authentication_failed", http.StatusFound)
		return
	}

	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		http.Redirect(w, r, clientURL+"/login?error=authentication_failed", http.StatusFound)
		return
	}

	state := strings.TrimSpace(r.URL.Query().Get("state"))

	tokens, err := s.exchangeProviderCode(r.Context(), provider, code, state)
	if err != nil {
		http.Redirect(w, r, clientURL+"/login?error="+url.QueryEscape(callbackErrorCode(err)), http.StatusFound)
		return
	}

	profile, err := s.fetchProviderProfile(r.Context(), provider, tokens.AccessToken)
	if err != nil {
		http.Redirect(w, r, clientURL+"/login?error="+url.QueryEscape(callbackErrorCode(err)), http.StatusFound)
		return
	}

	linkUserID, err := s.ConsumeRelayCode(r.Context(), state)
	if err != nil {
		http.Redirect(w, r, clientURL+"/login?error="+url.QueryEscape(callbackErrorCode(err)), http.StatusFound)
		return
	}
	if linkUserID != "" {
		if err := s.linkOAuthAccount(r.Context(), linkUserID, profile, tokens, nil); err != nil {
			http.Redirect(w, r, clientURL+"/login?error="+url.QueryEscape(callbackErrorCode(err)), http.StatusFound)
			return
		}
		_ = s.insertStandaloneAuditLog(r.Context(), &linkUserID, "OAUTH_LINK", map[string]any{
			"provider": strings.ToLower(strings.TrimSpace(provider)),
		}, requestIP(r), nil)
		http.Redirect(w, r, clientURL+"/settings?linked="+url.QueryEscape(strings.ToLower(strings.TrimSpace(provider))), http.StatusFound)
		return
	}

	result, err := s.findOrCreateOAuthUser(r.Context(), profile, tokens, nil, requestIP(r))
	if err != nil {
		http.Redirect(w, r, clientURL+"/login?error="+url.QueryEscape(callbackErrorCode(err)), http.StatusFound)
		return
	}

	if result.AllowlistDecision.Blocked {
		_ = s.insertStandaloneAuditLog(r.Context(), &result.UserID, "LOGIN_FAILURE", map[string]any{
			"reason": "ip_not_allowed",
		}, requestIP(r), nil)
		http.Redirect(w, r, clientURL+"/login?error=ip_not_allowed", http.StatusFound)
		return
	}

	if s.Auth == nil {
		http.Redirect(w, r, clientURL+"/login?error=authentication_failed", http.StatusFound)
		return
	}

	browserTokens, err := s.Auth.IssueBrowserTokensForUser(r.Context(), result.UserID, requestIP(r), r.UserAgent())
	if err != nil {
		http.Redirect(w, r, clientURL+"/login?error=authentication_failed", http.StatusFound)
		return
	}

	csrfToken, err := s.Auth.ApplyBrowserAuthCookies(r.Context(), w, browserTokens.User.ID, browserTokens.RefreshToken, browserTokens.RefreshExpires)
	if err != nil {
		http.Redirect(w, r, clientURL+"/login?error=authentication_failed", http.StatusFound)
		return
	}
	authCode, err := s.storeAuthCodeEntry(r.Context(), authCodeEntry{
		AccessToken:     browserTokens.AccessToken,
		CSRFToken:       csrfToken,
		NeedsVaultSetup: result.NeedsVaultSetup,
		UserID:          result.UserID,
		Email:           result.Email,
		Username:        deref(result.Username),
		AvatarData:      deref(result.AvatarData),
		TenantID:        browserTokens.User.TenantID,
		TenantRole:      browserTokens.User.TenantRole,
	})
	if err != nil {
		http.Redirect(w, r, clientURL+"/login?error=authentication_failed", http.StatusFound)
		return
	}

	_ = s.insertStandaloneAuditLog(r.Context(), &result.UserID, "LOGIN_OAUTH", map[string]any{
		"provider": strings.ToLower(strings.TrimSpace(provider)),
	}, requestIP(r), result.AllowlistDecision.Flags())

	http.Redirect(w, r, clientURL+"/oauth/callback?code="+url.QueryEscape(authCode), http.StatusFound)
}

func (s Service) resolveLinkUserID(r *http.Request) (string, error) {
	if code := strings.TrimSpace(r.URL.Query().Get("code")); code != "" {
		return s.ConsumeLinkCode(r.Context(), code)
	}

	if s.Authenticator == nil {
		return "", &requestError{status: http.StatusUnauthorized, message: "Missing authentication"}
	}
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" && !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
		r = r.Clone(r.Context())
		r.Header.Set("Authorization", "Bearer "+token)
	}
	claims, err := s.Authenticator.Authenticate(r)
	if err != nil {
		return "", &requestError{status: http.StatusUnauthorized, message: "Missing authentication"}
	}
	return claims.UserID, nil
}
