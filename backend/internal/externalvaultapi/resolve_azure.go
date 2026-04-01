package externalvaultapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type azureAuthPayload struct {
	TenantID     string `json:"tenantId"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

func (s Service) readAzureSecret(ctx context.Context, record providerRecord, secretPath string) (map[string]string, error) {
	client := providerHTTPClient(record.CACertificate)

	token, err := s.resolveAzureAccessToken(ctx, client, record)
	if err != nil {
		return nil, err
	}

	secretName, version := splitAzureSecretPath(secretPath)
	endpoint := strings.TrimRight(record.ServerURL, "/") + "/secrets/" + url.PathEscape(secretName)
	if version != "" {
		endpoint += "/" + url.PathEscape(version)
	}
	endpoint += "?api-version=7.4"

	parsed, statusCode, raw, err := doJSONRequest(ctx, client, http.MethodGet, endpoint, map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/json",
	}, nil)
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, &ResolveError{Status: 502, Message: fmt.Sprintf("Azure Key Vault API error (%d): %s", statusCode, strings.TrimSpace(string(raw)))}
	}

	value, _ := parsed["value"].(string)
	if strings.TrimSpace(value) == "" {
		return nil, &ResolveError{Status: 502, Message: fmt.Sprintf("Secret %q has no value", secretName)}
	}
	return parseStringMap(value), nil
}

func (s Service) resolveAzureAccessToken(ctx context.Context, client *http.Client, record providerRecord) (string, error) {
	var auth azureAuthPayload
	if err := s.decodeProviderAuth(record, &auth); err != nil {
		return "", err
	}

	switch record.AuthMethod {
	case "MANAGED_IDENTITY":
		params := url.Values{
			"api-version": {"2019-08-01"},
			"resource":    {"https://vault.azure.net"},
		}
		if strings.TrimSpace(auth.ClientID) != "" {
			params.Set("client_id", strings.TrimSpace(auth.ClientID))
		}
		endpoint := "http://169.254.169.254/metadata/identity/oauth2/token?" + params.Encode()
		parsed, statusCode, raw, err := doJSONRequest(ctx, client, http.MethodGet, endpoint, map[string]string{
			"Metadata": "true",
			"Accept":   "application/json",
		}, nil)
		if err != nil {
			return "", err
		}
		if statusCode < 200 || statusCode >= 300 {
			return "", &ResolveError{Status: 502, Message: fmt.Sprintf("Azure IMDS token request failed (%d): %s", statusCode, strings.TrimSpace(string(raw)))}
		}
		token, _ := parsed["access_token"].(string)
		if strings.TrimSpace(token) == "" {
			return "", &ResolveError{Status: 502, Message: "Azure IMDS token request did not return an access token"}
		}
		return token, nil
	case "CLIENT_CREDENTIALS":
		if strings.TrimSpace(auth.TenantID) == "" || strings.TrimSpace(auth.ClientID) == "" || strings.TrimSpace(auth.ClientSecret) == "" {
			return "", &ResolveError{Status: 400, Message: "Azure CLIENT_CREDENTIALS auth requires tenantId, clientId, and clientSecret"}
		}

		form := url.Values{
			"grant_type":    {"client_credentials"},
			"client_id":     {strings.TrimSpace(auth.ClientID)},
			"client_secret": {strings.TrimSpace(auth.ClientSecret)},
			"scope":         {"https://vault.azure.net/.default"},
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://login.microsoftonline.com/"+url.PathEscape(strings.TrimSpace(auth.TenantID))+"/oauth2/v2.0/token", strings.NewReader(form.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		var parsed map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return "", err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", &ResolveError{Status: 502, Message: fmt.Sprintf("Azure OAuth2 token request failed (%d)", resp.StatusCode)}
		}
		token, _ := parsed["access_token"].(string)
		if strings.TrimSpace(token) == "" {
			return "", &ResolveError{Status: 502, Message: "Azure OAuth2 token request did not return an access token"}
		}
		return token, nil
	default:
		return "", &ResolveError{Status: 502, Message: fmt.Sprintf("native provider resolution is not implemented for authMethod %s", record.AuthMethod)}
	}
}

func splitAzureSecretPath(secretPath string) (string, string) {
	parts := strings.SplitN(strings.Trim(strings.TrimSpace(secretPath), "/"), "/", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}
