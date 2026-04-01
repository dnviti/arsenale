package gateways

import (
	"database/sql"
	"reflect"
	"testing"
	"time"
)

type stubRowScanner struct {
	values []any
	err    error
}

func (s stubRowScanner) Scan(dest ...any) error {
	if s.err != nil {
		return s.err
	}
	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(s.values[i]))
	}
	return nil
}

func TestScanManagedGatewayInstance(t *testing.T) {
	lastHealthCheck := time.Date(2026, 3, 31, 4, 5, 6, 0, time.UTC)
	createdAt := time.Date(2026, 3, 31, 4, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)

	item, err := scanManagedGatewayInstance(stubRowScanner{
		values: []any{
			"instance-1",
			"gateway-1",
			"container-1",
			"gateway-1-a",
			"10.0.0.10",
			3022,
			sql.NullInt32{Int32: 8443, Valid: true},
			"RUNNING",
			"docker",
			sql.NullString{String: "healthy", Valid: true},
			sql.NullTime{Time: lastHealthCheck, Valid: true},
			sql.NullString{String: "none", Valid: true},
			2,
			sql.NullString{String: "proxy.internal", Valid: true},
			sql.NullInt32{Int32: 9443, Valid: true},
			createdAt,
			updatedAt,
		},
	})
	if err != nil {
		t.Fatalf("scanManagedGatewayInstance returned error: %v", err)
	}

	if item.ID != "instance-1" || item.GatewayID != "gateway-1" {
		t.Fatalf("unexpected identity fields: %#v", item)
	}
	if item.APIPort == nil || *item.APIPort != 8443 {
		t.Fatalf("unexpected apiPort: %#v", item.APIPort)
	}
	if item.HealthStatus == nil || *item.HealthStatus != "healthy" {
		t.Fatalf("unexpected healthStatus: %#v", item.HealthStatus)
	}
	if item.LastHealthCheck == nil || !item.LastHealthCheck.Equal(lastHealthCheck) {
		t.Fatalf("unexpected lastHealthCheck: %#v", item.LastHealthCheck)
	}
	if item.ErrorMessage == nil || *item.ErrorMessage != "none" {
		t.Fatalf("unexpected errorMessage: %#v", item.ErrorMessage)
	}
	if item.TunnelProxyHost == nil || *item.TunnelProxyHost != "proxy.internal" {
		t.Fatalf("unexpected tunnelProxyHost: %#v", item.TunnelProxyHost)
	}
	if item.TunnelProxyPort == nil || *item.TunnelProxyPort != 9443 {
		t.Fatalf("unexpected tunnelProxyPort: %#v", item.TunnelProxyPort)
	}
}

func TestScanManagedGatewayInstanceNullables(t *testing.T) {
	createdAt := time.Date(2026, 3, 31, 5, 0, 0, 0, time.UTC)

	item, err := scanManagedGatewayInstance(stubRowScanner{
		values: []any{
			"instance-2",
			"gateway-2",
			"container-2",
			"gateway-2-a",
			"10.0.0.11",
			4022,
			sql.NullInt32{},
			"PROVISIONING",
			"kubernetes",
			sql.NullString{},
			sql.NullTime{},
			sql.NullString{},
			0,
			sql.NullString{},
			sql.NullInt32{},
			createdAt,
			createdAt,
		},
	})
	if err != nil {
		t.Fatalf("scanManagedGatewayInstance returned error: %v", err)
	}

	if item.APIPort != nil || item.HealthStatus != nil || item.LastHealthCheck != nil || item.ErrorMessage != nil || item.TunnelProxyHost != nil || item.TunnelProxyPort != nil {
		t.Fatalf("expected nil nullable fields, got %#v", item)
	}
}
