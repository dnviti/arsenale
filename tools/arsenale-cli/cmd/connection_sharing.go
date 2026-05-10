package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var connectionShareColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "SHARED_WITH", Field: "sharedWith"},
	{Header: "PERMISSION", Field: "permission"},
}

var connectionSharesColumns = []Column{
	{Header: "USER_ID", Field: "userId"},
	{Header: "EMAIL", Field: "email"},
	{Header: "PERMISSION", Field: "permission"},
}

var connectionFavoriteColumns = []Column{
	{Header: "ID", Field: "id"},
	{Header: "FAVORITE", Field: "isFavorite"},
}

func runConnShare(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	permission, err := normalizeConnectionPermission(connSharePerm)
	if err != nil {
		fatal("%v", err)
	}
	payload := map[string]string{"permission": permission}
	if value := strings.TrimSpace(connShareUserID); value != "" {
		payload["userId"] = value
	}
	if value := strings.TrimSpace(connShareEmail); value != "" {
		payload["email"] = value
	}
	if payload["userId"] == "" && payload["email"] == "" {
		fatal("either --user-id or --email is required")
	}

	body, status, err := apiPost(fmt.Sprintf("/api/connections/%s/share", args[0]), payload, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, connectionShareColumns)
}

func runConnUnshare(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiDelete(fmt.Sprintf("/api/connections/%s/share/%s", args[0], args[1]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	if outputFormat == "json" || outputFormat == "yaml" {
		printer().PrintSingle(body, []Column{{Header: "DELETED", Field: "deleted"}})
		return
	}
	if !quiet {
		fmt.Println("Share removed")
	}
}

func runConnShares(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet(fmt.Sprintf("/api/connections/%s/shares", args[0]), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().Print(body, connectionSharesColumns)
}

func runConnBatchShare(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(connFromFile)
	if err != nil {
		fatal("%v", err)
	}
	normalized, err := normalizeBatchSharePayload(data)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPost("/api/connections/batch-share", json.RawMessage(normalized), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "SHARED", Field: "shared"},
		{Header: "FAILED", Field: "failed"},
		{Header: "ALREADY_SHARED", Field: "alreadyShared"},
	})
}

func runConnFavorite(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPatch(fmt.Sprintf("/api/connections/%s/favorite", args[0]), nil, cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, connectionFavoriteColumns)
}

func normalizeConnectionPermission(value string) (string, error) {
	switch strings.ToUpper(strings.NewReplacer("-", "_", " ", "_").Replace(strings.TrimSpace(value))) {
	case "READ", "READ_ONLY", "READONLY":
		return "READ_ONLY", nil
	case "WRITE", "FULL", "FULL_ACCESS", "FULLACCESS":
		return "FULL_ACCESS", nil
	default:
		return "", fmt.Errorf("permission must be READ_ONLY or FULL_ACCESS")
	}
}

func normalizeBatchSharePayload(data []byte) ([]byte, error) {
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if permission, ok := payload["permission"].(string); ok {
		normalized, err := normalizeConnectionPermission(permission)
		if err != nil {
			return nil, err
		}
		payload["permission"] = normalized
	}
	return json.Marshal(payload)
}
