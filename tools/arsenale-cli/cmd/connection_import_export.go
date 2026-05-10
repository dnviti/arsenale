package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var connectionImportColumns = []Column{
	{Header: "IMPORTED", Field: "imported"},
	{Header: "SKIPPED", Field: "skipped"},
	{Header: "FAILED", Field: "failed"},
}

func runConnExport(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	payload, err := buildConnectionExportPayload()
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/connections/export", payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	fmt.Fprintln(os.Stdout, string(body))
}

func runConnImport(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	fields, err := buildConnectionImportFields()
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiUploadWithFields("/api/connections/import", connImportFile, fields, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, connectionImportColumns)
}

func buildConnectionExportPayload() (map[string]any, error) {
	format, err := normalizeConnectionExportFormat(connExportFormat)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"format":             format,
		"includeCredentials": connExportIncludeCredentials,
	}
	if len(connExportIDs) > 0 {
		payload["connectionIds"] = connExportIDs
	}
	if folderID := strings.TrimSpace(connExportFolderID); folderID != "" {
		payload["folderId"] = folderID
	}
	return payload, nil
}

func buildConnectionImportFields() (map[string]string, error) {
	strategy, err := normalizeConnectionDuplicateStrategy(connImportDuplicateStrategy)
	if err != nil {
		return nil, err
	}

	fields := map[string]string{"duplicateStrategy": strategy}
	if format := strings.TrimSpace(connImportFormat); format != "" {
		normalized, err := normalizeConnectionImportFormat(format)
		if err != nil {
			return nil, err
		}
		fields["format"] = normalized
	}
	if mapping := strings.TrimSpace(connImportColumnMapping); mapping != "" {
		fields["columnMapping"] = mapping
	}
	return fields, nil
}

func normalizeConnectionExportFormat(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "JSON":
		return "JSON", nil
	case "CSV":
		return "CSV", nil
	default:
		return "", fmt.Errorf("format must be JSON or CSV")
	}
}

func normalizeConnectionImportFormat(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "CSV", "JSON", "MREMOTENG", "RDP":
		return strings.ToUpper(strings.TrimSpace(value)), nil
	default:
		return "", fmt.Errorf("format must be CSV, JSON, MREMOTENG, or RDP")
	}
}

func normalizeConnectionDuplicateStrategy(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", "SKIP":
		return "SKIP", nil
	case "OVERWRITE":
		return "OVERWRITE", nil
	case "RENAME":
		return "RENAME", nil
	default:
		return "", fmt.Errorf("duplicate strategy must be SKIP, OVERWRITE, or RENAME")
	}
}
