package externalvaultapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type gcpAuthPayload struct {
	ServiceAccountKey string `json:"serviceAccountKey"`
	ProjectID         string `json:"projectId"`
}

type gcpServiceAccount struct {
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	ProjectID   string `json:"project_id"`
}

func (s Service) readGCPSecret(ctx context.Context, record providerRecord, secretPath string) (map[string]string, error) {
	client := providerHTTPClient(record.CACertificate)

	token, projectID, err := s.resolveGCPAccessToken(ctx, client, record)
	if err != nil {
		return nil, err
	}

	endpoint := "https://secretmanager.googleapis.com/v1/" + secretPathResource(strings.TrimSpace(secretPath), strings.TrimSpace(projectID)) + ":access"
	parsed, statusCode, raw, err := doJSONRequest(ctx, client, http.MethodGet, endpoint, map[string]string{
		"Authorization": "Bearer " + token,
		"Accept":        "application/json",
	}, nil)
	if err != nil {
		return nil, err
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, &ResolveError{Status: 502, Message: fmt.Sprintf("GCP Secret Manager API error (%d): %s", statusCode, strings.TrimSpace(string(raw)))}
	}

	payloadSection, _ := parsed["payload"].(map[string]any)
	encoded, _ := payloadSection["data"].(string)
	if strings.TrimSpace(encoded) == "" {
		return nil, &ResolveError{Status: 502, Message: fmt.Sprintf("Secret %q has no payload data", secretPath)}
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, &ResolveError{Status: 502, Message: fmt.Sprintf("decode GCP secret payload: %v", err)}
	}
	return parseSecretResponse(decoded), nil
}

func (s Service) resolveGCPAccessToken(ctx context.Context, client *http.Client, record providerRecord) (string, string, error) {
	var auth gcpAuthPayload
	if err := s.decodeProviderAuth(record, &auth); err != nil {
		return "", "", err
	}

	switch record.AuthMethod {
	case "WORKLOAD_IDENTITY":
		projectID := strings.TrimSpace(auth.ProjectID)
		if projectID == "" {
			return "", "", &ResolveError{Status: 400, Message: "GCP WORKLOAD_IDENTITY auth requires projectId in auth payload"}
		}

		parsed, statusCode, raw, err := doJSONRequest(ctx, client, http.MethodGet, "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token", map[string]string{
			"Metadata-Flavor": "Google",
			"Accept":          "application/json",
		}, nil)
		if err != nil {
			return "", "", err
		}
		if statusCode < 200 || statusCode >= 300 {
			return "", "", &ResolveError{Status: 502, Message: fmt.Sprintf("GCP metadata token request failed (%d): %s", statusCode, strings.TrimSpace(string(raw)))}
		}
		token, _ := parsed["access_token"].(string)
		if strings.TrimSpace(token) == "" {
			return "", "", &ResolveError{Status: 502, Message: "GCP metadata token request did not return an access token"}
		}
		return token, projectID, nil
	case "SERVICE_ACCOUNT_KEY":
		if strings.TrimSpace(auth.ServiceAccountKey) == "" {
			return "", "", &ResolveError{Status: 400, Message: "GCP SERVICE_ACCOUNT_KEY auth requires serviceAccountKey in auth payload"}
		}
		var serviceAccount gcpServiceAccount
		if err := json.Unmarshal([]byte(auth.ServiceAccountKey), &serviceAccount); err != nil {
			return "", "", &ResolveError{Status: 400, Message: "GCP service account key JSON is malformed"}
		}
		projectID := strings.TrimSpace(auth.ProjectID)
		if projectID == "" {
			projectID = strings.TrimSpace(serviceAccount.ProjectID)
		}
		if projectID == "" {
			return "", "", &ResolveError{Status: 400, Message: "GCP auth payload must include projectId"}
		}

		signedJWT, err := buildGCPServiceAccountJWT(serviceAccount)
		if err != nil {
			return "", "", err
		}
		form := url.Values{
			"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
			"assertion":  {signedJWT},
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
		if err != nil {
			return "", "", err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return "", "", err
		}
		defer resp.Body.Close()

		var parsed map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return "", "", err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "", "", &ResolveError{Status: 502, Message: fmt.Sprintf("GCP OAuth2 token request failed (%d)", resp.StatusCode)}
		}
		token, _ := parsed["access_token"].(string)
		if strings.TrimSpace(token) == "" {
			return "", "", &ResolveError{Status: 502, Message: "GCP OAuth2 token request did not return an access token"}
		}
		return token, projectID, nil
	default:
		return "", "", &ResolveError{Status: 502, Message: fmt.Sprintf("native provider resolution is not implemented for authMethod %s", record.AuthMethod)}
	}
}

func buildGCPServiceAccountJWT(serviceAccount gcpServiceAccount) (string, error) {
	if strings.TrimSpace(serviceAccount.ClientEmail) == "" || strings.TrimSpace(serviceAccount.PrivateKey) == "" {
		return "", &ResolveError{Status: 400, Message: "GCP service account key must contain client_email and private_key"}
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(serviceAccount.PrivateKey))
	if err != nil {
		return "", &ResolveError{Status: 400, Message: fmt.Sprintf("parse GCP service account private key: %v", err)}
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":   strings.TrimSpace(serviceAccount.ClientEmail),
		"sub":   strings.TrimSpace(serviceAccount.ClientEmail),
		"aud":   "https://oauth2.googleapis.com/token",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
		"scope": "https://www.googleapis.com/auth/cloud-platform",
	})
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", &ResolveError{Status: 502, Message: fmt.Sprintf("sign GCP service account JWT: %v", err)}
	}
	return signed, nil
}
