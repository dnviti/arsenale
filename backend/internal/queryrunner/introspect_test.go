package queryrunner

import (
	"testing"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestValidateIntrospectionRequestRequiresType(t *testing.T) {
	if err := ValidateIntrospectionRequest(contracts.QueryIntrospectionRequest{}); err == nil {
		t.Fatal("ValidateIntrospectionRequest() unexpectedly accepted empty type")
	}
}

func TestValidateIntrospectionRequestRejectsUnknownType(t *testing.T) {
	if err := ValidateIntrospectionRequest(contracts.QueryIntrospectionRequest{Type: "unknown"}); err == nil {
		t.Fatal("ValidateIntrospectionRequest() unexpectedly accepted unknown type")
	}
}

func TestValidateIntrospectionRequestRequiresTargetForTableScopedTypes(t *testing.T) {
	if err := ValidateIntrospectionRequest(contracts.QueryIntrospectionRequest{Type: "indexes"}); err == nil {
		t.Fatal("ValidateIntrospectionRequest() unexpectedly accepted missing target")
	}
}

func TestValidateIntrospectionRequestAllowsDatabaseVersionWithoutTarget(t *testing.T) {
	if err := ValidateIntrospectionRequest(contracts.QueryIntrospectionRequest{Type: "database_version"}); err != nil {
		t.Fatalf("ValidateIntrospectionRequest() returned error: %v", err)
	}
}
