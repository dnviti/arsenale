package oauthapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
)

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
