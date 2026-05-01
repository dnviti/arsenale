package tenants

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/dnviti/arsenale/backend/internal/runtimefeatures"
)

func TestNormalizeTenantForRuntimeHidesRecordingFieldsWhenDisabled(t *testing.T) {
	retention := 90
	item := normalizeTenantForRuntime(tenantResponse{
		RecordingEnabled:       true,
		RecordingRetentionDays: &retention,
	}, false)

	if item.RecordingEnabled {
		t.Fatalf("expected recording flag to be forced off when recordings are disabled")
	}
	if item.RecordingRetentionDays != nil {
		t.Fatalf("expected retention to be hidden when recordings are disabled")
	}
}

func TestUpdateTenantRejectsRecordingFieldsWhenRecordingsDisabled(t *testing.T) {
	service := Service{Features: runtimefeatures.Manifest{RecordingsEnabled: false}}
	payload := map[string]json.RawMessage{
		"recordingEnabled": json.RawMessage(`true`),
	}

	_, err := service.UpdateTenant(context.Background(), "tenant-1", payload)
	if err == nil {
		t.Fatalf("expected recordings-disabled tenant update to fail")
	}

	var reqErr *requestError
	if !errors.As(err, &reqErr) {
		t.Fatalf("expected requestError, got %T", err)
	}
	if reqErr.status != 404 {
		t.Fatalf("expected 404, got %d", reqErr.status)
	}
}
