package queryrunner

import (
	"encoding/json"
	"fmt"
	"strings"
)

func parseMongoQuerySpec(raw string) (mongoQuerySpec, error) {
	trimmed := strings.TrimSpace(raw)
	var payload any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return mongoQuerySpec{}, fmt.Errorf("mongodb queries must use a JSON spec: %w", err)
	}
	root, ok := payload.(map[string]any)
	if !ok {
		return mongoQuerySpec{}, fmt.Errorf("mongodb queries must use a JSON object spec")
	}
	spec, err := normalizeMongoQuerySpecMap(root)
	if err != nil {
		return mongoQuerySpec{}, err
	}
	if spec.Operation == "" {
		return mongoQuerySpec{}, fmt.Errorf("mongodb query spec requires an operation")
	}
	return spec, nil
}

func normalizeMongoQuerySpecMap(root map[string]any) (mongoQuerySpec, error) {
	root = unwrapMongoQueryEnvelope(root)

	raw, err := json.Marshal(root)
	if err != nil {
		return mongoQuerySpec{}, fmt.Errorf("marshal mongodb query spec: %w", err)
	}

	var spec mongoQuerySpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return mongoQuerySpec{}, fmt.Errorf("decode mongodb query spec: %w", err)
	}

	spec.Operation = normalizeMongoOperation(spec.Operation)
	applyMongoCommonAliases(root, &spec)
	if spec.Operation == "" {
		inferMongoOperation(root, &spec)
	}
	normalizeMongoCollectionReference(&spec)
	return spec, nil
}

func normalizeMongoCollectionReference(spec *mongoQuerySpec) {
	if spec == nil {
		return
	}

	spec.Database = strings.TrimSpace(spec.Database)
	spec.Collection = strings.TrimSpace(spec.Collection)
	if spec.Collection == "" || strings.Count(spec.Collection, ".") != 1 {
		return
	}

	parts := strings.SplitN(spec.Collection, ".", 2)
	databaseName := strings.TrimSpace(parts[0])
	collectionName := strings.TrimSpace(parts[1])
	if databaseName == "" || collectionName == "" {
		return
	}

	if spec.Database == "" {
		spec.Database = databaseName
		spec.Collection = collectionName
		return
	}
	if strings.EqualFold(spec.Database, databaseName) {
		spec.Collection = collectionName
	}
}

func unwrapMongoQueryEnvelope(root map[string]any) map[string]any {
	for _, key := range []string{"query", "querySpec", "spec"} {
		nested, ok := root[key].(map[string]any)
		if !ok || len(nested) == 0 {
			continue
		}
		if key == "query" && hasMongoDirectSpecKeys(root) {
			continue
		}
		if !looksLikeMongoQuerySpecMap(nested) {
			continue
		}
		merged := make(map[string]any, len(root)+len(nested))
		for nestedKey, nestedValue := range nested {
			merged[nestedKey] = nestedValue
		}
		for _, passthrough := range []string{"database", "collection", "operation", "filter", "projection", "sort", "limit", "skip", "pipeline", "document", "documents", "update", "command", "field"} {
			if _, ok := merged[passthrough]; ok {
				continue
			}
			if value, ok := root[passthrough]; ok {
				merged[passthrough] = value
			}
		}
		return merged
	}
	return root
}

func hasMongoDirectSpecKeys(root map[string]any) bool {
	for _, key := range []string{"operation", "collection", "filter", "projection", "sort", "limit", "skip", "pipeline", "document", "documents", "update", "command", "field", "find", "aggregate", "count", "distinct", "insertOne", "insertMany", "updateOne", "updateMany", "deleteOne", "deleteMany", "runCommand", "runcommand"} {
		if _, ok := root[key]; ok {
			return true
		}
	}
	return false
}

func looksLikeMongoQuerySpecMap(root map[string]any) bool {
	for _, key := range []string{"operation", "collection", "filter", "projection", "sort", "limit", "skip", "pipeline", "document", "documents", "update", "command", "field", "find", "aggregate", "count", "distinct", "insertOne", "insertMany", "updateOne", "updateMany", "deleteOne", "deleteMany", "runCommand", "runcommand"} {
		if _, ok := root[key]; ok {
			return true
		}
	}
	return false
}

func normalizeMongoOperation(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	if normalized, ok := mongoOperationAliases[value]; ok {
		return normalized
	}
	return value
}

func applyMongoCommonAliases(root map[string]any, spec *mongoQuerySpec) {
	if spec == nil {
		return
	}
	if len(spec.Filter) == 0 {
		if value, ok := root["query"].(map[string]any); ok && len(value) > 0 {
			spec.Filter = value
		}
	}
	if len(spec.Projection) == 0 {
		if value, ok := root["fields"].(map[string]any); ok && len(value) > 0 {
			spec.Projection = value
		}
	}
	if spec.Field == "" {
		if value, ok := root["key"].(string); ok {
			spec.Field = strings.TrimSpace(value)
		}
	}
}

