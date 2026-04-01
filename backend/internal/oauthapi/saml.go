package oauthapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
)

type samlRequestEntry struct {
	RequestID  string `json:"requestId"`
	LinkUserID string `json:"linkUserId,omitempty"`
	ExpiresAt  int64  `json:"expiresAt"`
}

var (
	samlRequestMu    sync.Mutex
	samlRequestStore = map[string]samlRequestEntry{}
)

const samlRequestTTL = 5 * time.Minute

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

func (s Service) startSAMLFlow(w http.ResponseWriter, r *http.Request, linkUserID string) error {
	sp, err := s.buildSAMLServiceProvider(r.Context())
	if err != nil {
		return err
	}

	binding, idpURL := chooseSAMLBinding(sp)
	if binding == "" || idpURL == "" {
		return &requestError{status: http.StatusBadRequest, message: "SAML provider not available"}
	}

	authReq, err := sp.MakeAuthenticationRequest(idpURL, binding, saml.HTTPPostBinding)
	if err != nil {
		return fmt.Errorf("build saml auth request: %w", err)
	}

	state, err := s.storeSAMLRequest(r.Context(), authReq.ID, linkUserID)
	if err != nil {
		return err
	}

	switch binding {
	case saml.HTTPRedirectBinding:
		target, err := authReq.Redirect(state, sp)
		if err != nil {
			return fmt.Errorf("build saml redirect target: %w", err)
		}
		http.Redirect(w, r, target.String(), http.StatusFound)
		return nil
	case saml.HTTPPostBinding:
		w.Header().Set("content-security-policy", "default-src; script-src 'sha256-AjPdJSbZmeWHnEc5ykvJFay8FTWeTeRbs9dutfZ0HqE='; reflected-xss block; referrer no-referrer;")
		w.Header().Set("content-type", "text/html")
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body>"))
		_, _ = w.Write(authReq.Post(state))
		_, _ = w.Write([]byte("</body></html>"))
		return nil
	default:
		return fmt.Errorf("unsupported saml binding: %s", binding)
	}
}

func (s Service) buildSAMLServiceProvider(ctx context.Context) (*saml.ServiceProvider, error) {
	entryPoint := strings.TrimSpace(os.Getenv("SAML_ENTRY_POINT"))
	if entryPoint == "" {
		return nil, &requestError{status: http.StatusBadRequest, message: "SAML provider not available"}
	}

	callbackURL, err := url.Parse(strings.TrimSpace(oauthEnv("SAML_CALLBACK_URL", defaultPublicCallbackURL("/api/auth/saml/callback"))))
	if err != nil || !callbackURL.IsAbs() {
		return nil, fmt.Errorf("invalid SAML callback URL")
	}

	metadataURL := deriveSAMLMetadataURL(callbackURL)
	idpMetadata, err := s.loadSAMLIdentityProvider(ctx, entryPoint)
	if err != nil {
		return nil, err
	}

	sp := &saml.ServiceProvider{
		EntityID:          strings.TrimSpace(oauthEnv("SAML_ISSUER", "arsenale")),
		MetadataURL:       *metadataURL,
		AcsURL:            *callbackURL,
		IDPMetadata:       idpMetadata,
		AuthnNameIDFormat: saml.UnspecifiedNameIDFormat,
		HTTPClient:        s.client(),
	}
	return sp, nil
}

func deriveSAMLMetadataURL(callbackURL *url.URL) *url.URL {
	metadataURL := *callbackURL
	path := strings.TrimSuffix(metadataURL.Path, "/")
	if strings.HasSuffix(path, "/callback") {
		path = strings.TrimSuffix(path, "/callback")
	}
	metadataURL.Path = path + "/metadata"
	metadataURL.RawPath = ""
	metadataURL.RawQuery = ""
	metadataURL.Fragment = ""
	return &metadataURL
}

