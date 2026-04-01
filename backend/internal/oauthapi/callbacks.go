package oauthapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type oauthProfile struct {
	Provider       string
	ProviderUserID string
	Email          string
	DisplayName    string
}

type oauthProviderTokens struct {
	AccessToken  string
	RefreshToken string
}

type oauthLoginResult struct {
	UserID            string
	Email             string
	Username          *string
	AvatarData        *string
	NeedsVaultSetup   bool
	AllowlistDecision ipAllowlistDecision
}

type ipAllowlistDecision struct {
	Flagged bool
	Blocked bool
}

type tenantAllowlist struct {
	Enabled bool
	Mode    string
	Entries []string
}

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

func (s Service) findOrCreateOAuthUser(ctx context.Context, profile oauthProfile, tokens oauthProviderTokens, samlAttributes map[string]any, ipAddress string) (oauthLoginResult, error) {
	if s.DB == nil {
		return oauthLoginResult{}, errors.New("database is unavailable")
	}

	samlAttrsRaw, err := marshalSAMLAttributes(samlAttributes)
	if err != nil {
		return oauthLoginResult{}, err
	}

	var (
		userID            string
		username          *string
		avatarData        *string
		enabled           bool
		needsVaultSetup   bool
		allowlistDecision ipAllowlistDecision
	)

	row := s.DB.QueryRow(ctx, `
SELECT oa.id, u.id, u.username, u."avatarData", u.enabled, NOT COALESCE(u."vaultSetupComplete", false)
FROM "OAuthAccount" oa
JOIN "User" u ON u.id = oa."userId"
WHERE oa.provider = $1
  AND oa."providerUserId" = $2
`, profile.Provider, profile.ProviderUserID)
	var accountID string
	err = row.Scan(&accountID, &userID, &username, &avatarData, &enabled, &needsVaultSetup)
	switch {
	case err == nil:
		if !enabled {
			return oauthLoginResult{}, &requestError{status: http.StatusForbidden, message: "Your account has been disabled. Contact your administrator."}
		}
		if _, err := s.DB.Exec(ctx, `
UPDATE "OAuthAccount"
   SET "accessToken" = $2,
       "refreshToken" = NULLIF($3, ''),
       "providerEmail" = $4,
       "samlAttributes" = COALESCE($5::jsonb, "samlAttributes")
 WHERE id = $1
`, accountID, tokens.AccessToken, tokens.RefreshToken, profile.Email, samlAttrsRaw); err != nil {
			return oauthLoginResult{}, fmt.Errorf("update oauth account: %w", err)
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return oauthLoginResult{}, fmt.Errorf("load oauth account: %w", err)
	default:
		err = s.DB.QueryRow(ctx, `
SELECT id, username, "avatarData", enabled, NOT COALESCE("vaultSetupComplete", false)
FROM "User"
WHERE LOWER(email) = LOWER($1)
`, profile.Email).Scan(&userID, &username, &avatarData, &enabled, &needsVaultSetup)
		switch {
		case err == nil:
			if !enabled {
				return oauthLoginResult{}, &requestError{status: http.StatusForbidden, message: "Your account has been disabled. Contact your administrator."}
			}
			if _, err := s.DB.Exec(ctx, `
INSERT INTO "OAuthAccount" (id, "userId", provider, "providerUserId", "providerEmail", "accessToken", "refreshToken", "samlAttributes")
VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8::jsonb)
`, uuid.NewString(), userID, profile.Provider, profile.ProviderUserID, profile.Email, tokens.AccessToken, tokens.RefreshToken, samlAttrsRaw); err != nil {
				return oauthLoginResult{}, fmt.Errorf("create linked oauth account: %w", err)
			}
		case !errors.Is(err, pgx.ErrNoRows):
			return oauthLoginResult{}, fmt.Errorf("load oauth user by email: %w", err)
		default:
			enabledSignup, err := s.getSelfSignupEnabled(ctx)
			if err != nil {
				return oauthLoginResult{}, err
			}
			if !enabledSignup {
				return oauthLoginResult{}, &requestError{status: http.StatusForbidden, message: "Registration is currently disabled. Contact your administrator."}
			}
			tx, err := s.DB.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return oauthLoginResult{}, fmt.Errorf("begin oauth signup: %w", err)
			}
			defer func() { _ = tx.Rollback(ctx) }()

			userID = uuid.NewString()
			displayName := nullableString(profile.DisplayName)
			if err := tx.QueryRow(ctx, `
INSERT INTO "User" (id, email, username, "passwordHash", "vaultSalt", "encryptedVaultKey", "vaultKeyIV", "vaultKeyTag", "vaultSetupComplete", "emailVerified")
VALUES ($1, $2, $3, NULL, NULL, NULL, NULL, NULL, false, true)
RETURNING username, "avatarData", enabled, NOT COALESCE("vaultSetupComplete", false)
`, userID, profile.Email, displayName).Scan(&username, &avatarData, &enabled, &needsVaultSetup); err != nil {
				return oauthLoginResult{}, fmt.Errorf("create oauth user: %w", err)
			}
			if _, err := tx.Exec(ctx, `
INSERT INTO "OAuthAccount" (id, "userId", provider, "providerUserId", "providerEmail", "accessToken", "refreshToken", "samlAttributes")
VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8::jsonb)
`, uuid.NewString(), userID, profile.Provider, profile.ProviderUserID, profile.Email, tokens.AccessToken, tokens.RefreshToken, samlAttrsRaw); err != nil {
				return oauthLoginResult{}, fmt.Errorf("create oauth account: %w", err)
			}
			if err := tx.Commit(ctx); err != nil {
				return oauthLoginResult{}, fmt.Errorf("commit oauth signup: %w", err)
			}
		}
	}

	_ = s.inferDomainFromSAML(ctx, userID, samlAttributes, profile.Email)

	allowlist, err := s.loadTenantAllowlist(ctx, userID)
	if err != nil {
		return oauthLoginResult{}, err
	}
	allowlistDecision = evaluateIPAllowlist(allowlist, ipAddress)

	return oauthLoginResult{
		UserID:            userID,
		Email:             profile.Email,
		Username:          username,
		AvatarData:        avatarData,
		NeedsVaultSetup:   needsVaultSetup,
		AllowlistDecision: allowlistDecision,
	}, nil
}

