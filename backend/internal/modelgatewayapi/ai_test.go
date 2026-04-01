package modelgatewayapi

import (
	"testing"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestParsePlanningResponseReadsJSONTables(t *testing.T) {
	t.Parallel()

	raw := `{"tables":[{"name":"orders","schema":"public","reason":"contains order totals"},{"name":"customers","reason":"joins buyer details"}]}`
	items := parsePlanningResponse(raw)
	if len(items) != 2 {
		t.Fatalf("expected 2 table requests, got %d", len(items))
	}
	if items[1].Schema != "public" {
		t.Fatalf("expected missing schema to default to public, got %q", items[1].Schema)
	}
}

func TestFindUnapprovedTableReferenceRejectsExtraTable(t *testing.T) {
	t.Parallel()

	approved := []contracts.SchemaTable{
		{Name: "orders", Schema: "public"},
	}
	all := []contracts.SchemaTable{
		{Name: "orders", Schema: "public"},
		{Name: "users", Schema: "public"},
	}

	violation := findUnapprovedTableReference("select * from orders join users on users.id = orders.user_id", approved, all)
	if violation != "users" {
		t.Fatalf("expected users violation, got %q", violation)
	}
}

func TestParseFirstTurnResponseFiltersUnsupportedRequests(t *testing.T) {
	t.Parallel()

	raw := `{"needs_data":true,"data_requests":[{"type":"indexes","target":"orders","reason":"inspect indexes"},{"type":"database_version","target":"db","reason":"unsupported here"}]}`
	result := parseFirstTurnResponse(raw)
	if !result.NeedsData {
		t.Fatal("expected needsData=true")
	}
	if len(result.DataRequests) != 1 {
		t.Fatalf("expected 1 supported data request, got %d", len(result.DataRequests))
	}
	if result.DataRequests[0].Type != "indexes" {
		t.Fatalf("unexpected request type %q", result.DataRequests[0].Type)
	}
}
