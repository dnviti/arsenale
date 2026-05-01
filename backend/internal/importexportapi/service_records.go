package importexportapi

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

func normalizeJSONRecord(item map[string]any) (importRecord, error) {
	record := importRecord{
		Name: strings.TrimSpace(asString(item["name"])),
		Type: strings.TrimSpace(asString(item["type"])),
		Host: strings.TrimSpace(asString(item["host"])),
		Port: intFromAny(item["port"], 22),
	}
	if description := strings.TrimSpace(asString(item["description"])); description != "" {
		record.Description = &description
	}
	if folderName := strings.TrimSpace(asString(item["folderName"])); folderName != "" {
		record.FolderName = &folderName
	}
	record.Username = asString(item["username"])
	record.Password = asString(item["password"])
	if domain := strings.TrimSpace(asString(item["domain"])); domain != "" {
		record.Domain = &domain
	}
	return validateImportRecord(record)
}

func normalizeCSVRecord(headers, row []string, mapping columnMapping) (importRecord, error) {
	values := make(map[string]string, len(headers))
	for idx, header := range headers {
		if idx < len(row) {
			values[header] = strings.TrimSpace(row[idx])
		}
	}
	record := importRecord{
		Name:     values[mapping.resolve("name", "name")],
		Type:     firstNonEmpty(values[mapping.resolve("type", "type")], "SSH"),
		Host:     values[mapping.resolve("host", "host")],
		Port:     parsePort(values[mapping.resolve("port", "port")], 22),
		Username: values[mapping.resolve("username", "username")],
		Password: values[mapping.resolve("password", "password")],
	}
	if description := values[mapping.resolve("description", "description")]; description != "" {
		record.Description = &description
	}
	if folderName := values[mapping.resolve("folder", "folder")]; folderName != "" {
		record.FolderName = &folderName
	}
	if domain := values[mapping.resolve("domain", "domain")]; domain != "" {
		record.Domain = &domain
	}
	return validateImportRecord(record)
}

func validateImportRecord(record importRecord) (importRecord, error) {
	record.Name = strings.TrimSpace(record.Name)
	record.Type = normalizeConnectionType(record.Type)
	record.Host = strings.TrimSpace(record.Host)
	if record.Name == "" || record.Host == "" {
		return importRecord{}, fmt.Errorf("Name and host are required")
	}
	if record.Port < 1 || record.Port > 65535 {
		return importRecord{}, fmt.Errorf("Invalid port number")
	}
	if !slices.Contains([]string{"SSH", "RDP", "VNC"}, record.Type) {
		return importRecord{}, fmt.Errorf("Invalid connection type: %s", record.Type)
	}
	return record, nil
}

func normalizeConnectionType(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SSH", "SFTP", "TELNET":
		return "SSH"
	case "RDP":
		return "RDP"
	case "VNC":
		return "VNC"
	default:
		return strings.ToUpper(strings.TrimSpace(value))
	}
}

func detectFormat(header *multipart.FileHeader, explicit string) string {
	if value := strings.ToUpper(strings.TrimSpace(explicit)); value != "" {
		return value
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".csv":
		return "CSV"
	case ".json":
		return "JSON"
	case ".xml":
		return "MREMOTENG"
	case ".rdp":
		return "RDP"
	default:
		return "CSV"
	}
}

func normalizeDuplicateStrategy(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "RENAME":
		return "RENAME"
	case "OVERWRITE":
		return "OVERWRITE"
	default:
		return "SKIP"
	}
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func intFromAny(value any, fallback int) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed)
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
			return parsed
		}
	}
	return fallback
}

func parsePort(value string, fallback int) int {
	if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		return parsed
	}
	return fallback
}

func normalizeStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func intPtr(value int) *int {
	return &value
}