func (s Service) linkOAuthAccount(ctx context.Context, userID string, profile oauthProfile, tokens oauthProviderTokens, samlAttributes map[string]any) error {
	if s.DB == nil {
		return errors.New("database is unavailable")
	}

	samlAttrsRaw, err := marshalSAMLAttributes(samlAttributes)
	if err != nil {
		return err
	}

	var existingUserID string
	err = s.DB.QueryRow(ctx, `
SELECT "userId"
FROM "OAuthAccount"
WHERE provider = $1
  AND "providerUserId" = $2
`, profile.Provider, profile.ProviderUserID).Scan(&existingUserID)
	switch {
	case err == nil:
		if existingUserID == userID {
			return nil
		}
		return &requestError{status: http.StatusConflict, message: "This OAuth account is already linked to a different user."}
	case !errors.Is(err, pgx.ErrNoRows):
		return fmt.Errorf("load linked oauth account: %w", err)
	}

	var existingAccountID string
	err = s.DB.QueryRow(ctx, `
SELECT id
FROM "OAuthAccount"
WHERE "userId" = $1
  AND provider = $2
`, userID, profile.Provider).Scan(&existingAccountID)
	switch {
	case err == nil:
		return &requestError{status: http.StatusConflict, message: fmt.Sprintf("You already have a %s account linked.", strings.ToLower(profile.Provider))}
	case !errors.Is(err, pgx.ErrNoRows):
		return fmt.Errorf("load user oauth account: %w", err)
	}

	if _, err := s.DB.Exec(ctx, `
INSERT INTO "OAuthAccount" (id, "userId", provider, "providerUserId", "providerEmail", "accessToken", "refreshToken", "samlAttributes")
VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8::jsonb)
`, uuid.NewString(), userID, profile.Provider, profile.ProviderUserID, profile.Email, tokens.AccessToken, tokens.RefreshToken, samlAttrsRaw); err != nil {
		return fmt.Errorf("create oauth link: %w", err)
	}
	_ = s.inferDomainFromSAML(ctx, userID, samlAttributes, profile.Email)
	return nil
}

func (s Service) getSelfSignupEnabled(ctx context.Context) (bool, error) {
	if strings.TrimSpace(oauthEnv("SELF_SIGNUP_ENABLED", "")) != "true" {
		return false, nil
	}
	if s.DB == nil {
		return false, errors.New("database is unavailable")
	}

	var value string
	err := s.DB.QueryRow(ctx, `SELECT value FROM "AppConfig" WHERE key = 'selfSignupEnabled'`).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("query self-signup flag: %w", err)
	}
	return strings.TrimSpace(strings.ToLower(value)) == "true", nil
}

