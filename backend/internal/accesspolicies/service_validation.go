package accesspolicies

import (
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
)

func requireTenantAdmin(claims authn.Claims) *requestError {
	if strings.TrimSpace(claims.TenantID) == "" {
		return &requestError{status: http.StatusForbidden, message: "Tenant membership required"}
	}
	switch strings.ToUpper(strings.TrimSpace(claims.TenantRole)) {
	case "OWNER", "ADMIN":
		return nil
	default:
		return &requestError{status: http.StatusForbidden, message: "Insufficient tenant role"}
	}
}

func validateTargetType(targetType string) error {
	switch targetType {
	case "TENANT", "TEAM", "FOLDER":
		return nil
	default:
		return &requestError{status: http.StatusBadRequest, message: "Invalid target type"}
	}
}

func validateTimeWindows(value *string) error {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	for _, segment := range strings.Split(trimmed, ",") {
		segment = strings.TrimSpace(segment)
		if len(segment) != len("00:00-00:00") {
			return &requestError{status: http.StatusBadRequest, message: `Each time window must be in "HH:MM-HH:MM" format (hours 0-23, minutes 0-59)`}
		}
		parts := strings.Split(segment, "-")
		if len(parts) != 2 || !isValidClock(parts[0]) || !isValidClock(parts[1]) {
			return &requestError{status: http.StatusBadRequest, message: `Each time window must be in "HH:MM-HH:MM" format (hours 0-23, minutes 0-59)`}
		}
	}
	return nil
}

func isValidClock(value string) bool {
	if len(value) != 5 || value[2] != ':' {
		return false
	}
	hour := parseTwoDigits(value[:2])
	minute := parseTwoDigits(value[3:])
	return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59
}

func parseTwoDigits(value string) int {
	if len(value) != 2 || value[0] < '0' || value[0] > '9' || value[1] < '0' || value[1] > '9' {
		return -1
	}
	return int(value[0]-'0')*10 + int(value[1]-'0')
}

func defaultBool(value *bool) bool {
	return value != nil && *value
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

func policyBelongsToTenant(targetType, targetID, tenantID string, teamIDs, folderIDs map[string]struct{}) bool {
	switch targetType {
	case "TENANT":
		return targetID == tenantID
	case "TEAM":
		_, ok := teamIDs[targetID]
		return ok
	case "FOLDER":
		_, ok := folderIDs[targetID]
		return ok
	default:
		return false
	}
}
