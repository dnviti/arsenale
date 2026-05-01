package queryrunner

import (
	"context"
	"testing"

	"github.com/dnviti/arsenale/backend/pkg/contracts"
)

func TestValidateReadOnlySQLAcceptsSelect(t *testing.T) {
	if err := ValidateReadOnlySQL("SELECT current_database()"); err != nil {
		t.Fatalf("ValidateReadOnlySQL() returned error: %v", err)
	}
}

func TestValidateReadOnlySQLRejectsWrite(t *testing.T) {
	if err := ValidateReadOnlySQL("UPDATE users SET email = 'x'"); err == nil {
		t.Fatal("ValidateReadOnlySQL() unexpectedly accepted write query")
	}
}

func TestValidateReadOnlySQLRejectsMultipleStatements(t *testing.T) {
	if err := ValidateReadOnlySQL("SELECT 1; SELECT 2"); err == nil {
		t.Fatal("ValidateReadOnlySQL() unexpectedly accepted multiple statements")
	}
}

func TestResolvePoolRejectsUnsupportedTargetProtocol(t *testing.T) {
	_, _, err := resolvePool(context.Background(), nil, contracts.QueryExecutionRequest{
		SQL: "SELECT 1",
		Target: &contracts.DatabaseTarget{
			Protocol: "mysql",
			Host:     "db",
			Port:     3306,
			Username: "arsenale",
		},
	})
	if err == nil {
		t.Fatal("resolvePool() unexpectedly accepted unsupported protocol")
	}
}

func TestResolvePoolRejectsMissingTargetHost(t *testing.T) {
	_, _, err := resolvePool(context.Background(), nil, contracts.QueryExecutionRequest{
		SQL: "SELECT 1",
		Target: &contracts.DatabaseTarget{
			Protocol: "postgresql",
			Port:     5432,
			Username: "arsenale",
		},
	})
	if err == nil {
		t.Fatal("resolvePool() unexpectedly accepted empty target host")
	}
}
