package dbauditapi

import (
	"encoding/json"
	"strings"
)

func decodeString(raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	return value, nil
}

func decodeOptionalString(raw json.RawMessage) (*string, error) {
	if string(raw) == "null" {
		return nil, nil
	}
	value, err := decodeString(raw)
	if err != nil {
		return nil, err
	}
	return normalizeOptionalString(&value), nil
}

func decodeBool(raw json.RawMessage) (bool, error) {
	var value bool
	err := json.Unmarshal(raw, &value)
	return value, err
}

func decodeInt(raw json.RawMessage) (int, error) {
	var value int
	err := json.Unmarshal(raw, &value)
	return value, err
}

func decodeStringSlice(raw json.RawMessage) ([]string, error) {
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func decodeOptionalEnumString(raw json.RawMessage, normalize func(string) (string, error)) (*string, bool, error) {
	if string(raw) == "null" {
		return nil, false, nil
	}
	value, err := decodeString(raw)
	if err != nil {
		return nil, false, err
	}
	normalized, err := normalize(value)
	if err != nil {
		return nil, false, err
	}
	return &normalized, true, nil
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func defaultBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func defaultInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func defaultString(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return *value
}

func defaultStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
