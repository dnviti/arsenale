package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Manage and use AI features",
}

var aiConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage AI configuration",
}

var aiConfigGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get AI configuration",
	Run:   runAiConfigGet,
}

var aiConfigSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set AI configuration",
	Long:  `Set AI configuration from a JSON/YAML file: arsenale ai config set --from-file config.yaml`,
	Run:   runAiConfigSet,
}

var aiConfigFromFile string

func init() {
	rootCmd.AddCommand(aiCmd)

	aiCmd.AddCommand(aiConfigCmd)
	aiConfigCmd.AddCommand(aiConfigGetCmd)
	aiConfigCmd.AddCommand(aiConfigSetCmd)

	aiConfigSetCmd.Flags().StringVarP(&aiConfigFromFile, "from-file", "f", "", "JSON/YAML file (- for stdin)")
	aiConfigSetCmd.MarkFlagRequired("from-file")
}

func runAiConfigGet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	body, status, err := apiGet("/api/ai/config", cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)
	printer().PrintSingle(body, []Column{
		{Header: "GEN_ENABLED", Field: "enabled"},
		{Header: "GEN_BACKEND", Field: "queryGeneration.backend"},
		{Header: "GEN_MODEL", Field: "modelId"},
		{Header: "OPT_ENABLED", Field: "queryOptimizer.enabled"},
		{Header: "OPT_BACKEND", Field: "queryOptimizer.backend"},
		{Header: "OPT_MODEL", Field: "queryOptimizer.modelId"},
		{Header: "TEMP", Field: "temperature"},
		{Header: "TIMEOUT_MS", Field: "timeoutMs"},
	})
}

func runAiConfigSet(cmd *cobra.Command, args []string) {
	cfg := getCfg()
	if err := ensureAuthenticated(cfg); err != nil {
		fatal("%v", err)
	}

	data, err := readResourceFromFileOrStdin(aiConfigFromFile)
	if err != nil {
		fatal("%v", err)
	}
	data, err = normalizeAIConfigPayload(data)
	if err != nil {
		fatal("%v", err)
	}

	body, status, err := apiPut("/api/ai/config", json.RawMessage(data), cfg)
	if err != nil {
		fatal("%v", err)
	}
	checkAPIError(status, body)

	if outputFormat == "json" || outputFormat == "yaml" {
		if err := printer().PrintSingle(body, nil); err != nil {
			fatal("%v", err)
		}
		return
	}
	if !quiet {
		fmt.Println("AI configuration updated")
	}
}

func normalizeAIConfigPayload(data []byte) ([]byte, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse AI config: %w", err)
	}

	for _, key := range []string{
		"provider",
		"hasApiKey",
		"modelId",
		"baseUrl",
		"maxTokensPerRequest",
		"dailyRequestLimit",
		"enabled",
	} {
		delete(payload, key)
	}

	if rawBackends, ok := payload["backends"]; ok && string(rawBackends) != "null" {
		var backends []map[string]json.RawMessage
		if err := json.Unmarshal(rawBackends, &backends); err != nil {
			return nil, fmt.Errorf("parse AI backends: %w", err)
		}
		for i := range backends {
			delete(backends[i], "hasApiKey")
		}
		normalizedBackends, err := json.Marshal(backends)
		if err != nil {
			return nil, fmt.Errorf("normalize AI backends: %w", err)
		}
		payload["backends"] = normalizedBackends
	}

	normalized, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("normalize AI config: %w", err)
	}
	return normalized, nil
}
