package cmd

import (
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"
)

var connectionListColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
	{Header: "HOST", Field: "host"},
	{Header: "PORT", Field: "port"},
	{Header: "SCOPE", Field: "scope"},
}

type connectionListGroups struct {
	Own    []map[string]any `json:"own"`
	Shared []map[string]any `json:"shared"`
	Team   []map[string]any `json:"team"`
}

func runConnList(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/connections", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	displayBody, tableBody, err := normalizeConnectionListOutput(body, connSearch)
	if err != nil {
		fatal("%v", err)
	}

	if outputFormat == "json" || outputFormat == "yaml" {
		printer().Print(displayBody, connectionColumns)
		return
	}
	printer().Print(tableBody, connectionListColumns)
}

func runConnGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/connections/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, connectionColumns)
}

func runConnCreate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(connFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/connections", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintCreated(body, "id")
}

func runConnUpdate(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(connFromFile)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/connections/"+args[0], json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, connectionColumns)
}

func runConnDelete(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete("/api/connections/"+args[0], cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if outputFormat == "json" || outputFormat == "yaml" {
		printer().PrintSingle(body, []Column{{Header: "DELETED", Field: "deleted"}})
		return
	}
	printer().PrintDeleted("Connection", args[0])
}

func normalizeConnectionListOutput(body []byte, search string) ([]byte, []byte, error) {
	if groups, ok, err := parseConnectionGroups(body); err != nil {
		return nil, nil, err
	} else if ok {
		filterConnectionGroups(&groups, search)
		flat := flattenConnectionGroups(groups)
		displayBody, err := json.Marshal(groups)
		if err != nil {
			return nil, nil, err
		}
		tableBody, err := json.Marshal(flat)
		if err != nil {
			return nil, nil, err
		}
		return displayBody, tableBody, nil
	}

	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, nil, err
	}
	rows = filterConnectionRows(rows, search)
	tableBody, err := json.Marshal(rows)
	if err != nil {
		return nil, nil, err
	}
	return tableBody, tableBody, nil
}

func parseConnectionGroups(body []byte) (connectionListGroups, bool, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return connectionListGroups{}, false, nil
	}
	if _, ok := raw["own"]; !ok {
		if _, ok := raw["shared"]; !ok {
			if _, ok := raw["team"]; !ok {
				return connectionListGroups{}, false, nil
			}
		}
	}

	var groups connectionListGroups
	if err := json.Unmarshal(body, &groups); err != nil {
		return connectionListGroups{}, true, err
	}
	return groups, true, nil
}

func filterConnectionGroups(groups *connectionListGroups, search string) {
	groups.Own = filterConnectionRows(withConnectionScope(groups.Own, "private"), search)
	groups.Shared = filterConnectionRows(withConnectionScope(groups.Shared, "shared"), search)
	groups.Team = filterConnectionRows(withConnectionScope(groups.Team, "team"), search)
}

func flattenConnectionGroups(groups connectionListGroups) []map[string]any {
	total := len(groups.Own) + len(groups.Shared) + len(groups.Team)
	rows := make([]map[string]any, 0, total)
	rows = append(rows, groups.Own...)
	rows = append(rows, groups.Shared...)
	rows = append(rows, groups.Team...)
	return rows
}

func withConnectionScope(rows []map[string]any, fallback string) []map[string]any {
	for _, row := range rows {
		if _, ok := row["scope"]; !ok {
			row["scope"] = fallback
		}
	}
	return rows
}

func filterConnectionRows(rows []map[string]any, search string) []map[string]any {
	needle := strings.ToLower(strings.TrimSpace(search))
	if needle == "" {
		return rows
	}

	filtered := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if connectionRowMatches(row, needle) {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func connectionRowMatches(row map[string]any, needle string) bool {
	for _, field := range []string{"id", "name", "type", "host", "scope"} {
		if strings.Contains(strings.ToLower(formatValue(row[field])), needle) {
			return true
		}
	}
	return false
}
