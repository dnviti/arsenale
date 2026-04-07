package oauthapi

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func (s Service) HandleSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	sp, err := s.buildSAMLServiceProvider(r.Context())
	if err != nil {
		var reqErr *requestError
		if asRequestError(err, &reqErr) && reqErr.status == http.StatusBadRequest {
			http.NotFound(w, r)
			return
		}
		s.writeError(w, err)
		return
	}

	raw, err := xml.MarshalIndent(sp.Metadata(), "", "  ")
	if err != nil {
		s.writeError(w, fmt.Errorf("marshal saml metadata: %w", err))
		return
	}
	w.Header().Set("content-type", "application/samlmetadata+xml")
	_, _ = w.Write(raw)
}

func (s Service) HandleInitiateSAML(w http.ResponseWriter, r *http.Request) {
	if err := s.startSAMLFlow(w, r, ""); err != nil {
		s.writeError(w, err)
	}
}

func (s Service) HandleInitiateSAMLLink(w http.ResponseWriter, r *http.Request) {
	userID, err := s.resolveLinkUserID(r)
	if err != nil {
		s.writeError(w, err)
		return
	}
	if err := s.startSAMLFlow(w, r, userID); err != nil {
		s.writeError(w, err)
	}
}

func (s Service) HandleSAMLCallback(w http.ResponseWriter, r *http.Request) {
	clientURL := strings.TrimRight(strings.TrimSpace(s.ClientURL), "/")
	if clientURL == "" {
		clientURL = strings.TrimRight(oauthEnv("CLIENT_URL", "https://localhost:3000"), "/")
	}

	if strings.TrimSpace(os.Getenv("SAML_ENTRY_POINT")) == "" {
		http.Redirect(w, r, clientURL+"/login?error=provider_unavailable", http.StatusFound)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, clientURL+"/login?error=authentication_failed", http.StatusFound)
		return
	}

	tracked, err := s.consumeSAMLRequest(r.Context(), r.Form.Get("RelayState"))
	if err != nil {
		http.Redirect(w, r, clientURL+"/login?error=authentication_failed", http.StatusFound)
		return
	}

	sp, err := s.buildSAMLServiceProvider(r.Context())
	if err != nil {
		http.Redirect(w, r, clientURL+"/login?error=provider_unavailable", http.StatusFound)
		return
	}

	assertion, err := sp.ParseResponse(r, []string{tracked.RequestID})
	if err != nil {
		http.Redirect(w, r, clientURL+"/login?error="+url.QueryEscape(callbackErrorCode(err)), http.StatusFound)
		return
	}

	profile, samlAttributes, err := extractSAMLProfile(assertion)
	if err != nil {
		http.Redirect(w, r, clientURL+"/login?error="+url.QueryEscape(callbackErrorCode(err)), http.StatusFound)
		return
	}

	tokens := oauthProviderTokens{}
	if tracked.LinkUserID != "" {
		if err := s.linkOAuthAccount(r.Context(), tracked.LinkUserID, profile, tokens, samlAttributes); err != nil {
			http.Redirect(w, r, clientURL+"/login?error="+url.QueryEscape(callbackErrorCode(err)), http.StatusFound)
			return
		}
		_ = s.insertStandaloneAuditLog(r.Context(), &tracked.LinkUserID, "OAUTH_LINK", map[string]any{
			"provider": "saml",
		}, requestIP(r), nil)
		http.Redirect(w, r, clientURL+"/settings?linked=saml", http.StatusFound)
		return
	}

	result, err := s.findOrCreateOAuthUser(r.Context(), profile, tokens, samlAttributes, requestIP(r))
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
		"provider": "saml",
	}, requestIP(r), result.AllowlistDecision.Flags())
	http.Redirect(w, r, clientURL+"/oauth/callback?code="+url.QueryEscape(authCode), http.StatusFound)
}
