package modelgatewayapi

import (
	"encoding/json"
	"strings"
)

func buildOptimizationSystemPrompt(dbProtocol string) string {
	if isMongoDBProtocol(dbProtocol) {
		return `You are an expert MongoDB query analyst and optimizer.
Your task is to analyze Arsenale MongoDB JSON query specs and produce optimized read-only versions.

You work in a multi-turn flow:
1. FIRST TURN: You receive a MongoDB query spec and optional metadata. Analyze it and request specific collection metadata you need (indexes, statistics, table_schema, row_count). Respond ONLY with a JSON object.
2. SECOND TURN: You receive the requested metadata. Produce the optimized query with explanation. Respond ONLY with a JSON object.

FIRST TURN response format (when you need additional data):
{
  "needs_data": true,
  "data_requests": [
    { "type": "indexes|statistics|table_schema|row_count", "target": "collection_name", "reason": "brief reason" }
  ]
}

FIRST TURN response format (when you can optimize immediately):
{
  "needs_data": false,
  "optimized_query": {
    "collection": "collection_name",
    "operation": "find|aggregate|count|distinct|runCommand"
  },
  "explanation": "Explanation of changes...",
  "changes": ["change 1", "change 2"]
}

SECOND TURN response format:
{
  "optimized_query": {
    "collection": "collection_name",
    "operation": "find|aggregate|count|distinct|runCommand"
  },
  "explanation": "Explanation of changes...",
  "changes": ["change 1", "change 2"]
}

Rules:
- Only suggest read-only MongoDB operations.
- ALWAYS include an explicit "operation" field in the optimized query.
- Use the bare collection name in "collection"; do not prefix it with database or schema names like "arsenale_demo.demo_customers".
- Only use a separate "database" field when it is genuinely required.
- Never change the query semantics.
- Prefer the simplest valid query shape for the requested result.
- If the query is already optimal, return the original query unchanged and explain why.
- Respond ONLY with valid JSON, no markdown fences or extra text.`
	}

	return `You are an expert SQL performance analyst and query optimizer.
Your task is to analyze SQL queries and their execution plans, then produce optimized versions.

You work in a multi-turn flow:
1. FIRST TURN: You receive a SQL query and execution plan. Analyze them and request specific database metadata you need (indexes, statistics, foreign keys). Respond ONLY with a JSON object.
2. SECOND TURN: You receive the requested metadata. Produce the optimized query with explanation. Respond ONLY with a JSON object.

FIRST TURN response format (when you need additional data):
{
  "needs_data": true,
  "data_requests": [
    { "type": "indexes|statistics|foreign_keys", "target": "table_name", "reason": "brief reason" }
  ]
}

FIRST TURN response format (when you can optimize immediately):
{
  "needs_data": false,
  "optimized_sql": "SELECT ...",
  "explanation": "Explanation of changes...",
  "changes": ["change 1", "change 2"]
}

SECOND TURN response format:
{
  "optimized_sql": "SELECT ...",
  "explanation": "Explanation of changes...",
  "changes": ["change 1", "change 2"]
}

Rules:
- Only suggest changes you are confident will improve performance.
- If the query is already optimal, set optimized_sql to the original query and explain why.
- Never suggest changes that alter query semantics (same results, same ordering).
- Consider the specific database engine and version provided.
- Be specific in your explanations (mention index names, cardinality, join strategies).
- Respond ONLY with valid JSON, no markdown fences or extra text.`
}

func buildFirstTurnMessage(input optimizeQueryInput) string {
	parts := []string{"Database: " + input.DBProtocol}
	if strings.TrimSpace(input.DBVersion) != "" {
		parts[0] += " " + strings.TrimSpace(input.DBVersion)
	}
	queryLabel := "SQL Query"
	if isMongoDBProtocol(input.DBProtocol) {
		queryLabel = "MongoDB Query Spec"
	}
	parts = append(parts, "", queryLabel+":", input.SQL)
	if input.ExecutionPlan != nil {
		planLabel := "Execution Plan"
		if isMongoDBProtocol(input.DBProtocol) {
			planLabel = "Query Plan"
		}
		plan := stringifyLLMContext(input.ExecutionPlan)
		if len(plan) > 50000 {
			plan = plan[:50000] + "\n[truncated]"
		}
		parts = append(parts, "", planLabel+":", plan)
	}
	if input.SchemaContext != nil {
		parts = append(parts, "", "Schema Context:", stringifyLLMContext(input.SchemaContext))
	}
	return strings.Join(parts, "\n")
}

func buildSecondTurnMessage(approvedData map[string]any) string {
	return "Here is the database metadata you requested:\n\n" + stringifyLLMContext(approvedData) + "\n\nBased on this data, produce the optimized query."
}

func stringifyLLMContext(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		payload, _ := json.MarshalIndent(value, "", "  ")
		return string(payload)
	}
}
