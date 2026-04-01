package oauthapi

import (
	"context"
	"net/url"
	"testing"

	"github.com/crewjam/saml"
)

func TestExtractSAMLProfile(t *testing.T) {
	assertion := &saml.Assertion{
		Subject: &saml.Subject{
			NameID: &saml.NameID{
				Value:  "John@example.com",
				Format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
			},
		},
		AuthnStatements: []saml.AuthnStatement{
			{SessionIndex: "session-123"},
		},
		AttributeStatements: []saml.AttributeStatement{
			{
				Attributes: []saml.Attribute{
					{
						Name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
						Values: []saml.AttributeValue{
							{Value: "John Example"},
						},
					},
					{
						Name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/upn",
						Values: []saml.AttributeValue{
							{Value: "john@corp.example.com"},
						},
					},
					{
						Name: "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/windowsdomainname",
						Values: []saml.AttributeValue{
							{Value: "CORP"},
						},
					},
					{
						Name: "http://schemas.microsoft.com/ws/2008/06/identity/claims/groups",
						Values: []saml.AttributeValue{
							{Value: "admins"},
							{Value: "ops"},
						},
					},
				},
			},
		},
	}

	profile, attrs, err := extractSAMLProfile(assertion)
	if err != nil {
		t.Fatalf("extractSAMLProfile returned error: %v", err)
	}

	if profile.Provider != "SAML" {
		t.Fatalf("provider = %q, want SAML", profile.Provider)
	}
	if profile.ProviderUserID != "John@example.com" {
		t.Fatalf("providerUserID = %q, want John@example.com", profile.ProviderUserID)
	}
	if profile.Email != "john@example.com" {
		t.Fatalf("email = %q, want john@example.com", profile.Email)
	}
	if profile.DisplayName != "John Example" {
		t.Fatalf("displayName = %q, want John Example", profile.DisplayName)
	}

	if got := attrs["upn"]; got != "john@corp.example.com" {
		t.Fatalf("upn = %#v, want john@corp.example.com", got)
	}
	if got := attrs["domain"]; got != "CORP" {
		t.Fatalf("domain = %#v, want CORP", got)
	}
	if got := attrs["nameID"]; got != "John@example.com" {
		t.Fatalf("nameID = %#v, want John@example.com", got)
	}
	if got := attrs["sessionIndex"]; got != "session-123" {
		t.Fatalf("sessionIndex = %#v, want session-123", got)
	}
	groups, ok := attrs["groups"].([]string)
	if !ok {
		t.Fatalf("groups type = %T, want []string", attrs["groups"])
	}
	if len(groups) != 2 || groups[0] != "admins" || groups[1] != "ops" {
		t.Fatalf("groups = %#v, want [admins ops]", groups)
	}
}

func TestStoreAndConsumeSAMLRequest(t *testing.T) {
	svc := Service{}

	state, err := svc.storeSAMLRequest(context.Background(), "request-123", "user-456")
	if err != nil {
		t.Fatalf("storeSAMLRequest returned error: %v", err)
	}

	entry, err := svc.consumeSAMLRequest(context.Background(), state)
	if err != nil {
		t.Fatalf("consumeSAMLRequest returned error: %v", err)
	}
	if entry.RequestID != "request-123" {
		t.Fatalf("requestID = %q, want request-123", entry.RequestID)
	}
	if entry.LinkUserID != "user-456" {
		t.Fatalf("linkUserID = %q, want user-456", entry.LinkUserID)
	}

	if _, err := svc.consumeSAMLRequest(context.Background(), state); err == nil {
		t.Fatal("expected consumed saml state to be unavailable")
	}
}

func TestDeriveSAMLMetadataURL(t *testing.T) {
	callbackURL, err := url.Parse("https://api.example.test/api/auth/saml/callback")
	if err != nil {
		t.Fatalf("parse callback url: %v", err)
	}

	metadataURL := deriveSAMLMetadataURL(callbackURL)
	if metadataURL.String() != "https://api.example.test/api/auth/saml/metadata" {
		t.Fatalf("metadataURL = %q, want https://api.example.test/api/auth/saml/metadata", metadataURL.String())
	}
}