func (s Service) loadSAMLIdentityProvider(ctx context.Context, entryPoint string) (*saml.EntityDescriptor, error) {
	if rawURL := strings.TrimSpace(os.Getenv("SAML_METADATA_URL")); rawURL != "" {
		metadataURL, err := url.Parse(rawURL)
		if err != nil || !metadataURL.IsAbs() {
			return nil, fmt.Errorf("invalid SAML metadata URL")
		}
		metadata, err := samlsp.FetchMetadata(ctx, s.client(), *metadataURL)
		if err != nil {
			return nil, fmt.Errorf("fetch saml metadata: %w", err)
		}
		return metadata, nil
	}

	keyDescriptors, err := samlKeyDescriptorsFromPEM(strings.TrimSpace(os.Getenv("SAML_CERT")))
	if err != nil {
		return nil, err
	}

	return &saml.EntityDescriptor{
		EntityID: entryPoint,
		IDPSSODescriptors: []saml.IDPSSODescriptor{
			{
				SSODescriptor: saml.SSODescriptor{
					RoleDescriptor: saml.RoleDescriptor{
						ProtocolSupportEnumeration: "urn:oasis:names:tc:SAML:2.0:protocol",
						KeyDescriptors:             keyDescriptors,
					},
					NameIDFormats: []saml.NameIDFormat{saml.UnspecifiedNameIDFormat},
				},
				SingleSignOnServices: []saml.Endpoint{
					{Binding: saml.HTTPRedirectBinding, Location: entryPoint},
					{Binding: saml.HTTPPostBinding, Location: entryPoint},
				},
			},
		},
	}, nil
}

func chooseSAMLBinding(sp *saml.ServiceProvider) (string, string) {
	if location := strings.TrimSpace(sp.GetSSOBindingLocation(saml.HTTPRedirectBinding)); location != "" {
		return saml.HTTPRedirectBinding, location
	}
	if location := strings.TrimSpace(sp.GetSSOBindingLocation(saml.HTTPPostBinding)); location != "" {
		return saml.HTTPPostBinding, location
	}
	return "", ""
}

func samlKeyDescriptorsFromPEM(raw string) ([]saml.KeyDescriptor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var descriptors []saml.KeyDescriptor
	for raw != "" {
		block, rest := pemDecode(raw)
		raw = strings.TrimSpace(rest)
		if block == "" {
			break
		}
		descriptors = append(descriptors, saml.KeyDescriptor{
			Use: "signing",
			KeyInfo: saml.KeyInfo{
				X509Data: saml.X509Data{
					X509Certificates: []saml.X509Certificate{
						{Data: block},
					},
				},
			},
		})
	}
	if len(descriptors) == 0 {
		return nil, fmt.Errorf("invalid SAML certificate")
	}
	return descriptors, nil
}

func pemDecode(raw string) (string, string) {
	const (
		beginMarker = "-----BEGIN CERTIFICATE-----"
		endMarker   = "-----END CERTIFICATE-----"
	)
	begin := strings.Index(raw, beginMarker)
	if begin == -1 {
		return "", ""
	}
	raw = raw[begin+len(beginMarker):]
	end := strings.Index(raw, endMarker)
	if end == -1 {
		return "", ""
	}
	cert := strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', '\t', ' ':
			return -1
		default:
			return r
		}
	}, raw[:end])
	if _, err := base64.StdEncoding.DecodeString(cert); err != nil {
		return "", ""
	}
	return cert, raw[end+len(endMarker):]
}

func (s Service) storeSAMLRequest(ctx context.Context, requestID, linkUserID string) (string, error) {
	state, err := randomCode()
	if err != nil {
		return "", err
	}
	entry := samlRequestEntry{
		RequestID:  strings.TrimSpace(requestID),
		LinkUserID: strings.TrimSpace(linkUserID),
		ExpiresAt:  time.Now().Add(samlRequestTTL).UnixMilli(),
	}

	if s.Redis != nil {
		payload, err := json.Marshal(entry)
		if err != nil {
			return "", fmt.Errorf("marshal saml request entry: %w", err)
		}
		if err := s.Redis.Set(ctx, "saml:request:"+state, payload, samlRequestTTL).Err(); err != nil {
			return "", fmt.Errorf("store saml request: %w", err)
		}
		return state, nil
	}

	samlRequestMu.Lock()
	defer samlRequestMu.Unlock()
	cleanupExpiredSAMLRequestsLocked(time.Now().UnixMilli())
	samlRequestStore[state] = entry
	return state, nil
}