func inferMongoOperation(root map[string]any, spec *mongoQuerySpec) {
	if spec == nil {
		return
	}

	for _, candidate := range []string{"find", "aggregate", "count", "distinct", "insertOne", "insertMany", "updateOne", "updateMany", "deleteOne", "deleteMany", "runCommand", "runcommand"} {
		value, ok := root[candidate]
		if !ok {
			continue
		}
		spec.Operation = normalizeMongoOperation(candidate)
		applyMongoOperationShorthand(spec, value)
		return
	}

	switch {
	case len(spec.Command) > 0:
		spec.Operation = "runcommand"
	case spec.Collection != "" && len(spec.Pipeline) > 0:
		spec.Operation = "aggregate"
	case spec.Collection != "" && spec.Field != "":
		spec.Operation = "distinct"
	case spec.Collection != "":
		spec.Operation = "find"
	}
}

func applyMongoOperationShorthand(spec *mongoQuerySpec, value any) {
	if spec == nil {
		return
	}

	switch typed := value.(type) {
	case string:
		if spec.Operation != "runcommand" && spec.Collection == "" {
			spec.Collection = strings.TrimSpace(typed)
		}
	case map[string]any:
		raw, err := json.Marshal(typed)
		if err == nil {
			var nested mongoQuerySpec
			if json.Unmarshal(raw, &nested) == nil {
				nested.Operation = normalizeMongoOperation(nested.Operation)
				mergeMongoQuerySpec(spec, nested)
			}
		}
		applyMongoCommonAliases(typed, spec)
		if spec.Collection == "" {
			if collection, ok := typed["collection"].(string); ok {
				spec.Collection = strings.TrimSpace(collection)
			}
		}
		if spec.Operation == "distinct" && spec.Field == "" {
			if key, ok := typed["key"].(string); ok {
				spec.Field = strings.TrimSpace(key)
			}
		}
		if spec.Operation == "runcommand" && len(spec.Command) == 0 {
			spec.Command = typed
		}
	}
}

func mergeMongoQuerySpec(dst *mongoQuerySpec, src mongoQuerySpec) {
	if dst == nil {
		return
	}
	if dst.Database == "" {
		dst.Database = strings.TrimSpace(src.Database)
	}
	if dst.Collection == "" {
		dst.Collection = strings.TrimSpace(src.Collection)
	}
	if dst.Operation == "" {
		dst.Operation = normalizeMongoOperation(src.Operation)
	}
	if len(dst.Filter) == 0 && len(src.Filter) > 0 {
		dst.Filter = src.Filter
	}
	if len(dst.Projection) == 0 && len(src.Projection) > 0 {
		dst.Projection = src.Projection
	}
	if len(dst.Sort) == 0 && len(src.Sort) > 0 {
		dst.Sort = src.Sort
	}
	if dst.Limit == 0 && src.Limit != 0 {
		dst.Limit = src.Limit
	}
	if dst.Skip == 0 && src.Skip != 0 {
		dst.Skip = src.Skip
	}
	if len(dst.Pipeline) == 0 && len(src.Pipeline) > 0 {
		dst.Pipeline = src.Pipeline
	}
	if len(dst.Document) == 0 && len(src.Document) > 0 {
		dst.Document = src.Document
	}
	if len(dst.Documents) == 0 && len(src.Documents) > 0 {
		dst.Documents = src.Documents
	}
	if len(dst.Update) == 0 && len(src.Update) > 0 {
		dst.Update = src.Update
	}
	if len(dst.Command) == 0 && len(src.Command) > 0 {
		dst.Command = src.Command
	}
	if dst.Field == "" {
		dst.Field = strings.TrimSpace(src.Field)
	}
}

func ParseMongoQueryMetadata(raw string) (string, string, error) {
	spec, err := parseMongoQuerySpec(raw)
	if err != nil {
		return "", "", err
	}
	return spec.Operation, strings.TrimSpace(spec.Collection), nil
}

func NormalizeMongoQueryText(raw string) (string, string, string, error) {
	spec, err := parseMongoQuerySpec(raw)
	if err != nil {
		return "", "", "", err
	}
	normalized, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", "", "", fmt.Errorf("marshal mongodb query spec: %w", err)
	}
	return string(normalized), spec.Operation, strings.TrimSpace(spec.Collection), nil
}

func NormalizeMongoReadOnlyQueryText(raw string) (string, string, string, error) {
	spec, err := parseMongoQuerySpec(raw)
	if err != nil {
		return "", "", "", err
	}
	if err := validateMongoReadOnlySpec(spec); err != nil {
		return "", "", "", err
	}
	normalized, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", "", "", fmt.Errorf("marshal mongodb query spec: %w", err)
	}
	return string(normalized), spec.Operation, strings.TrimSpace(spec.Collection), nil
}

func validateMongoReadOnlySpec(spec mongoQuerySpec) error {
	switch spec.Operation {
	case "find", "aggregate", "count", "distinct", "runcmd", "runcommand":
		return nil
	default:
		return fmt.Errorf("mongodb read-only mode does not allow %q", spec.Operation)
	}
}
