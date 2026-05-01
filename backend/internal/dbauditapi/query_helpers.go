package dbauditapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func parseDBAuditQuery(r *http.Request) (dbAuditQuery, error) {
	query := dbAuditQuery{
		Page:      1,
		Limit:     50,
		SortBy:    "createdAt",
		SortOrder: "desc",
	}
	values := r.URL.Query()
	if raw := strings.TrimSpace(values.Get("page")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			return dbAuditQuery{}, fmt.Errorf("page must be a positive integer")
		}
		query.Page = value
	}
	if raw := strings.TrimSpace(values.Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 100 {
			return dbAuditQuery{}, fmt.Errorf("limit must be between 1 and 100")
		}
		query.Limit = value
	}
	query.UserID = strings.TrimSpace(values.Get("userId"))
	query.ConnectionID = strings.TrimSpace(values.Get("connectionId"))
	query.Search = strings.TrimSpace(values.Get("search"))
	if raw := strings.TrimSpace(values.Get("queryType")); raw != "" {
		raw = strings.ToUpper(raw)
		switch raw {
		case "SELECT", "INSERT", "UPDATE", "DELETE", "DDL", "OTHER":
			query.QueryType = raw
		default:
			return dbAuditQuery{}, fmt.Errorf("queryType must be SELECT, INSERT, UPDATE, DELETE, DDL, or OTHER")
		}
	}
	if raw := strings.TrimSpace(values.Get("blocked")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return dbAuditQuery{}, fmt.Errorf("blocked must be a boolean")
		}
		query.Blocked = &value
	}
	if raw := strings.TrimSpace(values.Get("startDate")); raw != "" {
		value, err := parseAuditTime(raw)
		if err != nil {
			return dbAuditQuery{}, fmt.Errorf("invalid startDate")
		}
		query.StartDate = &value
	}
	if raw := strings.TrimSpace(values.Get("endDate")); raw != "" {
		value, err := parseAuditTime(raw)
		if err != nil {
			return dbAuditQuery{}, fmt.Errorf("invalid endDate")
		}
		query.EndDate = &value
	}
	if raw := strings.TrimSpace(values.Get("sortBy")); raw != "" {
		switch raw {
		case "createdAt", "queryType", "executionTimeMs":
			query.SortBy = raw
		default:
			return dbAuditQuery{}, fmt.Errorf("sortBy must be createdAt, queryType, or executionTimeMs")
		}
	}
	if raw := strings.TrimSpace(values.Get("sortOrder")); raw != "" {
		value := strings.ToLower(raw)
		if value != "asc" && value != "desc" {
			return dbAuditQuery{}, fmt.Errorf("sortOrder must be asc or desc")
		}
		query.SortOrder = value
	}
	return query, nil
}

func buildFilters(query dbAuditQuery, tenantID string) (string, []any) {
	clauses := []string{`l."tenantId" = $1`}
	args := []any{tenantID}
	add := func(clause string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}

	if query.UserID != "" {
		add(`l."userId" = $%d`, query.UserID)
	}
	if query.ConnectionID != "" {
		add(`l."connectionId" = $%d`, query.ConnectionID)
	}
	if query.QueryType != "" {
		add(`l."queryType" = $%d::"DbQueryType"`, query.QueryType)
	}
	if query.Blocked != nil {
		add(`l.blocked = $%d`, *query.Blocked)
	}
	if query.StartDate != nil {
		add(`l."createdAt" >= $%d`, *query.StartDate)
	}
	if query.EndDate != nil {
		add(`l."createdAt" <= $%d`, *query.EndDate)
	}
	if query.Search != "" {
		term := strings.TrimSpace(query.Search)
		queryText := "%" + term + "%"
		blockReason := "%" + term + "%"
		tablesValue := strings.ToLower(term)
		args = append(args, queryText, tablesValue, blockReason)
		base := len(args) - 2
		clauses = append(clauses,
			fmt.Sprintf(`(l."queryText" ILIKE $%d OR EXISTS (SELECT 1 FROM unnest(l."tablesAccessed") AS t WHERE t = $%d) OR l."blockReason" ILIKE $%d)`, base, base+1, base+2),
		)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func orderByClause(query dbAuditQuery) string {
	column := `l."createdAt"`
	switch query.SortBy {
	case "queryType":
		column = `l."queryType"`
	case "executionTimeMs":
		column = `l."executionTimeMs"`
	}
	direction := "DESC"
	if query.SortOrder == "asc" {
		direction = "ASC"
	}
	return column + " " + direction
}

func parseAuditTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, time.DateOnly} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time")
}

func totalPages(total, limit int) int {
	if total == 0 {
		return 0
	}
	return (total + limit - 1) / limit
}

func readRawUpdatePayload(r *http.Request) (map[string]json.RawMessage, error) {
	defer r.Body.Close()
	var payload map[string]json.RawMessage
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]json.RawMessage{}
	}
	return payload, nil
}
