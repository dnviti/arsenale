package importexportapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/authn"
)

func (s Service) ExportConnections(ctx context.Context, claims authn.Claims, payload exportPayload, ip *string) (string, string, []byte, error) {
	format := strings.ToUpper(strings.TrimSpace(payload.Format))
	if format != "CSV" && format != "JSON" {
		return "", "", nil, &requestError{status: http.StatusBadRequest, message: "format must be CSV or JSON"}
	}

	items, err := s.loadExportConnections(ctx, claims.UserID, payload.ConnectionIDs, normalizeStringPtr(payload.FolderID))
	if err != nil {
		return "", "", nil, err
	}

	if payload.IncludeCredentials {
		key, err := s.getVaultKey(ctx, claims.UserID)
		if err != nil {
			return "", "", nil, err
		}
		if len(key) == 0 {
			return "", "", nil, &requestError{status: http.StatusForbidden, message: "Vault is locked. Cannot export credentials."}
		}
		defer zeroBytes(key)
		for i := range items {
			items[i].Username = decryptNullableField(key, items[i].EncryptedUsername)
			items[i].Password = decryptNullableField(key, items[i].EncryptedPassword)
			items[i].Domain = decryptNullableField(key, items[i].EncryptedDomain)
		}
	} else {
		for i := range items {
			items[i].Username = nil
			items[i].Password = nil
			items[i].Domain = nil
		}
	}

	if err := s.insertAuditLog(ctx, claims.UserID, "EXPORT_CONNECTIONS", "", map[string]any{
		"format":             format,
		"includeCredentials": payload.IncludeCredentials,
		"connectionCount":    len(items),
	}, ip); err != nil {
		return "", "", nil, err
	}

	switch format {
	case "JSON":
		body, err := json.MarshalIndent(map[string]any{
			"version":    "1.0",
			"exportedAt": time.Now().UTC().Format(time.RFC3339),
			"count":      len(items),
			"connections": func() []exportConnection {
				exported := make([]exportConnection, len(items))
				for i, item := range items {
					exported[i] = item.toExportConnection()
				}
				return exported
			}(),
		}, "", "  ")
		if err != nil {
			return "", "", nil, fmt.Errorf("marshal json export: %w", err)
		}
		filename := fmt.Sprintf("arsenale-connections-%s.json", time.Now().UTC().Format("2006-01-02"))
		return filename, "application/json", body, nil
	default:
		body, err := buildCSV(items)
		if err != nil {
			return "", "", nil, err
		}
		filename := fmt.Sprintf("connections-export-%s.csv", time.Now().UTC().Format("2006-01-02T15-04-05Z"))
		return filename, "text/csv", body, nil
	}
}

func (s Service) loadExportConnections(ctx context.Context, userID string, connectionIDs []string, folderID *string) ([]rawConnectionRow, error) {
	args := []any{userID}
	conditions := []string{`c."userId" = $1`}
	if folderID != nil {
		args = append(args, *folderID)
		conditions = append(conditions, fmt.Sprintf(`c."folderId" = $%d`, len(args)))
	}
	if len(connectionIDs) > 0 {
		args = append(args, connectionIDs)
		conditions = append(conditions, fmt.Sprintf(`c.id = ANY($%d)`, len(args)))
	}

	query := fmt.Sprintf(`
SELECT
	c.id,
	c.name,
	c.type::text,
	c.host,
	c.port,
	c.description,
	c."isFavorite",
	c."enableDrive",
	c."folderId",
	f.name,
	c."sshTerminalConfig",
	c."rdpSettings",
	c."vncSettings",
	c."defaultCredentialMode",
	c."createdAt",
	c."updatedAt",
	c."encryptedUsername",
	c."usernameIV",
	c."usernameTag",
	c."encryptedPassword",
	c."passwordIV",
	c."passwordTag",
	c."encryptedDomain",
	c."domainIV",
	c."domainTag"
FROM "Connection" c
LEFT JOIN "Folder" f ON f.id = c."folderId"
WHERE %s
ORDER BY c.name ASC
`, strings.Join(conditions, " AND "))

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query export connections: %w", err)
	}
	defer rows.Close()

	result := make([]rawConnectionRow, 0)
	for rows.Next() {
		item, err := scanExportRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate export connections: %w", err)
	}
	return result, nil
}

func buildCSV(items []rawConnectionRow) ([]byte, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)
	headers := []string{"Name", "Type", "Host", "Port", "Description", "Folder", "Username", "Password", "Domain", "IsFavorite", "EnableDrive", "CreatedAt", "UpdatedAt"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}
	for _, item := range items {
		record := []string{
			item.Name,
			item.Type,
			item.Host,
			strconv.Itoa(item.Port),
			stringOrEmpty(item.Description),
			stringOrEmpty(item.FolderName),
			stringOrEmpty(item.Username),
			stringOrEmpty(item.Password),
			stringOrEmpty(item.Domain),
			strconv.FormatBool(item.IsFavorite),
			strconv.FormatBool(item.EnableDrive),
			item.CreatedAt.Format(time.RFC3339),
			item.UpdatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush csv: %w", err)
	}
	return buffer.Bytes(), nil
}

func scanExportRow(row scanRow) (rawConnectionRow, error) {
	var (
		item                rawConnectionRow
		description, folder sql.NullString
		defaultCredential   sql.NullString
		encUser, userIV     sql.NullString
		userTag             sql.NullString
		encPassword, passIV sql.NullString
		passTag             sql.NullString
		encDomain, domainIV sql.NullString
		domainTag           sql.NullString
	)
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Type,
		&item.Host,
		&item.Port,
		&description,
		&item.IsFavorite,
		&item.EnableDrive,
		new(sql.NullString),
		&folder,
		&item.SSHTerminalConfig,
		&item.RDPSettings,
		&item.VNCSettings,
		&defaultCredential,
		&item.CreatedAt,
		&item.UpdatedAt,
		&encUser,
		&userIV,
		&userTag,
		&encPassword,
		&passIV,
		&passTag,
		&encDomain,
		&domainIV,
		&domainTag,
	); err != nil {
		return rawConnectionRow{}, fmt.Errorf("scan export connection: %w", err)
	}
	if description.Valid {
		item.Description = &description.String
	}
	if folder.Valid {
		item.FolderName = &folder.String
	}
	if defaultCredential.Valid {
		item.DefaultCredentialMode = &defaultCredential.String
	}
	item.EncryptedUsername = nullableEncryptedField(encUser, userIV, userTag)
	item.EncryptedPassword = nullableEncryptedField(encPassword, passIV, passTag)
	item.EncryptedDomain = nullableEncryptedField(encDomain, domainIV, domainTag)
	return item, nil
}
