package queryrunner

import "testing"

func TestValidateExplainSQLRejectsEmpty(t *testing.T) {
	if err := ValidateExplainSQL("   "); err == nil {
		t.Fatal("ValidateExplainSQL() unexpectedly accepted empty SQL")
	}
}

func TestValidateExplainSQLRejectsMultipleStatements(t *testing.T) {
	if err := ValidateExplainSQL("SELECT 1; SELECT 2"); err == nil {
		t.Fatal("ValidateExplainSQL() unexpectedly accepted multiple statements")
	}
}

func TestValidateExplainSQLAcceptsSingleStatement(t *testing.T) {
	if err := ValidateExplainSQL("select current_database()"); err != nil {
		t.Fatalf("ValidateExplainSQL() returned error: %v", err)
	}
}
