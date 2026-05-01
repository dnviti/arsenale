package dbauditapi

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var nestedQuantifierPattern = regexp.MustCompile(`(\+|\*|\{[^}]+\})\s*\)?\s*(\+|\*|\?|\{[^}]+\})`)

func validateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", &requestError{status: http.StatusBadRequest, message: "name is required"}
	}
	if len(name) > 200 {
		return "", &requestError{status: http.StatusBadRequest, message: "name must be 200 characters or fewer"}
	}
	return name, nil
}

func validateSafeRegex(pattern, label string) (string, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return "", &requestError{status: http.StatusBadRequest, message: label + " pattern is required"}
	}
	if len(pattern) > maxRegexLength || nestedQuantifierPattern.MatchString(pattern) {
		return "", &requestError{status: http.StatusBadRequest, message: fmt.Sprintf("Regex %s rejected: pattern too long or contains nested quantifiers", label)}
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return "", &requestError{status: http.StatusBadRequest, message: fmt.Sprintf("Invalid regex %s: %s", label, pattern)}
	}
	return pattern, nil
}

func normalizeFirewallAction(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "BLOCK", "ALERT", "LOG":
		return strings.ToUpper(strings.TrimSpace(value)), nil
	default:
		return "", &requestError{status: http.StatusBadRequest, message: "Invalid firewall action"}
	}
}

func normalizeMaskingStrategy(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "REDACT", "HASH", "PARTIAL":
		return strings.ToUpper(strings.TrimSpace(value)), nil
	default:
		return "", &requestError{status: http.StatusBadRequest, message: "Invalid masking strategy"}
	}
}

func normalizeDbQueryType(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SELECT", "INSERT", "UPDATE", "DELETE", "DDL", "OTHER":
		return strings.ToUpper(strings.TrimSpace(value)), nil
	default:
		return "", &requestError{status: http.StatusBadRequest, message: "Invalid db query type"}
	}
}

func normalizeOptionalDbQueryType(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	normalized, err := normalizeDbQueryType(*value)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func normalizeRateLimitAction(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "REJECT", "LOG_ONLY":
		return strings.ToUpper(strings.TrimSpace(value)), nil
	default:
		return "", &requestError{status: http.StatusBadRequest, message: "Invalid rate limit action"}
	}
}

func normalizeOptionalRateLimitAction(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	normalized, err := normalizeRateLimitAction(*value)
	if err != nil {
		return nil, err
	}
	return &normalized, nil
}

func validateRateLimitValues(windowMS, maxQueries, burstMax *int) error {
	if windowMS != nil && *windowMS < 1 {
		return &requestError{status: http.StatusBadRequest, message: "windowMs must be at least 1"}
	}
	if maxQueries != nil && *maxQueries < 1 {
		return &requestError{status: http.StatusBadRequest, message: "maxQueries must be at least 1"}
	}
	if burstMax != nil && *burstMax < 1 {
		return &requestError{status: http.StatusBadRequest, message: "burstMax must be at least 1"}
	}
	return nil
}
