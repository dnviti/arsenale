package gateways

import (
	"database/sql"
	"errors"
	"testing"
)

func TestMapLoadGatewayErrorNotFound(t *testing.T) {
	err := mapLoadGatewayError(sql.ErrNoRows)

	var reqErr *requestError
	if !errors.As(err, &reqErr) {
		t.Fatalf("expected requestError, got %T", err)
	}
	if reqErr.status != 404 {
		t.Fatalf("expected status 404, got %d", reqErr.status)
	}
	if reqErr.message != "Gateway not found" {
		t.Fatalf("expected gateway not found message, got %q", reqErr.message)
	}
}

func TestMapLoadGatewayErrorPassthrough(t *testing.T) {
	original := errors.New("boom")

	if got := mapLoadGatewayError(original); !errors.Is(got, original) {
		t.Fatalf("expected original error passthrough, got %v", got)
	}
}
