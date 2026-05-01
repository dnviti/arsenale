package tooling

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dnviti/arsenale/backend/internal/authz"
	"github.com/dnviti/arsenale/backend/internal/catalog"
	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

type memoryReadInput struct {
	NamespaceKey string `json:"namespaceKey"`
}

func LookupCapability(id string) (contracts.CapabilityDefinition, error) {
	for _, capability := range catalog.Capabilities() {
		if capability.ID == id {
			return capability, nil
		}
	}
	return contracts.CapabilityDefinition{}, fmt.Errorf("unknown capability %q", id)
}

func PlanToolCall(req contracts.ToolCallPlanRequest) (contracts.ToolCallPlanResponse, error) {
	capability, err := LookupCapability(req.Capability)
	if err != nil {
		return contracts.ToolCallPlanResponse{}, err
	}

	authzReq := req.Authz
	if authzReq.Action == "" {
		authzReq.Action = capability.Action
	}
	if authzReq.Resource.Type == "" {
		authzReq.Resource.Type = capability.ResourceType
	}

	decision := authz.NewStaticPDP().Evaluate(authzReq)
	return contracts.ToolCallPlanResponse{
		Capability: capability,
		Decision:   decision,
		DryRun:     true,
	}, nil
}

func ExecuteToolCall(ctx context.Context, req contracts.ToolCallExecuteRequest) (contracts.ToolCallExecuteResponse, error) {
	capability, err := LookupCapability(req.Capability)
	if err != nil {
		return contracts.ToolCallExecuteResponse{}, err
	}

	authzReq := req.Authz
	if authzReq.Action == "" {
		authzReq.Action = capability.Action
	}
	if authzReq.Resource.Type == "" {
		authzReq.Resource.Type = capability.ResourceType
	}

	decision := authz.NewStaticPDP().Evaluate(authzReq)
	response := contracts.ToolCallExecuteResponse{
		Capability: capability,
		Decision:   decision,
		DryRun:     false,
	}
	if decision.Effect != contracts.AuthzAllow {
		return response, nil
	}

	switch capability.ID {
	case "connection.connect.ssh":
		var input contracts.TerminalSessionGrant
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return contracts.ToolCallExecuteResponse{}, fmt.Errorf("decode terminal grant input: %w", err)
		}
		output, err := executeTerminalGrant(ctx, input)
		if err != nil {
			return contracts.ToolCallExecuteResponse{}, err
		}
		response.Output = output
		return response, nil
	case "db.schema.read":
		var input contracts.SchemaFetchRequest
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return contracts.ToolCallExecuteResponse{}, fmt.Errorf("decode schema read input: %w", err)
		}
		output, err := executeSchemaRead(ctx, input)
		if err != nil {
			return contracts.ToolCallExecuteResponse{}, err
		}
		response.Output = output
		return response, nil
	case "db.introspection.read":
		var input contracts.QueryIntrospectionRequest
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return contracts.ToolCallExecuteResponse{}, fmt.Errorf("decode introspection input: %w", err)
		}
		output, err := executeIntrospection(ctx, input)
		if err != nil {
			return contracts.ToolCallExecuteResponse{}, err
		}
		response.Output = output
		return response, nil
	case "db.query.execute.readonly":
		var input contracts.QueryExecutionRequest
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return contracts.ToolCallExecuteResponse{}, fmt.Errorf("decode readonly query input: %w", err)
		}
		output, err := executeReadOnlyQuery(ctx, input)
		if err != nil {
			return contracts.ToolCallExecuteResponse{}, err
		}
		response.Output = output
		return response, nil
	case "memory.read":
		var input memoryReadInput
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return contracts.ToolCallExecuteResponse{}, fmt.Errorf("decode memory read input: %w", err)
		}
		output, err := executeMemoryRead(ctx, input)
		if err != nil {
			return contracts.ToolCallExecuteResponse{}, err
		}
		response.Output = output
		return response, nil
	case "memory.write":
		var input contracts.MemoryWriteRequest
		if err := json.Unmarshal(req.Input, &input); err != nil {
			return contracts.ToolCallExecuteResponse{}, fmt.Errorf("decode memory write input: %w", err)
		}
		output, err := executeMemoryWrite(ctx, input)
		if err != nil {
			return contracts.ToolCallExecuteResponse{}, err
		}
		response.Output = output
		return response, nil
	default:
		return contracts.ToolCallExecuteResponse{}, fmt.Errorf("capability %q is not executable yet", capability.ID)
	}
}

