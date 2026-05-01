package externalvaultapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const resolveRequestTimeout = 10 * time.Second

func (s Service) decodeProviderAuth(record providerRecord, dest any) error {
	rawAuth, err := decryptValue(s.ServerEncryptionKey, record.EncryptedAuthPayload, record.AuthPayloadIV, record.AuthPayloadTag)
	if err != nil {
		return &ResolveError{Status: 500, Message: "failed to decrypt auth payload"}
	}
	if err := json.Unmarshal([]byte(rawAuth), dest); err != nil {
		return &ResolveError{Status: 500, Message: "invalid auth payload"}
	}
	return nil
}

func (s Service) readSecret(ctx context.Context, record providerRecord, secretPath string) (map[string]string, error) {
	switch record.ProviderType {
	case "HASHICORP_VAULT":
		return s.readHashiCorpSecret(ctx, record, secretPath)
	case "AWS_SECRETS_MANAGER":
		return s.readAWSSecret(ctx, record, secretPath)
	case "AZURE_KEY_VAULT":
		return s.readAzureSecret(ctx, record, secretPath)
	case "GCP_SECRET_MANAGER":
		return s.readGCPSecret(ctx, record, secretPath)
	case "CYBERARK_CONJUR":
		return s.readConjurSecret(ctx, record, secretPath)
	default:
		return nil, &ResolveError{
			Status:  502,
			Message: fmt.Sprintf("native provider resolution is not implemented for providerType %s", record.ProviderType),
		}
	}
}

func providerHTTPClient(caCertificate *string) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: resolveProviderTLSConfig(caCertificate),
	}
	return &http.Client{
		Timeout:   resolveRequestTimeout,
		Transport: transport,
	}
}

func doJSONRequest(ctx context.Context, client *http.Client, method, endpoint string, headers map[string]string, body any) (map[string]any, int, []byte, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, 0, nil, err
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, nil, err
	}

	var parsed map[string]any
	if len(raw) != 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, resp.StatusCode, raw, nil
		}
	}
	return parsed, resp.StatusCode, raw, nil
}

func doTextRequest(ctx context.Context, client *http.Client, method, endpoint string, headers map[string]string, body string) (string, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", resp.StatusCode, err
	}
	return string(raw), resp.StatusCode, nil
}

func parseStringMap(raw string) map[string]string {
	var parsed map[string]string
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return parsed
	}
	return map[string]string{"value": raw}
}

func parseSecretResponse(raw []byte) map[string]string {
	var parsed map[string]string
	if err := json.Unmarshal(raw, &parsed); err == nil {
		return parsed
	}
	return map[string]string{"value": string(raw)}
}

func encodeConjurToken(token string) string {
	return base64.StdEncoding.EncodeToString([]byte(token))
}

func resolveProviderTLSConfig(caCertificate *string) *tls.Config {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if caCertificate == nil || strings.TrimSpace(*caCertificate) == "" {
		return cfg
	}
	pool := x509.NewCertPool()
	if pool.AppendCertsFromPEM([]byte(*caCertificate)) {
		cfg.RootCAs = pool
	}
	return cfg
}

func secretPathResource(secretPath, projectID string) string {
	if strings.HasPrefix(secretPath, "projects/") {
		return secretPath
	}
	parts := strings.Split(secretPath, "/")
	secretName := strings.TrimSpace(parts[0])
	version := "versions/latest"
	if len(parts) >= 3 {
		version = strings.Join(parts[1:], "/")
	}
	return fmt.Sprintf("projects/%s/secrets/%s/%s", url.PathEscape(projectID), url.PathEscape(secretName), version)
}

func kubernetesServiceAccountToken() (string, error) {
	payload, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", err
	}
	return string(payload), nil
}
