package modelgatewayapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/queryrunner"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func normalizePlanningIdentifier(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`\"'")
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	return value
}

func splitQualifiedPlanningName(name string) (string, string) {
	name = normalizePlanningIdentifier(name)
	if !strings.Contains(name, ".") {
		return "", name
	}
	parts := strings.SplitN(name, ".", 2)
	return normalizePlanningIdentifier(parts[0]), normalizePlanningIdentifier(parts[1])
}

func resolvePlanningRequests(requests []objectRequest, schema []contracts.SchemaTable) []objectRequest {
	qualified := make(map[string]contracts.SchemaTable, len(schema))
	byName := make(map[string][]contracts.SchemaTable, len(schema))
	for _, table := range schema {
		normalizedSchema := normalizePlanningIdentifier(table.Schema)
		normalizedName := normalizePlanningIdentifier(table.Name)
		qualified[strings.ToLower(normalizedSchema+"."+normalizedName)] = table
		byName[strings.ToLower(normalizedName)] = append(byName[strings.ToLower(normalizedName)], table)
	}

	resolved := make([]objectRequest, 0, len(requests))
	seen := make(map[string]struct{}, len(requests))
	appendResolved := func(table contracts.SchemaTable, reason string) {
		key := strings.ToLower(normalizePlanningIdentifier(table.Schema) + "." + normalizePlanningIdentifier(table.Name))
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		resolved = append(resolved, objectRequest{
			Name:   strings.TrimSpace(table.Name),
			Schema: strings.TrimSpace(table.Schema),
			Reason: strings.TrimSpace(reason),
		})
	}

	for _, item := range requests {
		name := normalizePlanningIdentifier(item.Name)
		schemaName := normalizePlanningIdentifier(item.Schema)
		if name == "" {
			continue
		}
		if schemaName == "" && strings.Contains(name, ".") {
			schemaName, name = splitQualifiedPlanningName(name)
		}
		if schemaName != "" {
			if table, ok := qualified[strings.ToLower(schemaName+"."+name)]; ok {
				appendResolved(table, item.Reason)
				continue
			}
		}
		candidates := byName[strings.ToLower(name)]
		if len(candidates) == 1 {
			appendResolved(candidates[0], item.Reason)
			continue
		}
		if table, ok := fuzzyResolvePlanningTable(name, schemaName, schema); ok {
			appendResolved(table, item.Reason)
		}
	}

	return resolved
}

func fuzzyResolvePlanningTable(name, schemaName string, schema []contracts.SchemaTable) (contracts.SchemaTable, bool) {
	requestTokens := tokenizePlanningText(strings.TrimSpace(schemaName + " " + name))
	if len(requestTokens) == 0 {
		return contracts.SchemaTable{}, false
	}

	bestScore := 0
	bestIndex := -1
	ambiguous := false
	for idx, table := range schema {
		score := scorePlanningTableTokens(requestTokens, table)
		if schemaName != "" && strings.EqualFold(strings.TrimSpace(table.Schema), strings.TrimSpace(schemaName)) {
			score += 2
		}
		if score <= 0 {
			continue
		}
		if score > bestScore {
			bestScore = score
			bestIndex = idx
			ambiguous = false
			continue
		}
		if score == bestScore {
			ambiguous = true
		}
	}

	if bestIndex < 0 || ambiguous {
		return contracts.SchemaTable{}, false
	}
	return schema[bestIndex], true
}

func heuristicPlanningFallback(prompt string, schema []contracts.SchemaTable) []objectRequest {
	promptTokens := tokenizePlanningText(prompt)
	if len(promptTokens) == 0 {
		return nil
	}

	type candidate struct {
		table contracts.SchemaTable
		score int
	}

	candidates := make([]candidate, 0, len(schema))
	for _, table := range schema {
		score := scorePlanningTableTokens(promptTokens, table)
		if score <= 0 {
			continue
		}
		candidates = append(candidates, candidate{table: table, score: score})
	}
	if len(candidates) == 0 {
		return nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			left := strings.ToLower(candidates[i].table.Schema + "." + candidates[i].table.Name)
			right := strings.ToLower(candidates[j].table.Schema + "." + candidates[j].table.Name)
			return left < right
		}
		return candidates[i].score > candidates[j].score
	})

	limit := len(candidates)
	if limit > 5 {
		limit = 5
	}

	resolved := make([]objectRequest, 0, limit)
	for _, item := range candidates[:limit] {
		resolved = append(resolved, objectRequest{
			Name:   strings.TrimSpace(item.table.Name),
			Schema: strings.TrimSpace(item.table.Schema),
			Reason: "Matched prompt keywords heuristically after AI planning returned no direct table match.",
		})
	}
	return resolved
}

func tokenizePlanningText(value string) map[string]struct{} {
	parts := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	if len(parts) == 0 {
		return nil
	}

	stopwords := map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "all": {}, "by": {}, "for": {}, "from": {},
		"get": {}, "give": {}, "in": {}, "list": {}, "me": {}, "of": {}, "on": {},
		"show": {}, "the": {}, "to": {}, "top": {}, "with": {},
	}

	tokens := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) < 3 {
			continue
		}
		if _, skip := stopwords[part]; skip {
			continue
		}
		tokens[part] = struct{}{}
		if singular := strings.TrimSuffix(part, "s"); singular != part && len(singular) >= 3 {
			tokens[singular] = struct{}{}
		}
	}
	return tokens
}

