package externalvaultapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func (s Service) normalizeCreatePayload(payload providerCreatePayload) (normalizedPayload, encryptedField, error) {
	normalized := normalizedPayload{
		Name:            strings.TrimSpace(payload.Name),
		ProviderType:    strings.ToUpper(strings.TrimSpace(defaultString(payload.ProviderType, "HASHICORP_VAULT"))),
		ServerURL:       strings.TrimSpace(payload.ServerURL),
		AuthMethod:      strings.ToUpper(strings.TrimSpace(payload.AuthMethod)),
		Namespace:       normalizeOptional(payload.Namespace),
		MountPath:       strings.TrimSpace(defaultOptional(payload.MountPath, "secret")),
		AuthPayload:     strings.TrimSpace(payload.AuthPayload),
		CACertificate:   normalizeOptional(payload.CACertificate),
		CacheTTLSeconds: defaultInt(payload.CacheTTLSeconds, 300),
		Enabled:         true,
	}
	if err := validateNormalized(normalized.ProviderType, normalized.AuthMethod, normalized.ServerURL, normalized.AuthPayload, normalized.CacheTTLSeconds); err != nil {
		return normalizedPayload{}, encryptedField{}, err
	}
	encryptedAuth, err := encryptValue(s.ServerEncryptionKey, normalized.AuthPayload)
	if err != nil {
		return normalizedPayload{}, encryptedField{}, fmt.Errorf("encrypt auth payload: %w", err)
	}
	return normalized, encryptedAuth, nil
}

func (s Service) normalizeUpdatePayload(existing providerRecord, payload providerUpdatePayload) (normalizedPayload, encryptedField, []string, error) {
	normalized := normalizedPayload{
		Name:            existing.Name,
		ProviderType:    existing.ProviderType,
		ServerURL:       existing.ServerURL,
		AuthMethod:      existing.AuthMethod,
		Namespace:       existing.Namespace,
		MountPath:       existing.MountPath,
		AuthPayload:     "",
		CACertificate:   existing.CACertificate,
		CacheTTLSeconds: existing.CacheTTLSeconds,
		Enabled:         existing.Enabled,
	}
	encrypted := encryptedField{
		Ciphertext: existing.EncryptedAuthPayload,
		IV:         existing.AuthPayloadIV,
		Tag:        existing.AuthPayloadTag,
	}
	changed := make([]string, 0)

	if payload.Name != nil {
		normalized.Name = strings.TrimSpace(*payload.Name)
		changed = append(changed, "name")
	}
	if payload.ProviderType != nil {
		normalized.ProviderType = strings.ToUpper(strings.TrimSpace(*payload.ProviderType))
		changed = append(changed, "providerType")
	}
	if payload.ServerURL != nil {
		normalized.ServerURL = strings.TrimSpace(*payload.ServerURL)
		changed = append(changed, "serverUrl")
	}
	if payload.AuthMethod != nil {
		normalized.AuthMethod = strings.ToUpper(strings.TrimSpace(*payload.AuthMethod))
		changed = append(changed, "authMethod")
	}
	if payload.Namespace != nil {
		normalized.Namespace = normalizeOptional(payload.Namespace)
		changed = append(changed, "namespace")
	}
	if payload.MountPath != nil {
		normalized.MountPath = strings.TrimSpace(*payload.MountPath)
		changed = append(changed, "mountPath")
	}
	if payload.CACertificate != nil {
		normalized.CACertificate = normalizeOptional(payload.CACertificate)
		changed = append(changed, "caCertificate")
	}
	if payload.CacheTTLSeconds != nil {
		normalized.CacheTTLSeconds = *payload.CacheTTLSeconds
		changed = append(changed, "cacheTtlSeconds")
	}
	if payload.Enabled != nil {
		normalized.Enabled = *payload.Enabled
		changed = append(changed, "enabled")
	}
	if payload.AuthPayload != nil {
		normalized.AuthPayload = strings.TrimSpace(*payload.AuthPayload)
		var err error
		encrypted, err = encryptValue(s.ServerEncryptionKey, normalized.AuthPayload)
		if err != nil {
			return normalizedPayload{}, encryptedField{}, nil, fmt.Errorf("encrypt auth payload: %w", err)
		}
		changed = append(changed, "authPayload")
	}

	if normalized.AuthPayload == "" {
		normalized.AuthPayload = "{}"
	}
	if err := validateNormalized(normalized.ProviderType, normalized.AuthMethod, normalized.ServerURL, normalized.AuthPayload, normalized.CacheTTLSeconds); err != nil {
		if payload.AuthPayload == nil {
			// Preserve existing encrypted auth data on updates that only change other fields.
			if _, ok := err.(*requestError); ok {
				if strings.Contains(err.Error(), "authPayload") {
					return normalized, encrypted, changed, nil
				}
			}
		}
		return normalizedPayload{}, encryptedField{}, nil, err
	}

	return normalized, encrypted, changed, nil
}