func (s Service) consumeSAMLRequest(ctx context.Context, state string) (samlRequestEntry, error) {
	state = strings.TrimSpace(state)
	if state == "" {
		return samlRequestEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired SAML request"}
	}

	if s.Redis != nil {
		payload, err := s.Redis.GetDel(ctx, "saml:request:"+state).Bytes()
		if err == nil {
			var entry samlRequestEntry
			if err := json.Unmarshal(payload, &entry); err != nil {
				return samlRequestEntry{}, fmt.Errorf("decode saml request entry: %w", err)
			}
			if entry.ExpiresAt <= time.Now().UnixMilli() || strings.TrimSpace(entry.RequestID) == "" {
				return samlRequestEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired SAML request"}
			}
			return entry, nil
		}
		return samlRequestEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired SAML request"}
	}

	samlRequestMu.Lock()
	defer samlRequestMu.Unlock()
	cleanupExpiredSAMLRequestsLocked(time.Now().UnixMilli())
	entry, ok := samlRequestStore[state]
	if !ok {
		return samlRequestEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired SAML request"}
	}
	delete(samlRequestStore, state)
	if entry.ExpiresAt <= time.Now().UnixMilli() || strings.TrimSpace(entry.RequestID) == "" {
		return samlRequestEntry{}, &requestError{status: http.StatusBadRequest, message: "Invalid or expired SAML request"}
	}
	return entry, nil
}

func cleanupExpiredSAMLRequestsLocked(now int64) {
	for state, entry := range samlRequestStore {
		if entry.ExpiresAt <= now {
			delete(samlRequestStore, state)
		}
	}
}

func extractSAMLProfile(assertion *saml.Assertion) (oauthProfile, map[string]any, error) {
	if assertion == nil {
		return oauthProfile{}, nil, &requestError{status: http.StatusUnauthorized, message: "OAuth authentication failed"}
	}

	nameID := ""
	nameIDFormat := ""
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		nameID = strings.TrimSpace(assertion.Subject.NameID.Value)
		nameIDFormat = strings.TrimSpace(assertion.Subject.NameID.Format)
	}

	email := ""
	if nameID != "" && strings.Contains(strings.ToLower(nameIDFormat), "emailaddress") {
		email = nameID
	}
	if email == "" {
		email = firstSAMLAttributeValue(assertion,
			"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"email",
			"mail",
			"emailAddress",
		)
	}
	if email == "" {
		email = firstSAMLAttributeValue(assertion,
			"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/upn",
			"upn",
			"userPrincipalName",
		)
	}
	if email == "" {
		email = nameID
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return oauthProfile{}, nil, &requestError{status: http.StatusUnauthorized, message: "OAuth authentication failed"}
	}

	displayName := strings.TrimSpace(firstSAMLAttributeValue(assertion,
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
		"displayName",
		"name",
		"cn",
	))

	attributes := map[string]any{}
	if upn := strings.TrimSpace(firstSAMLAttributeValue(assertion,
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/upn",
		"upn",
		"userPrincipalName",
	)); upn != "" {
		attributes["upn"] = upn
	}
	if domain := strings.TrimSpace(firstSAMLAttributeValue(assertion,
		"http://schemas.xmlsoap.org/ws/2005/05/identity/claims/windowsdomainname",
		"http://schemas.microsoft.com/identity/claims/tenantid",
		"domain",
		"windowsdomainname",
	)); domain != "" {
		attributes["domain"] = domain
	}
	if groups := samlAttributeValues(assertion,
		"http://schemas.xmlsoap.org/claims/Group",
		"http://schemas.microsoft.com/ws/2008/06/identity/claims/groups",
		"groups",
		"group",
	); len(groups) > 0 {
		attributes["groups"] = groups
	}
	if nameID != "" {
		attributes["nameID"] = nameID
	}
	if nameIDFormat != "" {
		attributes["nameIDFormat"] = nameIDFormat
	}
	if len(assertion.AuthnStatements) > 0 && strings.TrimSpace(assertion.AuthnStatements[0].SessionIndex) != "" {
		attributes["sessionIndex"] = strings.TrimSpace(assertion.AuthnStatements[0].SessionIndex)
	}
	if len(attributes) == 0 {
		attributes = nil
	}

	return oauthProfile{
		Provider:       "SAML",
		ProviderUserID: firstNonEmpty(nameID, email),
		Email:          email,
		DisplayName:    displayName,
	}, attributes, nil
}

