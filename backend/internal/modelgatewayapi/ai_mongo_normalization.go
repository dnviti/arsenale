package modelgatewayapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/queryrunner"
)

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
