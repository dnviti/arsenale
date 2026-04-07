package systemsettingsapi

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

func parseValue(raw string, valueType SettingType, defaultValue any) any {
	if raw == "" {
		return defaultValue
	}
	switch valueType {
	case "boolean":
		return raw == "true" || raw == "1"
	case "number":
		parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return defaultValue
		}
		return parsed
	case "select", "string", "string[]":
		return raw
	default:
		return raw
	}
}

func serializeValue(value any, def SettingDef) (string, any, error) {
	switch typed := value.(type) {
	case string:
		return typed, redactValue(typed, def.Sensitive), nil
	case bool:
		if typed {
			return "true", redactValue(typed, def.Sensitive), nil
		}
		return "false", redactValue(typed, def.Sensitive), nil
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return "", nil, errors.New("value must be a finite number")
		}
		if math.Trunc(typed) == typed {
			return strconv.FormatInt(int64(typed), 10), redactValue(typed, def.Sensitive), nil
		}
		return strconv.FormatFloat(typed, 'f', -1, 64), redactValue(typed, def.Sensitive), nil
	case int:
		return strconv.Itoa(typed), redactValue(typed, def.Sensitive), nil
	case int32:
		return strconv.FormatInt(int64(typed), 10), redactValue(typed, def.Sensitive), nil
	case int64:
		return strconv.FormatInt(typed, 10), redactValue(typed, def.Sensitive), nil
	default:
		return "", nil, errors.New("value must be a string, number, or boolean")
	}
}

func redactValue(value any, sensitive bool) any {
	if sensitive {
		return "[REDACTED]"
	}
	return value
}
