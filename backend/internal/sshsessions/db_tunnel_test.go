package sshsessions

import "testing"

func TestBuildDBTunnelConnectionString(t *testing.T) {
	postgres := "postgresql"
	if got := buildDBTunnelConnectionString(&postgres, "127.0.0.1", 15432, "db-user", "s3cret!", "warehouse"); got == nil || *got != "postgresql://db-user:s3cret%21@127.0.0.1:15432/warehouse" {
		t.Fatalf("unexpected postgres connection string: %#v", got)
	}

	mssql := "mssql"
	if got := buildDBTunnelConnectionString(&mssql, "127.0.0.1", 11433, "sa", "Password1!", "inventory"); got == nil || *got != "Server=127.0.0.1,11433;Database=inventory;User Id=sa;Password=Password1!;" {
		t.Fatalf("unexpected mssql connection string: %#v", got)
	}

	if got := buildDBTunnelConnectionString(nil, "127.0.0.1", 16379, "", "", ""); got == nil || *got != "127.0.0.1:16379" {
		t.Fatalf("unexpected default connection string: %#v", got)
	}
}
