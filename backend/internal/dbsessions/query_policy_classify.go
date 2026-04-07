package dbsessions

import (
	"regexp"
	"strings"

	"github.com/dnviti/arsenale/backend/internal/queryrunner"
)

func compileCaseInsensitiveRegex(pattern string) (*regexp.Regexp, bool) {
	cacheKey := "(?i)" + pattern
	if value, ok := compiledPatternCache.Load(cacheKey); ok {
		entry := value.(compiledRegexCacheEntry)
		return entry.re, entry.ok
	}

	re, err := regexp.Compile(cacheKey)
	entry := compiledRegexCacheEntry{re: re, ok: err == nil}
	actual, _ := compiledPatternCache.LoadOrStore(cacheKey, entry)
	cached := actual.(compiledRegexCacheEntry)
	return cached.re, cached.ok
}

func classifyDBQuery(queryText string) dbQueryType {
	if operation, ok := parseMongoOperation(queryText); ok {
		switch operation {
		case "find", "aggregate", "count", "distinct", "runcmd", "runcommand":
			return dbQueryTypeSelect
		case "insertone", "insertmany":
			return dbQueryTypeInsert
		case "updateone", "updatemany":
			return dbQueryTypeUpdate
		case "deleteone", "deletemany":
			return dbQueryTypeDelete
		default:
			return dbQueryTypeOther
		}
	}

	trimmed := stripLeadingSQLComments(queryText)

	switch {
	case dbQueryTypeDDLPattern.MatchString(trimmed):
		return dbQueryTypeDDL
	case dbQueryTypeSelectPattern.MatchString(trimmed):
		return dbQueryTypeSelect
	case dbQueryTypeInsertPattern.MatchString(trimmed):
		return dbQueryTypeInsert
	case dbQueryTypeUpdatePattern.MatchString(trimmed):
		return dbQueryTypeUpdate
	case dbQueryTypeDeletePattern.MatchString(trimmed):
		return dbQueryTypeDelete
	case dbQueryTypeWithPattern.MatchString(trimmed):
		switch {
		case dbCteSelectPattern.MatchString(trimmed):
			return dbQueryTypeSelect
		case dbCteInsertPattern.MatchString(trimmed):
			return dbQueryTypeInsert
		case dbCteUpdatePattern.MatchString(trimmed):
			return dbQueryTypeUpdate
		case dbCteDeletePattern.MatchString(trimmed):
			return dbQueryTypeDelete
		default:
			return dbQueryTypeSelect
		}
	case dbQueryTypeExplainPattern.MatchString(trimmed):
		return dbQueryTypeSelect
	case dbQueryTypeShowPattern.MatchString(trimmed):
		return dbQueryTypeSelect
	case dbQueryTypeSetPattern.MatchString(trimmed):
		return dbQueryTypeDDL
	case dbQueryTypeGrantPattern.MatchString(trimmed):
		return dbQueryTypeDDL
	case dbQueryTypeMergePattern.MatchString(trimmed):
		return dbQueryTypeUpdate
	case dbQueryTypeCallPattern.MatchString(trimmed):
		return dbQueryTypeOther
	default:
		return dbQueryTypeOther
	}
}

func stripLeadingSQLComments(sqlText string) string {
	i := 0
	for i < len(sqlText) {
		switch sqlText[i] {
		case ' ', '\t', '\n', '\r':
			i++
			continue
		}
		if i+1 < len(sqlText) && sqlText[i] == '-' && sqlText[i+1] == '-' {
			i += 2
			for i < len(sqlText) && sqlText[i] != '\n' {
				i++
			}
			if i < len(sqlText) {
				i++
			}
			continue
		}
		if i+1 < len(sqlText) && sqlText[i] == '/' && sqlText[i+1] == '*' {
			i += 2
			for i+1 < len(sqlText) && !(sqlText[i] == '*' && sqlText[i+1] == '/') {
				i++
			}
			if i+1 < len(sqlText) {
				i += 2
			}
			continue
		}
		break
	}
	return sqlText[i:]
}

func extractTablesAccessed(queryText string) []string {
	if collection, ok := parseMongoCollection(queryText); ok {
		return []string{collection}
	}

	seen := map[string]struct{}{}
	items := make([]string, 0)
	for _, pattern := range dbTableAccessPatterns {
		matches := pattern.FindAllStringSubmatch(queryText, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			name := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(match[1], `"`, ""), "`", "")))
			if name == "" {
				continue
			}
			if _, reserved := reservedTableWords[name]; reserved {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			items = append(items, name)
		}
	}
	return items
}

func parseMongoOperation(queryText string) (string, bool) {
	operation, _, err := queryrunner.ParseMongoQueryMetadata(queryText)
	if err != nil {
		return "", false
	}
	if operation == "" {
		return "", false
	}
	return operation, true
}

func parseMongoCollection(queryText string) (string, bool) {
	_, collection, err := queryrunner.ParseMongoQueryMetadata(queryText)
	if err != nil {
		return "", false
	}
	collection = strings.ToLower(strings.TrimSpace(collection))
	if collection == "" {
		return "", false
	}
	return collection, true
}
