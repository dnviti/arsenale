package importexportapi

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/authn"
	"github.com/dnviti/arsenale/backend/internal/connections"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s Service) ImportConnections(ctx context.Context, r *http.Request, claims authn.Claims) (importResult, error) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		return importResult{}, &requestError{status: http.StatusBadRequest, message: "invalid multipart form"}
	}
	mapping, err := parseColumnMapping(r.FormValue("columnMapping"))
	if err != nil {
		return importResult{}, err
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return importResult{}, &requestError{status: http.StatusBadRequest, message: "No file uploaded"}
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return importResult{}, fmt.Errorf("read import file: %w", err)
	}

	format := detectFormat(header, r.FormValue("format"))
	switch format {
	case "CSV":
		return s.importCSV(ctx, claims, header.Filename, content, mapping, normalizeDuplicateStrategy(r.FormValue("duplicateStrategy")), requestIP(r))
	case "JSON":
		return s.importJSON(ctx, claims, header.Filename, content, normalizeDuplicateStrategy(r.FormValue("duplicateStrategy")), requestIP(r))
	case "MREMOTENG", "RDP":
		if format == "MREMOTENG" {
			return s.importMRemoteNG(ctx, claims, header.Filename, content, normalizeDuplicateStrategy(r.FormValue("duplicateStrategy")), requestIP(r))
		}
		return s.importRDP(ctx, claims, header.Filename, content, normalizeDuplicateStrategy(r.FormValue("duplicateStrategy")), requestIP(r))
	default:
		return importResult{}, &requestError{status: http.StatusBadRequest, message: "Unsupported format"}
	}
}

func (s Service) importJSON(ctx context.Context, claims authn.Claims, filename string, content []byte, duplicateStrategy string, ip *string) (importResult, error) {
	var payload any
	if err := json.Unmarshal(content, &payload); err != nil {
		return importResult{}, &requestError{status: http.StatusBadRequest, message: "Invalid JSON format"}
	}

	var connectionsToImport []map[string]any
	switch value := payload.(type) {
	case []any:
		for _, item := range value {
			if mapped, ok := item.(map[string]any); ok {
				connectionsToImport = append(connectionsToImport, mapped)
			}
		}
	case map[string]any:
		if rawConnections, ok := value["connections"].([]any); ok {
			for _, item := range rawConnections {
				if mapped, ok := item.(map[string]any); ok {
					connectionsToImport = append(connectionsToImport, mapped)
				}
			}
		}
	}

	result := importResult{Errors: make([]importResultError, 0)}
	for idx, item := range connectionsToImport {
		record, err := normalizeJSONRecord(item)
		if err != nil {
			row := idx + 1
			result.Failed++
			result.Errors = append(result.Errors, importResultError{Row: &row, Filename: filename, Error: err.Error()})
			continue
		}
		if err := s.importOne(ctx, claims, record, duplicateStrategy, ip); err != nil {
			if reqErr, ok := err.(*requestError); ok {
				row := idx + 1
				if reqErr.status == http.StatusConflict {
					result.Skipped++
				} else {
					result.Failed++
					result.Errors = append(result.Errors, importResultError{Row: &row, Filename: filename, Error: reqErr.message})
				}
				continue
			}
			row := idx + 1
			result.Failed++
			result.Errors = append(result.Errors, importResultError{Row: &row, Filename: filename, Error: err.Error()})
			continue
		}
		result.Imported++
	}

	if err := s.insertAuditLog(ctx, claims.UserID, "IMPORT_CONNECTIONS", "", map[string]any{
		"format":   "JSON",
		"imported": result.Imported,
		"skipped":  result.Skipped,
		"failed":   result.Failed,
	}, ip); err != nil {
		return importResult{}, err
	}
	return result, nil
}

