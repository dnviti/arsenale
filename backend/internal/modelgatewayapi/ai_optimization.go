package modelgatewayapi

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/queryrunner"
	"github.com/google/uuid"
)

func (s Service) optimizeQuery(ctx context.Context, input optimizeQueryInput, userID, tenantID, ipAddress string) (optimizeQueryResult, error) {
	aiContext, err := s.DatabaseSessions.ResolveOwnedAIContext(ctx, userID, tenantID, input.SessionID)
	if err != nil {
		return optimizeQueryResult{}, err
	}
	platform, err := s.loadPlatformConfig(ctx, tenantID)
	if err != nil {
		return optimizeQueryResult{}, err
	}
	execution, err := resolveFeatureExecution(platform, aiContext, "query-optimizer")
	if err != nil {
		return optimizeQueryResult{}, err
	}
	if !execution.Enabled {
		return optimizeQueryResult{}, &requestError{
			status:  http.StatusForbidden,
			message: "AI query optimization is not enabled",
		}
	}
	overrides := llmOverridesFromExecution(execution)

	conversationID := uuid.NewString()
	messages := []llmMessage{
		{Role: "system", Content: buildOptimizationSystemPrompt(input.DBProtocol)},
		{Role: "user", Content: buildFirstTurnMessage(input)},
	}

	raw, err := s.completeLLM(ctx, llmCompletionOptions{Messages: messages}, overrides)
	var parsed firstTurnResponse
	if err != nil {
		parsed = buildHeuristicDataRequests(input)
	} else {
		parsed = parseFirstTurnResponse(raw.Content)
	}

	state := s.ensureAIState()
	now := time.Now().UTC()
	state.mu.Lock()
	pruneOptimizationLocked(state, now)
	state.optimizationSessions[conversationID] = optimizationConversation{
		ID:           conversationID,
		UserID:       userID,
		TenantID:     tenantID,
		Input:        input,
		Rounds:       0,
		ApprovedData: map[string]any{},
		Messages:     append([]llmMessage(nil), messages...),
		Overrides:    overrides,
		CreatedAt:    now,
	}
	state.mu.Unlock()

	provider, modelID := providerAndModelFromOverrides(overrides)
	_ = s.insertAuditLog(ctx, userID, "DB_QUERY_AI_OPTIMIZED", "DatabaseQuery", input.SessionID, map[string]any{
		"conversationId":   conversationID,
		"phase":            "initial",
		"provider":         provider,
		"model":            modelID,
		"dataRequestCount": len(parsed.DataRequests),
		"dataRequestTypes": collectDataRequestTypes(parsed.DataRequests),
	}, ipAddress)

	if !parsed.NeedsData {
		state.mu.Lock()
		delete(state.optimizationSessions, conversationID)
		state.mu.Unlock()
		optimizedSQL := parsed.OptimizedSQL
		explanation := parsed.Explanation
		if optimizedSQL == "" {
			optimizedSQL = input.SQL
		}
		if normalized, normErr := normalizeQueryForProtocol(input.DBProtocol, optimizedSQL, true); normErr == nil {
			optimizedSQL = normalized
		}
		if isMongoDBProtocol(input.DBProtocol) {
			if stabilized, changed, stabErr := stabilizeMongoOptimizedQuery(input.SQL, optimizedSQL); stabErr == nil {
				optimizedSQL = stabilized
				if changed {
					explanation = appendMongoSemanticsNote(explanation)
				}
			}
		}
		if explanation == "" {
			explanation = "No optimization opportunities identified."
		}
		return optimizeQueryResult{
			Status:         "complete",
			ConversationID: conversationID,
			OptimizedSQL:   optimizedSQL,
			Explanation:    explanation,
			Changes:        parsed.Changes,
		}, nil
	}

	return optimizeQueryResult{
		Status:         "needs_data",
		ConversationID: conversationID,
		DataRequests:   parsed.DataRequests,
	}, nil
}

