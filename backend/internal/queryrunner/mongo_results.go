package queryrunner

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func mongoDocumentsToResult(docs []bson.M) contracts.QueryExecutionResponse {
	rows := normalizeMongoDocuments(docs)
	columnSet := make(map[string]struct{})
	for _, row := range rows {
		for key := range row {
			columnSet[key] = struct{}{}
		}
	}
	columns := make([]string, 0, len(columnSet))
	for key := range columnSet {
		columns = append(columns, key)
	}
	sort.Strings(columns)

	return contracts.QueryExecutionResponse{
		Columns:  columns,
		Rows:     rows,
		RowCount: len(rows),
	}
}

func singleRowMongoResult(row map[string]any) contracts.QueryExecutionResponse {
	columns := make([]string, 0, len(row))
	for key := range row {
		columns = append(columns, key)
	}
	sort.Strings(columns)
	return contracts.QueryExecutionResponse{
		Columns:  columns,
		Rows:     []map[string]any{row},
		RowCount: 1,
	}
}

func normalizeMongoDocuments(docs []bson.M) []map[string]any {
	rows := make([]map[string]any, 0, len(docs))
	for _, doc := range docs {
		rows = append(rows, normalizeMongoDocument(doc))
	}
	return rows
}

func normalizeMongoDocument(doc bson.M) map[string]any {
	row := make(map[string]any, len(doc))
	for key, value := range doc {
		row[key] = normalizeMongoValue(value)
	}
	return row
}

func normalizeMongoValue(value any) any {
	payload, err := bson.MarshalExtJSON(value, false, false)
	if err != nil {
		return value
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return string(payload)
	}
	return decoded
}

func mongoTypeName(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case int32, int64, int, float32, float64:
		return "number"
	case string:
		return "string"
	case time.Time:
		return "date"
	case bson.M, map[string]any:
		return "document"
	case []any, bson.A:
		return "array"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func defaultMongoMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	return value
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
