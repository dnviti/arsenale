package modelgatewayapi

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func buildPlanningSystemPrompt(dbProtocol string) string {
	objectLabel := "tables"
	if isMongoDBProtocol(dbProtocol) {
		objectLabel = "collections"
	}

	return fmt.Sprintf(`You are a database query planning assistant. Given a user's request and a list of available database %s, determine which %s are needed to write the query.

Return ONLY valid JSON with no markdown fences:
{"tables": [{"name": "table_name", "schema": "schema_name", "reason": "brief reason this table is needed"}]}

Rules:
- Only include %s that are genuinely needed to answer the user's request.
- Do not invent %s that are not in the provided list.
- Include relationship or lookup %s if the query requires them.
- Keep reasons concise (one sentence).`, objectLabel, objectLabel, objectLabel, objectLabel, objectLabel)
}

func formatTableList(tables []contracts.SchemaTable, dbProtocol string) string {
	objectLabel := "tables"
	if isMongoDBProtocol(dbProtocol) {
		objectLabel = "collections"
	}
	if len(tables) == 0 {
		return "No " + objectLabel + " available."
	}
	lines := []string{"Available " + objectLabel + ":"}
	limit := len(tables)
	if limit > 100 {
		limit = 100
	}
	for _, table := range tables[:limit] {
		displayName := table.Name
		if trimmed := strings.TrimSpace(table.Schema); trimmed != "" && trimmed != "public" {
			if isMongoDBProtocol(dbProtocol) {
				displayName = table.Name + " (database " + trimmed + ")"
			} else {
				displayName = trimmed + "." + table.Name
			}
		}
		columns := make([]string, 0, len(table.Columns))
		for _, column := range table.Columns {
			columns = append(columns, column.Name)
		}
		lines = append(lines, "- "+displayName+" ("+strings.Join(columns, ", ")+")")
	}
	return strings.Join(lines, "\n")
}

func parsePlanningResponse(raw string) []objectRequest {
	extracted, err := extractJSON(raw)
	if err != nil {
		return nil
	}
	root, ok := extracted.(map[string]any)
	if !ok {
		return nil
	}
	items, ok := root["tables"].([]any)
	if !ok {
		return nil
	}
	result := make([]objectRequest, 0, len(items))
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, _ := record["name"].(string)
		if strings.TrimSpace(name) == "" {
			continue
		}
		schema, _ := record["schema"].(string)
		reason, _ := record["reason"].(string)
		result = append(result, objectRequest{
			Name:   normalizePlanningIdentifier(name),
			Schema: normalizePlanningIdentifier(schema),
			Reason: strings.TrimSpace(reason),
		})
	}
	return result
}

