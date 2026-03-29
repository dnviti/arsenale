package storage

import "testing"

func TestAugmentDatabaseURLAddsSSLRootCert(t *testing.T) {
	got := augmentDatabaseURL("postgresql://user:pass@postgres:5432/arsenale?sslmode=verify-full", "/certs/postgres/ca.pem")
	want := "postgresql://user:pass@postgres:5432/arsenale?sslmode=verify-full&sslrootcert=%2Fcerts%2Fpostgres%2Fca.pem"
	if got != want {
		t.Fatalf("augmentDatabaseURL() = %q, want %q", got, want)
	}
}

func TestAugmentDatabaseURLPreservesExistingSSLRootCert(t *testing.T) {
	raw := "postgresql://user:pass@postgres:5432/arsenale?sslmode=verify-full&sslrootcert=%2Fexisting%2Fca.pem"
	if got := augmentDatabaseURL(raw, "/certs/postgres/ca.pem"); got != raw {
		t.Fatalf("augmentDatabaseURL() = %q, want original %q", got, raw)
	}
}
