package externalvaultapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type hashiCorpTokenAuth struct {
	Token string `json:"token"`
}

type hashiCorpAppRoleAuth struct {
	RoleID   string `json:"roleId"`
	SecretID string `json:"secretId"`
}

func (s Service) readHashiCorpSecret(ctx context.Context, record providerRecord, secretPath string) (map[string]string, error) {
	client := providerHTTPClient(record.CACertificate)

	token, err := s.resolveHashiCorpToken(ctx, client, record)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(record.ServerURL, "/") + "/v1/" + strings.Trim(strings.TrimSpace(record.MountPath), "/")
	endpoint += "/data/" + strings.TrimLeft(strings.TrimSpace(secretPath), "/")

	headers := map[string]string{
		"Accept":        "application/json",
		"X-Vault-Token": token,
	}
	if record.Namespace != nil && strings.TrimSpace(*record.Namespace) != "" {
		headers["X-Vault-Namespace"] = strings.TrimSpace(*record.Namespace)
	}

	parsed, statusCode, raw, err := doJSONRequest(ctx, client, http.MethodGet, endpoint, headers, nil)
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, &ResolveError{Status: 502, Message: fmt.Sprintf("HashiCorp Vault API error (%d): %s", statusCode, strings.TrimSpace(string(raw)))}
	}

	dataSection, _ := parsed["data"].(map[string]any)
	innerData, _ := dataSection["data"].(map[string]any)
	if len(innerData) == 0 {
		return nil, &ResolveError{Status: 502, Message: fmt.Sprintf("Secret at %q is empty or has unexpected format", secretPath)}
	}

	result := make(map[string]string, len(innerData))
	for key, value := range innerData {
		switch typed := value.(type) {
		case string:
			result[key] = typed
		default:
			result[key] = strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return result, nil
}

func (s Service) resolveHashiCorpToken(ctx context.Context, client *http.Client, record providerRecord) (string, error) {
	switch record.AuthMethod {
	case "TOKEN":
		var auth hashiCorpTokenAuth
		if err := s.decodeProviderAuth(record, &auth); err != nil {
			return "", err
		}
		if strings.TrimSpace(auth.Token) == "" {
			return "", &ResolveError{Status: 400, Message: "HashiCorp Vault TOKEN auth requires token"}
		}
		return strings.TrimSpace(auth.Token), nil
	case "APPROLE":
		var auth hashiCorpAppRoleAuth
		if err := s.decodeProviderAuth(record, &auth); err != nil {
			return "", err
		}
		if strings.TrimSpace(auth.RoleID) == "" || strings.TrimSpace(auth.SecretID) == "" {
			return "", &ResolveError{Status: 400, Message: "HashiCorp Vault APPROLE auth requires roleId and secretId"}
		}

		endpoint := strings.TrimRight(record.ServerURL, "/") + "/v1/auth/approle/login"
		headers := map[string]string{"Accept": "application/json"}
		if record.Namespace != nil && strings.TrimSpace(*record.Namespace) != "" {
			headers["X-Vault-Namespace"] = strings.TrimSpace(*record.Namespace)
		}

		parsed, statusCode, raw, err := doJSONRequest(ctx, client, http.MethodPost, endpoint, headers, map[string]string{
			"role_id":   strings.TrimSpace(auth.RoleID),
			"secret_id": strings.TrimSpace(auth.SecretID),
		})
		if err != nil {
			return "", err
		}
		if statusCode < 200 || statusCode >= 300 {
			return "", &ResolveError{Status: 502, Message: fmt.Sprintf("HashiCorp Vault AppRole login failed (%d): %s", statusCode, strings.TrimSpace(string(raw)))}
		}

		authSection, _ := parsed["auth"].(map[string]any)
		clientToken, _ := authSection["client_token"].(string)
		if strings.TrimSpace(clientToken) == "" {
			return "", &ResolveError{Status: 502, Message: "AppRole login did not return a client token"}
		}
		return strings.TrimSpace(clientToken), nil
	default:
		return "", &ResolveError{Status: 502, Message: fmt.Sprintf("native provider resolution is not implemented for authMethod %s", record.AuthMethod)}
	}
}
