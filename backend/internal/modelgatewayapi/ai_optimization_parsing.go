package modelgatewayapi

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

func extractJSON(text string) (any, error) {
	var direct any
	if err := json.Unmarshal([]byte(text), &direct); err == nil {
		return direct, nil
	}

	if match := regexp.MustCompile("(?is)```(?:json)?\\s*(.*?)```").FindStringSubmatch(text); len(match) == 2 {
		var fenced any
		if err := json.Unmarshal([]byte(strings.TrimSpace(match[1])), &fenced); err == nil {
			return fenced, nil
		}
	}

	if match := regexp.MustCompile(`(?s)\{.*\}`).FindString(text); strings.TrimSpace(match) != "" {
		var embedded any
		if err := json.Unmarshal([]byte(match), &embedded); err == nil {
			return embedded, nil
		}
	}

	return nil, &requestError{status: http.StatusBadGateway, message: "AI returned an invalid response format."}
}

func parseFirstTurnResponse(content string) firstTurnResponse {
	extracted, err := extractJSON(content)
	if err != nil {
		return firstTurnResponse{}
	}
	root, ok := extracted.(map[string]any)
	if !ok {
		return firstTurnResponse{}
	}

	if needsData, _ := root["needs_data"].(bool); needsData {
		requests := parseDataRequests(root["data_requests"])
		if len(requests) > 0 {
			return firstTurnResponse{NeedsData: true, DataRequests: requests}
		}
	}

	result := firstTurnResponse{}
	if value := extractQueryTextValue(root["optimized_sql"]); value != "" {
		result.OptimizedSQL = value
	} else if value := extractQueryTextValue(root["optimized_query"]); value != "" {
		result.OptimizedSQL = value
	}
	if value, ok := root["explanation"].(string); ok {
		result.Explanation = value
	}
	result.Changes = parseStringItems(root["changes"])
	return result
}

func parseSecondTurnResponse(content, originalSQL string) secondTurnResponse {
	extracted, err := extractJSON(content)
	if err != nil {
		return defaultSecondTurnResponse(originalSQL)
	}
	root, ok := extracted.(map[string]any)
	if !ok {
		return defaultSecondTurnResponse(originalSQL)
	}

	result := defaultSecondTurnResponse(originalSQL)
	if value := extractQueryTextValue(root["optimized_sql"]); value != "" {
		result.OptimizedSQL = value
	} else if value := extractQueryTextValue(root["optimized_query"]); value != "" {
		result.OptimizedSQL = value
	}
	if value, ok := root["explanation"].(string); ok && strings.TrimSpace(value) != "" {
		result.Explanation = strings.TrimSpace(value)
	}
	result.Changes = parseStringItems(root["changes"])
	return result
}

func parseDataRequests(value any) []dataRequest {
	rawItems, _ := value.([]any)
	requests := make([]dataRequest, 0, len(rawItems))
	for _, item := range rawItems {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		reqType, _ := record["type"].(string)
		target, _ := record["target"].(string)
		reason, _ := record["reason"].(string)
		reqType = strings.TrimSpace(reqType)
		target = strings.TrimSpace(target)
		reason = strings.TrimSpace(reason)
		if reqType == "" || target == "" || reason == "" {
			continue
		}
		if _, ok := validOptimizationIntrospectionTypes[reqType]; !ok {
			continue
		}
		requests = append(requests, dataRequest{Type: reqType, Target: target, Reason: reason})
	}
	return requests
}

func parseStringItems(value any) []string {
	items, _ := value.([]any)
	result := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			result = append(result, value)
		}
	}
	return result
}

func defaultSecondTurnResponse(originalSQL string) secondTurnResponse {
	return secondTurnResponse{
		OptimizedSQL: originalSQL,
		Explanation:  "Analysis complete. The query appears to be reasonably optimized.",
		Changes:      []string{},
	}
}