func buildGenerationSystemPrompt(dbProtocol string) string {
	if isMongoDBProtocol(dbProtocol) {
		return `You are a MongoDB query assistant. You generate Arsenale MongoDB JSON query specs from natural-language requests.

CRITICAL CONSTRAINT:
You may ONLY reference collections that appear in the schema below. The user has explicitly approved only these collections. You MUST NOT reference any other collection.

Return ONLY valid JSON with two fields:
{
  "query": {
    "collection": "collection_name",
    "operation": "find|aggregate|count|distinct|runCommand",
    "...": "other supported fields"
  },
  "explanation": "brief explanation"
}

Rules:
1. Only generate read-only MongoDB operations: find, aggregate, count, distinct, or runCommand.
2. ALWAYS include an explicit "operation" field.
3. Set "collection" to the bare collection name only. Do NOT prefix it with database or schema names like "arsenale_demo.demo_customers".
4. Only use a separate "database" field when you intentionally need a different database; otherwise omit it.
5. Use ONLY collection and field names from the provided schema.
6. Never return shell syntax, JavaScript, db.collection.find(...), or SQL.
7. For simple retrievals, prefer "find". Use "aggregate" only when grouping, joining-like lookup, or computed totals are needed.
8. For "find", include a reasonable "limit" when the user did not specify one.
9. For "distinct", include both "collection" and "field".
10. For "aggregate", include both "collection" and "pipeline".
11. For "runCommand", include a "command" object.
12. If the approved collections are insufficient, write the best read-only query you can using ONLY the approved collections and explain the limitation.`
	}

	dialect := strings.ToUpper(strings.TrimSpace(dbProtocol))
	if dialect == "" {
		dialect = "POSTGRESQL"
	}
	return fmt.Sprintf(`You are a SQL query assistant. You generate SQL queries from natural language descriptions.

CRITICAL CONSTRAINT:
You may ONLY reference tables that appear in the schema below. The user has explicitly approved only these tables. You MUST NOT reference, join, subquery, or otherwise use ANY table not listed in the schema. If the approved tables are insufficient to fully answer the request, write the best query you can using ONLY the approved tables and explain the limitation.

RULES:
1. ONLY generate SELECT queries. NEVER generate INSERT, UPDATE, DELETE, DROP, ALTER, CREATE, TRUNCATE, or any DML/DDL statements.
2. Use the correct SQL dialect for %s.
3. ONLY use table and column names from the provided schema — do not reference any other tables.
4. If the request is ambiguous, make reasonable assumptions and explain them.
5. When the user does not specify a limit, add a reasonable limit on the number of returned rows. Use the appropriate limiting syntax for the %s dialect (for example, LIMIT for PostgreSQL/MySQL, TOP for MSSQL, FETCH FIRST for DB2/Oracle).
6. Use table aliases for readability.
7. Return your response as a JSON object with two fields:
   - "sql": the generated SELECT query (using ONLY approved tables)
   - "explanation": a brief explanation of what the query does and any assumptions made

Example response:
{"sql": "SELECT o.id, o.total FROM orders o WHERE o.total > 1000", "explanation": "Retrieves orders where the total is greater than 1000."}`, dialect, dialect)
}

func formatSchemaContext(tables []contracts.SchemaTable, dbProtocol string) string {
	if len(tables) == 0 {
		return "No schema information available."
	}
	lines := []string{"Database type: " + dbProtocol, "", "Schema:"}
	objectLabel := "TABLE"
	if isMongoDBProtocol(dbProtocol) {
		objectLabel = "COLLECTION"
	}
	limit := len(tables)
	if limit > 50 {
		limit = 50
	}
	for _, table := range tables[:limit] {
		displayName := table.Name
		if trimmed := strings.TrimSpace(table.Schema); trimmed != "" && trimmed != "public" {
			if isMongoDBProtocol(dbProtocol) {
				displayName = table.Name + " (database " + trimmed + ")"
			} else {
				displayName = trimmed + "." + table.Name
			}
		}
		lines = append(lines, "", objectLabel+" "+displayName+":")
		for _, column := range table.Columns {
			nullable := " NOT NULL"
			if column.Nullable {
				nullable = " NULL"
			}
			pk := ""
			if column.IsPrimaryKey {
				pk = " PK"
			}
			lines = append(lines, "  "+column.Name+" "+column.DataType+nullable+pk)
		}
	}
	return strings.Join(lines, "\n")
}

type generationResponse struct {
	SQL         string
	Explanation string
}

