package oauthapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/crewjam/saml"
)

func extractSAMLProfile(assertion *saml.Assertion) (oauthProfile, map[string]any, error) {
	if assertion == nil {
		return oauthProfile{}, nil, &requestError{status: 401, message: "OAuth authentication failed"}
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
		return oauthProfile{}, nil, &requestError{status: 401, message: "OAuth authentication failed"}
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
