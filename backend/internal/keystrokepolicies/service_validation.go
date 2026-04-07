package keystrokepolicies

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
)

func requireTenantAdmin(claims authn.Claims) *requestError {
	if claims.TenantID == "" {
		return &requestError{status: http.StatusForbidden, message: "Tenant context is required"}
	}
	switch strings.ToUpper(strings.TrimSpace(claims.TenantRole)) {
	case "ADMIN", "OWNER":
		return nil
	default:
		return &requestError{status: http.StatusForbidden, message: "Admin role required"}
	}
}

func validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return &requestError{status: http.StatusBadRequest, message: "Name is required"}
	}
	if len(name) > 200 {
		return &requestError{status: http.StatusBadRequest, message: "Name must be 200 characters or fewer"}
	}
	return nil
}

func normalizeAction(action string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "BLOCK_AND_TERMINATE", "ALERT_ONLY":
		return strings.ToUpper(strings.TrimSpace(action)), nil
	default:
		return "", &requestError{status: http.StatusBadRequest, message: "Invalid keystroke policy action"}
	}
}

func validatePatterns(patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, &requestError{status: http.StatusBadRequest, message: "At least one regex pattern is required"}
	}
	if len(patterns) > maxPatternsPerPolicy {
		return nil, &requestError{status: http.StatusBadRequest, message: fmt.Sprintf("Too many regex patterns (max %d)", maxPatternsPerPolicy)}
	}
	normalized := make([]string, 0, len(patterns))
	for i, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			return nil, &requestError{status: http.StatusBadRequest, message: fmt.Sprintf("Regex pattern at index %d is required", i)}
		}
		if len(pattern) > maxPatternLength {
			return nil, &requestError{status: http.StatusBadRequest, message: fmt.Sprintf("Regex pattern at index %d exceeds maximum length of %d characters", i, maxPatternLength)}
		}
		if _, err := regexp.Compile(pattern); err != nil {
			return nil, &requestError{status: http.StatusBadRequest, message: fmt.Sprintf("Invalid regular expression pattern at index %d", i)}
		}
		normalized = append(normalized, pattern)
	}
	return normalized, nil
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