func parseGenerationResponse(raw string) generationResponse {
	extracted, err := extractJSON(raw)
	if err == nil {
		if record, ok := extracted.(map[string]any); ok {
			if queryText := extractQueryTextValue(record["sql"]); queryText != "" {
				explanation, _ := record["explanation"].(string)
				return generationResponse{
					SQL:         strings.TrimSpace(queryText),
					Explanation: strings.TrimSpace(explanation),
				}
			}
			if queryText := extractQueryTextValue(record["query"]); queryText != "" {
				explanation, _ := record["explanation"].(string)
				return generationResponse{
					SQL:         strings.TrimSpace(queryText),
					Explanation: strings.TrimSpace(explanation),
				}
			}
			if queryText := extractQueryTextValue(record["query_spec"]); queryText != "" {
				explanation, _ := record["explanation"].(string)
				return generationResponse{
					SQL:         strings.TrimSpace(queryText),
					Explanation: strings.TrimSpace(explanation),
				}
			}
		}
	}

	blockPatterns := []*regexp.Regexp{
		regexp.MustCompile("(?is)```sql\\s*(.*?)```"),
		regexp.MustCompile("(?is)```\\s*(.*?)```"),
	}
	for _, pattern := range blockPatterns {
		match := pattern.FindStringSubmatch(raw)
		if len(match) == 2 {
			return generationResponse{SQL: strings.TrimSpace(match[1])}
		}
	}
	return generationResponse{SQL: strings.TrimSpace(raw)}
}

func extractQueryTextValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		payload, err := json.MarshalIndent(typed, "", "  ")
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(payload))
	case []any:
		payload, err := json.MarshalIndent(typed, "", "  ")
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(payload))
	default:
		return ""
	}
}

func findUnapprovedTableReference(sqlText string, approvedTables, allTables []contracts.SchemaTable) string {
	approved := make(map[string]struct{}, len(approvedTables)*2)
	for _, table := range approvedTables {
		approved[strings.ToLower(table.Name)] = struct{}{}
		approved[strings.ToLower(table.Schema+"."+table.Name)] = struct{}{}
	}

	lowered := strings.ToLower(sqlText)
	for _, table := range allTables {
		unqualified := strings.ToLower(table.Name)
		qualified := strings.ToLower(table.Schema + "." + table.Name)
		if _, ok := approved[unqualified]; ok {
			continue
		}
		if _, ok := approved[qualified]; ok {
			continue
		}
		if regexp.MustCompile(`\b` + regexp.QuoteMeta(unqualified) + `\b`).MatchString(lowered) {
			if table.Schema != "" && table.Schema != "public" {
				return table.Schema + "." + table.Name
			}
			return table.Name
		}
		if regexp.MustCompile(`\b` + regexp.QuoteMeta(qualified) + `\b`).MatchString(lowered) {
			return table.Schema + "." + table.Name
		}
	}
	return ""
}

func collectDeniedTables(filteredSchema, fullSchema []contracts.SchemaTable) []string {
	allowed := make(map[string]struct{}, len(filteredSchema))
	for _, table := range filteredSchema {
		allowed[strings.ToLower(table.Schema+"."+table.Name)] = struct{}{}
	}
	var denied []string
	for _, table := range fullSchema {
		key := strings.ToLower(table.Schema + "." + table.Name)
		if _, ok := allowed[key]; ok {
			continue
		}
		if table.Schema != "" && table.Schema != "public" {
			denied = append(denied, table.Schema+"."+table.Name)
		} else {
			denied = append(denied, table.Name)
		}
	}
	return denied
}

func pruneGenerationLocked(state *aiState, now time.Time) {
	for id, conversation := range state.generationConversations {
		if now.Sub(conversation.CreatedAt) > generationConversationTTL {
			delete(state.generationConversations, id)
		}
	}
}

func pruneOptimizationLocked(state *aiState, now time.Time) {
	for id, conversation := range state.optimizationSessions {
		if now.Sub(conversation.CreatedAt) > optimizationConversationTTL {
			delete(state.optimizationSessions, id)
		}
	}
}

func cloneSchemaTables(tables []contracts.SchemaTable) []contracts.SchemaTable {
	cloned := make([]contracts.SchemaTable, 0, len(tables))
	for _, table := range tables {
		item := table
		if len(table.Columns) > 0 {
			item.Columns = append([]contracts.SchemaColumn(nil), table.Columns...)
		}
		cloned = append(cloned, item)
	}
	return cloned
}