func scorePlanningTableTokens(promptTokens map[string]struct{}, table contracts.SchemaTable) int {
	if len(promptTokens) == 0 {
		return 0
	}

	tableTokens := tokenizePlanningText(table.Schema + " " + table.Name)
	score := 0
	for token := range tableTokens {
		if isPlanningNoiseToken(token) {
			continue
		}
		if _, ok := promptTokens[token]; ok {
			score += 3
		}
	}
	for _, column := range table.Columns {
		columnTokens := tokenizePlanningText(column.Name)
		for token := range columnTokens {
			if isPlanningNoiseToken(token) {
				continue
			}
			if _, ok := promptTokens[token]; ok {
				score++
			}
		}
	}
	return score
}

func isPlanningNoiseToken(token string) bool {
	switch token {
	case "all", "arsenale", "data", "dbo", "demo", "field", "public", "record", "table", "value":
		return true
	default:
		return false
	}
}

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

func normalizeQueryForProtocol(dbProtocol, queryText string, readOnly bool) (string, error) {
	queryText = strings.TrimSpace(queryText)
	if queryText == "" {
		return "", &requestError{status: http.StatusBadRequest, message: "The AI did not return a query."}
	}

	if isMongoDBProtocol(dbProtocol) {
		var (
			normalized string
			err        error
		)
		if readOnly {
			normalized, _, _, err = queryrunner.NormalizeMongoReadOnlyQueryText(queryText)
		} else {
			normalized, _, _, err = queryrunner.NormalizeMongoQueryText(queryText)
		}
		if err != nil {
			return "", &requestError{status: http.StatusBadRequest, message: "The AI generated an invalid MongoDB query spec: " + err.Error()}
		}
		return normalized, nil
	}

	if err := validateGeneratedSQL(queryText); err != nil {
		return "", err
	}
	return queryText, nil
}

func stabilizeMongoOptimizedQuery(originalQuery, optimizedQuery string) (string, bool, error) {
	originalNormalized, err := normalizeQueryForProtocol("mongodb", originalQuery, true)
	if err != nil {
		return optimizedQuery, false, err
	}
	optimizedNormalized, err := normalizeQueryForProtocol("mongodb", optimizedQuery, true)
	if err != nil {
		return optimizedQuery, false, err
	}

	var originalSpec map[string]any
	if err := json.Unmarshal([]byte(originalNormalized), &originalSpec); err != nil {
		return optimizedNormalized, false, err
	}
	var optimizedSpec map[string]any
	if err := json.Unmarshal([]byte(optimizedNormalized), &optimizedSpec); err != nil {
		return optimizedNormalized, false, err
	}

	originalOp := normalizeMongoOptimizationOperation(originalSpec["operation"])
	optimizedOp := normalizeMongoOptimizationOperation(optimizedSpec["operation"])
	if originalOp == "" {
		return optimizedNormalized, false, nil
	}

	changed := false
	if optimizedOp == "" {
		optimizedSpec["operation"] = originalSpec["operation"]
		optimizedOp = originalOp
		changed = true
	}
	if optimizedOp != originalOp {
		return originalNormalized, true, nil
	}

	changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "database") || changed
	changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "collection") || changed

	switch originalOp {
	case "find":
		changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "filter") || changed
		changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "projection") || changed
		changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "sort") || changed
		changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "limit") || changed
		changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "skip") || changed
	case "count":
		changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "filter") || changed
	case "distinct":
		changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "filter") || changed
		changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "field") || changed
	case "aggregate":
		changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "pipeline") || changed
	case "runcommand":
		changed = copyMongoValueIfMissing(optimizedSpec, originalSpec, "command") || changed
	}

	payload, err := json.MarshalIndent(optimizedSpec, "", "  ")
	if err != nil {
		return optimizedNormalized, changed, err
	}
	stabilized, err := normalizeQueryForProtocol("mongodb", string(payload), true)
	if err != nil {
		return optimizedNormalized, changed, err
	}
	return stabilized, changed, nil
}

func normalizeMongoOptimizationOperation(value any) string {
	text, _ := value.(string)
	text = strings.ToLower(strings.TrimSpace(text))
	text = strings.ReplaceAll(text, "_", "")
	text = strings.ReplaceAll(text, "-", "")
	switch text {
	case "countdocument", "countdocuments", "estimateddocumentcount":
		return "count"
	case "runcmd":
		return "runcommand"
	default:
		return text
	}
}

func copyMongoValueIfMissing(dst, src map[string]any, key string) bool {
	if !mongoValueMissing(dst[key]) {
		return false
	}
	value, ok := src[key]
	if !ok || mongoValueMissing(value) {
		return false
	}
	dst[key] = value
	return true
}

func mongoValueMissing(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	case float64:
		return typed == 0
	case int:
		return typed == 0
	case int32:
		return typed == 0
	case int64:
		return typed == 0
	default:
		return false
	}
}

func appendMongoSemanticsNote(explanation string) string {
	note := "Missing MongoDB filter/sort/limit fields were preserved from the original query to keep the same result set semantics."
	explanation = strings.TrimSpace(explanation)
	if explanation == "" {
		return note
	}
	if strings.Contains(explanation, note) {
		return explanation
	}
	return explanation + " " + note
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
