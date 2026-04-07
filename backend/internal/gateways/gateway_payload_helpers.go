package gateways

import "strings"

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func intValue(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func stringValue(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return strings.ToUpper(strings.TrimSpace(*value))
}

func trimStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func intPtr(value int) *int {
	return &value
}

func chooseString(current string, update optionalString) string {
	if !update.Present || update.Value == nil {
		return current
	}
	return strings.TrimSpace(*update.Value)
}

func chooseNullableString(current *string, update optionalString) *string {
	if !update.Present {
		return current
	}
	if update.Value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*update.Value)
	return &trimmed
}

func chooseInt(current int, update optionalInt) int {
	if !update.Present || update.Value == nil {
		return current
	}
	return *update.Value
}

func chooseNullableInt(current *int, update optionalInt) *int {
	if !update.Present {
		return current
	}
	if update.Value == nil {
		return nil
	}
	value := *update.Value
	return &value
}

func chooseBool(current bool, update optionalBool) bool {
	if !update.Present || update.Value == nil {
		return current
	}
	return *update.Value
}

func hasText(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}