func (s Service) loadTenantAllowlist(ctx context.Context, userID string) (*tenantAllowlist, error) {
	if s.DB == nil {
		return nil, errors.New("database is unavailable")
	}

	rows, err := s.DB.Query(ctx, `
SELECT tm.status::text, tm."isActive", tm."joinedAt",
       t."ipAllowlistEnabled", t."ipAllowlistMode", t."ipAllowlistEntries"
FROM "TenantMember" tm
JOIN "Tenant" t ON t.id = tm."tenantId"
WHERE tm."userId" = $1
  AND (tm."expiresAt" IS NULL OR tm."expiresAt" > NOW())
ORDER BY tm."joinedAt" ASC
`, userID)
	if err != nil {
		return nil, fmt.Errorf("query oauth memberships: %w", err)
	}
	defer rows.Close()

	type membership struct {
		Status   string
		IsActive bool
		JoinedAt time.Time
		Allow    tenantAllowlist
	}
	var memberships []membership
	for rows.Next() {
		var (
			item    membership
			mode    sql.NullString
			entries []string
		)
		if err := rows.Scan(&item.Status, &item.IsActive, &item.JoinedAt, &item.Allow.Enabled, &mode, &entries); err != nil {
			return nil, fmt.Errorf("scan oauth membership: %w", err)
		}
		item.Allow.Mode = "flag"
		if mode.Valid && strings.TrimSpace(mode.String) != "" {
			item.Allow.Mode = strings.TrimSpace(mode.String)
		}
		item.Allow.Entries = entries
		memberships = append(memberships, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate oauth memberships: %w", err)
	}

	var accepted []membership
	for _, item := range memberships {
		if item.Status == "ACCEPTED" {
			accepted = append(accepted, item)
		}
		if item.Status == "ACCEPTED" && item.IsActive {
			allow := item.Allow
			return &allow, nil
		}
	}
	if len(accepted) == 1 {
		allow := accepted[0].Allow
		return &allow, nil
	}
	return nil, nil
}

func evaluateIPAllowlist(allowlist *tenantAllowlist, ipAddress string) ipAllowlistDecision {
	if allowlist == nil || !allowlist.Enabled {
		return ipAllowlistDecision{}
	}
	if isIPAllowed(ipAddress, allowlist.Entries) {
		return ipAllowlistDecision{}
	}
	if strings.EqualFold(strings.TrimSpace(allowlist.Mode), "block") {
		return ipAllowlistDecision{Blocked: true}
	}
	return ipAllowlistDecision{Flagged: true}
}

func (d ipAllowlistDecision) Flags() []string {
	if !d.Flagged {
		return nil
	}
	return []string{"UNTRUSTED_IP"}
}

func isIPAllowed(ipAddress string, entries []string) bool {
	if len(entries) == 0 {
		return true
	}
	for _, entry := range entries {
		if isIPInCIDR(ipAddress, entry) {
			return true
		}
	}
	return false
}

func isIPInCIDR(ipAddress, cidr string) bool {
	ipAddress = normalizeIP(ipAddress)
	cidr = strings.TrimSpace(cidr)
	if ipAddress == "" || cidr == "" {
		return false
	}

	addr, err := netip.ParseAddr(ipAddress)
	if err != nil {
		return false
	}

	slash := strings.LastIndexByte(cidr, '/')
	if slash == -1 {
		target, err := netip.ParseAddr(normalizeIP(cidr))
		if err != nil {
			return false
		}
		return target == addr
	}

	base := normalizeIP(cidr[:slash])
	prefixLen, err := strconv.Atoi(strings.TrimSpace(cidr[slash+1:]))
	if err != nil {
		return false
	}

	baseAddr, err := netip.ParseAddr(base)
	if err != nil || baseAddr.BitLen() != addr.BitLen() {
		return false
	}
	if prefixLen < 0 || prefixLen > baseAddr.BitLen() {
		return false
	}

	return netip.PrefixFrom(baseAddr, prefixLen).Masked().Contains(addr)
}

func (s Service) insertStandaloneAuditLog(ctx context.Context, userID *string, action string, details map[string]any, ipAddress string, flags []string) error {
	if s.DB == nil {
		return nil
	}
	rawDetails, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}

	query := `INSERT INTO "AuditLog" (id, "userId", action, details, "ipAddress") VALUES ($1, $2, $3::"AuditAction", $4::jsonb, NULLIF($5, ''))`
	args := []any{uuid.NewString(), userID, action, string(rawDetails), ipAddress}
	if len(flags) > 0 {
		query = `INSERT INTO "AuditLog" (id, "userId", action, details, "ipAddress", flags) VALUES ($1, $2, $3::"AuditAction", $4::jsonb, NULLIF($5, ''), $6::text[])`
		args = append(args, flags)
	}
	if _, err := s.DB.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func callbackErrorCode(err error) string {
	var reqErr *requestError
	if errors.As(err, &reqErr) && reqErr.status == http.StatusForbidden {
		if strings.Contains(strings.ToLower(reqErr.message), "disabled") {
			return "account_disabled"
		}
		return "registration_disabled"
	}
	return "authentication_failed"
}

func requestIP(r *http.Request) string {
	for _, value := range []string{
		r.Header.Get("X-Real-IP"),
		firstForwardedFor(r.Header.Get("X-Forwarded-For")),
		r.RemoteAddr,
	} {
		if ip := stripIP(value); ip != "" {
			return ip
		}
	}
	return ""
}

func firstForwardedFor(value string) string {
	for i, ch := range value {
		if ch == ',' {
			return value[:i]
		}
	}
	return value
}

func stripIP(value string) string {
	value = normalizeIP(value)
	if value == "" {
		return ""
	}
	return value
}

func normalizeIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	return strings.TrimPrefix(value, "::ffff:")
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}