func firstSAMLAttributeValue(assertion *saml.Assertion, names ...string) string {
	values := samlAttributeValues(assertion, names...)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func samlAttributeValues(assertion *saml.Assertion, names ...string) []string {
	if assertion == nil {
		return nil
	}

	nameSet := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		nameSet[strings.ToLower(name)] = struct{}{}
	}

	var values []string
	for _, statement := range assertion.AttributeStatements {
		for _, attribute := range statement.Attributes {
			if !matchesSAMLAttribute(attribute, nameSet) {
				continue
			}
			for _, value := range attribute.Values {
				text := strings.TrimSpace(value.Value)
				if text == "" && value.NameID != nil {
					text = strings.TrimSpace(value.NameID.Value)
				}
				if text != "" {
					values = append(values, text)
				}
			}
		}
	}
	return values
}

func matchesSAMLAttribute(attribute saml.Attribute, names map[string]struct{}) bool {
	for _, value := range []string{attribute.Name, attribute.FriendlyName} {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if _, ok := names[value]; ok {
			return true
		}
	}
	return false
}

func marshalSAMLAttributes(samlAttributes map[string]any) (any, error) {
	if len(samlAttributes) == 0 {
		return nil, nil
	}
	payload, err := json.Marshal(samlAttributes)
	if err != nil {
		return nil, fmt.Errorf("marshal saml attributes: %w", err)
	}
	return string(payload), nil
}

func (s Service) inferDomainFromSAML(ctx context.Context, userID string, samlAttributes map[string]any, email string) error {
	if s.DB == nil || len(samlAttributes) == 0 {
		return nil
	}

	var existingDomainName *string
	if err := s.DB.QueryRow(ctx, `SELECT "domainName" FROM "User" WHERE id = $1`, userID).Scan(&existingDomainName); err != nil {
		return fmt.Errorf("load user domain profile: %w", err)
	}
	if existingDomainName != nil && strings.TrimSpace(*existingDomainName) != "" {
		return nil
	}

	var domainName string
	if value := samlString(samlAttributes["domain"]); value != "" {
		domainName = strings.ToUpper(value)
	}

	var domainUsername string
	if upn := samlString(samlAttributes["upn"]); upn != "" {
		localPart, domainPart, ok := strings.Cut(upn, "@")
		if ok {
			domainUsername = strings.TrimSpace(localPart)
			if domainName == "" {
				domainName = netbiosDomain(domainPart)
			}
		}
	}

	if domainName == "" && domainUsername == "" {
		localPart, domainPart, ok := strings.Cut(strings.TrimSpace(email), "@")
		if ok {
			domainUsername = strings.TrimSpace(localPart)
			domainName = netbiosDomain(domainPart)
		}
	}

	if domainName == "" && domainUsername == "" {
		return nil
	}
	if _, err := s.DB.Exec(ctx, `
UPDATE "User"
   SET "domainName" = $2,
       "domainUsername" = $3
 WHERE id = $1
`, userID, nullableString(domainName), nullableString(domainUsername)); err != nil {
		return fmt.Errorf("update inferred domain profile: %w", err)
	}
	return nil
}

func samlString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

func netbiosDomain(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	segment := strings.Split(value, ".")[0]
	return strings.ToUpper(strings.TrimSpace(segment))
}

func asRequestError(err error, target **requestError) bool {
	if err == nil {
		return false
	}
	reqErr, ok := err.(*requestError)
	if ok {
		*target = reqErr
		return true
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
