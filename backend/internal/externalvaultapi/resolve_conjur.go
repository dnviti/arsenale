package externalvaultapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type conjurAuthPayload struct {
	Login     string `json:"login"`
	APIKey    string `json:"apiKey"`
	Account   string `json:"account"`
	ServiceID string `json:"serviceId"`
	HostID    string `json:"hostId"`
}

func (s Service) readConjurSecret(ctx context.Context, record providerRecord, secretPath string) (map[string]string, error) {
	client := providerHTTPClient(record.CACertificate)

	token, account, err := s.resolveConjurAccessToken(ctx, client, record)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(record.ServerURL, "/") + "/secrets/" + url.PathEscape(account) + "/variable/" + url.PathEscape(strings.TrimSpace(secretPath))
	text, statusCode, err := doTextRequest(ctx, client, http.MethodGet, endpoint, map[string]string{
		"Authorization": `Token token="` + encodeConjurToken(token) + `"`,
	}, "")
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, &ResolveError{Status: 502, Message: fmt.Sprintf("Conjur API error (%d): %s", statusCode, strings.TrimSpace(text))}
	}
	return parseStringMap(text), nil
}

func (s Service) resolveConjurAccessToken(ctx context.Context, client *http.Client, record providerRecord) (string, string, error) {
	var auth conjurAuthPayload
	if err := s.decodeProviderAuth(record, &auth); err != nil {
		return "", "", err
	}

	account := strings.TrimSpace(auth.Account)
	if account == "" {
		return "", "", &ResolveError{Status: 400, Message: "Conjur auth payload must include account"}
	}

	switch record.AuthMethod {
	case "CONJUR_API_KEY":
		if strings.TrimSpace(auth.Login) == "" || strings.TrimSpace(auth.APIKey) == "" {
			return "", "", &ResolveError{Status: 400, Message: "Conjur CONJUR_API_KEY auth requires login and apiKey"}
		}
		endpoint := strings.TrimRight(record.ServerURL, "/") + "/authn/" + url.PathEscape(account) + "/" + url.PathEscape(strings.TrimSpace(auth.Login)) + "/authenticate"
		text, statusCode, err := doTextRequest(ctx, client, http.MethodPost, endpoint, map[string]string{
			"Accept":       "text/plain",
			"Content-Type": "text/plain",
		}, strings.TrimSpace(auth.APIKey))
		if err != nil {
			return "", "", err
		}
		if statusCode < 200 || statusCode >= 300 {
			return "", "", &ResolveError{Status: 502, Message: fmt.Sprintf("Conjur API key auth failed (%d): %s", statusCode, strings.TrimSpace(text))}
		}
		return text, account, nil
	case "CONJUR_AUTHN_K8S":
		if strings.TrimSpace(auth.ServiceID) == "" {
			return "", "", &ResolveError{Status: 400, Message: "Conjur CONJUR_AUTHN_K8S auth requires serviceId"}
		}
		k8sToken, err := kubernetesServiceAccountToken()
		if err != nil {
			return "", "", &ResolveError{Status: 400, Message: "Cannot read Kubernetes service account token"}
		}
		endpoint := strings.TrimRight(record.ServerURL, "/") + "/authn-k8s/" + url.PathEscape(strings.TrimSpace(auth.ServiceID)) + "/" + url.PathEscape(account) + "/" + url.PathEscape(strings.TrimSpace(auth.HostID)) + "/authenticate"
		text, statusCode, err := doTextRequest(ctx, client, http.MethodPost, endpoint, map[string]string{
			"Accept":       "text/plain",
			"Content-Type": "text/plain",
		}, k8sToken)
		if err != nil {
			return "", "", err
		}
		if statusCode < 200 || statusCode >= 300 {
			return "", "", &ResolveError{Status: 502, Message: fmt.Sprintf("Conjur K8s auth failed (%d): %s", statusCode, strings.TrimSpace(text))}
		}
		return text, account, nil
	default:
		return "", "", &ResolveError{Status: 502, Message: fmt.Sprintf("native provider resolution is not implemented for authMethod %s", record.AuthMethod)}
	}
}
