package recordingsapi

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func parseListQuery(r *http.Request) (listQuery, error) {
	query := listQuery{Limit: 50, Offset: 0}
	if value := strings.TrimSpace(r.URL.Query().Get("connectionId")); value != "" {
		if _, err := uuid.Parse(value); err != nil {
			return listQuery{}, &requestError{status: http.StatusBadRequest, message: "connectionId must be a valid UUID"}
		}
		query.ConnectionID = &value
	}
	if value := strings.TrimSpace(r.URL.Query().Get("protocol")); value != "" {
		switch value {
		case "SSH", "RDP", "VNC":
		default:
			return listQuery{}, &requestError{status: http.StatusBadRequest, message: "protocol must be SSH, RDP, or VNC"}
		}
		query.Protocol = &value
	}
	if value := strings.TrimSpace(r.URL.Query().Get("status")); value != "" {
		switch value {
		case "RECORDING", "COMPLETE", "ERROR":
		default:
			return listQuery{}, &requestError{status: http.StatusBadRequest, message: "status must be RECORDING, COMPLETE, or ERROR"}
		}
		query.Status = &value
	}
	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			return listQuery{}, &requestError{status: http.StatusBadRequest, message: "limit must be between 1 and 100"}
		}
		query.Limit = parsed
	}
	if value := strings.TrimSpace(r.URL.Query().Get("offset")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return listQuery{}, &requestError{status: http.StatusBadRequest, message: "offset must be 0 or greater"}
		}
		query.Offset = parsed
	}
	return query, nil
}

func scanRecording(row interface{ Scan(dest ...any) error }, includeUser bool) (recordingResponse, error) {
	var (
		item        recordingResponse
		sessionID   sql.NullString
		fileSize    sql.NullInt32
		duration    sql.NullInt32
		width       sql.NullInt32
		height      sql.NullInt32
		completedAt sql.NullTime
		user        recordingUser
		username    sql.NullString
	)
	dest := []any{
		&item.ID,
		&sessionID,
		&item.UserID,
		&item.ConnectionID,
		&item.Protocol,
		&item.FilePath,
		&fileSize,
		&duration,
		&width,
		&height,
		&item.Format,
		&item.Status,
		&item.CreatedAt,
		&completedAt,
		&item.Connection.ID,
		&item.Connection.Name,
		&item.Connection.Type,
		&item.Connection.Host,
	}
	if includeUser {
		dest = append(dest, &user.ID, &user.Email, &username)
	}
	if err := row.Scan(dest...); err != nil {
		return recordingResponse{}, err
	}
	if sessionID.Valid {
		item.SessionID = &sessionID.String
	}
	if fileSize.Valid {
		value := int(fileSize.Int32)
		item.FileSize = &value
	}
	if duration.Valid {
		value := int(duration.Int32)
		item.Duration = &value
	}
	if width.Valid {
		value := int(width.Int32)
		item.Width = &value
	}
	if height.Valid {
		value := int(height.Int32)
		item.Height = &value
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	if includeUser {
		if username.Valid {
			user.Username = &username.String
		}
		item.User = &user
	}
	return item, nil
}

func requestIP(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Real-IP")); value != "" {
		return value
	}
	if value := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); value != "" {
		parts := strings.Split(value, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return strings.TrimSpace(r.RemoteAddr)
}