func (s Service) continueOptimization(ctx context.Context, conversationID string, approvedData map[string]any, userID, tenantID, ipAddress string) (optimizeQueryResult, error) {
	state := s.ensureAIState()
	state.mu.Lock()
	pruneOptimizationLocked(state, time.Now().UTC())
	conversation, ok := state.optimizationSessions[conversationID]
	state.mu.Unlock()
	if !ok {
		return optimizeQueryResult{}, &requestError{status: http.StatusNotFound, message: "Conversation not found or expired."}
	}
	if conversation.UserID != userID || conversation.TenantID != tenantID {
		return optimizeQueryResult{}, &requestError{status: http.StatusNotFound, message: "Conversation not found or expired."}
	}

	conversation.Rounds++
	if conversation.ApprovedData == nil {
		conversation.ApprovedData = map[string]any{}
	}
	for key, value := range approvedData {
		conversation.ApprovedData[key] = value
	}

	messages := append([]llmMessage{}, conversation.Messages...)
	messages = append(messages,
		llmMessage{Role: "assistant", Content: `{"needs_data": true, "data_requests": [...]}`},
		llmMessage{Role: "user", Content: buildSecondTurnMessage(approvedData)},
	)

	raw, err := s.completeLLM(ctx, llmCompletionOptions{Messages: messages}, conversation.Overrides)
	if err != nil {
		return optimizeQueryResult{}, err
	}
	parsed := parseSecondTurnResponse(raw.Content, conversation.Input.SQL)
	if normalized, normErr := normalizeQueryForProtocol(conversation.Input.DBProtocol, parsed.OptimizedSQL, true); normErr == nil {
		parsed.OptimizedSQL = normalized
	}
	if isMongoDBProtocol(conversation.Input.DBProtocol) {
		if stabilized, changed, stabErr := stabilizeMongoOptimizedQuery(conversation.Input.SQL, parsed.OptimizedSQL); stabErr == nil {
			parsed.OptimizedSQL = stabilized
			if changed {
				parsed.Explanation = appendMongoSemanticsNote(parsed.Explanation)
			}
		}
	}

	provider, modelID := providerAndModelFromOverrides(conversation.Overrides)
	_ = s.insertAuditLog(ctx, userID, "DB_QUERY_AI_OPTIMIZED", "DatabaseQuery", conversation.Input.SessionID, map[string]any{
		"conversationId":   conversationID,
		"phase":            "continue",
		"round":            conversation.Rounds,
		"provider":         provider,
		"model":            modelID,
		"approvedDataKeys": mapKeys(approvedData),
	}, ipAddress)

	state.mu.Lock()
	delete(state.optimizationSessions, conversationID)
	state.mu.Unlock()

	return optimizeQueryResult{
		Status:         "complete",
		ConversationID: conversationID,
		OptimizedSQL:   parsed.OptimizedSQL,
		Explanation:    parsed.Explanation,
		Changes:        parsed.Changes,
	}, nil
}

func buildHeuristicDataRequests(input optimizeQueryInput) firstTurnResponse {
	if isMongoDBProtocol(input.DBProtocol) {
		collections := extractCollectionsFromMongoQuery(input.SQL)
		requests := make([]dataRequest, 0, len(collections)*3)
		for _, collection := range collections {
			requests = append(requests, dataRequest{
				Type:   "indexes",
				Target: collection,
				Reason: fmt.Sprintf("Inspect indexes on `%s` to identify query-shape improvements", collection),
			})
			requests = append(requests, dataRequest{
				Type:   "statistics",
				Target: collection,
				Reason: fmt.Sprintf("Read collection statistics for `%s` to understand scan cost", collection),
			})
			requests = append(requests, dataRequest{
				Type:   "table_schema",
				Target: collection,
				Reason: fmt.Sprintf("Inspect sampled schema for `%s` to validate field usage", collection),
			})
		}
		if len(requests) == 0 {
			return firstTurnResponse{}
		}
		return firstTurnResponse{NeedsData: true, DataRequests: requests}
	}

	tables := extractTablesFromSQL(input.SQL)
	requests := make([]dataRequest, 0, len(tables)*2)
	for _, table := range tables {
		requests = append(requests, dataRequest{
			Type:   "indexes",
			Target: table,
			Reason: fmt.Sprintf("Inspect indexes on `%s` to identify missing index opportunities", table),
		})
		requests = append(requests, dataRequest{
			Type:   "statistics",
			Target: table,
			Reason: fmt.Sprintf("Read column statistics for `%s` to understand data distribution", table),
		})
	}
	if len(tables) > 1 {
		limit := len(tables)
		if limit > 3 {
			limit = 3
		}
		for _, table := range tables[:limit] {
			requests = append(requests, dataRequest{
				Type:   "foreign_keys",
				Target: table,
				Reason: fmt.Sprintf("Check foreign key relationships on `%s` for join optimization", table),
			})
		}
	}
	if len(requests) == 0 {
		return firstTurnResponse{}
	}
	return firstTurnResponse{NeedsData: true, DataRequests: requests}
}

func extractCollectionsFromMongoQuery(queryText string) []string {
	_, collection, err := queryrunner.ParseMongoQueryMetadata(queryText)
	if err != nil || strings.TrimSpace(collection) == "" {
		return nil
	}
	return []string{strings.TrimSpace(collection)}
}

func extractTablesFromSQL(sqlText string) []string {
	pattern := regexp.MustCompile(`(?i)(?:FROM|JOIN)\s+(?:` + "`" + `|"|')?(\w+)(?:` + "`" + `|"|')?`)
	matches := pattern.FindAllStringSubmatch(sqlText, -1)
	seen := map[string]struct{}{}
	var tables []string
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		tables = append(tables, name)
	}
	return tables
}

func collectDataRequestTypes(requests []dataRequest) []string {
	items := make([]string, 0, len(requests))
	for _, item := range requests {
		items = append(items, item.Type+":"+item.Target)
	}
	return items
}

func mapKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