func executeTerminalGrant(ctx context.Context, input contracts.TerminalSessionGrant) (map[string]any, error) {
	payload, err := json.Marshal(contracts.TerminalSessionGrantIssueRequest{Grant: input})
	if err != nil {
		return nil, fmt.Errorf("marshal terminal grant request: %w", err)
	}

	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		strings.TrimRight(terminalBrokerURL(), "/")+"/v1/session-grants:issue",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("build terminal-broker grant request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call terminal-broker for grant issuance: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var failure map[string]any
		_ = json.NewDecoder(response.Body).Decode(&failure)
		if message, ok := failure["error"].(string); ok && message != "" {
			return nil, fmt.Errorf("terminal-broker rejected grant request: %s", message)
		}
		return nil, fmt.Errorf("terminal-broker returned status %d for grant request", response.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode terminal-broker grant response: %w", err)
	}

	if _, ok := result["webSocketUrl"]; !ok {
		if token, ok := result["token"].(string); ok && token != "" {
			result["webSocketUrl"] = strings.TrimRight(publicTerminalBrokerURL(), "/") + "/ws/terminal?token=" + url.QueryEscape(token)
		}
	}

	return result, nil
}

func executeMemoryRead(ctx context.Context, input memoryReadInput) (map[string]any, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodGet,
		strings.TrimRight(memoryServiceURL(), "/")+"/v1/memory/items?namespaceKey="+url.QueryEscape(strings.TrimSpace(input.NamespaceKey)),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build memory-service read request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call memory-service for read: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var failure map[string]any
		_ = json.NewDecoder(response.Body).Decode(&failure)
		if message, ok := failure["error"].(string); ok && message != "" {
			return nil, fmt.Errorf("memory-service rejected read request: %s", message)
		}
		return nil, fmt.Errorf("memory-service returned status %d for read request", response.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode memory-service read response: %w", err)
	}

	return result, nil
}

func executeMemoryWrite(ctx context.Context, input contracts.MemoryWriteRequest) (map[string]any, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal memory write request: %w", err)
	}

	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		strings.TrimRight(memoryServiceURL(), "/")+"/v1/memory/items",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("build memory-service write request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call memory-service for write: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		var failure map[string]any
		_ = json.NewDecoder(response.Body).Decode(&failure)
		if message, ok := failure["error"].(string); ok && message != "" {
			return nil, fmt.Errorf("memory-service rejected write request: %s", message)
		}
		return nil, fmt.Errorf("memory-service returned status %d for write request", response.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode memory-service write response: %w", err)
	}

	return result, nil
}

func executeIntrospection(ctx context.Context, input contracts.QueryIntrospectionRequest) (contracts.QueryIntrospectionResponse, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return contracts.QueryIntrospectionResponse{}, fmt.Errorf("marshal introspection request: %w", err)
	}

	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		strings.TrimRight(queryRunnerURL(), "/")+"/v1/introspection:run",
		bytes.NewReader(payload),
	)
	if err != nil {
		return contracts.QueryIntrospectionResponse{}, fmt.Errorf("build introspection request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return contracts.QueryIntrospectionResponse{}, fmt.Errorf("call query-runner for introspection: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var failure map[string]any
		_ = json.NewDecoder(response.Body).Decode(&failure)
		if message, ok := failure["error"].(string); ok && message != "" {
			return contracts.QueryIntrospectionResponse{}, fmt.Errorf("query-runner rejected introspection request: %s", message)
		}
		return contracts.QueryIntrospectionResponse{}, fmt.Errorf("query-runner returned status %d for introspection request", response.StatusCode)
	}

	var result contracts.QueryIntrospectionResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return contracts.QueryIntrospectionResponse{}, fmt.Errorf("decode introspection response: %w", err)
	}

	return result, nil
}

func executeSchemaRead(ctx context.Context, input contracts.SchemaFetchRequest) (contracts.SchemaInfo, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("marshal schema read request: %w", err)
	}

	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		strings.TrimRight(queryRunnerURL(), "/")+"/v1/schema:fetch",
		bytes.NewReader(payload),
	)
	if err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("build query-runner schema request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("call query-runner for schema: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var failure map[string]any
		_ = json.NewDecoder(response.Body).Decode(&failure)
		if message, ok := failure["error"].(string); ok && message != "" {
			return contracts.SchemaInfo{}, fmt.Errorf("query-runner rejected schema request: %s", message)
		}
		return contracts.SchemaInfo{}, fmt.Errorf("query-runner returned status %d for schema request", response.StatusCode)
	}

	var result contracts.SchemaInfo
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return contracts.SchemaInfo{}, fmt.Errorf("decode query-runner schema response: %w", err)
	}

	return result, nil
}

func executeReadOnlyQuery(ctx context.Context, input contracts.QueryExecutionRequest) (contracts.QueryExecutionResponse, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return contracts.QueryExecutionResponse{}, fmt.Errorf("marshal readonly query request: %w", err)
	}

	requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		strings.TrimRight(queryRunnerURL(), "/")+"/v1/query-runs:execute",
		bytes.NewReader(payload),
	)
	if err != nil {
		return contracts.QueryExecutionResponse{}, fmt.Errorf("build query-runner request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return contracts.QueryExecutionResponse{}, fmt.Errorf("call query-runner: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		var failure map[string]any
		_ = json.NewDecoder(response.Body).Decode(&failure)
		if message, ok := failure["error"].(string); ok && message != "" {
			return contracts.QueryExecutionResponse{}, fmt.Errorf("query-runner rejected request: %s", message)
		}
		return contracts.QueryExecutionResponse{}, fmt.Errorf("query-runner returned status %d", response.StatusCode)
	}

	var result contracts.QueryExecutionResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return contracts.QueryExecutionResponse{}, fmt.Errorf("decode query-runner response: %w", err)
	}

	return result, nil
}

func queryRunnerURL() string {
	if value := strings.TrimSpace(os.Getenv("QUERY_RUNNER_URL")); value != "" {
		return value
	}
	return "http://query-runner:8093"
}

func memoryServiceURL() string {
	if value := strings.TrimSpace(os.Getenv("MEMORY_SERVICE_URL")); value != "" {
		return value
	}
	return "http://memory-service:8086"
}

func terminalBrokerURL() string {
	if value := strings.TrimSpace(os.Getenv("TERMINAL_BROKER_URL")); value != "" {
		return value
	}
	return "http://terminal-broker:8090"
}

func publicTerminalBrokerURL() string {
	if value := strings.TrimSpace(os.Getenv("PUBLIC_TERMINAL_BROKER_URL")); value != "" {
		return value
	}
	return terminalBrokerURL()
}
