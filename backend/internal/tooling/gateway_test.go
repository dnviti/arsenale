package tooling

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestExecuteToolCallReadOnlyQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/query-runs:execute" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(contracts.QueryExecutionResponse{
			Columns:  []string{"value"},
			Rows:     []map[string]any{{"value": "ok"}},
			RowCount: 1,
		})
	}))
	defer server.Close()

	t.Setenv("QUERY_RUNNER_URL", server.URL)

	input, err := json.Marshal(contracts.QueryExecutionRequest{
		SQL:     "select 1 as value",
		MaxRows: 1,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	response, err := ExecuteToolCall(context.Background(), contracts.ToolCallExecuteRequest{
		Capability: "db.query.execute.readonly",
		Authz: contracts.AuthzRequest{
			Subject:  contracts.PrincipalRef{Type: contracts.PrincipalSystem, ID: "test"},
			Resource: contracts.ResourceRef{Type: "database", ID: "control-plane"},
		},
		Input: input,
	})
	if err != nil {
		t.Fatalf("ExecuteToolCall() error = %v", err)
	}
	if response.Decision.Effect != contracts.AuthzAllow {
		t.Fatalf("decision effect = %s, want allow", response.Decision.Effect)
	}
}

func TestExecuteToolCallTerminalGrant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/session-grants:issue" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":     "terminal-token",
			"expiresAt": "2030-01-01T00:00:00Z",
		})
	}))
	defer server.Close()

	t.Setenv("TERMINAL_BROKER_URL", server.URL)
	t.Setenv("PUBLIC_TERMINAL_BROKER_URL", "ws://terminal-broker-go:8090")

	input, err := json.Marshal(contracts.TerminalSessionGrant{
		Target: contracts.TerminalEndpoint{
			Host:     "terminal-target",
			Port:     2224,
			Username: "acceptance",
			Password: "acceptance",
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	response, err := ExecuteToolCall(context.Background(), contracts.ToolCallExecuteRequest{
		Capability: "connection.connect.ssh",
		Authz: contracts.AuthzRequest{
			Subject:  contracts.PrincipalRef{Type: contracts.PrincipalAgentRun, ID: "run-1"},
			Resource: contracts.ResourceRef{Type: "connection", ID: "terminal-target"},
			Context:  map[string]string{"approved": "true"},
		},
		Input: input,
	})
	if err != nil {
		t.Fatalf("ExecuteToolCall() error = %v", err)
	}
	if response.Decision.Effect != contracts.AuthzAllow {
		t.Fatalf("decision effect = %s, want allow", response.Decision.Effect)
	}

	output, ok := response.Output.(map[string]any)
	if !ok {
		t.Fatalf("response.Output type = %T, want map[string]any", response.Output)
	}
	if output["webSocketUrl"] != "ws://terminal-broker-go:8090/ws/terminal?token=terminal-token" {
		t.Fatalf("webSocketUrl = %v", output["webSocketUrl"])
	}
}

func TestExecuteToolCallSchemaRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/schema:fetch" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(contracts.SchemaInfo{
			Tables: []contracts.SchemaTable{
				{
					Name:   "users",
					Schema: "public",
					Columns: []contracts.SchemaColumn{
						{Name: "id", DataType: "uuid", Nullable: false, IsPrimaryKey: true},
					},
				},
			},
			Views:      []contracts.SchemaView{},
			Functions:  []contracts.SchemaRoutine{},
			Procedures: []contracts.SchemaRoutine{},
			Triggers:   []contracts.SchemaTrigger{},
			Sequences:  []contracts.SchemaSequence{},
			Packages:   []contracts.SchemaPackage{},
			Types:      []contracts.SchemaNamedType{},
		})
	}))
	defer server.Close()

	t.Setenv("QUERY_RUNNER_URL", server.URL)

	input, err := json.Marshal(contracts.SchemaFetchRequest{})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	response, err := ExecuteToolCall(context.Background(), contracts.ToolCallExecuteRequest{
		Capability: "db.schema.read",
		Authz: contracts.AuthzRequest{
			Subject:  contracts.PrincipalRef{Type: contracts.PrincipalSystem, ID: "test"},
			Resource: contracts.ResourceRef{Type: "database", ID: "control-plane"},
		},
		Input: input,
	})
	if err != nil {
		t.Fatalf("ExecuteToolCall() error = %v", err)
	}
	if response.Decision.Effect != contracts.AuthzAllow {
		t.Fatalf("decision effect = %s, want allow", response.Decision.Effect)
	}
}

func TestExecuteToolCallIntrospectionRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/introspection:run" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(contracts.QueryIntrospectionResponse{
			Supported: true,
			Data: map[string]any{
				"version": "PostgreSQL 16",
			},
		})
	}))
	defer server.Close()

	t.Setenv("QUERY_RUNNER_URL", server.URL)

	input, err := json.Marshal(contracts.QueryIntrospectionRequest{
		Type: "database_version",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	response, err := ExecuteToolCall(context.Background(), contracts.ToolCallExecuteRequest{
		Capability: "db.introspection.read",
		Authz: contracts.AuthzRequest{
			Subject:  contracts.PrincipalRef{Type: contracts.PrincipalSystem, ID: "test"},
			Resource: contracts.ResourceRef{Type: "database", ID: "control-plane"},
		},
		Input: input,
	})
	if err != nil {
		t.Fatalf("ExecuteToolCall() error = %v", err)
	}
	if response.Decision.Effect != contracts.AuthzAllow {
		t.Fatalf("decision effect = %s, want allow", response.Decision.Effect)
	}
}

func TestExecuteToolCallDeniedWithoutApproval(t *testing.T) {
	input, err := json.Marshal(map[string]string{"noop": "true"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	response, err := ExecuteToolCall(context.Background(), contracts.ToolCallExecuteRequest{
		Capability: "db.query.execute.write",
		Authz: contracts.AuthzRequest{
			Subject:  contracts.PrincipalRef{Type: contracts.PrincipalSystem, ID: "test"},
			Resource: contracts.ResourceRef{Type: "database", ID: "control-plane"},
		},
		Input: input,
	})
	if err != nil {
		t.Fatalf("ExecuteToolCall() error = %v", err)
	}
	if response.Decision.Effect != contracts.AuthzDeny {
		t.Fatalf("decision effect = %s, want deny", response.Decision.Effect)
	}
}