func (s Service) importCSV(ctx context.Context, claims authn.Claims, filename string, content []byte, mapping columnMapping, duplicateStrategy string, ip *string) (importResult, error) {
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return importResult{}, &requestError{status: http.StatusBadRequest, message: "Invalid CSV format"}
	}
	if len(rows) == 0 {
		return importResult{Errors: []importResultError{}}, nil
	}

	headers := rows[0]
	normalized := make([]string, len(headers))
	for i, header := range headers {
		normalized[i] = strings.ToLower(strings.TrimSpace(header))
	}

	result := importResult{Errors: make([]importResultError, 0)}
	for idx, row := range rows[1:] {
		record, err := normalizeCSVRecord(normalized, row, mapping)
		if err != nil {
			line := idx + 2
			result.Failed++
			result.Errors = append(result.Errors, importResultError{Row: &line, Filename: filename, Error: err.Error()})
			continue
		}
		if err := s.importOne(ctx, claims, record, duplicateStrategy, ip); err != nil {
			line := idx + 2
			var reqErr *requestError
			if errors.As(err, &reqErr) && reqErr.status == http.StatusConflict {
				result.Skipped++
			} else {
				result.Failed++
				result.Errors = append(result.Errors, importResultError{Row: &line, Filename: filename, Error: err.Error()})
			}
			continue
		}
		result.Imported++
	}

	if err := s.insertAuditLog(ctx, claims.UserID, "IMPORT_CONNECTIONS", "", map[string]any{
		"format":   "CSV",
		"imported": result.Imported,
		"skipped":  result.Skipped,
		"failed":   result.Failed,
	}, ip); err != nil {
		return importResult{}, err
	}
	return result, nil
}

func (s Service) importMRemoteNG(ctx context.Context, claims authn.Claims, filename string, content []byte, duplicateStrategy string, ip *string) (importResult, error) {
	parsed, err := parseMRemoteNGXML(string(content))
	if err != nil {
		return importResult{}, &requestError{status: http.StatusBadRequest, message: err.Error()}
	}

	result := importResult{Errors: make([]importResultError, 0)}
	for idx, item := range parsed {
		connType := mapMRemoteProtocol(item.Protocol)
		if connType == "" {
			result.Skipped++
			continue
		}

		record, err := validateImportRecord(importRecord{
			Name:        item.Name,
			Type:        connType,
			Host:        item.Hostname,
			Port:        parsePort(item.Port, 22),
			Username:    item.Username,
			Password:    item.Password,
			Description: normalizeStringPointer(item.Description),
			FolderName:  normalizeStringPointer(item.Panel),
		})
		if err != nil {
			line := idx + 1
			result.Failed++
			result.Errors = append(result.Errors, importResultError{Row: &line, Filename: filename, Error: err.Error()})
			continue
		}

		if err := s.importOne(ctx, claims, record, duplicateStrategy, ip); err != nil {
			line := idx + 1
			var reqErr *requestError
			if errors.As(err, &reqErr) && reqErr.status == http.StatusConflict {
				result.Skipped++
			} else {
				result.Failed++
				result.Errors = append(result.Errors, importResultError{Row: &line, Filename: filename, Error: err.Error()})
			}
			continue
		}
		result.Imported++
	}

	if err := s.insertAuditLog(ctx, claims.UserID, "IMPORT_CONNECTIONS", "", map[string]any{
		"format":   "MREMOTENG",
		"imported": result.Imported,
		"skipped":  result.Skipped,
		"failed":   result.Failed,
	}, ip); err != nil {
		return importResult{}, err
	}
	return result, nil
}

func (s Service) importRDP(ctx context.Context, claims authn.Claims, filename string, content []byte, duplicateStrategy string, ip *string) (importResult, error) {
	parsed := parseRDPFile(string(content))
	record, err := validateImportRecord(importRecord{
		Name:     firstNonEmpty(parsed.Hostname, "RDP Connection"),
		Type:     "RDP",
		Host:     parsed.Hostname,
		Port:     parsed.Port,
		Username: parsed.Username,
	})
	if err != nil {
		return importResult{
			Failed: 1,
			Errors: []importResultError{{Row: intPtr(1), Filename: filename, Error: err.Error()}},
		}, nil
	}

	result := importResult{Errors: make([]importResultError, 0)}
	if err := s.importOne(ctx, claims, record, duplicateStrategy, ip); err != nil {
		var reqErr *requestError
		switch {
		case errors.As(err, &reqErr) && reqErr.status == http.StatusConflict:
			result.Skipped = 1
		default:
			result.Failed = 1
			result.Errors = append(result.Errors, importResultError{Row: intPtr(1), Filename: filename, Error: err.Error()})
		}
	} else {
		result.Imported = 1
	}

	if err := s.insertAuditLog(ctx, claims.UserID, "IMPORT_CONNECTIONS", "", map[string]any{
		"format":   "RDP",
		"imported": result.Imported,
		"skipped":  result.Skipped,
		"failed":   result.Failed,
	}, ip); err != nil {
		return importResult{}, err
	}
	return result, nil
}

