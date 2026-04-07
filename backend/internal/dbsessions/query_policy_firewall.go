package dbsessions

import (
	"context"
	"strings"
)

func (s Service) evaluateFirewall(ctx context.Context, tenantID, queryText, database, table string) firewallEvaluation {
	if s.DB == nil || strings.TrimSpace(tenantID) == "" {
		return firewallEvaluation{Allowed: true}
	}

	rows, err := s.DB.Query(ctx, `
SELECT name, pattern, action::text, scope
FROM "DbFirewallRule"
WHERE "tenantId" = $1 AND enabled = true
ORDER BY priority DESC, "createdAt" DESC
`, tenantID)
	if err == nil {
		defer rows.Close()

		for rows.Next() {
			var rule firewallRuleRecord
			if scanErr := rows.Scan(&rule.Name, &rule.Pattern, &rule.Action, &rule.Scope); scanErr != nil {
				break
			}
			if matchesScopedRegex(rule.Pattern, rule.Scope.String, queryText, database, table) {
				action := strings.ToUpper(strings.TrimSpace(rule.Action))
				return firewallEvaluation{
					Allowed:  action != "BLOCK",
					Action:   action,
					RuleName: rule.Name,
					Matched:  true,
				}
			}
		}
	}

	for _, builtin := range builtinDBFirewallPatterns {
		re, ok := compileCaseInsensitiveRegex(builtin.Pattern)
		if !ok {
			continue
		}
		if re.MatchString(queryText) {
			return firewallEvaluation{
				Allowed:  builtin.Action != "BLOCK",
				Action:   builtin.Action,
				RuleName: "[Built-in] " + builtin.Name,
				Matched:  true,
			}
		}
	}

	return firewallEvaluation{Allowed: true}
}

func matchesScopedRegex(pattern, scope, queryText, database, table string) bool {
	if trimmed := strings.ToLower(strings.TrimSpace(scope)); trimmed != "" {
		database = strings.ToLower(strings.TrimSpace(database))
		table = strings.ToLower(strings.TrimSpace(table))
		if database != trimmed && table != trimmed {
			return false
		}
	}

	re, ok := compileCaseInsensitiveRegex(pattern)
	if !ok {
		return false
	}
	return re.MatchString(queryText)
}
