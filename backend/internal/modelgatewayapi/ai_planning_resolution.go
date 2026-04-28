package modelgatewayapi

import (
	"sort"
	"strings"

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