func (s Service) importOne(ctx context.Context, claims authn.Claims, record importRecord, duplicateStrategy string, ip *string) error {
	var folderID *string
	if record.FolderName != nil && strings.TrimSpace(*record.FolderName) != "" {
		resolvedFolderID, err := s.findOrCreateFolder(ctx, claims.UserID, *record.FolderName)
		if err != nil {
			return err
		}
		folderID = &resolvedFolderID
	}

	if duplicateStrategy == "SKIP" {
		exists, err := s.checkDuplicate(ctx, claims.UserID, record.Host, record.Port, record.Type)
		if err != nil {
			return err
		}
		if exists {
			return &requestError{status: http.StatusConflict, message: "duplicate skipped"}
		}
	}

	if duplicateStrategy == "RENAME" {
		exists, err := s.checkDuplicate(ctx, claims.UserID, record.Host, record.Port, record.Type)
		if err != nil {
			return err
		}
		if exists {
			base := record.Name
			counter := 1
			for {
				nextName := fmt.Sprintf("%s (%d)", base, counter)
				used, err := s.checkDuplicateByName(ctx, claims.UserID, nextName)
				if err != nil {
					return err
				}
				if !used {
					record.Name = nextName
					break
				}
				counter++
			}
		}
	}

	if strings.TrimSpace(record.Username) == "" || record.Password == "" {
		return &requestError{status: http.StatusBadRequest, message: "username and password are required"}
	}

	if s.Connections == nil {
		return fmt.Errorf("connections service is not configured")
	}

	_, err := s.Connections.ImportSimpleConnection(ctx, claims, connections.ImportPayload{
		Name:        record.Name,
		Type:        record.Type,
		Host:        record.Host,
		Port:        record.Port,
		Username:    record.Username,
		Password:    record.Password,
		Domain:      record.Domain,
		FolderID:    folderID,
		Description: record.Description,
	}, ip)
	return err
}

func (s Service) findOrCreateFolder(ctx context.Context, userID, name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", &requestError{status: http.StatusBadRequest, message: "folder name is required"}
	}

	var folderID string
	err := s.DB.QueryRow(ctx, `
SELECT id
FROM "Folder"
WHERE "userId" = $1
  AND name = $2
  AND "parentId" IS NULL
LIMIT 1
`, userID, trimmed).Scan(&folderID)
	if err == nil {
		return folderID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("find folder: %w", err)
	}

	folderID = uuid.NewString()
	if _, err := s.DB.Exec(ctx, `
INSERT INTO "Folder" (id, name, "parentId", "userId", "teamId", "sortOrder", "createdAt", "updatedAt")
VALUES ($1, $2, NULL, $3, NULL, 0, NOW(), NOW())
`, folderID, trimmed, userID); err != nil {
		return "", fmt.Errorf("create folder: %w", err)
	}
	return folderID, nil
}

func (s Service) checkDuplicate(ctx context.Context, userID, host string, port int, connectionType string) (bool, error) {
	var exists bool
	if err := s.DB.QueryRow(ctx, `
SELECT EXISTS(
	SELECT 1 FROM "Connection"
	WHERE "userId" = $1
	  AND host = $2
	  AND port = $3
	  AND type = $4::"ConnectionType"
)`, userID, host, port, strings.ToUpper(connectionType)).Scan(&exists); err != nil {
		return false, fmt.Errorf("check duplicate connection: %w", err)
	}
	return exists, nil
}

func (s Service) checkDuplicateByName(ctx context.Context, userID, name string) (bool, error) {
	var exists bool
	if err := s.DB.QueryRow(ctx, `
SELECT EXISTS(
	SELECT 1 FROM "Connection"
	WHERE "userId" = $1
	  AND name = $2
)`, userID, name).Scan(&exists); err != nil {
		return false, fmt.Errorf("check duplicate connection name: %w", err)
	}
	return exists, nil
}