func validateNormalized(providerType, authMethod, serverURL, authPayload string, cacheTTLSeconds int) error {
	if strings.TrimSpace(providerType) == "" {
		return &requestError{status: http.StatusBadRequest, message: "providerType is required"}
	}
	if strings.TrimSpace(authMethod) == "" {
		return &requestError{status: http.StatusBadRequest, message: "authMethod is required"}
	}
	allowed, ok := allowedAuthMethods[providerType]
	if !ok {
		return &requestError{status: http.StatusBadRequest, message: "providerType is not supported"}
	}
	if _, ok := allowed[authMethod]; !ok {
		return &requestError{status: http.StatusBadRequest, message: "authMethod is not supported for the selected providerType"}
	}
	parsedURL, err := url.ParseRequestURI(serverURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return &requestError{status: http.StatusBadRequest, message: "serverUrl must be a valid URL"}
	}
	if cacheTTLSeconds < 0 || cacheTTLSeconds > 86400 {
		return &requestError{status: http.StatusBadRequest, message: "cacheTtlSeconds must be between 0 and 86400"}
	}
	if err := validateAuthPayload(authMethod, authPayload); err != nil {
		return err
	}
	return nil
}

func validateAuthPayload(authMethod, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return &requestError{status: http.StatusBadRequest, message: "authPayload must be valid JSON with the expected keys for the selected authMethod"}
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return &requestError{status: http.StatusBadRequest, message: "authPayload must be valid JSON with the expected keys for the selected authMethod"}
	}
	has := func(key string) bool {
		value, ok := parsed[key]
		if !ok {
			return false
		}
		text, ok := value.(string)
		return ok && strings.TrimSpace(text) != ""
	}
	switch authMethod {
	case "TOKEN":
		if !has("token") {
			return &requestError{status: http.StatusBadRequest, message: "authPayload must be valid JSON with the expected keys for the selected authMethod"}
		}
	case "APPROLE":
		if !has("roleId") || !has("secretId") {
			return &requestError{status: http.StatusBadRequest, message: "authPayload must be valid JSON with the expected keys for the selected authMethod"}
		}
	case "IAM_ACCESS_KEY":
		if !has("accessKeyId") || !has("secretAccessKey") {
			return &requestError{status: http.StatusBadRequest, message: "authPayload must be valid JSON with the expected keys for the selected authMethod"}
		}
	case "CLIENT_CREDENTIALS":
		if !has("tenantId") || !has("clientId") || !has("clientSecret") {
			return &requestError{status: http.StatusBadRequest, message: "authPayload must be valid JSON with the expected keys for the selected authMethod"}
		}
	case "SERVICE_ACCOUNT_KEY":
		if !has("serviceAccountKey") {
			return &requestError{status: http.StatusBadRequest, message: "authPayload must be valid JSON with the expected keys for the selected authMethod"}
		}
	case "WORKLOAD_IDENTITY":
		if !has("projectId") {
			return &requestError{status: http.StatusBadRequest, message: "authPayload must be valid JSON with the expected keys for the selected authMethod"}
		}
	case "CONJUR_API_KEY":
		if !has("login") || !has("apiKey") || !has("account") {
			return &requestError{status: http.StatusBadRequest, message: "authPayload must be valid JSON with the expected keys for the selected authMethod"}
		}
	case "CONJUR_AUTHN_K8S":
		if !has("serviceId") || !has("account") {
			return &requestError{status: http.StatusBadRequest, message: "authPayload must be valid JSON with the expected keys for the selected authMethod"}
		}
	}
	return nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func defaultOptional(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return strings.TrimSpace(*value)
}

func defaultInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
