package secretsmeta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type createSecretInput struct {
	Name        string
	Description *string
	Type        string
	Scope       string
	TeamID      *string
	FolderID    *string
	Data        json.RawMessage
	Metadata    map[string]any
	Tags        []string
	ExpiresAt   *time.Time
}

type updateSecretInput struct {
	Name           *string
	DescriptionSet bool
	Description    *string
	Data           json.RawMessage
	DataSet        bool
	MetadataSet    bool
	Metadata       map[string]any
	Tags           *[]string
	FolderIDSet    bool
	FolderID       *string
	IsFavorite     *bool
	ExpiresAtSet   bool
	ExpiresAt      *time.Time
	ChangeNote     *string
	Fields         []string
}

type createExternalShareInput struct {
	ExpiresInMinutes int
	MaxAccessCount   *int
	Pin              *string
}

type shareSecretInput struct {
	Email      *string
	UserID     *string
	Permission string
}

type updateSecretSharePermissionInput struct {
	Permission string
}

type loginSecretPayload struct {
	Type     string  `json:"type"`
	Username string  `json:"username"`
	Password string  `json:"password"`
	Domain   *string `json:"domain,omitempty"`
	URL      *string `json:"url,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

type sshKeySecretPayload struct {
	Type       string  `json:"type"`
	Username   *string `json:"username,omitempty"`
	PrivateKey string  `json:"privateKey"`
	PublicKey  *string `json:"publicKey,omitempty"`
	Passphrase *string `json:"passphrase,omitempty"`
	Algorithm  *string `json:"algorithm,omitempty"`
	Notes      *string `json:"notes,omitempty"`
}

type certificateSecretPayload struct {
	Type        string  `json:"type"`
	Certificate string  `json:"certificate"`
	PrivateKey  string  `json:"privateKey"`
	Chain       *string `json:"chain,omitempty"`
	Passphrase  *string `json:"passphrase,omitempty"`
	ExpiresAt   *string `json:"expiresAt,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

type apiKeySecretPayload struct {
	Type     string            `json:"type"`
	APIKey   string            `json:"apiKey"`
	Endpoint *string           `json:"endpoint,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Notes    *string           `json:"notes,omitempty"`
}

type secureNoteSecretPayload struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func parseCreateSecretInput(body []byte) (createSecretInput, error) {
	fields, err := decodeJSONObject(body)
	if err != nil {
		return createSecretInput{}, err
	}

	var input createSecretInput
	if input.Name, err = requiredNonEmptyStringField(fields, "name"); err != nil {
		return createSecretInput{}, err
	}
	if input.Description, err = optionalStringField(fields, "description", false); err != nil {
		return createSecretInput{}, err
	}
	if input.Type, err = requiredEnumField(fields, "type", "LOGIN", "SSH_KEY", "CERTIFICATE", "API_KEY", "SECURE_NOTE"); err != nil {
		return createSecretInput{}, err
	}
	if input.Scope, err = requiredEnumField(fields, "scope", "PERSONAL", "TEAM", "TENANT"); err != nil {
		return createSecretInput{}, err
	}
	if input.TeamID, err = optionalUUIDField(fields, "teamId", false); err != nil {
		return createSecretInput{}, err
	}
	if input.FolderID, err = optionalUUIDField(fields, "folderId", false); err != nil {
		return createSecretInput{}, err
	}
	if input.Metadata, err = optionalJSONObjectField(fields, "metadata", false); err != nil {
		return createSecretInput{}, err
	}
	if input.Tags, err = optionalStringArrayField(fields, "tags", false); err != nil {
		return createSecretInput{}, err
	}
	if input.ExpiresAt, err = optionalTimeField(fields, "expiresAt", false); err != nil {
		return createSecretInput{}, err
	}
	if input.Data, err = requiredSecretPayloadField(fields, "data"); err != nil {
		return createSecretInput{}, err
	}
	if payloadType, err := secretPayloadType(input.Data); err != nil {
		return createSecretInput{}, err
	} else if payloadType != input.Type {
		return createSecretInput{}, fmt.Errorf("Secret data type does not match declared type")
	}
	if input.Scope == "TEAM" && input.TeamID == nil {
		return createSecretInput{}, fmt.Errorf("teamId is required for team-scoped secrets")
	}

	return input, nil
}

func parseUpdateSecretInput(body []byte) (updateSecretInput, error) {
	fields, err := decodeJSONObject(body)
	if err != nil {
		return updateSecretInput{}, err
	}

	var input updateSecretInput

	if _, ok := fields["name"]; ok {
		value, err := requiredNonEmptyStringField(fields, "name")
		if err != nil {
			return updateSecretInput{}, err
		}
		input.Name = &value
		input.Fields = append(input.Fields, "name")
	}
	if _, ok := fields["description"]; ok {
		value, err := optionalStringField(fields, "description", true)
		if err != nil {
			return updateSecretInput{}, err
		}
		input.DescriptionSet = true
		input.Description = value
		input.Fields = append(input.Fields, "description")
	}
	if raw, ok := fields["data"]; ok {
		payload, err := sanitizeSecretPayload(raw)
		if err != nil {
			return updateSecretInput{}, err
		}
		input.Data = payload
		input.DataSet = true
		input.Fields = append(input.Fields, "data")
	}
	if _, ok := fields["metadata"]; ok {
		value, err := optionalJSONObjectField(fields, "metadata", true)
		if err != nil {
			return updateSecretInput{}, err
		}
		input.MetadataSet = true
		input.Metadata = value
		input.Fields = append(input.Fields, "metadata")
	}
	if _, ok := fields["tags"]; ok {
		value, err := optionalStringArrayField(fields, "tags", false)
		if err != nil {
			return updateSecretInput{}, err
		}
		input.Tags = &value
		input.Fields = append(input.Fields, "tags")
	}
	if _, ok := fields["folderId"]; ok {
		value, err := optionalUUIDField(fields, "folderId", true)
		if err != nil {
			return updateSecretInput{}, err
		}
		input.FolderIDSet = true
		input.FolderID = value
		input.Fields = append(input.Fields, "folderId")
	}
	if _, ok := fields["isFavorite"]; ok {
		value, err := boolField(fields, "isFavorite")
		if err != nil {
			return updateSecretInput{}, err
		}
		input.IsFavorite = &value
		input.Fields = append(input.Fields, "isFavorite")
	}
	if _, ok := fields["expiresAt"]; ok {
		value, err := optionalTimeField(fields, "expiresAt", true)
		if err != nil {
			return updateSecretInput{}, err
		}
		input.ExpiresAtSet = true
		input.ExpiresAt = value
		input.Fields = append(input.Fields, "expiresAt")
	}
	if _, ok := fields["changeNote"]; ok {
		value, err := optionalStringField(fields, "changeNote", false)
		if err != nil {
			return updateSecretInput{}, err
		}
		input.ChangeNote = value
		input.Fields = append(input.Fields, "changeNote")
	}

	if len(input.Fields) == 0 {
		return updateSecretInput{}, fmt.Errorf("No fields to update")
	}

	return input, nil
}

func parseCreateExternalShareInput(body []byte) (createExternalShareInput, error) {
	fields, err := decodeJSONObject(body)
	if err != nil {
		return createExternalShareInput{}, err
	}

	var input createExternalShareInput
	if input.ExpiresInMinutes, err = requiredIntField(fields, "expiresInMinutes"); err != nil {
		return createExternalShareInput{}, err
	}
	if input.ExpiresInMinutes < 5 || input.ExpiresInMinutes > 43200 {
		return createExternalShareInput{}, fmt.Errorf("expiresInMinutes must be between 5 and 43200")
	}
	if _, ok := fields["maxAccessCount"]; ok {
		value, err := optionalIntField(fields, "maxAccessCount", false)
		if err != nil {
			return createExternalShareInput{}, err
		}
		if value != nil && (*value < 1 || *value > 1000) {
			return createExternalShareInput{}, fmt.Errorf("maxAccessCount must be between 1 and 1000")
		}
		input.MaxAccessCount = value
	}
	if _, ok := fields["pin"]; ok {
		value, err := optionalStringField(fields, "pin", false)
		if err != nil {
			return createExternalShareInput{}, err
		}
		if value != nil {
			pin := strings.TrimSpace(*value)
			if len(pin) < 4 || len(pin) > 8 {
				return createExternalShareInput{}, fmt.Errorf("PIN must be 4-8 digits")
			}
			for _, r := range pin {
				if r < '0' || r > '9' {
					return createExternalShareInput{}, fmt.Errorf("PIN must be 4-8 digits")
				}
			}
			input.Pin = &pin
		}
	}

	return input, nil
}

func parseShareSecretInput(body []byte) (shareSecretInput, error) {
	fields, err := decodeJSONObject(body)
	if err != nil {
		return shareSecretInput{}, err
	}

	var input shareSecretInput
	if input.Email, err = optionalStringField(fields, "email", false); err != nil {
		return shareSecretInput{}, err
	}
	if input.UserID, err = optionalStringField(fields, "userId", false); err != nil {
		return shareSecretInput{}, err
	}
	if input.Permission, err = requiredEnumField(fields, "permission", "READ_ONLY", "FULL_ACCESS"); err != nil {
		return shareSecretInput{}, err
	}
	if input.Email == nil && input.UserID == nil {
		return shareSecretInput{}, fmt.Errorf("Either email or userId is required")
	}
	return input, nil
}

func parseUpdateSecretSharePermissionInput(body []byte) (updateSecretSharePermissionInput, error) {
	fields, err := decodeJSONObject(body)
	if err != nil {
		return updateSecretSharePermissionInput{}, err
	}

	permission, err := requiredEnumField(fields, "permission", "READ_ONLY", "FULL_ACCESS")
	if err != nil {
		return updateSecretSharePermissionInput{}, err
	}
	return updateSecretSharePermissionInput{Permission: permission}, nil
}

func decodeJSONObject(body []byte) (map[string]json.RawMessage, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, fmt.Errorf("request body must be a JSON object")
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("request body must be a JSON object")
	}
	if fields == nil {
		return nil, fmt.Errorf("request body must be a JSON object")
	}
	return fields, nil
}

func requiredNonEmptyStringField(fields map[string]json.RawMessage, name string) (string, error) {
	value, err := stringField(fields, name, false)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func requiredEnumField(fields map[string]json.RawMessage, name string, allowed ...string) (string, error) {
	value, err := requiredNonEmptyStringField(fields, name)
	if err != nil {
		return "", err
	}
	for _, item := range allowed {
		if value == item {
			return value, nil
		}
	}
	return "", fmt.Errorf("%s must be one of %s", name, strings.Join(allowed, ", "))
}

func stringField(fields map[string]json.RawMessage, name string, allowNull bool) (string, error) {
	raw, ok := fields[name]
	if !ok {
		return "", fmt.Errorf("%s is required", name)
	}
	if allowNull && isJSONNull(raw) {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("%s must be a string", name)
	}
	return value, nil
}

func optionalStringField(fields map[string]json.RawMessage, name string, allowNull bool) (*string, error) {
	raw, ok := fields[name]
	if !ok {
		return nil, nil
	}
	if isJSONNull(raw) {
		if allowNull {
			return nil, nil
		}
		return nil, fmt.Errorf("%s must be a string", name)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be a string", name)
	}
	return &value, nil
}

func optionalUUIDField(fields map[string]json.RawMessage, name string, allowNull bool) (*string, error) {
	value, err := optionalStringField(fields, name, allowNull)
	if err != nil || value == nil {
		return value, err
	}
	if _, err := uuid.Parse(strings.TrimSpace(*value)); err != nil {
		return nil, fmt.Errorf("%s must be a valid UUID", name)
	}
	return value, nil
}

func optionalJSONObjectField(fields map[string]json.RawMessage, name string, allowNull bool) (map[string]any, error) {
	raw, ok := fields[name]
	if !ok {
		return nil, nil
	}
	if isJSONNull(raw) {
		if allowNull {
			return nil, nil
		}
		return nil, fmt.Errorf("%s must be an object", name)
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, fmt.Errorf("%s must be an object", name)
	}
	return value, nil
}

func optionalStringArrayField(fields map[string]json.RawMessage, name string, allowNull bool) ([]string, error) {
	raw, ok := fields[name]
	if !ok {
		return nil, nil
	}
	if isJSONNull(raw) {
		if allowNull {
			return nil, nil
		}
		return nil, fmt.Errorf("%s must be an array of strings", name)
	}
	var value []string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be an array of strings", name)
	}
	return value, nil
}

func boolField(fields map[string]json.RawMessage, name string) (bool, error) {
	raw, ok := fields[name]
	if !ok {
		return false, fmt.Errorf("%s is required", name)
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, fmt.Errorf("%s must be a boolean", name)
	}
	return value, nil
}

func requiredIntField(fields map[string]json.RawMessage, name string) (int, error) {
	raw, ok := fields[name]
	if !ok {
		return 0, fmt.Errorf("%s is required", name)
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fmt.Errorf("%s must be an integer", name)
	}
	return value, nil
}

func optionalIntField(fields map[string]json.RawMessage, name string, allowNull bool) (*int, error) {
	raw, ok := fields[name]
	if !ok {
		return nil, nil
	}
	if isJSONNull(raw) {
		if allowNull {
			return nil, nil
		}
		return nil, fmt.Errorf("%s must be an integer", name)
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be an integer", name)
	}
	return &value, nil
}

func optionalTimeField(fields map[string]json.RawMessage, name string, allowNull bool) (*time.Time, error) {
	value, err := optionalStringField(fields, name, allowNull)
	if err != nil || value == nil {
		return nil, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, *value)
	if err != nil {
		return nil, fmt.Errorf("%s must be a valid RFC3339 datetime", name)
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func requiredSecretPayloadField(fields map[string]json.RawMessage, name string) (json.RawMessage, error) {
	raw, ok := fields[name]
	if !ok || isJSONNull(raw) {
		return nil, fmt.Errorf("%s is required", name)
	}
	return sanitizeSecretPayload(raw)
}

func sanitizeSecretPayload(raw json.RawMessage) (json.RawMessage, error) {
	payloadType, err := secretPayloadType(raw)
	if err != nil {
		return nil, err
	}

	switch payloadType {
	case "LOGIN":
		var payload loginSecretPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("data must be a valid secret payload")
		}
		if strings.TrimSpace(payload.Username) == "" {
			return nil, fmt.Errorf("data.username is required")
		}
		if strings.TrimSpace(payload.Password) == "" {
			return nil, fmt.Errorf("data.password is required")
		}
		return marshalSanitizedPayload(payload)
	case "SSH_KEY":
		var payload sshKeySecretPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("data must be a valid secret payload")
		}
		if strings.TrimSpace(payload.PrivateKey) == "" {
			return nil, fmt.Errorf("data.privateKey is required")
		}
		return marshalSanitizedPayload(payload)
	case "CERTIFICATE":
		var payload certificateSecretPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("data must be a valid secret payload")
		}
		if strings.TrimSpace(payload.Certificate) == "" {
			return nil, fmt.Errorf("data.certificate is required")
		}
		if strings.TrimSpace(payload.PrivateKey) == "" {
			return nil, fmt.Errorf("data.privateKey is required")
		}
		return marshalSanitizedPayload(payload)
	case "API_KEY":
		var payload apiKeySecretPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("data must be a valid secret payload")
		}
		if strings.TrimSpace(payload.APIKey) == "" {
			return nil, fmt.Errorf("data.apiKey is required")
		}
		return marshalSanitizedPayload(payload)
	case "SECURE_NOTE":
		var payload secureNoteSecretPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("data must be a valid secret payload")
		}
		if strings.TrimSpace(payload.Content) == "" {
			return nil, fmt.Errorf("data.content is required")
		}
		return marshalSanitizedPayload(payload)
	default:
		return nil, fmt.Errorf("data.type must be one of LOGIN, SSH_KEY, CERTIFICATE, API_KEY, or SECURE_NOTE")
	}
}

func secretPayloadType(raw json.RawMessage) (string, error) {
	if isJSONNull(raw) || !json.Valid(raw) {
		return "", fmt.Errorf("data must be a valid secret payload")
	}
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return "", fmt.Errorf("data must be a valid secret payload")
	}
	switch strings.TrimSpace(header.Type) {
	case "LOGIN", "SSH_KEY", "CERTIFICATE", "API_KEY", "SECURE_NOTE":
		return header.Type, nil
	default:
		return "", fmt.Errorf("data.type must be one of LOGIN, SSH_KEY, CERTIFICATE, API_KEY, or SECURE_NOTE")
	}
}

func marshalSanitizedPayload(value any) (json.RawMessage, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal secret payload: %w", err)
	}
	return json.RawMessage(encoded), nil
}

func isJSONNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
